package sprocessor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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
	length int32          // tamaño actual de la cola
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
	atomic.AddInt32(&q.length, -1)

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
	buffer       []trace.ReadOnlySpan
	spansNeeded  int32
	waitTime     int
	exportSignal chan struct{}
	mu           sync.Mutex

	exporter sdktrace.SpanExporter
}

var _ sdktrace.SpanProcessor = (*MPSCSpanProcessor)(nil)

func NewMPSCSpanProcessor(exporter sdktrace.SpanExporter, maxSize, waitTime int) *MPSCSpanProcessor {
	return &MPSCSpanProcessor{
		queue:        &MPSCQueue{}, // Inicializa tu MPSCQueue aquí
		buffer:       make([]trace.ReadOnlySpan, 0, maxSize),
		spansNeeded:  int32(maxSize),
		exportSignal: make(chan struct{}, 1),
		waitTime:     waitTime,
		exporter:     exporter,
	}
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
			bsp.exportBatch()
		case <-time.After(time.Second): // Reemplaza con tu WAIT_TIME
			bsp.exportBatch()
		}
	}
}

func (bsp *MPSCSpanProcessor) exportBatch() {
	bsp.mu.Lock()
	defer bsp.mu.Unlock()

	for !bsp.queue.IsEmpty() && len(bsp.buffer) < int(bsp.spansNeeded) {
		if span, ok := bsp.queue.Dequeue(); ok {
			bsp.buffer = append(bsp.buffer, span)
		}
	}

	if len(bsp.buffer) >= int(bsp.spansNeeded) {
		bsp.export(bsp.buffer)
		bsp.buffer = bsp.buffer[:0] // Vaciar el buffer
	}

	if bsp.queue.IsEmpty() {
		atomic.StoreInt32(&bsp.spansNeeded, int32(bsp.waitTime-len(bsp.buffer)))
	}
}

func (bsp *MPSCSpanProcessor) export(spans []trace.ReadOnlySpan) {
	err := bsp.exporter.ExportSpans(context.Background(), bsp.buffer)
	if err != nil {
		fmt.Println("err exporting spans:", err)
	}
}

func (bsp *MPSCSpanProcessor) Shutdown(ctx context.Context) error {
	return bsp.exporter.Shutdown(ctx)
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *MPSCSpanProcessor) ForceFlush(ctx context.Context) error {
	batch := bsp.buffer
	if len(batch) > 0 {
		return bsp.exporter.ExportSpans(ctx, batch)
	}
	return nil
}

func (bps *MPSCSpanProcessor) OnStart(ctx context.Context, s sdktrace.ReadWriteSpan) {
	// do nothing
}
