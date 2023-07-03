package trace

import (
	"github.com/TykTechnologies/opentelemetry/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func spanProcessorFactory(cfg config.OpenTelemetry, exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	switch cfg.SpanProcessorType {
	case "simple":
		return newSimpleSpanProcessor(exporter)
	default:
		// Default to BatchSpanProcessor
		return newBatchSpanProcessor(exporter)
	}
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}

func newBatchSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewBatchSpanProcessor(exporter)
}
