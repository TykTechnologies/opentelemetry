package trace

import (
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/TykTechnologies/opentelemetry/trace/sprocessor"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func spanProcessorFactory(spanProcessorType string, exporter sdktrace.SpanExporter, cfg *config.OpenTelemetry) sdktrace.SpanProcessor {
	switch spanProcessorType {
	case "simple":
		return newSimpleSpanProcessor(exporter)
	case "tyk":
		return sprocessor.NewAnalyticsHandler(exporter, cfg)
	case "mpsc":
		return sprocessor.NewMPSCSpanProcessor(exporter, cfg.BatchSize, cfg.BatchTimeout)
	default:
		// Default to BatchSpanProcessor
		return newBatchSpanProcessor(exporter, cfg)
	}
}

func newSimpleSpanProcessor(exporter sdktrace.SpanExporter) sdktrace.SpanProcessor {
	return sdktrace.NewSimpleSpanProcessor(exporter)
}

func newBatchSpanProcessor(exporter sdktrace.SpanExporter, cfg *config.OpenTelemetry) sdktrace.SpanProcessor {
	opts := []sdktrace.BatchSpanProcessorOption{}
	if cfg.BatchSize > 0 {
		opts = append(opts, sdktrace.WithMaxExportBatchSize(cfg.BatchSize))
	}
	if cfg.BatchQueueSize > 0 {
		opts = append(opts, sdktrace.WithMaxQueueSize(cfg.BatchQueueSize))
	}
	if cfg.BatchTimeout > 0 {
		opts = append(opts, sdktrace.WithBatchTimeout(time.Duration(cfg.BatchTimeout)*time.Millisecond))
	}
	if cfg.BatchExportTimeout > 0 {
		opts = append(opts, sdktrace.WithExportTimeout(time.Duration(cfg.BatchExportTimeout)*time.Millisecond))
	}

	return sdktrace.NewBatchSpanProcessor(exporter, opts...)
}
