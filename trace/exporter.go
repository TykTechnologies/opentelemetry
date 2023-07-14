package trace

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func exporterFactory(ctx context.Context, provider *traceProvider) (sdktrace.SpanExporter, error) {
	var client otlptrace.Client

	switch provider.cfg.Exporter {
	case config.GRPCEXPORTER:
		client = newGRPCClient(ctx, provider.cfg)

		if !isGRPCEndpointActive(provider.cfg.Endpoint) {
			provider.logger.Error("GRPC endpoint: ", provider.cfg.Endpoint, " is down")
		}
	case config.HTTPEXPORTER:
		client = newHTTPClient(ctx, provider.cfg)

		if !isHTTPEndpointActive(provider.cfg.Endpoint) {
			provider.logger.Error("HTTP endpoint: ", provider.cfg.Endpoint, " is down")
		}
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", provider.cfg.Exporter)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(provider.cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Create the trace exporter
	return otlptrace.New(ctx, client)
}

func isGRPCEndpointActive(endpoint string) bool {
	conn, err := net.DialTimeout("tcp", endpoint, time.Second)

	if err != nil {
		return false
	}

	conn.Close()

	return true
}

func newGRPCClient(ctx context.Context, cfg *config.OpenTelemetry) otlptrace.Client {
	return otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(cfg.Headers),
	)
}

func isHTTPEndpointActive(endpoint string) bool {
	client := http.Client{
		Timeout: time.Second,
	}

	// Check if the endpoint has a protocol scheme (http/https), if not, append one
	if !strings.HasPrefix(strings.ToLower(endpoint), "http") {
		endpoint = "http://" + endpoint
	}

	// Using Head method to avoid downloading the whole body
	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, endpoint, nil)
	if err != nil {
		return false
	}

	_, err = client.Do(req)

	return err == nil
}

func newHTTPClient(ctx context.Context, cfg *config.OpenTelemetry) otlptrace.Client {
	return otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithInsecure(),
	)
}
