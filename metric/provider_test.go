package metric

import (
	"context"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/stretchr/testify/assert"
)

func TestNewProvider_Disabled(t *testing.T) {
	cfg := &config.OpenTelemetry{
		Enabled: false,
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
	metricsEnabled := false
	cfg := &config.OpenTelemetry{
		Enabled:  true,
		Exporter: "grpc",
		Endpoint: "localhost:4317",
		Metrics: config.MetricsConfig{
			Enabled: &metricsEnabled,
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
	cfg := &config.OpenTelemetry{
		Enabled: false,
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
	cfg := &config.OpenTelemetry{
		Enabled: false,
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
