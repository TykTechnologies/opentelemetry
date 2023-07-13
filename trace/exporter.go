package trace

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func exporterFactory(ctx context.Context, cfg *config.OpenTelemetry) (sdktrace.SpanExporter, error) {
	var client otlptrace.Client
	switch cfg.Exporter {
	case config.GRPCEXPORTER:
		client = newGRPCClient(ctx, cfg)
	case config.HTTPEXPORTER:
		client = newHTTPClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Create the trace exporter
	return otlptrace.New(ctx, client)
}

func newGRPCClient(ctx context.Context, cfg *config.OpenTelemetry) otlptrace.Client {
	clientOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(time.Duration(cfg.ConnectionTimeout) * time.Second),
		otlptracegrpc.WithHeaders(cfg.Headers),
	}

	if cfg.TLSConfig.Insecure {
		clientOptions = append(clientOptions, otlptracegrpc.WithInsecure())
	} else {
		TLSConf := &tls.Config{
			InsecureSkipVerify: cfg.TLSConfig.InsecureSkipVerify,
		}
		clientOptions = append(clientOptions, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(TLSConf)))
	}

	return otlptracegrpc.NewClient(clientOptions...)
}

func newHTTPClient(ctx context.Context, cfg *config.OpenTelemetry) otlptrace.Client {
	// OTel SDK does not support URL with scheme nor path, so we need to parse it
	// The scheme will be added automatically, depending on the TLSInsure setting
	endpoint := parseEndpoint(cfg)

	var clientOptions []otlptracehttp.Option
	clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
		otlptracehttp.WithHeaders(cfg.Headers))

	if cfg.TLSConfig.Insecure {
		clientOptions = append(clientOptions, otlptracehttp.WithInsecure())
	} else {
		TLSConf := &tls.Config{
			InsecureSkipVerify: cfg.TLSConfig.InsecureSkipVerify,
		}
		clientOptions = append(clientOptions, otlptracehttp.WithTLSClientConfig(TLSConf))
	}

	return otlptracehttp.NewClient(clientOptions...)
}

func parseEndpoint(cfg *config.OpenTelemetry) string {
	endpoint := cfg.Endpoint
	// Temporary adding scheme to get the host and port
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return cfg.Endpoint
	}

	host := u.Hostname()
	port := u.Port()

	if port == "" {
		return host
	}

	return net.JoinHostPort(host, port)
}
