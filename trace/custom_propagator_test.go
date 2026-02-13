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
			name:          "valid hex trace ID",
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
			name:          "short correlation ID with padding",
			headerName:    "X-Correlation-ID",
			headerValue:   "abc123",
			expectValid:   true,
			expectTraceID: "abc12300000000000000000000000000",
			expectSpanID:  "abc1230000000000",
			expectSampled: true,
		},
		{
			name:          "arbitrary correlation ID",
			headerName:    "X-Request-ID",
			headerValue:   "request-abc-123",
			expectValid:   true,
			expectTraceID: "eeabc123000000000000000000000000", // 'e' from 'request', 'e' again, then 'abc123', padded
			expectSpanID:  "eeabc12300000000",                 // First 16 chars
			expectSampled: true,
		},
		{
			name:        "empty header value",
			headerName:  "X-Correlation-ID",
			headerValue: "",
			expectValid: false,
		},
		{
			name:        "only non-hex characters",
			headerName:  "X-Correlation-ID",
			headerValue: "xyz-ghi-jkl",
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
			name:         "inject enabled with sampled trace but no original value",
			headerName:   "X-Correlation-ID",
			inject:       true,
			traceID:      "0102030405060708090a0b0c0d0e0f10",
			spanID:       "1112131415161718",
			sampled:      true,
			expectHeader: false,
		},
		{
			name:         "inject enabled with non-sampled trace but no original value",
			headerName:   "X-Request-ID",
			inject:       true,
			traceID:      "0102030405060708090a0b0c0d0e0f10",
			spanID:       "1112131415161718",
			sampled:      false,
			expectHeader: false,
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

func TestCustomHeaderPropagator_Inject_RemovesPassthroughHeader(t *testing.T) {
	// This test simulates the hybrid mode scenario where the reverse proxy
	// clones all incoming request headers to the outgoing request. When
	// inject=false (hybrid mode), the propagator must actively remove the
	// custom header from the carrier so only standard headers reach upstream.
	propagator := NewCustomHeaderPropagator("X-Correlation-ID", false)

	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("1112131415161718")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	// Pre-populate the carrier with the custom header (simulating reverse proxy clone)
	carrier := propagation.HeaderCarrier(http.Header{})
	carrier.Set("X-Correlation-ID", "45aa334455667788aabbccddeeff11bb")

	// Verify the header is present before injection
	assert.Equal(t, "45aa334455667788aabbccddeeff11bb", carrier.Get("X-Correlation-ID"))

	// Inject should remove the custom header when inject=false
	propagator.Inject(ctx, carrier)

	// The custom header must be gone
	assert.Empty(t, carrier.Get("X-Correlation-ID"), "custom header should be removed in hybrid mode")
}

func TestCustomHeaderPropagator_HybridModeEndToEnd(t *testing.T) {
	// End-to-end test for hybrid mode: extract from custom header, inject only traceparent.
	// This simulates: incoming request with custom header → gateway → upstream with only traceparent.
	customPropagator := NewCustomHeaderPropagator("X-Correlation-ID", false)
	standardPropagator := propagation.TraceContext{}
	composite := propagation.NewCompositeTextMapPropagator(customPropagator, standardPropagator)

	// Simulate incoming request with custom header
	incomingCarrier := propagation.HeaderCarrier(http.Header{})
	incomingCarrier.Set("X-Correlation-ID", "45aa334455667788aabbccddeeff11bb")

	// Extract trace context from incoming request
	ctx := composite.Extract(context.Background(), incomingCarrier)
	sc := trace.SpanContextFromContext(ctx)
	assert.True(t, sc.IsValid(), "should extract valid span context from custom header")
	assert.Equal(t, "45aa334455667788aabbccddeeff11bb", sc.TraceID().String())

	// Simulate outgoing request: reverse proxy clones ALL headers from incoming request
	outgoingCarrier := propagation.HeaderCarrier(http.Header{})
	outgoingCarrier.Set("X-Correlation-ID", "45aa334455667788aabbccddeeff11bb") // cloned by proxy

	// Inject trace context into outgoing request
	composite.Inject(ctx, outgoingCarrier)

	// Verify: only traceparent should be present, custom header should be removed
	assert.Empty(t, outgoingCarrier.Get("X-Correlation-ID"),
		"custom header must NOT reach upstream in hybrid mode")
	assert.NotEmpty(t, outgoingCarrier.Get("traceparent"),
		"traceparent must be set in hybrid mode")
	assert.Contains(t, outgoingCarrier.Get("traceparent"), "45aa334455667788aabbccddeeff11bb",
		"traceparent must contain the same trace ID extracted from custom header")
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
		name           string
		headerName     string
		inject         bool
		originalValue  string
		expectInjected string
		expectTraceID  string
		expectSpanID   string
	}{
		{
			name:           "round trip with inject enabled - preserves original value",
			headerName:     "X-Correlation-ID",
			inject:         true,
			originalValue:  "request-abc-123",
			expectInjected: "request-abc-123",                  // Original value preserved
			expectTraceID:  "eeabc123000000000000000000000000", // Normalised for OTel: 'e' from 'request', 'e' again, then 'abc123', padded
			expectSpanID:   "eeabc12300000000",                 // First 16 chars of normalised trace ID
		},
		{
			name:           "round trip with valid hex trace ID",
			headerName:     "X-Correlation-ID",
			inject:         true,
			originalValue:  "0102030405060708090a0b0c0d0e0f10",
			expectInjected: "0102030405060708090a0b0c0d0e0f10", // Original preserved
			expectTraceID:  "0102030405060708090a0b0c0d0e0f10",
			expectSpanID:   "0102030405060708",
		},
		{
			name:           "round trip with UUID format",
			headerName:     "X-Request-ID",
			inject:         true,
			originalValue:  "550e8400-e29b-41d4-a716-446655440000",
			expectInjected: "550e8400-e29b-41d4-a716-446655440000", // Original preserved with dashes
			expectTraceID:  "550e8400e29b41d4a716446655440000",     // Normalised (dashes removed)
			expectSpanID:   "550e8400e29b41d4",
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

			if tt.inject && tt.originalValue != "" {
				// Set up carrier with original value
				incomingCarrier := propagation.HeaderCarrier(http.Header{})
				incomingCarrier.Set(tt.headerName, tt.originalValue)

				// Extract
				extractedCtx := propagator.Extract(context.Background(), incomingCarrier)
				extractedSC := trace.SpanContextFromContext(extractedCtx)

				// Verify extraction worked
				assert.True(t, extractedSC.IsValid(), "expected valid span context after extraction")
				assert.Equal(t, tt.expectTraceID, extractedSC.TraceID().String(), "trace ID mismatch")
				assert.Equal(t, tt.expectSpanID, extractedSC.SpanID().String(), "span ID mismatch")

				// Inject to new carrier
				outgoingCarrier := propagation.HeaderCarrier(http.Header{})
				propagator.Inject(extractedCtx, outgoingCarrier)

				// Verify original value is preserved
				injectedValue := outgoingCarrier.Get(tt.headerName)
				assert.Equal(t, tt.expectInjected, injectedValue, "injected value should match original")
			} else if !tt.inject {
				// Test inject disabled
				originalTraceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
				originalSpanID, _ := trace.SpanIDFromHex("1112131415161718")
				originalSC := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    originalTraceID,
					SpanID:     originalSpanID,
					TraceFlags: trace.FlagsSampled,
				})

				ctx := trace.ContextWithSpanContext(context.Background(), originalSC)

				carrier := propagation.HeaderCarrier(http.Header{})
				propagator.Inject(ctx, carrier)

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
