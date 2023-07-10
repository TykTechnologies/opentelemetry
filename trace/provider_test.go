package trace

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func Test_Shutdown(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name        string
		givenCfg    *config.OpenTelemetry
		setupFn     func() (string, func())
		expectedErr error
	}{
		{
			name: "shutdown - otel disabled", // otel disabled should trigger the use of the noop provider
			givenCfg: &config.OpenTelemetry{
				Enabled: false,
			},
			expectedErr: nil,
		},
		{
			name: "shutdown - http otel enabled", // otel enabled should trigger the use of the sdk provider
			givenCfg: &config.OpenTelemetry{
				Enabled:           true,
				Exporter:          "http",
				Endpoint:          "http://localhost:4317",
				ConnectionTimeout: 10,
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
			name: "shutdown - grpc otel enabled", // otel enabled should trigger the use of the sdk provider
			givenCfg: &config.OpenTelemetry{
				Enabled:           true,
				Exporter:          "grpc",
				Endpoint:          "http://localhost:4317",
				ConnectionTimeout: 10,
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
				tc.givenCfg.Endpoint = endpoint
				defer teardown()
			}

			provider, err := NewProvider(WithContext(ctx), WithConfig(tc.givenCfg))
			assert.Nil(t, err)
			assert.NotNil(t, provider)

			err = provider.Shutdown(ctx)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func Test_Tracer(t *testing.T) {
	tcs := []struct {
		name                  string
		givenCfg              *config.OpenTelemetry
		setupFn               func() (string, func())
		expectedTraceProvider interface{}
	}{
		{
			name: "no op tracer",
			givenCfg: &config.OpenTelemetry{
				Enabled: false,
			},
			expectedTraceProvider: oteltrace.NewNoopTracerProvider(),
		},
		{
			name: "sdk tracer",
			givenCfg: &config.OpenTelemetry{
				Enabled:           true,
				Exporter:          "http",
				Endpoint:          "http://localhost:4317",
				ConnectionTimeout: 10,
			},
			setupFn: func() (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Here you can check the request and return a response
					w.WriteHeader(http.StatusOK)
				}))

				return server.URL, server.Close
			},
			expectedTraceProvider: sdktrace.NewTracerProvider(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.setupFn != nil {
				endpoint, teardown := tc.setupFn()
				tc.givenCfg.Endpoint = endpoint
				defer teardown()
			}

			// first check if we are setting the internal tracer provider
			provider, err := NewProvider(WithContext(ctx), WithConfig(tc.givenCfg))
			assert.Nil(t, err)
			assert.NotNil(t, provider)

			tp, ok := provider.(*traceProvider)
			assert.True(t, ok)

			assert.IsType(t, tc.expectedTraceProvider, tp.traceProvider)

			// now check if we are setting the OTel global tracer provider
			globalProvider := otel.GetTracerProvider()
			assert.NotNil(t, globalProvider)

			// lastly, check the tracer
			tracer := provider.Tracer()
			assert.NotNil(t, tracer)
		})
	}
}

func Test_Type(t *testing.T) {
	tcs := []struct {
		name         string
		givenCfg     *config.OpenTelemetry
		expectedType string
	}{
		{
			name: "no op tracer",
			givenCfg: &config.OpenTelemetry{
				Enabled: false,
			},
			expectedType: "noop",
		},
		{
			name: "otel tracer",
			givenCfg: &config.OpenTelemetry{
				Enabled: true,
			},
			expectedType: "otel",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			provider, err := NewProvider(WithContext(ctx), WithConfig(tc.givenCfg))
			assert.Nil(t, err)
			assert.NotNil(t, provider)

			assert.Equal(t, tc.expectedType, provider.Type())
		})
	}
}

func Test_GetSampler(t *testing.T) {
	tests := []struct {
		name         string
		samplingType string
		samplingRate float64
		parentBased  bool
		expectedDesc string
	}{
		{"AlwaysOn", "AlwaysOn", 0, false, "AlwaysOnSampler"},
		{"AlwaysOff", "AlwaysOff", 0, false, "AlwaysOffSampler"},
		{"TraceIDRatioBased-0", "TraceIDRatioBased", 0, true,
			"ParentBased{root:TraceIDRatioBased{0},remoteParentSampled:AlwaysOnSampler," +
				"remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}"},
		{"TraceIDRatioBased-0.5", "TraceIDRatioBased", 0.5, false, "TraceIDRatioBased{0.5}"},
		{"TraceIDRatioBased-1", "TraceIDRatioBased", 1, false, "AlwaysOnSampler"},
		{"Invalid", "Invalid", 0, false, "AlwaysOnSampler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := getSampler(tt.samplingType, tt.samplingRate, tt.parentBased)

			if got := sampler.Description(); got != tt.expectedDesc {
				t.Errorf("getSampler() = %v, want %v", got, tt.expectedDesc)
			}
		})
	}
}
