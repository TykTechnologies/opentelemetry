package metric

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc/credentials"
)

func exporterFactory(ctx context.Context, cfg *config.OpenTelemetry) (sdkmetric.Exporter, error) {
	switch cfg.Exporter {
	case config.GRPCEXPORTER:
		return newGRPCExporter(ctx, cfg)
	case config.HTTPEXPORTER:
		return newHTTPExporter(ctx, cfg)
	default:
		return nil, fmt.Errorf("invalid exporter type: %s", cfg.Exporter)
	}
}

func newGRPCExporter(ctx context.Context, cfg *config.OpenTelemetry) (sdkmetric.Exporter, error) {
	clientOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithTimeout(time.Duration(cfg.ConnectionTimeout) * time.Second),
		otlpmetricgrpc.WithHeaders(cfg.Headers),
	}

	// Configure retry if enabled.
	if cfg.Metrics.Retry.Enabled != nil && *cfg.Metrics.Retry.Enabled {
		clientOptions = append(clientOptions, otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Duration(cfg.Metrics.Retry.InitialInterval) * time.Millisecond,
			MaxInterval:     time.Duration(cfg.Metrics.Retry.MaxInterval) * time.Millisecond,
			MaxElapsedTime:  time.Duration(cfg.Metrics.Retry.MaxElapsedTime) * time.Millisecond,
		}))
	}

	isTLSDisabled := !cfg.TLS.Enable

	if isTLSDisabled {
		clientOptions = append(clientOptions, otlpmetricgrpc.WithInsecure())
	} else {
		tlsConf, err := handleTLS(&cfg.TLS)
		if err != nil {
			return nil, err
		}
		clientOptions = append(clientOptions, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConf)))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()

	return otlpmetricgrpc.New(ctx, clientOptions...)
}

func newHTTPExporter(ctx context.Context, cfg *config.OpenTelemetry) (sdkmetric.Exporter, error) {
	// OTel SDK does not support URL with scheme nor path, so we need to parse it.
	// The scheme will be added automatically, depending on the TLS setting.
	endpoint := parseEndpoint(cfg)

	clientOptions := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithTimeout(time.Duration(cfg.ConnectionTimeout) * time.Second),
		otlpmetrichttp.WithHeaders(cfg.Headers),
	}

	// Configure retry if enabled.
	if cfg.Metrics.Retry.Enabled != nil && *cfg.Metrics.Retry.Enabled {
		clientOptions = append(clientOptions, otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Duration(cfg.Metrics.Retry.InitialInterval) * time.Millisecond,
			MaxInterval:     time.Duration(cfg.Metrics.Retry.MaxInterval) * time.Millisecond,
			MaxElapsedTime:  time.Duration(cfg.Metrics.Retry.MaxElapsedTime) * time.Millisecond,
		}))
	}

	isTLSDisabled := !cfg.TLS.Enable

	if isTLSDisabled {
		clientOptions = append(clientOptions, otlpmetrichttp.WithInsecure())
	} else {
		tlsConf, err := handleTLS(&cfg.TLS)
		if err != nil {
			return nil, err
		}
		clientOptions = append(clientOptions, otlpmetrichttp.WithTLSClientConfig(tlsConf))
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()

	return otlpmetrichttp.New(ctx, clientOptions...)
}

func parseEndpoint(cfg *config.OpenTelemetry) string {
	endpoint := cfg.Endpoint
	// Temporarily adding scheme to get the host and port.
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

func handleTLS(cfg *config.TLS) (*tls.Config, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}

		tlsConf.Certificates = []tls.Certificate{cert}
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

		tlsConf.RootCAs = certPool
	}

	minVersion, maxVersion, err := handleTLSVersion(cfg)
	if err != nil {
		return nil, err
	}

	tlsConf.MinVersion = uint16(minVersion)
	tlsConf.MaxVersion = uint16(maxVersion)

	return tlsConf, nil
}

func handleTLSVersion(cfg *config.TLS) (minVersion, maxVersion int, err error) {
	validVersions := map[string]int{
		"1.0": tls.VersionTLS10,
		"1.1": tls.VersionTLS11,
		"1.2": tls.VersionTLS12,
		"1.3": tls.VersionTLS13,
	}

	if cfg.MaxVersion == "" {
		cfg.MaxVersion = "1.3"
	}

	if _, ok := validVersions[cfg.MaxVersion]; ok {
		maxVersion = validVersions[cfg.MaxVersion]
	} else {
		err = errors.New("Invalid MaxVersion specified. Please specify a valid TLS version: 1.0, 1.1, 1.2, or 1.3")
		return
	}

	if cfg.MinVersion == "" {
		cfg.MinVersion = "1.2"
	}

	if _, ok := validVersions[cfg.MinVersion]; ok {
		minVersion = validVersions[cfg.MinVersion]
	} else {
		err = errors.New("Invalid MinVersion specified. Please specify a valid TLS version: 1.0, 1.1, 1.2, or 1.3")
		return
	}

	if minVersion > maxVersion {
		err = errors.New(
			"MinVersion is higher than MaxVersion. Please specify a valid MinVersion that is lower or equal to MaxVersion",
		)
		return
	}

	return
}
