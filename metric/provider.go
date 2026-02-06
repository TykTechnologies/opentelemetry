package metric

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/otel"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	// NoopProvider indicates a noop provider type.
	NoopProvider = "noop"
	// OtelProvider indicates an OpenTelemetry provider type.
	OtelProvider = "otel"
)

// ExportStats contains statistics about metric exports.
type ExportStats struct {
	// TotalExports is the total number of export attempts.
	TotalExports int64
	// SuccessfulExports is the number of successful exports.
	SuccessfulExports int64
	// FailedExports is the number of failed exports.
	FailedExports int64
	// LastExportTime is the time of the last export attempt.
	LastExportTime time.Time
	// LastSuccessTime is the time of the last successful export.
	LastSuccessTime time.Time
}

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
	// NewGauge creates a new gauge with the given name, description, and unit.
	// Use gauges for values that can go up and down, like pool sizes or temperatures.
	// Returns a nil-safe Gauge that can be used even if the provider is disabled.
	NewGauge(name, description, unit string) (*Gauge, error)
	// NewUpDownCounter creates a new up-down counter with the given name, description, and unit.
	// Use up-down counters for values that can increase or decrease, like active connections.
	// Returns a nil-safe UpDownCounter that can be used even if the provider is disabled.
	NewUpDownCounter(name, description, unit string) (*UpDownCounter, error)

	// Healthy returns whether the exporter is healthy (last export succeeded).
	Healthy() bool
	// LastExportError returns the last export error, if any.
	LastExportError() error
	// GetExportStats returns statistics about metric exports.
	GetExportStats() ExportStats
	// IsMetricDisabled returns whether a metric is disabled by configuration.
	IsMetricDisabled(name string) bool
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

	// Health and stats tracking
	healthy          atomic.Bool
	lastExportError  atomic.Value // stores error
	totalExports     atomic.Int64
	successExports   atomic.Int64
	failedExports    atomic.Int64
	lastExportTime   atomic.Value // stores time.Time
	lastSuccessTime  atomic.Value // stores time.Time
	disabledMetrics  map[string]struct{}
	disabledMetricsMu sync.RWMutex
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
		disabledMetrics:    make(map[string]struct{}),
	}

	// Apply the given options.
	for _, opt := range opts {
		opt.apply(provider)
	}

	// Set the config defaults - this does not override the config values.
	provider.cfg.SetDefaults()

	// Build disabled metrics map for O(1) lookups.
	for _, name := range provider.cfg.Metrics.DisabledMetrics {
		provider.disabledMetrics[name] = struct{}{}
	}

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

	// Create the exporter with retry configuration.
	exporter, err := exporterFactory(provider.ctx, provider.cfg)
	if err != nil {
		provider.logger.Error("failed to create metric exporter", err)
		return provider, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Wrap exporter with stats tracking.
	wrappedExporter := &statsExporter{
		exporter: exporter,
		provider: provider,
	}

	// Create the periodic reader with the configured export interval.
	exportInterval := time.Duration(provider.cfg.Metrics.ExportInterval) * time.Second
	readerOpts := []sdkmetric.PeriodicReaderOption{
		sdkmetric.WithInterval(exportInterval),
	}

	reader := sdkmetric.NewPeriodicReader(wrappedExporter, readerOpts...)

	// Build meter provider options.
	meterProvOpts := []sdkmetric.Option{
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(reader),
	}


	// Create the meter provider.
	meterProv := sdkmetric.NewMeterProvider(meterProvOpts...)

	// Set the local meter provider.
	provider.meterProvider = meterProv
	provider.providerShutdownFn = meterProv.Shutdown
	provider.providerType = OtelProvider
	provider.enabled = true
	provider.healthy.Store(true)

	// Set global otel meter provider.
	otel.SetMeterProvider(meterProv)

	// Set the global otel error handler.
	otel.SetErrorHandler(&errHandler{
		logger: provider.logger,
	})

	provider.logger.Info("Meter provider initialized successfully")

	return provider, nil
}

// deltaTemporalitySelector returns delta temporality for all instruments.
func deltaTemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

// statsExporter wraps an exporter to track export statistics.
type statsExporter struct {
	exporter sdkmetric.Exporter
	provider *meterProvider
}

func (e *statsExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	e.provider.totalExports.Add(1)
	e.provider.lastExportTime.Store(time.Now())

	err := e.exporter.Export(ctx, rm)
	if err != nil {
		e.provider.failedExports.Add(1)
		e.provider.lastExportError.Store(err)
		e.provider.healthy.Store(false)
		e.provider.logger.Error("metric export failed", err)
		return err
	}

	e.provider.successExports.Add(1)
	e.provider.lastSuccessTime.Store(time.Now())
	e.provider.healthy.Store(true)
	e.provider.lastExportError.Store(error(nil))
	return nil
}

func (e *statsExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return e.exporter.Temporality(kind)
}

func (e *statsExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return e.exporter.Aggregation(kind)
}

func (e *statsExporter) Shutdown(ctx context.Context) error {
	return e.exporter.Shutdown(ctx)
}

func (e *statsExporter) ForceFlush(ctx context.Context) error {
	return e.exporter.ForceFlush(ctx)
}

func (mp *meterProvider) Shutdown(ctx context.Context) error {
	if mp.providerShutdownFn == nil {
		return nil
	}

	// Use ShutdownTimeout if configured, otherwise fall back to ConnectionTimeout.
	timeout := mp.cfg.Metrics.ShutdownTimeout
	if timeout == 0 {
		timeout = mp.cfg.ConnectionTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
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

func (mp *meterProvider) Healthy() bool {
	if !mp.enabled {
		return true // Noop provider is always "healthy"
	}
	return mp.healthy.Load()
}

func (mp *meterProvider) LastExportError() error {
	if !mp.enabled {
		return nil
	}
	if v := mp.lastExportError.Load(); v != nil {
		if err, ok := v.(error); ok {
			return err
		}
	}
	return nil
}

func (mp *meterProvider) GetExportStats() ExportStats {
	stats := ExportStats{
		TotalExports:      mp.totalExports.Load(),
		SuccessfulExports: mp.successExports.Load(),
		FailedExports:     mp.failedExports.Load(),
	}

	if v := mp.lastExportTime.Load(); v != nil {
		if t, ok := v.(time.Time); ok {
			stats.LastExportTime = t
		}
	}

	if v := mp.lastSuccessTime.Load(); v != nil {
		if t, ok := v.(time.Time); ok {
			stats.LastSuccessTime = t
		}
	}

	return stats
}

func (mp *meterProvider) IsMetricDisabled(name string) bool {
	mp.disabledMetricsMu.RLock()
	defer mp.disabledMetricsMu.RUnlock()
	_, disabled := mp.disabledMetrics[name]
	return disabled
}

func (mp *meterProvider) NewCounter(name, description, unit string) (*Counter, error) {
	if !mp.enabled || mp.IsMetricDisabled(name) {
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
	if !mp.enabled || mp.IsMetricDisabled(name) {
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

func (mp *meterProvider) NewGauge(name, description, unit string) (*Gauge, error) {
	if !mp.enabled || mp.IsMetricDisabled(name) {
		return &Gauge{enabled: false}, nil
	}

	gauge, err := mp.Meter().Float64Gauge(
		name,
		otelmetric.WithDescription(description),
		otelmetric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}

	return &Gauge{
		gauge:   gauge,
		enabled: true,
	}, nil
}

func (mp *meterProvider) NewUpDownCounter(name, description, unit string) (*UpDownCounter, error) {
	if !mp.enabled || mp.IsMetricDisabled(name) {
		return &UpDownCounter{enabled: false}, nil
	}

	counter, err := mp.Meter().Int64UpDownCounter(
		name,
		otelmetric.WithDescription(description),
		otelmetric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}

	return &UpDownCounter{
		counter: counter,
		enabled: true,
	}, nil
}
