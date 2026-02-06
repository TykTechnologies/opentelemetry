package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// NoopProvider indicates a noop provider type.
	NoopProvider = "noop"
	// OtelProvider indicates an OpenTelemetry provider type.
	OtelProvider = "otel"
)

// Provider is the interface that wraps the basic methods of a meter provider.
// If misconfigured or disabled, the provider will return a noop meter and
// instruments that silently do nothing.
type Provider interface {
	// Shutdown executes the underlying exporter shutdown function.
	Shutdown(context.Context) error
	// Meter returns a meter with pre-configured name. It's used to create metrics.
	Meter() otelmetric.Meter
	// Type returns the type of the provider, it can be either "noop" or "otel".
	Type() string
	// Enabled returns whether the provider is enabled and recording metrics.
	Enabled() bool
	// NewCounter creates a new counter with the given name, description, and unit.
	// Returns a nil-safe Counter that can be used even if the provider is disabled.
	NewCounter(name, description, unit string) (*Counter, error)
	// NewHistogram creates a new histogram with the given name, description, unit, and bucket boundaries.
	// If buckets is nil or empty, DefaultLatencyBuckets will be used.
	// Returns a nil-safe Histogram that can be used even if the provider is disabled.
	NewHistogram(name, description, unit string, buckets []float64) (*Histogram, error)
}

type meterProvider struct {
	meterProvider      otelmetric.MeterProvider
	providerShutdownFn func(context.Context) error

	cfg    *config.OpenTelemetry
	logger Logger

	ctx          context.Context
	providerType string
	enabled      bool

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
//
//	counter, _ := provider.NewCounter("my.counter", "A counter", "1")
//	counter.Add(ctx, 1, attribute.String("key", "value"))
func NewProvider(opts ...Option) (Provider, error) {
	provider := &meterProvider{
		meterProvider:      otel.GetMeterProvider(),
		providerShutdownFn: nil,
		logger:             &noopLogger{},
		cfg:                &config.OpenTelemetry{},
		ctx:                context.Background(),
		providerType:       NoopProvider,
		enabled:            false,
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
	provider.enabled = true

	// Set global otel meter provider.
	otel.SetMeterProvider(meterProv)

	// Set the global otel error handler.
	otel.SetErrorHandler(&errHandler{
		logger: provider.logger,
	})

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

func (mp *meterProvider) Meter() otelmetric.Meter {
	return mp.meterProvider.Meter(mp.cfg.ResourceName)
}

func (mp *meterProvider) Type() string {
	return mp.providerType
}

func (mp *meterProvider) Enabled() bool {
	return mp.enabled
}

func (mp *meterProvider) NewCounter(name, description, unit string) (*Counter, error) {
	if !mp.enabled {
		return &Counter{enabled: false}, nil
	}

	counter, err := mp.Meter().Int64Counter(
		name,
		otelmetric.WithDescription(description),
		otelmetric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}

	return &Counter{
		counter: counter,
		enabled: true,
	}, nil
}

func (mp *meterProvider) NewHistogram(name, description, unit string, buckets []float64) (*Histogram, error) {
	if !mp.enabled {
		return &Histogram{enabled: false}, nil
	}

	if len(buckets) == 0 {
		buckets = DefaultLatencyBuckets
	}

	histogram, err := mp.Meter().Float64Histogram(
		name,
		otelmetric.WithDescription(description),
		otelmetric.WithUnit(unit),
		otelmetric.WithExplicitBucketBoundaries(buckets...),
	)
	if err != nil {
		return nil, err
	}

	return &Histogram{
		histogram: histogram,
		enabled:   true,
	}, nil
}
