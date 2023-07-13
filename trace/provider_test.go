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

func TestSampler(t *testing.T) {
	// take a good amount of samples, so it works better with ratio based sampler
	const samples = 2000

	type testCase struct {
		name         string
		samplerName  string
		expected     int
		samplingRate float64
		parentBased  bool
		samples      int
	}

	testCases := []testCase{
		{
			name:        "basic always sample",
			samplerName: config.ALWAYSON,
			expected:    samples,
			samples:     samples,
		},
		{
			name:        "basic never sample",
			samplerName: config.ALWAYSOFF,
			expected:    0,
			samples:     samples,
		},
		{
			// it should return AlwaysOn Sampler
			name:     "all defaults",
			expected: samples,
			samples:  samples,
		},
		{
			// Should behave as AlwaysOn
			name:         "Ratio ID Based with sampling rate of 1",
			samplerName:  config.TRACEIDRATIOBASED,
			samplingRate: 1,
			expected:     samples,
			samples:      samples,
		},
		{
			// should behave as AlwaysOn
			name:         "Ratio ID Based with sampling rate of 2",
			samplerName:  config.TRACEIDRATIOBASED,
			samplingRate: 2,
			expected:     samples,
			samples:      samples,
		},
		{
			// should behave as AlwaysOff
			name:         "Ratio ID Based with negative sampling rate",
			samplerName:  config.TRACEIDRATIOBASED,
			samplingRate: -1,
			expected:     0,
			samples:      samples,
		},
		{
			name:         "Ratio ID Based with sampling rate of 50%",
			samplerName:  config.TRACEIDRATIOBASED,
			samplingRate: 0.5,
			parentBased:  true,
			expected:     samples / 2,
			samples:      samples,
		},
	}

	idGenerator := defaultIDGenerator()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sampler := getSampler(tc.samplerName, tc.samplingRate, false)
			var sampled int
			for i := 0; i < tc.samples; i++ {
				traceID, _ := idGenerator.NewIDs(context.Background())
				samplingParameters := sdktrace.SamplingParameters{TraceID: traceID}

				samplerDecision := sampler.ShouldSample(samplingParameters).Decision
				if samplerDecision == sdktrace.RecordAndSample {
					sampled++
				}
			}

			if tc.samplerName == config.TRACEIDRATIOBASED && tc.samplingRate > 0 && tc.samplingRate < 1 {
				tolerance := 0.015
				floatSamples := float64(tc.samples)
				lowLimit := floatSamples * (tc.samplingRate - tolerance)
				highLimit := floatSamples * (tc.samplingRate + tolerance)
				if float64(sampled) > highLimit || float64(sampled) < lowLimit {
					t.Errorf("number of samples is not in range. Got: %v, expected to be between %v and %v", sampled, lowLimit, highLimit)
				}
			} else {
				assert.Equal(t, tc.expected, sampled)
			}
		})
	}
}
