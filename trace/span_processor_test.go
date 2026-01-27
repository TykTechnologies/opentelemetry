package trace

import (
	"context"
	"strconv"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
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

	// Create a new tracer provider
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

	// create span and trace ids
	wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
	assert.Nil(t, err)

	spanID, err := trace.SpanIDFromHex("0102040810203040")
	assert.Nil(t, err)

	// Register the span processor with the tracer provider
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
		processor := newBatchSpanProcessor(config.SpanBatchConfig{}, &te)
		assert.NotNil(t, processor)
		// Create a new tracer provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		// create span and trace ids
		wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
		assert.Nil(t, err)

		spanID, err := trace.SpanIDFromHex("0102040810203040")
		assert.Nil(t, err)

		// Register the span processor with the tracer provider
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
		processor := newBatchSpanProcessor(config.SpanBatchConfig{}, &te)
		assert.NotNil(t, processor)
		// Create a new tracer provider
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

		// Register the span processor with the tracer provider
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
		processor := newBatchSpanProcessor(config.SpanBatchConfig{}, &te)
		assert.NotNil(t, processor)
		// Create a new tracer provider
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

		// Register the span processor with the tracer provider
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

	t.Run("with custom batch config", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor with custom config
		cfg := config.SpanBatchConfig{
			MaxQueueSize:       8192,
			MaxExportBatchSize: 1024,
			BatchTimeout:       3,
		}
		processor := newBatchSpanProcessor(cfg, &te)
		assert.NotNil(t, processor)

		// Create a new tracer provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		// create span and trace ids
		wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
		assert.Nil(t, err)

		spanID, err := trace.SpanIDFromHex("0102040810203040")
		assert.Nil(t, err)

		// Register the span processor with the tracer provider
		tp.RegisterSpanProcessor(processor)

		spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
		span := spans[0]
		span.End()
		tp.ForceFlush(context.Background()) // forcing flush to avoid waiting for the batch timeout
		assert.Equal(t, 1, len(te.spans))
		gotTraceID := te.spans[0].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})

	t.Run("with partial batch config", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor with only MaxQueueSize set
		cfg := config.SpanBatchConfig{
			MaxQueueSize: 4096,
		}
		processor := newBatchSpanProcessor(cfg, &te)
		assert.NotNil(t, processor)

		// Create a new tracer provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		// create span and trace ids
		wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
		assert.Nil(t, err)

		spanID, err := trace.SpanIDFromHex("0102040810203040")
		assert.Nil(t, err)

		// Register the span processor with the tracer provider
		tp.RegisterSpanProcessor(processor)

		spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
		span := spans[0]
		span.End()
		tp.ForceFlush(context.Background())
		assert.Equal(t, 1, len(te.spans))
		gotTraceID := te.spans[0].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})

	t.Run("with zero values", func(t *testing.T) {
		// Create a new exporter
		te := testExporter{}

		// Create a new span processor with zero values (should use SDK defaults)
		cfg := config.SpanBatchConfig{
			MaxQueueSize:       0,
			MaxExportBatchSize: 0,
		}
		processor := newBatchSpanProcessor(cfg, &te)
		assert.NotNil(t, processor)

		// Create a new tracer provider
		tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

		// create span and trace ids
		wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
		assert.Nil(t, err)

		spanID, err := trace.SpanIDFromHex("0102040810203040")
		assert.Nil(t, err)

		// Register the span processor with the tracer provider
		tp.RegisterSpanProcessor(processor)

		spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
		span := spans[0]
		span.End()
		tp.ForceFlush(context.Background())
		assert.Equal(t, 1, len(te.spans))
		gotTraceID := te.spans[0].SpanContext().TraceID()
		assert.Equal(t, wantTraceID, gotTraceID)
	})
}

func TestSpanProcessorFactory_SimpleIgnoresBatchConfig(t *testing.T) {
	t.Parallel()

	// Create a new exporter
	te := testExporter{}

	// Create batch config that should be ignored
	cfg := config.SpanBatchConfig{
		MaxQueueSize:       8192,
		MaxExportBatchSize: 1024,
		BatchTimeout:       3,
	}

	// Create a simple processor - batch config should be ignored
	processor := spanProcessorFactory("simple", cfg, &te)
	assert.NotNil(t, processor)

	// Create a new tracer provider
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))

	// create span and trace ids
	wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
	assert.Nil(t, err)

	spanID, err := trace.SpanIDFromHex("0102040810203040")
	assert.Nil(t, err)

	// Register the span processor with the tracer provider
	tp.RegisterSpanProcessor(processor)

	// Create a new span
	spans := startTestSpan(t, tp, spanID, wantTraceID, 1)
	span := spans[0]
	span.End()
	// Simple processor exports immediately, no need to flush
	assert.Equal(t, 1, len(te.spans))
	gotTraceID := te.spans[0].SpanContext().TraceID()
	assert.Equal(t, wantTraceID, gotTraceID)
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
