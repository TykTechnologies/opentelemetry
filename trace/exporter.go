package trace

import (
	"context"
	"fmt"
	"opentelemetry/config"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	HTTPEXPORTER = "http"
	GRPCEXPORTER = "grpc"
)

func exporterFactory(ctx context.Context, cfg config.OpenTelemetry) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case GRPCEXPORTER:
		return newGRPCExporter(ctx, cfg)
	case HTTPEXPORTER:
		return newHTTPExporter(ctx, cfg)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}
}

func newGRPCExporter(ctx context.Context, cfg config.OpenTelemetry) (*otlptrace.Exporter, error) {
	// Set the timeout for establishing a connection to the collector
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout))
	defer cancel()

	// Create the gRPC connection
	conn, err := grpc.DialContext(ctx, cfg.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Create the trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	return traceExporter, nil
}

func newHTTPExporter(ctx context.Context, cfg config.OpenTelemetry) (*otlptrace.Exporter, error) {
	// Set the timeout for establishing a connection to the collector
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout))
	defer cancel()

	client := otlptracehttp.NewClient(
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	)

	return otlptrace.New(ctx, client)
}
