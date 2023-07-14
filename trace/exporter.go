package trace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

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
	var err error
	switch cfg.Exporter {
	case config.GRPCEXPORTER:
		client, err = newGRPCClient(ctx, cfg)
	case config.HTTPEXPORTER:
		client, err = newHTTPClient(ctx, cfg)
	default:
		err = fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Create the trace exporter
	return otlptrace.New(ctx, client)
}

func newGRPCClient(ctx context.Context, cfg *config.OpenTelemetry) (otlptrace.Client, error) {
	clientOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(time.Duration(cfg.ConnectionTimeout) * time.Second),
		otlptracegrpc.WithHeaders(cfg.Headers),
	}

	if !cfg.TLSConfig.Enable {
		clientOptions = append(clientOptions, otlptracegrpc.WithInsecure())
	} else {
		TLSConf, err := handleTLS(&cfg.TLSConfig)
		if err != nil {
			return nil, err
		}
		clientOptions = append(clientOptions, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(TLSConf)))
	}

	return otlptracegrpc.NewClient(clientOptions...), nil
}

func newHTTPClient(ctx context.Context, cfg *config.OpenTelemetry) (otlptrace.Client, error) {
	// OTel SDK does not support URL with scheme nor path, so we need to parse it
	// The scheme will be added automatically, depending on the TLSInsure setting
	endpoint := parseEndpoint(cfg)

	var clientOptions []otlptracehttp.Option
	clientOptions = append(clientOptions, otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTimeout(time.Duration(cfg.ConnectionTimeout)*time.Second),
		otlptracehttp.WithHeaders(cfg.Headers))

	if !cfg.TLSConfig.Enable {
		clientOptions = append(clientOptions, otlptracehttp.WithInsecure())
	} else {
		TLSConf, err := handleTLS(&cfg.TLSConfig)
		if err != nil {
			return nil, err
		}
		clientOptions = append(clientOptions, otlptracehttp.WithTLSClientConfig(TLSConf))
	}

	return otlptracehttp.NewClient(clientOptions...), nil
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

func handleTLS(cfg *config.TLSConfig) (*tls.Config, error) {
	TLSConf := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}

		TLSConf.Certificates = []tls.Certificate{cert}
	}

	if cfg.CAFile != "" {
		caPem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caPem) {
			return nil, fmt.Errorf("failed to add CA certificate")
		}

		TLSConf.RootCAs = certPool
	}

	return TLSConf, nil
}
