package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
				Enabled:            true,
				Exporter:           "http",
				Endpoint:           "test",
				ConnectionTimeout:  10,
				ResourceName:       "test-resource",
				SpanProcessorType:  "simple",
				ContextPropagation: "b3",
				Sampling: Sampling{
					Type: TRACEIDRATIOBASED,
					Rate: 0.8,
				},
			},
			expectedCfg: OpenTelemetry{
				Enabled:            true,
				Exporter:           "http",
				Endpoint:           "test",
				ConnectionTimeout:  10,
				ResourceName:       "test-resource",
				SpanProcessorType:  "simple",
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
				Enabled:            true,
				Exporter:           "grpc",
				Endpoint:           "localhost:4317",
				ConnectionTimeout:  1,
				ResourceName:       "tyk",
				SpanProcessorType:  "batch",
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
				Enabled:            true,
				Exporter:           "grpc",
				Endpoint:           "localhost:4317",
				ConnectionTimeout:  1,
				ResourceName:       "tyk",
				SpanProcessorType:  "batch",
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
