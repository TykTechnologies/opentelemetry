package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TykTechnologies/opentelemetry/config"
)

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{"plain host:port", "localhost:4317", "localhost:4317"},
		{"with http scheme", "http://localhost:4317", "localhost:4317"},
		{"with https scheme", "https://collector.example.com:4318", "collector.example.com:4318"},
		{"host only no port", "localhost", "localhost"},
		{"with scheme no port", "http://localhost", "localhost"},
		{"with path", "http://localhost:4317/v1/metrics", "localhost:4317"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEndpoint(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExporterFactory_InvalidType(t *testing.T) {
	cfg := &config.MetricsConfig{
		ExporterConfig: config.ExporterConfig{
			Exporter: "invalid",
		},
	}
	_, err := exporterFactory(context.TODO(), cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid exporter type")
}

func TestHandleTLSVersion_Valid(t *testing.T) {
	cfg := &config.TLS{MinVersion: "1.2", MaxVersion: "1.3"}
	min, max, err := handleTLSVersion(cfg)
	assert.NoError(t, err)
	assert.Greater(t, min, 0)
	assert.Greater(t, max, 0)
	assert.LessOrEqual(t, min, max)
}

func TestHandleTLSVersion_InvalidMin(t *testing.T) {
	cfg := &config.TLS{MinVersion: "invalid", MaxVersion: "1.3"}
	_, _, err := handleTLSVersion(cfg)
	assert.Error(t, err)
}

func TestHandleTLSVersion_InvalidMax(t *testing.T) {
	cfg := &config.TLS{MinVersion: "1.2", MaxVersion: "invalid"}
	_, _, err := handleTLSVersion(cfg)
	assert.Error(t, err)
}

func TestHandleTLSVersion_MinGreaterThanMax(t *testing.T) {
	cfg := &config.TLS{MinVersion: "1.3", MaxVersion: "1.2"}
	_, _, err := handleTLSVersion(cfg)
	assert.Error(t, err)
}

func TestHandleTLSVersion_Defaults(t *testing.T) {
	cfg := &config.TLS{}
	min, max, err := handleTLSVersion(cfg)
	assert.NoError(t, err)
	assert.Greater(t, min, 0)
	assert.Greater(t, max, 0)
}
