package trace

import (
	"context"
	"fmt"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Provider interface {
	Shutdown(context.Context) error
	Tracer() Tracer
}

type Tracer = oteltrace.Tracer

type traceProvider struct {
	traceProvider      oteltrace.TracerProvider
	providerShutdownFn func(context.Context) error

	cfg config.OpenTelemetry
}

// NewProvider creates a new trace provider with the given configuration
// The trace provider is responsible for creating spans and sending them to the exporter
// it also register the trace provider as a global trace provider, and connects the	trace provider to the exporter
func NewProvider(ctx context.Context, cfg config.OpenTelemetry) (Provider, error) {
	if !cfg.Enabled {
		return &traceProvider{
			traceProvider:      oteltrace.NewNoopTracerProvider(),
			providerShutdownFn: nil,
			cfg:                cfg,
		}, nil
	}

	// set the config defaults
	cfg.SetDefaults()

	// create the resource
	resource, err := resourceFactory(ctx, cfg.ResourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// create the exporter - here's where connecting to the collector happens
	exporter, err := exporterFactory(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// create the span processor - this is what will send the spans to the exporter.
	spanProcesor := spanProcessorFactory(cfg.SpanProcessorType, exporter)

	// Create the trace provider
	// The trace provider will use the resource and exporter created previously
	// to generate spans and send them to the exporter
	// The trace provider must be registered as a global trace provider
	// so that any other package can use it

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
		sdktrace.WithSpanProcessor(spanProcesor),
	)

	// set global otel trace provider
	otel.SetTracerProvider(tracerProvider)

	// set the global otel context propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &traceProvider{
		traceProvider:      tracerProvider,
		providerShutdownFn: tracerProvider.Shutdown,
		cfg:                cfg,
	}, nil
}

func (tp *traceProvider) Shutdown(ctx context.Context) error {
	if tp.providerShutdownFn == nil {
		return nil
	}

	return tp.providerShutdownFn(ctx)
}

func (tp *traceProvider) Tracer() Tracer {
	return tp.traceProvider.Tracer(tp.cfg.ResourceName)
}
