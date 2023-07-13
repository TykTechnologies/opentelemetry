package trace

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func Test_NewGRPCClient(t *testing.T) {
	ctx := context.Background()
	endpoint := "localhost:4317"

	cfg := &config.OpenTelemetry{
		Endpoint: endpoint,
	}

	client, err := newGRPCClient(ctx, cfg)
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

func Test_NewHTTPClient(t *testing.T) {
	ctx := context.Background()
	endpoint := "localhost:4317"

	cfg := &config.OpenTelemetry{
		Endpoint: endpoint,
	}

	client, err := newHTTPClient(ctx, cfg)
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

func Test_ExporterFactory(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name        string
		givenConfig *config.OpenTelemetry
		expectedErr error
		setupFn     func() (string, func())
	}{
		{
			name: "invalid exporter type",
			givenConfig: &config.OpenTelemetry{
				Exporter: "invalid",
			},
			expectedErr: fmt.Errorf("invalid exporter type: %s", "invalid"),
		},
		{
			name: "http exporter",
			givenConfig: &config.OpenTelemetry{
				Exporter:          "http",
				Endpoint:          "to be replace by setupFn",
				ConnectionTimeout: 1,
			},
			setupFn: func() (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Here you can check the request and return a response
					w.WriteHeader(http.StatusOK)
				}))

				return server.URL, server.Close
			},
			expectedErr: nil,
		},

		{
			name: "gRPC exporter",
			givenConfig: &config.OpenTelemetry{
				Exporter:          "grpc",
				Endpoint:          "to be replace by setupFn",
				ConnectionTimeout: 1,
			},
			setupFn: func() (string, func()) {
				lis, err := net.Listen("tcp", "localhost:0")
				if err != nil {
					t.Fatalf("failed to listen: %v", err)
				}

				// Create a gRPC server and serve on the listener
				s := grpc.NewServer()
				go func() {
					if err := s.Serve(lis); err != nil {
						t.Logf("failed to serve: %v", err)
					}
				}()

				return lis.Addr().String(), s.Stop
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if tc.setupFn != nil {
				endpoint, teardown := tc.setupFn()
				defer teardown()

				tc.givenConfig.Endpoint = endpoint
			}

			exporter, err := exporterFactory(ctx, tc.givenConfig)
			if tc.expectedErr != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, exporter)
			}
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string
		want     string
	}{
		{"with http and port", "http://example.com:8080", "example.com:8080"},
		{"with https and port", "https://example.com:8080", "example.com:8080"},
		{"with http without port", "http://example.com", "example.com"},
		{"with https without port", "https://example.com", "example.com"},
		{"without http and with port", "example.com:8080", "example.com:8080"},
		{"without http and port", "example.com", "example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.OpenTelemetry{Endpoint: tc.endpoint}
			got := parseEndpoint(cfg)
			if got != tc.want {
				t.Errorf("parseEndpoint(%q) = %q; want %q", tc.endpoint, got, tc.want)
			}
		})
	}
}
