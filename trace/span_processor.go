package trace

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type spanProcessorCreator func(sdktrace.SpanExporter) sdktrace.SpanProcessor

var spanProcessorCreators = map[string]spanProcessorCreator{
	"simple": newSimpleSpanProcessor,
	"batch":  newBatchSpanProcessor,
}

func spanProcessorFactory(spanProcessorType string, exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	if creator, exists := spanProcessorCreators[spanProcessorType]; exists {
		return creator(exporter)
	}

	// Default to BatchSpanProcessor if the spanProcessorType does not exist
	return newBatchSpanProcessor(exporter)
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}

func newBatchSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewBatchSpanProcessor(exporter)
}
