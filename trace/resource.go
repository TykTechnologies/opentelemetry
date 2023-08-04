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
	withProcess   bool
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

	if cfg.withProcess {
		// adding all the resource.WithProcess() options, except WithProcessOwner() since it's failing in k8s environments
		opts = append(opts, resource.WithProcessPID())
		opts = append(opts, resource.WithProcessExecutableName())
		opts = append(opts, resource.WithProcessCommandArgs())
		opts = append(opts, resource.WithProcessRuntimeName())
		opts = append(opts, resource.WithProcessRuntimeVersion())
		opts = append(opts, resource.WithProcessRuntimeDescription())
	}

	return resource.New(ctx, opts...)
}
