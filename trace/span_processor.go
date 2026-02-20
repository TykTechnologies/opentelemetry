package trace

import (
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/TykTechnologies/opentelemetry/config"
)

func spanProcessorFactory(spanProcessorType string, cfg config.SpanBatchConfig, exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	switch spanProcessorType {
	case "simple":
		return newSimpleSpanProcessor(exporter)
	default:
		// Default to BatchSpanProcessor
		return newBatchSpanProcessor(cfg, exporter)
	}
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}

func newBatchSpanProcessor(cfg config.SpanBatchConfig, exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	var opts []sdktrace.BatchSpanProcessorOption

	if cfg.MaxQueueSize > 0 {
		opts = append(opts, sdktrace.WithMaxQueueSize(cfg.MaxQueueSize))
	}

	if cfg.MaxExportBatchSize > 0 {
		opts = append(opts, sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatchSize))
	}

	if cfg.BatchTimeout > 0 {
		opts = append(opts, sdktrace.WithBatchTimeout(time.Duration(cfg.BatchTimeout)*time.Second))
	}

	return sdktrace.NewBatchSpanProcessor(exporter, opts...)
}
