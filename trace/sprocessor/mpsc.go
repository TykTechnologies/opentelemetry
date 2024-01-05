package sprocessor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Node represents an element in the queue
type Node struct {
	value sdktrace.ReadOnlySpan
	next  unsafe.Pointer // *Node
}

// MPSCQueue represents a lock-free multi-producer, single-consumer queue
type MPSCQueue struct {
	head   unsafe.Pointer // *Node
	tail   unsafe.Pointer // *Node
	length int32          // actual queue length
}

// NewMPSCQueue creates a new MPSCQueue
func NewMPSCQueue() *MPSCQueue {
	node := unsafe.Pointer(&Node{})
	return &MPSCQueue{head: node, tail: node}
}

// Enqueue adds an element to the queue
func (q *MPSCQueue) Enqueue(value sdktrace.ReadOnlySpan) {
	newNode := &Node{value: value}
	var tail, next unsafe.Pointer
	atomic.AddInt32(&q.length, 1)

	for {
		tail = atomic.LoadPointer(&q.tail)
		next = atomic.LoadPointer(&((*Node)(tail)).next)

		if tail == atomic.LoadPointer(&q.tail) {
			if next == nil && atomic.CompareAndSwapPointer(&((*Node)(tail)).next, next, unsafe.Pointer(newNode)) {
				atomic.CompareAndSwapPointer(&q.tail, tail, unsafe.Pointer(newNode))
				return
			} else if next != nil {
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			}
		}
	}
}

// Dequeue removes and returns an element from the queue
func (q *MPSCQueue) Dequeue() (sdktrace.ReadOnlySpan, bool) {
	var head, tail, next unsafe.Pointer

	for {
		head = atomic.LoadPointer(&q.head)
		tail = atomic.LoadPointer(&q.tail)
		next = atomic.LoadPointer(&((*Node)(head)).next)

		if head == atomic.LoadPointer(&q.head) {
			if head == tail {
				if next == nil {
					return nil, false // Queue is empty
				}

				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			} else {
				val := (*Node)(next).value
				if atomic.CompareAndSwapPointer(&q.head, head, next) {
					return val, true
				}
			}
		}
	}
}

// Length returns actual queue length
func (q *MPSCQueue) Length() int {
	return int(atomic.LoadInt32(&q.length))
}

// Length returns actual queue length
func (q *MPSCQueue) IsEmpty() bool {
	return int(atomic.LoadInt32(&q.length)) == 0
}

type MPSCSpanProcessor struct {
	queue        *MPSCQueue
	spansNeeded  int32
	waitTime     int
	exportSignal chan struct{}
	mu           sync.Mutex

	exporter sdktrace.SpanExporter
}

var _ sdktrace.SpanProcessor = (*MPSCSpanProcessor)(nil)

func NewMPSCSpanProcessor(exporter sdktrace.SpanExporter, maxSize, waitTime int) *MPSCSpanProcessor {
	queue := &MPSCSpanProcessor{
		queue:        NewMPSCQueue(),
		spansNeeded:  int32(maxSize),
		exportSignal: make(chan struct{}, 2048),
		waitTime:     waitTime,
		exporter:     exporter,
	}

	go queue.ExporterThread(context.Background())

	return queue
}

func (bsp *MPSCSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	bsp.queue.Enqueue(s)

	if bsp.queue.Length() >= int(atomic.LoadInt32(&bsp.spansNeeded)) {
		select {
		case bsp.exportSignal <- struct{}{}:
		default:
		}
	}
}

func (bsp *MPSCSpanProcessor) ExporterThread(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-bsp.exportSignal:
			bsp.exportBatch(false)
		}
	}
}

func (bsp *MPSCSpanProcessor) exportBatch(force bool) {
	buffer := make([]trace.ReadOnlySpan, 0, bsp.spansNeeded)
	for !bsp.queue.IsEmpty() && len(buffer) < int(bsp.spansNeeded) {
		if span, ok := bsp.queue.Dequeue(); ok {
			buffer = append(buffer, span)
		}
	}

	if (len(buffer) >= int(bsp.spansNeeded)) || force {
		bsp.export(buffer)
		buffer = buffer[:0] // emptying buffer
	}

	if bsp.queue.IsEmpty() {
		atomic.StoreInt32(&bsp.spansNeeded, int32(bsp.waitTime-len(buffer)))
	}
}

func (bsp *MPSCSpanProcessor) export(spans []trace.ReadOnlySpan) {
	err := bsp.exporter.ExportSpans(context.Background(), spans)
	if err != nil {
		fmt.Println("err exporting spans:", err)
	}
}

func (bsp *MPSCSpanProcessor) Shutdown(ctx context.Context) error {
	bsp.ForceFlush(ctx)
	return bsp.exporter.Shutdown(ctx)
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *MPSCSpanProcessor) ForceFlush(ctx context.Context) error {
	if !bsp.queue.IsEmpty() {
		buffer := make([]trace.ReadOnlySpan, 0, bsp.spansNeeded)
		for !bsp.queue.IsEmpty() && len(buffer) < int(bsp.spansNeeded) {
			if span, ok := bsp.queue.Dequeue(); ok {
				buffer = append(buffer, span)
			}
		}

		return bsp.exporter.ExportSpans(ctx, buffer)
	}

	return nil
}

func (bps *MPSCSpanProcessor) OnStart(ctx context.Context, s sdktrace.ReadWriteSpan) {
	// do nothing
}
