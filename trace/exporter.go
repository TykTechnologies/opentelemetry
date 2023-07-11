package trace

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func exporterFactory(ctx context.Context, cfg *config.OpenTelemetry) (sdktrace.SpanExporter, error) {
	var client otlptrace.Client
	var err error
	switch cfg.Exporter {
	case config.GRPCEXPORTER:
		client, err = newGRPCClient(ctx, cfg)
		if err != nil {
			return nil, err
		}
	case config.HTTPEXPORTER:
		client, err = newHTTPClient(ctx, cfg)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Create the trace exporter
	return otlptrace.New(ctx, client)
}

func newGRPCClient(ctx context.Context, cfg *config.OpenTelemetry) (otlptrace.Client, error) {
	return otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(cfg.Headers),
	), nil
}

func newHTTPClient(ctx context.Context, cfg *config.OpenTelemetry) (otlptrace.Client, error) {
	// The endpoint must not contain any URL path.
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %s", err)
	}

	u.Path = ""
	u.RawPath = ""

	// Use the cleaned URL as the endpoint
	return otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(u.String()),
		otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithInsecure(),
	), nil
}
