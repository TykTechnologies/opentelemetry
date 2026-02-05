package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// NoopProvider indicates a noop provider type.
	NoopProvider = "noop"
	// OtelProvider indicates an OpenTelemetry provider type.
	OtelProvider = "otel"
)

// Provider is the interface that wraps the basic methods of a meter provider.
// If misconfigured or disabled, the provider will return a noop meter.
type Provider interface {
	// Shutdown executes the underlying exporter shutdown function.
	Shutdown(context.Context) error
	// Meter returns a meter with pre-configured name. It's used to create metrics.
	Meter() metric.Meter
	// Type returns the type of the provider, it can be either "noop" or "otel".
	Type() string
	// Recorder returns the common RED metrics recorder.
	Recorder() *Recorder
}

type meterProvider struct {
	meterProvider      metric.MeterProvider
	providerShutdownFn func(context.Context) error
	recorder           *Recorder

	cfg    *config.OpenTelemetry
	logger Logger

	ctx          context.Context
	providerType string

	resources resourceConfig
}

// NewProvider creates a new meter provider with the given options.
// The meter provider is responsible for creating metrics and sending them to the exporter.
//
// Example:
//
//	provider, err := metric.NewProvider(
//		metric.WithContext(context.Background()),
//		metric.WithConfig(&config.OpenTelemetry{
//			Enabled:  true,
//			Exporter: "grpc",
//			Endpoint: "localhost:4317",
//			Metrics: config.MetricsConfig{
//				Enabled:        ptr(true),
//				ExportInterval: 60,
//			},
//		}),
//		metric.WithLogger(logrus.New().WithField("component", "tyk")),
//	)
//	if err != nil {
//		panic(err)
//	}
func NewProvider(opts ...Option) (Provider, error) {
	provider := &meterProvider{
		meterProvider:      otel.GetMeterProvider(),
		providerShutdownFn: nil,
		logger:             &noopLogger{},
		cfg:                &config.OpenTelemetry{},
		ctx:                context.Background(),
		providerType:       NoopProvider,
		recorder:           nil,
	}

	// Apply the given options.
	for _, opt := range opts {
		opt.apply(provider)
	}

	// Set the config defaults - this does not override the config values.
	provider.cfg.SetDefaults()

	// Check if metrics are enabled.
	metricsEnabled := provider.cfg.Metrics.Enabled != nil && *provider.cfg.Metrics.Enabled

	// If the provider is not enabled or metrics are not enabled, return a noop provider.
	if !provider.cfg.Enabled || !metricsEnabled {
		provider.recorder = newNoopRecorder()
		return provider, nil
	}

	// Create the resource.
	resource, err := resourceFactory(provider.ctx, provider.cfg.ResourceName, provider.resources)
	if err != nil {
		provider.logger.Error("failed to create resource", err)
		return provider, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create the exporter - here's where connecting to the collector happens.
	exporter, err := exporterFactory(provider.ctx, provider.cfg)
	if err != nil {
		provider.logger.Error("failed to create metric exporter", err)
		return provider, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create the periodic reader with the configured export interval.
	exportInterval := time.Duration(provider.cfg.Metrics.ExportInterval) * time.Second
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(exportInterval))

	// Create the meter provider.
	meterProv := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(reader),
	)

	// Set the local meter provider.
	provider.meterProvider = meterProv
	provider.providerShutdownFn = meterProv.Shutdown
	provider.providerType = OtelProvider

	// Set global otel meter provider.
	otel.SetMeterProvider(meterProv)

	// Set the global otel error handler.
	otel.SetErrorHandler(&errHandler{
		logger: provider.logger,
	})

	// Create the recorder with the meter.
	meter := meterProv.Meter(provider.cfg.ResourceName)
	recorder, err := newRecorder(meter)
	if err != nil {
		provider.logger.Error("failed to create metrics recorder", err)
		return provider, fmt.Errorf("failed to create metrics recorder: %w", err)
	}
	provider.recorder = recorder

	provider.logger.Info("Meter provider initialized successfully")

	return provider, nil
}

func (mp *meterProvider) Shutdown(ctx context.Context) error {
	if mp.providerShutdownFn == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(mp.cfg.ConnectionTimeout)*time.Second)
	defer cancel()

	return mp.providerShutdownFn(ctx)
}

func (mp *meterProvider) Meter() metric.Meter {
	return mp.meterProvider.Meter(mp.cfg.ResourceName)
}

func (mp *meterProvider) Type() string {
	return mp.providerType
}

func (mp *meterProvider) Recorder() *Recorder {
	return mp.recorder
}
