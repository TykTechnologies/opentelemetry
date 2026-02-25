package metric

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/config"
)

// Attribute is an alias for OpenTelemetry attribute.KeyValue.
type Attribute = attribute.KeyValue

// Option is an interface for configuring the meter provider.
type Option interface {
	apply(*meterProvider)
}

type opts struct {
	fn func(*meterProvider)
}

func (o *opts) apply(mp *meterProvider) {
	o.fn(mp)
}

// WithConfig sets the configuration options for the meter provider.
//
// Example:
//
//	metricsEnabled := true
//	cfg := &config.MetricsConfig{
//		Enabled:  &metricsEnabled,
//		ExporterConfig: config.ExporterConfig{
//			Exporter: "grpc",
//			Endpoint: "localhost:4317",
//		},
//	}
//	provider, err := metric.NewProvider(metric.WithConfig(cfg))
//	if err != nil {
//		panic(err)
//	}
func WithConfig(cfg *config.MetricsConfig) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.cfg = cfg
		},
	}
}

// WithLogger sets the logger for the meter provider.
// This is used to log errors and info messages for underlying operations.
//
// Example:
//
//	logger := logrus.New().WithField("component", "metric")
//	provider, err := metric.NewProvider(metric.WithLogger(logger))
//	if err != nil {
//		panic(err)
//	}
func WithLogger(logger Logger) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.logger = logger
		},
	}
}

// WithContext sets the context for the meter provider.
//
// Example:
//
//	ctx := context.Background()
//	provider, err := metric.NewProvider(metric.WithContext(ctx))
//	if err != nil {
//		panic(err)
//	}
func WithContext(ctx context.Context) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.ctx = ctx
		},
	}
}

// WithServiceID sets the resource service.id for the meter provider.
// This is useful to identify service instance on the metric resource.
//
// Example:
//
//	provider, err := metric.NewProvider(metric.WithServiceID("instance-id"))
//	if err != nil {
//		panic(err)
//	}
func WithServiceID(id string) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.id = id
		},
	}
}

// WithServiceVersion sets the resource service.version for the meter provider.
// This is useful to identify service version on the metric resource.
//
// Example:
//
//	provider, err := metric.NewProvider(metric.WithServiceVersion("v4.0.5"))
//	if err != nil {
//		panic(err)
//	}
func WithServiceVersion(version string) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.version = version
		},
	}
}

// WithHostDetector adds attributes from the host to the configured resource.
//
// Example:
//
//	provider, err := metric.NewProvider(metric.WithHostDetector())
//	if err != nil {
//		panic(err)
//	}
func WithHostDetector() Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.withHost = true
		},
	}
}

// WithContainerDetector adds attributes from the container to the configured resource.
//
// Example:
//
//	provider, err := metric.NewProvider(metric.WithContainerDetector())
//	if err != nil {
//		panic(err)
//	}
func WithContainerDetector() Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.withContainer = true
		},
	}
}

// WithProcessDetector adds attributes from the process to the configured resource.
//
// Example:
//
//	provider, err := metric.NewProvider(metric.WithProcessDetector())
//	if err != nil {
//		panic(err)
//	}
func WithProcessDetector() Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.withProcess = true
		},
	}
}

// WithCustomResourceAttributes adds custom attributes to the configured resource.
//
// Example:
//
//	attrs := []metric.Attribute{attribute.String("key", "value")}
//	provider, err := metric.NewProvider(metric.WithCustomResourceAttributes(attrs...))
//	if err != nil {
//		panic(err)
//	}
func WithCustomResourceAttributes(attrs ...Attribute) Option {
	return &opts{
		fn: func(mp *meterProvider) {
			mp.resources.customAttrs = attrs
		},
	}
}
