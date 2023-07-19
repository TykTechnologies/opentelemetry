package trace

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type resourceConfig struct {
	id      string
	version string

	withHost      bool
	withContainer bool
}

func resourceFactory(ctx context.Context, resourceName string, cfg resourceConfig) (*resource.Resource, error) {
	opts := []resource.Option{}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(resourceName),
	}

	if cfg.id != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(cfg.id))
	}

	if cfg.version != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.version))
	}

	opts = append(opts, resource.WithAttributes(attrs...))

	if cfg.withContainer {
		opts = append(opts, resource.WithContainer())
	}

	if cfg.withHost {
		opts = append(opts, resource.WithHost())
	}

	return resource.New(ctx, opts...)
}
