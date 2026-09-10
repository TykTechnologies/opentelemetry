package metric

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

	customAttrs []Attribute

	// fromEnv adds resource.WithFromEnv() so OTEL_RESOURCE_ATTRIBUTES and
	// OTEL_SERVICE_NAME are honoured. It is applied first so explicit
	// attributes win: resource.New merges detectors in order and later values
	// overwrite earlier ones for the same key.
	fromEnv bool
}

func resourceFactory(ctx context.Context, resourceName string, cfg resourceConfig) (*resource.Resource, error) {
	opts := []resource.Option{}

	// Env attributes go first so anything set explicitly below overrides them.
	if cfg.fromEnv {
		opts = append(opts, resource.WithFromEnv())
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(resourceName),
	}

	if cfg.id != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(cfg.id))
	}

	if cfg.version != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.version))
	}

	// Add custom attributes.
	attrs = append(attrs, cfg.customAttrs...)

	opts = append(opts, resource.WithAttributes(attrs...))

	if cfg.withContainer {
		opts = append(opts, resource.WithContainer())
	}

	if cfg.withHost {
		opts = append(opts, resource.WithHost())
	}

	if cfg.withProcess {
		// Adding all the resource.WithProcess() options, except WithProcessOwner() since it's failing in k8s environments.
		opts = append(opts, resource.WithProcessPID(),
			resource.WithProcessExecutableName(),
			resource.WithProcessCommandArgs(),
			resource.WithProcessRuntimeName(),
			resource.WithProcessRuntimeVersion(),
			resource.WithProcessRuntimeDescription())
	}

	return resource.New(ctx, opts...)
}
