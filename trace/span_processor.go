package trace

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func spanProcessorFactory(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return newSimpleSpanProcessor(exporter)
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}
