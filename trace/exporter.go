package trace

import (
	"context"
	"fmt"
	"net"
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
	// Parse the endpoint as a URL
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not parse endpoint URL: %v", err)
	}

	// Clear any path, raw path, and scheme on the URL to make sure it's just the base URL
	u.Path = ""
	u.RawPath = ""

	// Concatenate host and port as endpoint
	endpoint := net.JoinHostPort(u.Hostname(), u.Port())

	// Use the modified Insecure setting
	var clientOptions []otlptracehttp.Option
	clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint))
	clientOptions = append(clientOptions, otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second))
	clientOptions = append(clientOptions, otlptracehttp.WithHeaders(cfg.Headers))
	if u.Scheme != "https" {
		clientOptions = append(clientOptions, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.NewClient(clientOptions...), nil
}
