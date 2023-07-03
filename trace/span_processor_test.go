package trace

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type testExporter struct {
	spans    []sdktrace.ReadOnlySpan
	shutdown bool
}

func (t *testExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	t.spans = append(t.spans, spans...)
	return nil
}

func (t *testExporter) Shutdown(ctx context.Context) error {
	t.shutdown = true
	select {
	case <-ctx.Done():
		// Ensure context deadline tests receive the expected error.
		return ctx.Err()
	default:
		return nil
	}
}

var _ sdktrace.SpanExporter = (*testExporter)(nil)

func Test_NewSimpleSpanProcessor(t *testing.T) {
	t.Parallel()

	// Create a new exporter
	te := testExporter{}

	// Create a new span processor
	processor := newSimpleSpanProcessor(&te)
	assert.NotNil(t, processor)

	tp, wantTraceID, spanID := prepareTestProvider(t)

	// Register the span processor with the trace provider
	tp.RegisterSpanProcessor(processor)

	// Create a new span
	spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
	span := spans[0]
	span.End()
	assert.Equal(t, 1, len(te.spans))

	gotTraceID := te.spans[0].SpanContext().TraceID()

	assert.Equal(t, wantTraceID, gotTraceID)
}

func Test_NewBatchSpanProcessor(t *testing.T) {
	t.Parallel()

	// Create a new exporter
	te := testExporter{}

	// Create a new span processor
	processor := newBatchSpanProcessor(&te)
	assert.NotNil(t, processor)

	tp, wantTraceID, spanID := prepareTestProvider(t)

	// Register the span processor with the trace provider
	tp.RegisterSpanProcessor(processor)

	// Create a new span
	t.Run("single trace", func(t *testing.T) {
		spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
		span := spans[0]
		span.End()
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 1, len(te.spans))
		gotTraceID := te.spans[0].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})
	t.Run("multiple traces in a single batch", func(t *testing.T) {
		spans := startTestSpan(t, tp, spanID, wantTraceID, 5)
		for _, span := range spans {
			span.End()
		}
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 6, len(te.spans))   // 5 spans + 1 from the prior test
		gotTraceID := te.spans[5].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})

}

func prepareTestProvider(t *testing.T) (*sdktrace.TracerProvider, trace.TraceID, trace.SpanID) {
	t.Helper()

	// Create a new trace provider
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

	// create span and trace ids
	wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
	assert.Nil(t, err)

	spanID, err := trace.SpanIDFromHex("0102040810203040")
	assert.Nil(t, err)

	return tp, wantTraceID, spanID
}

func startTestSpan(t *testing.T, tp trace.TracerProvider, sid trace.SpanID, tid trace.TraceID, numberOfTraces int) []trace.Span {
	t.Helper()

	tr := tp.Tracer("SpanProcessor")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: 0x1,
	})

	response := make([]trace.Span, numberOfTraces)
	for i := 0; i < numberOfTraces; i++ {
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		_, span := tr.Start(ctx, "OnEnd"+strconv.Itoa(i))
		response[i] = span
	}

	return response
}
