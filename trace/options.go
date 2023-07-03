package trace

import (
	"context"

	"github.com/TykTechnologies/opentelemetry/config"
)

type Option interface {
	apply(*traceProvider)
}

type opts struct {
	fn func(*traceProvider)
}

func (o *opts) apply(tp *traceProvider) {
	o.fn(tp)
}

// WithConfig sets the config for the trace provider
func WithConfig(cfg config.OpenTelemetry) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.cfg = &cfg
		},
	}
}

// WithLogger sets the logger for the trace provider
// This is used to log errors and info messages for underlying operations
func WithLogger(logger Logger) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.logger = logger
		},
	}
}

// WithContext sets the context for the trace provider
func WithContext(ctx context.Context) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.ctx = ctx
		},
	}
}
