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
	assert.NotNil(t, provider.Recorder())
	assert.False(t, provider.Recorder().Enabled())
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
	assert.NotNil(t, provider.Recorder())
	assert.False(t, provider.Recorder().Enabled())
}

func TestRecorder_Record_NoopWhenNil(t *testing.T) {
	var recorder *Recorder
	// Should not panic
	recorder.Record(context.Background(), Attributes{}, Latency{})
}

func TestRecorder_Record_NoopWhenDisabled(t *testing.T) {
	recorder := newNoopRecorder()
	// Should not panic
	recorder.Record(context.Background(), Attributes{}, Latency{})
	assert.False(t, recorder.Enabled())
}
