package trace

import (
	"fmt"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"
)

func propagatorFactory(cfg *config.OpenTelemetry) (propagation.TextMapPropagator, error) {
	if cfg.ContextPropagation == config.PROPAGATOR_CUSTOM {
		if cfg.CustomTraceHeader == "" {
			return nil, fmt.Errorf("custom_trace_header required when context_propagation is 'custom'")
		}
		return NewCustomHeaderPropagator(cfg.CustomTraceHeader, true), nil
	}

	var propagators []propagation.TextMapPropagator

	if cfg.CustomTraceHeader != "" {
		shouldInject := cfg.ContextPropagation == config.PROPAGATOR_COMPOSITE
		propagators = append(propagators, NewCustomHeaderPropagator(cfg.CustomTraceHeader, shouldInject))
	}

	switch cfg.ContextPropagation {
	case config.PROPAGATOR_B3:
		propagators = append(propagators, b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)))
	case config.PROPAGATOR_TRACECONTEXT, config.PROPAGATOR_COMPOSITE:
		propagators = append(propagators, propagation.TraceContext{})
	default:
		return nil, fmt.Errorf("invalid context propagation type: %s", cfg.ContextPropagation)
	}

	if len(propagators) > 1 {
		return propagation.NewCompositeTextMapPropagator(propagators...), nil
	}
	return propagators[0], nil
}
