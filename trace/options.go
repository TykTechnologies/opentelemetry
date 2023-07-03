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

/*
	WithConfig sets the configuraiton options for the trace provider

Example

	config := &config.OpenTelemetry{
		Enabled:  true,
		Exporter: "grpc",
		Endpoint: "localhost:4317",
	}
	provider, err := trace.NewProvider(trace.WithConfig(config))
	if err != nil {
		panic(err)
	}
*/
func WithConfig(cfg *config.OpenTelemetry) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.cfg = cfg
		},
	}
}

/*
	WithLogger sets the logger for the trace provider
	This is used to log errors and info messages for underlying operations

Example

	logger := logrus.New().WithField("component", "trace")
	provider, err := trace.NewProvider(trace.WithLogger(logger))
	if err != nil {
		panic(err)
	}
*/
func WithLogger(logger Logger) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.logger = logger
		},
	}
}

/*
	WithContext sets the context for the trace provider

Example

	ctx := context.Background()
	provider, err := trace.NewProvider(trace.WithContext(ctx))
	if err != nil {
		panic(err)
	}
*/
func WithContext(ctx context.Context) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.ctx = ctx
		},
	}
}
