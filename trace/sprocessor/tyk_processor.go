package sprocessor

import (
	"context"
	mathRand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// RedisAnalyticsHandler will record analytics data to a redis back end
// as defined in the Config object
type AnalyticsHandler struct {
	cfg              *config.OpenTelemetry
	recordsChan      chan *sdktrace.ReadOnlySpan
	workerBufferSize uint64
	shouldStop       uint32
	poolWg           sync.WaitGroup

	poolSize int
	mu       sync.Mutex
	exporter sdktrace.SpanExporter
}

var _ sdktrace.SpanProcessor = (*AnalyticsHandler)(nil)

const recordsBufferSize uint64 = 1000

func NewAnalyticsHandler(exporter sdktrace.SpanExporter, cfg *config.OpenTelemetry) *AnalyticsHandler {
	r := &AnalyticsHandler{
		exporter: exporter,
		cfg:      cfg,
	}

	r.poolSize = runtime.NumCPU()

	r.workerBufferSize = recordsBufferSize / uint64(r.poolSize)

	r.Start()

	return r
}

// Start initialize the records channel and spawn the record workers
func (r *AnalyticsHandler) Start() {
	r.recordsChan = make(chan *sdktrace.ReadOnlySpan, recordsBufferSize)
	atomic.SwapUint32(&r.shouldStop, 0)
	for i := 0; i < r.poolSize; i++ {
		r.poolWg.Add(1)
		go r.recordWorker()
	}
}

// Stop stops the analytics processing
func (r *AnalyticsHandler) Shutdown(ctx context.Context) error {
	// flag to stop sending records into channel
	atomic.SwapUint32(&r.shouldStop, 1)

	// close channel to stop workers
	r.mu.Lock()
	close(r.recordsChan)
	r.mu.Unlock()

	// wait for all workers to be done
	r.poolWg.Wait()
	return nil
}

// Flush will stop the analytics processing and empty the analytics buffer and then re-init the workers again
func (r *AnalyticsHandler) ForceFlush(ctx context.Context) error {
	r.Shutdown(ctx)

	r.Start()
	return nil
}

func (r *AnalyticsHandler) RecordHit(span sdktrace.ReadOnlySpan) error {
	// check if we should stop sending records 1st
	if atomic.LoadUint32(&r.shouldStop) > 0 {
		return nil
	}

	// just send record to channel consumed by pool of workers
	// leave all data crunching and Redis I/O work for pool workers
	r.mu.Lock()
	r.recordsChan <- &span
	r.mu.Unlock()

	return nil
}

func (r *AnalyticsHandler) recordWorker() {
	defer r.poolWg.Done()

	// this is buffer to send one pipelined command to redis
	// use r.recordsBufferSize as cap to reduce slice re-allocations
	recordsBuffer := make([]sdktrace.ReadOnlySpan, 0, r.workerBufferSize)
	mathRand.Seed(time.Now().Unix())

	// read records from channel and process
	lastSentTs := time.Now()
	for {
		readyToSend := false

		flushTimer := time.NewTimer(200 * time.Millisecond)

		select {
		case record, ok := <-r.recordsChan:
			if !flushTimer.Stop() {
				// if the timer has been stopped then read from the channel to avoid leak
				<-flushTimer.C
			}

			// check if channel was closed and it is time to exit from worker
			if !ok {
				// send what is left in buffer
				_ = r.exporter.ExportSpans(context.Background(), recordsBuffer)
				return
			}

			// we have new record - prepare it and add to buffer

			recordsBuffer = append(recordsBuffer, *record)

			// identify that buffer is ready to be sent
			readyToSend = uint64(len(recordsBuffer)) == r.workerBufferSize
		case <-flushTimer.C:
			// nothing was received for that period of time
			// anyways send whatever we have, don't hold data too long in buffer
			readyToSend = true
		}

		// send data to Redis and reset buffer
		if len(recordsBuffer) > 0 && (readyToSend || time.Since(lastSentTs) >= time.Duration(1000*time.Millisecond)) {
			_ = r.exporter.ExportSpans(context.Background(), recordsBuffer)
			recordsBuffer = recordsBuffer[:0]
			lastSentTs = time.Now()
		}
	}
}

func (r *AnalyticsHandler) OnEnd(s sdktrace.ReadOnlySpan) {
	_ = r.RecordHit(s)
}

func (r *AnalyticsHandler) OnStart(ctx context.Context, s sdktrace.ReadWriteSpan) {
	// do nothing
}
