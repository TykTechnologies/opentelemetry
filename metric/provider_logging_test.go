package metric

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/config"
)

// invalidExporterConfig is enabled but names an exporter that does not exist,
// so NewProvider fails at exporter creation without touching the network.
func invalidExporterConfig() *config.MetricsConfig {
	return &config.MetricsConfig{
		Enabled: ptr(true),
		ExporterConfig: config.ExporterConfig{
			Exporter: "invalid",
			Endpoint: "localhost:4317",
		},
	}
}

func TestNewProvider_InitError_LogsByDefault(t *testing.T) {
	logger := &captureLogger{}

	_, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(invalidExporterConfig()),
		WithLogger(logger),
	)

	assert.Error(t, err)
	assert.Len(t, logger.errs, 1, "default behaviour logs the init failure once")
}

func TestNewProvider_InitError_WithQuietInitErrors(t *testing.T) {
	logger := &captureLogger{}

	_, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(invalidExporterConfig()),
		WithLogger(logger),
		WithQuietInitErrors(),
	)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid exporter type")
	assert.Empty(t, logger.errs, "quiet init must not log; the caller owns the error")
}

func TestWithQuietInitErrors_KeepsExportErrorLogging(t *testing.T) {
	logger := &captureLogger{}

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithReader(sdkmetric.NewManualReader()),
		WithLogger(logger),
		WithQuietInitErrors(),
	)
	assert.NoError(t, err)

	mp, ok := provider.(*meterProvider)
	assert.True(t, ok)

	exporter := &statsExporter{
		exporter: &fakeExporter{errs: []error{fmt.Errorf("collector unreachable")}},
		provider: mp,
	}

	assert.Error(t, exporter.Export(context.Background(), &metricdata.ResourceMetrics{}))
	assert.Len(t, logger.errs, 1, "runtime export failures must still be logged")
}
