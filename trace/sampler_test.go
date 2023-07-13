package trace

import (
	"context"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestGetSampler(t *testing.T) {
	tests := []struct {
		name         string
		samplingType string
		samplingRate float64
		parentBased  bool
		expectedDesc string
	}{
		{"AlwaysOn", "AlwaysOn", 0, false, "AlwaysOnSampler"},
		{"lowered case alwayson", "alwayson", 0, false, "AlwaysOnSampler"},
		{"AlwaysOff", "AlwaysOff", 0, false, "AlwaysOffSampler"},
		{"lowered case alwaysoff", "alwaysoff", 0, false, "AlwaysOffSampler"},
		{"TraceIDRatioBased-0", "TraceIDRatioBased", 0, true,
			"ParentBased{root:TraceIDRatioBased{0},remoteParentSampled:AlwaysOnSampler," +
				"remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler," +
				"localParentNotSampled:AlwaysOffSampler}"},
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
					t.Errorf("number of samples is not in range."+
						" Got: %v, expected to be between %v and %v", sampled, lowLimit, highLimit)
				}
			} else {
				assert.Equal(t, tc.expected, sampled)
			}
		})
	}
}
