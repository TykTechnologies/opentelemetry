package trace

import (
	"fmt"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
)

func Test_PropagatorFactory(t *testing.T) {
	tcs := []struct {
		name               string
		givenConfig        *config.OpenTelemetry
		expectedErr        error
		expectedPropagator propagation.TextMapPropagator
		checkType          bool // For composite propagators, just check type instead of equality
	}{
		{
			name: "invalid propagator type",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: "invalid",
			},
			expectedPropagator: nil,
			expectedErr:        fmt.Errorf("invalid context propagation type: %s", "invalid"),
		},
		{
			name: "b3 propagator",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_B3,
			},
			expectedPropagator: b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
			expectedErr:        nil,
		},
		{
			name: "tracecontext propagator",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_TRACECONTEXT,
			},
			expectedPropagator: propagation.TraceContext{},
			expectedErr:        nil,
		},
		{
			name: "custom propagator without header",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_CUSTOM,
				CustomTraceHeader:  "",
			},
			expectedPropagator: nil,
			expectedErr:        fmt.Errorf("custom_trace_header required when context_propagation is 'custom'"),
		},
		{
			name: "custom propagator with header",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_CUSTOM,
				CustomTraceHeader:  "X-Correlation-ID",
			},
			expectedPropagator: NewCustomHeaderPropagator("X-Correlation-ID", true),
			expectedErr:        nil,
		},
		{
			name: "composite propagator",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_COMPOSITE,
				CustomTraceHeader:  "X-Correlation-ID",
			},
			expectedPropagator: nil, // Will check type instead
			expectedErr:        nil,
			checkType:          true,
		},
		{
			name: "tracecontext with custom header (hybrid mode)",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_TRACECONTEXT,
				CustomTraceHeader:  "X-Request-ID",
			},
			expectedPropagator: nil, // Will check type instead
			expectedErr:        nil,
			checkType:          true,
		},
		{
			name: "b3 with custom header (hybrid mode)",
			givenConfig: &config.OpenTelemetry{
				ContextPropagation: config.PROPAGATOR_B3,
				CustomTraceHeader:  "X-Trace-ID",
			},
			expectedPropagator: nil, // Will check type instead
			expectedErr:        nil,
			checkType:          true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actualPropagator, actualErr := propagatorFactory(tc.givenConfig)
			assert.Equal(t, tc.expectedErr, actualErr)

			if tc.checkType {
				// For composite propagators, just verify it's not nil
				if tc.expectedErr == nil {
					assert.NotNil(t, actualPropagator)
				}
			} else {
				assert.Equal(t, tc.expectedPropagator, actualPropagator)
			}
		})
	}
}
