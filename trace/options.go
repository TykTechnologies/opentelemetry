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
	WithConfig sets the configuration options for the tracer provider

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
	WithLogger sets the logger for the tracer provider
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
	WithContext sets the context for the tracer provider

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

func WithServiceID(id string) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.resources.id = id
		},
	}
}

func WithServiceVersion(version string) Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.resources.id = version
		},
	}
}

func WithHostDetector() Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.resources.withHost = true
		},
	}
}

func WithContainerDetector() Option {
	return &opts{
		fn: func(tp *traceProvider) {
			tp.resources.withContainer = true
		},
	}
}
