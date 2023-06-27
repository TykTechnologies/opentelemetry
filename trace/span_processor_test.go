package trace

import (
	"context"
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

	// Create a new trace provider
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	// Create a new exporter
	te := testExporter{}

	// create span and trace ids
	wantTraceID, err := trace.TraceIDFromHex("01020304050607080102040810203040")
	assert.Nil(t, err)

	spanID, err := trace.SpanIDFromHex("0102040810203040")
	assert.Nil(t, err)

	// Create a new span processor
	processor := newSimpleSpanProcessor(&te)
	assert.NotNil(t, processor)

	// Register the span processor with the trace provider
	tp.RegisterSpanProcessor(processor)

	// Create a new span
	startTestSpan(t, tp, spanID, wantTraceID).End()
	assert.Equal(t, 1, len(te.spans))

	gotTraceID := te.spans[0].SpanContext().TraceID()

	assert.Equal(t, wantTraceID, gotTraceID)
}

func startTestSpan(t *testing.T, tp trace.TracerProvider, sid trace.SpanID, tid trace.TraceID) trace.Span {
	t.Helper()

	tr := tp.Tracer("SimpleSpanProcessor")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: 0x1,
	})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	_, span := tr.Start(ctx, "OnEnd")

	return span
}
