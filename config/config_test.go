package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

func Test_SetDefault(t *testing.T) {
	tcs := []struct {
		name        string
		givenCfg    OpenTelemetry
		expectedCfg OpenTelemetry
	}{
		{
			name: "otel disabled",
			givenCfg: OpenTelemetry{
				Enabled: false,
			},
			expectedCfg: OpenTelemetry{
				Enabled: false,
			},
		},
		{
			name: "custom values",
			givenCfg: OpenTelemetry{
				Enabled: true,
				ExporterConfig: ExporterConfig{
					Exporter:          "http",
					Endpoint:          "test",
					ConnectionTimeout: 10,
					ResourceName:      "test-resource",
				},
				SpanProcessorType:  "simple",
				SpanBatchConfig: SpanBatchConfig{
					MaxQueueSize:       1000,
					MaxExportBatchSize: 100,
					BatchTimeout:       10,
				},
				ContextPropagation: "b3",
				Sampling: Sampling{
					Type: TRACEIDRATIOBASED,
					Rate: 0.8,
				},
			},
			expectedCfg: OpenTelemetry{
				Enabled: true,
				ExporterConfig: ExporterConfig{
					Exporter:          "http",
					Endpoint:          "test",
					ConnectionTimeout: 10,
					ResourceName:      "test-resource",
				},
				SpanProcessorType:  "simple",
				SpanBatchConfig: SpanBatchConfig{
					MaxQueueSize:       1000,
					MaxExportBatchSize: 100,
					BatchTimeout:       10,
				},
				ContextPropagation: "b3",
				Sampling: Sampling{
					Type: TRACEIDRATIOBASED,
					Rate: 0.8,
				},
			},
		},
		{
			name: "default values",
			givenCfg: OpenTelemetry{
				Enabled: true,
			},
			expectedCfg: OpenTelemetry{
				Enabled: true,
				ExporterConfig: ExporterConfig{
					Exporter:          "grpc",
					Endpoint:          "localhost:4317",
					ConnectionTimeout: 1,
					ResourceName:      "tyk",
				},
				SpanProcessorType:  "batch",
				SpanBatchConfig: SpanBatchConfig{
					MaxQueueSize:       2048,
					MaxExportBatchSize: 512,
					BatchTimeout:       5,
				},
				ContextPropagation: "tracecontext",
				Sampling: Sampling{
					Type: ALWAYSON,
				},
			},
		},
		{
			name: "default sampling rate",
			givenCfg: OpenTelemetry{
				Enabled: true,
				Sampling: Sampling{
					Type: TRACEIDRATIOBASED,
				},
			},
			expectedCfg: OpenTelemetry{
				Enabled: true,
				ExporterConfig: ExporterConfig{
					Exporter:          "grpc",
					Endpoint:          "localhost:4317",
					ConnectionTimeout: 1,
					ResourceName:      "tyk",
				},
				SpanProcessorType:  "batch",
				SpanBatchConfig: SpanBatchConfig{
					MaxQueueSize:       2048,
					MaxExportBatchSize: 512,
					BatchTimeout:       5,
				},
				ContextPropagation: "tracecontext",
				Sampling: Sampling{
					Type: TRACEIDRATIOBASED,
					Rate: 0.5,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.givenCfg.SetDefaults()

			if diff := cmp.Diff(tc.expectedCfg, tc.givenCfg); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExporterConfig_SetDefaults(t *testing.T) {
	tcs := []struct {
		name     string
		given    ExporterConfig
		expected ExporterConfig
	}{
		{
			name:  "all defaults",
			given: ExporterConfig{},
			expected: ExporterConfig{
				Exporter:          "grpc",
				Endpoint:          "localhost:4317",
				ConnectionTimeout: 1,
				ResourceName:      "tyk",
			},
		},
		{
			name: "custom values preserved",
			given: ExporterConfig{
				Exporter:          "http",
				Endpoint:          "collector:4318",
				ConnectionTimeout: 5,
				ResourceName:      "my-service",
			},
			expected: ExporterConfig{
				Exporter:          "http",
				Endpoint:          "collector:4318",
				ConnectionTimeout: 5,
				ResourceName:      "my-service",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.given.SetDefaults()

			if diff := cmp.Diff(tc.expected, tc.given); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMetricsConfig_SetDefaults(t *testing.T) {
	tcs := []struct {
		name     string
		given    MetricsConfig
		expected MetricsConfig
	}{
		{
			name:  "all defaults",
			given: MetricsConfig{},
			expected: MetricsConfig{
				Enabled: nil,
				ExporterConfig: ExporterConfig{
					Exporter:          "grpc",
					Endpoint:          "localhost:4317",
					ConnectionTimeout: 1,
					ResourceName:      "tyk",
				},
				ExportInterval:  60,
				Temporality:     TEMPORALITY_CUMULATIVE,
				ShutdownTimeout: 30,
				Retry: MetricsRetryConfig{
					Enabled:         ptr(true),
					InitialInterval: 5000,
					MaxInterval:     30000,
					MaxElapsedTime:  60000,
				},
			},
		},
		{
			name: "custom values preserved",
			given: MetricsConfig{
				Enabled: ptr(true),
				ExporterConfig: ExporterConfig{
					Exporter: "http",
					Endpoint: "metrics-collector:4318",
				},
				ExportInterval:  30,
				Temporality:     TEMPORALITY_DELTA,
				ShutdownTimeout: 10,
				Retry: MetricsRetryConfig{
					Enabled:         ptr(false),
					InitialInterval: 1000,
					MaxInterval:     10000,
					MaxElapsedTime:  30000,
				},
			},
			expected: MetricsConfig{
				Enabled: ptr(true),
				ExporterConfig: ExporterConfig{
					Exporter:          "http",
					Endpoint:          "metrics-collector:4318",
					ConnectionTimeout: 1,
					ResourceName:      "tyk",
				},
				ExportInterval:  30,
				Temporality:     TEMPORALITY_DELTA,
				ShutdownTimeout: 10,
				Retry: MetricsRetryConfig{
					Enabled:         ptr(false),
					InitialInterval: 1000,
					MaxInterval:     10000,
					MaxElapsedTime:  30000,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.given.SetDefaults()

			if diff := cmp.Diff(tc.expected, tc.given); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
