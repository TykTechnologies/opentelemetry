package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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
