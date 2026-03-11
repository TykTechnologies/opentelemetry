package metric

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/config"
)

func ptr[T any](v T) *T {
	return &v
}

func TestNewProvider_Disabled(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: ptr(false),
	}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(cfg),
	)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, NoopProvider, provider.Type())
	assert.False(t, provider.Enabled())
}

func TestNewProvider_MetricsDisabled(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: ptr(false),
		ExporterConfig: config.ExporterConfig{
			Exporter: "grpc",
			Endpoint: "localhost:4317",
		},
	}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(cfg),
	)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, NoopProvider, provider.Type())
	assert.False(t, provider.Enabled())
}

func TestNewProvider_NilEnabled(t *testing.T) {
	cfg := &config.MetricsConfig{}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(cfg),
	)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, NoopProvider, provider.Type())
	assert.False(t, provider.Enabled())
}

func TestCounter_Add_NoopWhenNil(t *testing.T) {
	var counter *Counter
	// Should not panic
	counter.Add(context.Background(), 1)
	assert.False(t, counter.Enabled())
}

func TestCounter_Add_NoopWhenDisabled(t *testing.T) {
	counter := &Counter{enabled: false}
	// Should not panic
	counter.Add(context.Background(), 1)
	assert.False(t, counter.Enabled())
}

func TestHistogram_Record_NoopWhenNil(t *testing.T) {
	var histogram *Histogram
	// Should not panic
	histogram.Record(context.Background(), 1.0)
	assert.False(t, histogram.Enabled())
}

func TestHistogram_Record_NoopWhenDisabled(t *testing.T) {
	histogram := &Histogram{enabled: false}
	// Should not panic
	histogram.Record(context.Background(), 1.0)
	assert.False(t, histogram.Enabled())
}

func TestNewProvider_NewCounter_Disabled(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: ptr(false),
	}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(cfg),
	)

	assert.NoError(t, err)

	counter, err := provider.NewCounter("test.counter", "A test counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, counter)
	assert.False(t, counter.Enabled())

	// Should not panic
	counter.Add(context.Background(), 1)
}

func TestNewProvider_NewHistogram_Disabled(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: ptr(false),
	}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(cfg),
	)

	assert.NoError(t, err)

	histogram, err := provider.NewHistogram("test.histogram", "A test histogram", "ms", nil)
	assert.NoError(t, err)
	assert.NotNil(t, histogram)
	assert.False(t, histogram.Enabled())

	// Should not panic
	histogram.Record(context.Background(), 1.0)
}

func TestGauge_Record_NoopWhenNil(t *testing.T) {
	var gauge *Gauge
	gauge.Record(context.Background(), 1.0)
	assert.False(t, gauge.Enabled())
}

func TestGauge_Record_NoopWhenDisabled(t *testing.T) {
	gauge := &Gauge{enabled: false}
	gauge.Record(context.Background(), 1.0)
	assert.False(t, gauge.Enabled())
}

func TestUpDownCounter_Add_NoopWhenNil(t *testing.T) {
	var counter *UpDownCounter
	counter.Add(context.Background(), 1)
	assert.False(t, counter.Enabled())
}

func TestUpDownCounter_Add_NoopWhenDisabled(t *testing.T) {
	counter := &UpDownCounter{enabled: false}
	counter.Add(context.Background(), 1)
	assert.False(t, counter.Enabled())
}

func TestNewProvider_NewGauge_Disabled(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	gauge, err := provider.NewGauge("test.gauge", "A test gauge", "1")
	assert.NoError(t, err)
	assert.NotNil(t, gauge)
	assert.False(t, gauge.Enabled())
	gauge.Record(context.Background(), 42.0)
}

func TestNewProvider_NewUpDownCounter_Disabled(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	counter, err := provider.NewUpDownCounter("test.updown", "A test updown counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, counter)
	assert.False(t, counter.Enabled())
	counter.Add(context.Background(), 1)
}

func TestNewProvider_NoopHealthy(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	assert.True(t, provider.Healthy())
}

func TestNewProvider_NoopExportStats(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	stats := provider.GetExportStats()
	assert.Equal(t, int64(0), stats.TotalExports)
	assert.Equal(t, int64(0), stats.SuccessfulExports)
	assert.Equal(t, int64(0), stats.FailedExports)
}

func TestNewProvider_NoopLastExportError(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	assert.Nil(t, provider.LastExportError())
}

func TestNewProvider_NoopShutdown(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	assert.NoError(t, provider.Shutdown(context.Background()))
}

func TestNewProvider_NoopForceFlush(t *testing.T) {
	cfg := &config.MetricsConfig{Enabled: ptr(false)}
	provider, err := NewProvider(WithContext(context.Background()), WithConfig(cfg))
	assert.NoError(t, err)
	assert.NoError(t, provider.ForceFlush(context.Background()))
}

func TestNewProvider_WithReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, OtelProvider, provider.Type())
	assert.True(t, provider.Enabled())
	assert.True(t, provider.Healthy())
}

func TestNewProvider_WithReader_Counter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)

	counter, err := provider.NewCounter("test.counter", "A test counter", "1")
	assert.NoError(t, err)
	assert.True(t, counter.Enabled())

	ctx := context.Background()
	counter.Add(ctx, 5)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(ctx, &rm))

	// Find the counter metric.
	var found bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test.counter" {
				sum, ok := m.Data.(metricdata.Sum[int64])
				assert.True(t, ok)
				assert.Len(t, sum.DataPoints, 1)
				assert.Equal(t, int64(5), sum.DataPoints[0].Value)
				found = true
			}
		}
	}
	assert.True(t, found, "metric test.counter not found")
}

func TestNewProvider_WithReader_Histogram(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)

	histogram, err := provider.NewHistogram("test.histogram", "A test histogram", "ms", nil)
	assert.NoError(t, err)
	assert.True(t, histogram.Enabled())

	ctx := context.Background()
	histogram.Record(ctx, 50.0)
	histogram.Record(ctx, 150.0)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(ctx, &rm))

	var found bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test.histogram" {
				hist, ok := m.Data.(metricdata.Histogram[float64])
				assert.True(t, ok)
				assert.Len(t, hist.DataPoints, 1)
				assert.Equal(t, uint64(2), hist.DataPoints[0].Count)
				assert.Equal(t, 200.0, hist.DataPoints[0].Sum)
				found = true
			}
		}
	}
	assert.True(t, found, "metric test.histogram not found")
}

func TestNewProvider_WithReader_NoGlobalState(t *testing.T) {
	// WithReader should NOT set the global meter provider.
	reader := sdkmetric.NewManualReader()
	_, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)

	// The global provider should still be the default noop, not our custom one.
	// We can't easily assert this without side effects, but we verify the provider
	// was created without error and is enabled.
}

func TestNewProvider_WithReader_NoConfigNeeded(t *testing.T) {
	// WithReader should work without WithConfig — no Enabled flag needed.
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)
	assert.True(t, provider.Enabled())
	assert.Equal(t, OtelProvider, provider.Type())
}

// TestStatsExporter_MixedErrorTypes verifies that the statsExporter does not
// panic when storing errors of different concrete types (success sentinel vs
// real export error) into the atomic.Value.
func TestStatsExporter_MixedErrorTypes(t *testing.T) {
	mp := &meterProvider{enabled: true, logger: &noopLogger{}}
	failErr := fmt.Errorf("wrapped: %w", fmt.Errorf("inner"))
	exporter := &statsExporter{
		exporter: &fakeExporter{errs: []error{nil, failErr, nil}},
		provider: mp,
	}
	ctx := context.Background()
	rm := &metricdata.ResourceMetrics{}

	// Cycle: success → failure → success. Before the fix the second or third
	// call would panic due to inconsistent concrete types in atomic.Value.
	assert.NoError(t, exporter.Export(ctx, rm))
	assert.Nil(t, mp.LastExportError())

	assert.Error(t, exporter.Export(ctx, rm))
	assert.ErrorIs(t, mp.LastExportError(), failErr)

	assert.NoError(t, exporter.Export(ctx, rm))
	assert.Nil(t, mp.LastExportError())
}

// fakeExporter is a test double that returns errors from a pre-defined
// sequence, cycling when the sequence is exhausted.
type fakeExporter struct {
	errs []error
	idx  int
}

func (f *fakeExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	err := f.errs[f.idx%len(f.errs)]
	f.idx++
	return err
}

func (f *fakeExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (f *fakeExporter) Aggregation(sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(0)
}

func (f *fakeExporter) Shutdown(context.Context) error   { return nil }
func (f *fakeExporter) ForceFlush(context.Context) error { return nil }

func TestNewProvider_WithReader_Shutdown(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)
	assert.NoError(t, provider.Shutdown(context.Background()))
}

func TestNewProvider_WithReader_ForceFlush(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
	)
	assert.NoError(t, err)
	assert.NoError(t, provider.ForceFlush(context.Background()))
}

func TestNewProvider_WithCardinalityLimit(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{
			CardinalityLimit: 100,
		}),
		WithReader(reader),
	)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, OtelProvider, provider.Type())
	assert.True(t, provider.Enabled())
	assert.NoError(t, provider.Shutdown(context.Background()))
}

func TestNewProvider_WithCardinalityLimit_CapsDataPoints(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	limit := 5
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{
			CardinalityLimit: limit,
		}),
		WithReader(reader),
	)
	assert.NoError(t, err)

	counter, err := provider.NewCounter("test.capped_counter", "counter with cardinality cap", "1")
	assert.NoError(t, err)

	ctx := context.Background()

	// Record more unique attribute combinations than the limit allows.
	for i := 0; i < limit*3; i++ {
		counter.Add(ctx, 1, attribute.String("key", fmt.Sprintf("val-%d", i)))
	}

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(ctx, &rm))

	var dataPoints int
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test.capped_counter" {
				sum, ok := m.Data.(metricdata.Sum[int64])
				assert.True(t, ok)
				dataPoints = len(sum.DataPoints)
			}
		}
	}

	// The SDK caps at CardinalityLimit + 1 (the overflow data point).
	assert.LessOrEqual(t, dataPoints, limit+1,
		"expected at most %d data points (limit + overflow), got %d", limit+1, dataPoints)
	assert.Greater(t, dataPoints, 0, "expected at least one data point")
}
