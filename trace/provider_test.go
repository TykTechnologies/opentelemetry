package trace

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/TykTechnologies/opentelemetry/config"
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
				Enabled: true,
				ExporterConfig: config.ExporterConfig{
					Exporter:          "http",
					Endpoint:          "http://localhost:4317",
					ConnectionTimeout: 10,
				},
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
				Enabled: true,
				ExporterConfig: config.ExporterConfig{
					Exporter:          "grpc",
					Endpoint:          "http://localhost:4317",
					ConnectionTimeout: 10,
				},
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
				Enabled: true,
				ExporterConfig: config.ExporterConfig{
					Exporter:          "http",
					Endpoint:          "http://localhost:4317",
					ConnectionTimeout: 10,
				},
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

func TestProvider_WithSpanBatchConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup HTTP server to receive spans
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.OpenTelemetry{
		Enabled: true,
		ExporterConfig: config.ExporterConfig{
			Exporter:          "http",
			Endpoint:          server.URL,
			ConnectionTimeout: 10,
		},
		SpanProcessorType: "batch",
		SpanBatchConfig: config.SpanBatchConfig{
			MaxQueueSize:       8192,
			MaxExportBatchSize: 1024,
			BatchTimeout:       3,
		},
	}

	provider, err := NewProvider(WithContext(ctx), WithConfig(cfg))
	assert.Nil(t, err)
	assert.NotNil(t, provider)
	defer provider.Shutdown(ctx)

	// Verify provider is created successfully
	assert.Equal(t, "otel", provider.Type())

	// Create a tracer and span to verify it works
	tracer := provider.Tracer()
	assert.NotNil(t, tracer)

	_, span := tracer.Start(ctx, "test-span")
	assert.NotNil(t, span)
	span.End()

	// Force flush to ensure span is exported
	tp, ok := provider.(*traceProvider)
	assert.True(t, ok)

	if sdkProvider, ok := tp.traceProvider.(*sdktrace.TracerProvider); ok {
		err := sdkProvider.ForceFlush(ctx)
		assert.Nil(t, err)
	}
}

func TestProvider_BatchConfigValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Setup HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("warning when MaxExportBatchSize > MaxQueueSize", func(t *testing.T) {
		cfg := &config.OpenTelemetry{
			Enabled: true,
			ExporterConfig: config.ExporterConfig{
				Exporter:          "http",
				Endpoint:          server.URL,
				ConnectionTimeout: 10,
			},
			SpanProcessorType: "batch",
			SpanBatchConfig: config.SpanBatchConfig{
				MaxQueueSize:       512,
				MaxExportBatchSize: 1024, // Greater than MaxQueueSize
				BatchTimeout:       3,
			},
		}

		// Create provider with mock logger
		provider, err := NewProvider(
			WithContext(ctx),
			WithConfig(cfg),
			WithLogger(&testLogger{t: t, expectWarning: true}),
		)
		assert.Nil(t, err)
		assert.NotNil(t, provider)
		defer provider.Shutdown(ctx)

		// Verify provider is created successfully despite warning
		assert.Equal(t, "otel", provider.Type())
	})

	t.Run("no warning when MaxExportBatchSize <= MaxQueueSize", func(t *testing.T) {
		cfg := &config.OpenTelemetry{
			Enabled: true,
			ExporterConfig: config.ExporterConfig{
				Exporter:          "http",
				Endpoint:          server.URL,
				ConnectionTimeout: 10,
			},
			SpanProcessorType: "batch",
			SpanBatchConfig: config.SpanBatchConfig{
				MaxQueueSize:       2048,
				MaxExportBatchSize: 512, // Less than MaxQueueSize
				BatchTimeout:       3,
			},
		}

		provider, err := NewProvider(
			WithContext(ctx),
			WithConfig(cfg),
			WithLogger(&testLogger{t: t, expectWarning: false}),
		)
		assert.Nil(t, err)
		assert.NotNil(t, provider)
		defer provider.Shutdown(ctx)

		assert.Equal(t, "otel", provider.Type())
	})
}

// testLogger is a simple logger for testing that tracks warnings
type testLogger struct {
	t             *testing.T
	expectWarning bool
	gotWarning    bool
}

func (l *testLogger) Info(args ...interface{}) {
	msg := ""
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			msg += s
		}
	}

	// Check if this is a warning message
	if len(msg) > 0 && (msg[:7] == "Warning" || msg[:8] == "Warning:") {
		l.gotWarning = true
		if !l.expectWarning {
			l.t.Errorf("Unexpected warning: %s", msg)
		}
	}
}

func (l *testLogger) Error(args ...interface{}) {
	// Not used in this test
}
