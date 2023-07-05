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
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actualPropagator, actualErr := propagatorFactory(tc.givenConfig)
			assert.Equal(t, tc.expectedErr, actualErr)
			assert.Equal(t, tc.expectedPropagator, actualPropagator)
		})
	}
}
