package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// spanQueueNode represents a node in the queue.
type spanQueueNode struct {
	span sdktrace.ReadOnlySpan
	next *spanQueueNode
}

// spanQueue is a basic lock-free queue for spans.
type spanQueue struct {
	head    atomic.Pointer[spanQueueNode]
	tail    atomic.Pointer[spanQueueNode]
	padding [128]byte // Padding to avoid false sharing between head and tail
}

// newSpanQueue creates a new spanQueue.
func newSpanQueue() *spanQueue {
	q := &spanQueue{}
	node := &spanQueueNode{} // Dummy node
	q.head.Store(node)
	q.tail.Store(node)
	return q
}

// enqueue adds a span to the queue.
func (q *spanQueue) enqueue(span sdktrace.ReadOnlySpan) {
	node := &spanQueueNode{span: span}
	for {
		tail := q.tail.Load()
		tailNext := (*unsafe.Pointer)(unsafe.Pointer(tail.next))
		if atomic.CompareAndSwapPointer(tailNext, nil, unsafe.Pointer(node)) {
			q.tail.CompareAndSwap(tail, node)
			return
		}
		q.tail.CompareAndSwap(tail, tail.next)
	}
}

// dequeue removes and returns the next span from the queue.
func (q *spanQueue) dequeue() (sdktrace.ReadOnlySpan, bool) {
	for {
		head := q.head.Load()
		next := head.next
		if next == nil {
			return nil, false // Queue is empty
		}
		if q.head.CompareAndSwap(head, next) {
			return next.span, true
		}
	}
}

// BatchSpanProcessor is an implementation of the SpanProcessor that batches spans for async processing.
type BatchSpanProcessor struct {
	queue      *spanQueue
	maxBatch   int
	exporter   sdktrace.SpanExporter
	shutdownCh chan struct{}
	wg         sync.WaitGroup
}

// NewBatchSpanProcessor creates a new BatchSpanProcessor.
func NewMPSCSpanProcessor(exporter sdktrace.SpanExporter, maxBatchSize int) *BatchSpanProcessor {
	bsp := &BatchSpanProcessor{
		queue:      newSpanQueue(),
		maxBatch:   maxBatchSize,
		exporter:   exporter,
		shutdownCh: make(chan struct{}),
	}

	bsp.wg.Add(1)
	go bsp.processQueue()

	return bsp
}

// OnStart is called when a span is started.
func (bsp *BatchSpanProcessor) OnStart(_ context.Context, _ sdktrace.ReadWriteSpan) {
	// Do nothing on start.
}

// OnEnd is called when a span is finished.
func (bsp *BatchSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	bsp.queue.enqueue(s)
}

// Shutdown is called when the SDK shuts down.
func (bsp *BatchSpanProcessor) Shutdown(ctx context.Context) error {
	close(bsp.shutdownCh)
	bsp.wg.Wait()

	return bsp.exporter.Shutdown(ctx)
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *BatchSpanProcessor) ForceFlush(ctx context.Context) error {
	batch := bsp.collectBatch()
	if len(batch) > 0 {
		return bsp.exporter.ExportSpans(ctx, batch)
	}
	return nil
}

// processQueue processes the span queue in batches.
func (bsp *BatchSpanProcessor) processQueue() {
	defer bsp.wg.Done()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]sdktrace.ReadOnlySpan, 0, bsp.maxBatch)

	for {
		select {
		case <-bsp.shutdownCh:
			return
		case <-ticker.C:
			if len(batch) > 0 {
				bsp.exportBatch(batch)
				batch = make([]sdktrace.ReadOnlySpan, 0, bsp.maxBatch)
			}
		default:
			if span, ok := bsp.queue.dequeue(); ok {
				batch = append(batch, span)
				if len(batch) >= bsp.maxBatch {
					bsp.exportBatch(batch)
					batch = make([]sdktrace.ReadOnlySpan, 0, bsp.maxBatch)
				}
			}
		}
	}
}

// collectBatch collects a batch of spans from the queue.
func (bsp *BatchSpanProcessor) collectBatch() []sdktrace.ReadOnlySpan {
	var batch []sdktrace.ReadOnlySpan
	for {
		if span, ok := bsp.queue.dequeue(); ok {
			batch = append(batch, span)
		} else {
			break
		}
	}
	return batch
}

// exportBatch exports a batch of spans.
func (bsp *BatchSpanProcessor) exportBatch(batch []sdktrace.ReadOnlySpan) {
	_ = bsp.exporter.ExportSpans(context.Background(), batch)
}
