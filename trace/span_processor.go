package trace

import (
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func spanProcessorFactory(spanProcessorType string, exporter sdktrace.SpanExporter, cfg *config.OpenTelemetry) sdktrace.SpanProcessor {
	switch spanProcessorType {
	case "simple":
		return newSimpleSpanProcessor(exporter)
	default:
		// Default to BatchSpanProcessor
		return newBatchSpanProcessor(exporter, cfg)
	}
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}

func newBatchSpanProcessor(exporter sdktrace.SpanExporter, cfg *config.OpenTelemetry) sdktrace.SpanProcessor {
	opts := []sdktrace.BatchSpanProcessorOption{
		sdktrace.WithMaxExportBatchSize(cfg.BatchSize),
		sdktrace.WithMaxQueueSize(cfg.BatchQueueSize),
		sdktrace.WithBatchTimeout(time.Duration(cfg.BatchTimeout) * time.Millisecond),
		sdktrace.WithExportTimeout(time.Duration(cfg.BatchExportTimeout) * time.Millisecond),
	}

	return sdktrace.NewBatchSpanProcessor(exporter, opts...)
}
