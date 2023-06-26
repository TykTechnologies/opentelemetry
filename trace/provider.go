package trace

import (
	"context"
	"fmt"
	"opentelemetry/config"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Provider interface {
	Shutdown(context.Context) error
	Tracer() *sdktrace.TracerProvider
}

type traceProvider struct {
	traceProvider *sdktrace.TracerProvider
}

// NewProvider creates a new trace provider with the given configuration
// The trace provider is responsible for creating spans and sending them to the exporter
// it also register the trace provider as a global trace provider, and connects the	trace provider to the exporter
func NewProvider(ctx context.Context, cfg config.OpenTelemetry) (Provider, error) {
	resource, err := resourceFactory(ctx, cfg.ResourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	exporter, err := exporterFactory(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create the trace provider
	// The trace provider will use the resource and exporter created previously
	// to generate spans and send them to the exporter
	// The trace provider must be registered as a global trace provider
	// so that any other package can use it

	spanProcesor := spanProcessorFactory(exporter)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
		sdktrace.WithSpanProcessor(spanProcesor),
	)

	otel.SetTracerProvider(tracerProvider)

	return &traceProvider{
		traceProvider: tracerProvider,
	}, nil
}

func (tp *traceProvider) Shutdown(ctx context.Context) error {
	return tp.traceProvider.Shutdown(ctx)
}

func (tp *traceProvider) Tracer() *sdktrace.TracerProvider {
	return tp.traceProvider
}
