package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

const (
	HTTPEXPORTER = "http"
	GRPCEXPORTER = "grpc"
)

func exporterFactory(ctx context.Context, cfg config.OpenTelemetry) (sdktrace.SpanExporter, error) {
	var client otlptrace.Client

	switch cfg.Exporter {
	case GRPCEXPORTER:
		client = newGRPCClient(ctx, cfg)
	case HTTPEXPORTER:
		client = newHTTPClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Create the trace exporter
	return otlptrace.New(ctx, client)
}

func newGRPCClient(ctx context.Context, cfg config.OpenTelemetry) otlptrace.Client {
	return otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(cfg.Headers),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
		otlptracegrpc.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
	)
}

func newHTTPClient(ctx context.Context, cfg config.OpenTelemetry) otlptrace.Client {
	return otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithInsecure(),
	)
}
