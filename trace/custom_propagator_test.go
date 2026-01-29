package trace

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestCustomHeaderPropagator_Extract(t *testing.T) {
	tests := []struct {
		name          string
		headerName    string
		headerValue   string
		expectValid   bool
		expectTraceID string
		expectSpanID  string
		expectSampled bool
	}{
		{
			name:          "valid trace context with flags",
			headerName:    "X-Correlation-ID",
			headerValue:   "0102030405060708090a0b0c0d0e0f10-1112131415161718-01",
			expectValid:   true,
			expectTraceID: "0102030405060708090a0b0c0d0e0f10",
			expectSpanID:  "1112131415161718",
			expectSampled: true,
		},
		{
			name:          "valid trace context without flags",
			headerName:    "X-Request-ID",
			headerValue:   "0102030405060708090a0b0c0d0e0f10-1112131415161718",
			expectValid:   true,
			expectTraceID: "0102030405060708090a0b0c0d0e0f10",
			expectSpanID:  "1112131415161718",
			expectSampled: true,
		},
		{
			name:          "trace ID only",
			headerName:    "X-Trace-ID",
			headerValue:   "0102030405060708090a0b0c0d0e0f10",
			expectValid:   true,
			expectTraceID: "0102030405060708090a0b0c0d0e0f10",
			expectSpanID:  "0102030405060708",
			expectSampled: true,
		},
		{
			name:          "UUID format",
			headerName:    "X-Correlation-ID",
			headerValue:   "550e8400-e29b-41d4-a716-446655440000",
			expectValid:   true,
			expectTraceID: "550e8400e29b41d4a716446655440000",
			expectSpanID:  "550e8400e29b41d4",
			expectSampled: true,
		},
		{
			name:          "short trace ID with padding",
			headerName:    "X-Correlation-ID",
			headerValue:   "abc123",
			expectValid:   true,
			expectTraceID: "abc12300000000000000000000000000",
			expectSpanID:  "abc1230000000000",
			expectSampled: true,
		},
		{
			name:          "not sampled flag",
			headerName:    "X-Correlation-ID",
			headerValue:   "0102030405060708090a0b0c0d0e0f10-1112131415161718-00",
			expectValid:   true,
			expectTraceID: "0102030405060708090a0b0c0d0e0f10",
			expectSpanID:  "1112131415161718",
			expectSampled: false,
		},
		{
			name:        "empty header value",
			headerName:  "X-Correlation-ID",
			headerValue: "",
			expectValid: false,
		},
		{
			name:        "invalid characters",
			headerName:  "X-Correlation-ID",
			headerValue: "invalid-trace-id-with-non-hex-chars!!!",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewCustomHeaderPropagator(tt.headerName, true)

			carrier := propagation.HeaderCarrier(http.Header{})
			if tt.headerValue != "" {
				carrier.Set(tt.headerName, tt.headerValue)
			}

			ctx := propagator.Extract(context.Background(), carrier)
			sc := trace.SpanContextFromContext(ctx)

			if tt.expectValid {
				assert.True(t, sc.IsValid(), "expected valid span context")
				assert.Equal(t, tt.expectTraceID, sc.TraceID().String(), "trace ID mismatch")
				assert.Equal(t, tt.expectSpanID, sc.SpanID().String(), "span ID mismatch")
				assert.Equal(t, tt.expectSampled, sc.IsSampled(), "sampled flag mismatch")
				assert.True(t, sc.IsRemote(), "expected remote span context")
			} else {
				assert.False(t, sc.IsValid(), "expected invalid span context")
			}
		})
	}
}

func TestCustomHeaderPropagator_Inject(t *testing.T) {
	tests := []struct {
		name         string
		headerName   string
		inject       bool
		traceID      string
		spanID       string
		sampled      bool
		expectHeader bool
		expectValue  string
	}{
		{
			name:         "inject enabled with sampled trace",
			headerName:   "X-Correlation-ID",
			inject:       true,
			traceID:      "0102030405060708090a0b0c0d0e0f10",
			spanID:       "1112131415161718",
			sampled:      true,
			expectHeader: true,
			expectValue:  "0102030405060708090a0b0c0d0e0f10-1112131415161718-01",
		},
		{
			name:         "inject enabled with non-sampled trace",
			headerName:   "X-Request-ID",
			inject:       true,
			traceID:      "0102030405060708090a0b0c0d0e0f10",
			spanID:       "1112131415161718",
			sampled:      false,
			expectHeader: true,
			expectValue:  "0102030405060708090a0b0c0d0e0f10-1112131415161718-00",
		},
		{
			name:         "inject disabled",
			headerName:   "X-Correlation-ID",
			inject:       false,
			traceID:      "0102030405060708090a0b0c0d0e0f10",
			spanID:       "1112131415161718",
			sampled:      true,
			expectHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewCustomHeaderPropagator(tt.headerName, tt.inject)

			traceID, _ := trace.TraceIDFromHex(tt.traceID)
			spanID, _ := trace.SpanIDFromHex(tt.spanID)

			var flags trace.TraceFlags
			if tt.sampled {
				flags = trace.FlagsSampled
			}

			sc := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: flags,
			})

			ctx := trace.ContextWithSpanContext(context.Background(), sc)

			carrier := propagation.HeaderCarrier(http.Header{})
			propagator.Inject(ctx, carrier)

			value := carrier.Get(tt.headerName)
			if tt.expectHeader {
				assert.Equal(t, tt.expectValue, value, "injected header value mismatch")
			} else {
				assert.Empty(t, value, "expected no header to be injected")
			}
		})
	}
}

func TestCustomHeaderPropagator_Fields(t *testing.T) {
	tests := []struct {
		name         string
		headerName   string
		inject       bool
		expectFields []string
	}{
		{
			name:         "inject enabled",
			headerName:   "X-Correlation-ID",
			inject:       true,
			expectFields: []string{"X-Correlation-ID"},
		},
		{
			name:         "inject disabled",
			headerName:   "X-Correlation-ID",
			inject:       false,
			expectFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewCustomHeaderPropagator(tt.headerName, tt.inject)
			fields := propagator.Fields()
			assert.Equal(t, tt.expectFields, fields)
		})
	}
}

func TestCustomHeaderPropagator_RoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		inject     bool
	}{
		{
			name:       "round trip with inject enabled",
			headerName: "X-Correlation-ID",
			inject:     true,
		},
		{
			name:       "round trip with inject disabled (extract only)",
			headerName: "X-Request-ID",
			inject:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewCustomHeaderPropagator(tt.headerName, tt.inject)

			// Create original span context
			originalTraceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
			originalSpanID, _ := trace.SpanIDFromHex("1112131415161718")
			originalSC := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    originalTraceID,
				SpanID:     originalSpanID,
				TraceFlags: trace.FlagsSampled,
			})

			ctx := trace.ContextWithSpanContext(context.Background(), originalSC)

			// Inject
			carrier := propagation.HeaderCarrier(http.Header{})
			propagator.Inject(ctx, carrier)

			if tt.inject {
				// Extract
				extractedCtx := propagator.Extract(context.Background(), carrier)
				extractedSC := trace.SpanContextFromContext(extractedCtx)

				// Verify round trip
				assert.True(t, extractedSC.IsValid())
				assert.Equal(t, originalTraceID, extractedSC.TraceID())
				assert.Equal(t, originalSpanID, extractedSC.SpanID())
				assert.Equal(t, originalSC.IsSampled(), extractedSC.IsSampled())
			} else {
				// When inject is disabled, header should not be set
				value := carrier.Get(tt.headerName)
				assert.Empty(t, value)
			}
		})
	}
}

func TestCustomHeaderPropagator_NormaliseTraceID(t *testing.T) {
	propagator := NewCustomHeaderPropagator("X-Test", true)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 32 char hex",
			input:    "0102030405060708090a0b0c0d0e0f10",
			expected: "0102030405060708090a0b0c0d0e0f10",
		},
		{
			name:     "uppercase to lowercase",
			input:    "0102030405060708090A0B0C0D0E0F10",
			expected: "0102030405060708090a0b0c0d0e0f10",
		},
		{
			name:     "UUID with dashes",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400e29b41d4a716446655440000",
		},
		{
			name:     "short ID with padding",
			input:    "abc123",
			expected: "abc12300000000000000000000000000",
		},
		{
			name:     "long ID truncated",
			input:    "0102030405060708090a0b0c0d0e0f101112131415161718",
			expected: "0102030405060708090a0b0c0d0e0f10",
		},
		{
			name:     "non-hex characters removed",
			input:    "abc-123-xyz",
			expected: "abc12300000000000000000000000000", // xyz removed, then padded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := propagator.normaliseTraceID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomHeaderPropagator_NormaliseSpanID(t *testing.T) {
	propagator := NewCustomHeaderPropagator("X-Test", true)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 16 char hex",
			input:    "1112131415161718",
			expected: "1112131415161718",
		},
		{
			name:     "uppercase to lowercase",
			input:    "1112131415161718",
			expected: "1112131415161718",
		},
		{
			name:     "short ID with padding",
			input:    "abc123",
			expected: "abc1230000000000",
		},
		{
			name:     "long ID truncated",
			input:    "11121314151617181920",
			expected: "1112131415161718",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := propagator.normaliseSpanID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
