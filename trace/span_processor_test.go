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

	// Create a new trace provider
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

	// create span and trace ids
	wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
	assert.Nil(t, err)

	spanID, err := trace.SpanIDFromHex("0102040810203040")
	assert.Nil(t, err)

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

	// Create a new span
	t.Run("single trace", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor
		processor := newBatchSpanProcessor(&te)
		assert.NotNil(t, processor)
		// Create a new trace provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		// create span and trace ids
		wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
		assert.Nil(t, err)

		spanID, err := trace.SpanIDFromHex("0102040810203040")
		assert.Nil(t, err)

		// Register the span processor with the trace provider
		tp.RegisterSpanProcessor(processor)

		spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
		span := spans[0]
		span.End()
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 1, len(te.spans))
		gotTraceID := te.spans[0].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})
	t.Run("multiple traces with only one span", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor
		processor := newBatchSpanProcessor(&te)
		assert.NotNil(t, processor)
		// Create a new trace provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		traceIDs := make([]trace.TraceID, 5)
		spanIDs := make([]trace.SpanID, 5)
		for i := 0; i < 5; i++ {
			traceID, err := trace.TraceIDFromHex("0102030405060708010204081020304" + strconv.Itoa(i))
			assert.Nil(t, err)
			traceIDs[i] = traceID

			spanID, err := trace.SpanIDFromHex("010204081020304" + strconv.Itoa(i))
			assert.Nil(t, err)
			spanIDs[i] = spanID
		}

		// Register the span processor with the trace provider
		tp.RegisterSpanProcessor(processor)

		for i := 0; i < 5; i++ {
			spans := startTestSpan(t, tp, spanIDs[i], traceIDs[i], 1)
			for _, span := range spans {
				span.End()
			}
		}
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 5, len(te.spans))
		for i := 0; i < 5; i++ {
			gotTraceID := te.spans[i].SpanContext().TraceID()
			assert.Equal(t, traceIDs[i], gotTraceID)
		}
	})
	t.Run("multiple traces with multiple spans", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor
		processor := newBatchSpanProcessor(&te)
		assert.NotNil(t, processor)
		// Create a new trace provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		traceIDs := make([]trace.TraceID, 5)
		spanIDs := make([]trace.SpanID, 5)
		for i := 0; i < 5; i++ {
			traceID, err := trace.TraceIDFromHex("0102030405060708010204081020304" + strconv.Itoa(i))
			assert.Nil(t, err)
			traceIDs[i] = traceID

			spanID, err := trace.SpanIDFromHex("010204081020304" + strconv.Itoa(i))
			assert.Nil(t, err)
			spanIDs[i] = spanID
		}

		// Register the span processor with the trace provider
		tp.RegisterSpanProcessor(processor)

		for i := 0; i < 5; i++ {
			spans := startTestSpan(t, tp, spanIDs[i], traceIDs[i], 5)
			for _, span := range spans {
				span.End()
			}
		}
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 25, len(te.spans))  // 5 traces with 5 spans each = 25
		for i := 0; i < 25; i++ {
			gotTraceID := te.spans[i].SpanContext().TraceID()
			assert.Equal(t, traceIDs[i/5], gotTraceID) // 5 spans per trace, that's why we divide by 5
		}
	})
}

func startTestSpan(t *testing.T,
	tp trace.TracerProvider,
	sid trace.SpanID,
	tid trace.TraceID,
	numberOfTraces int) []trace.Span {
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
