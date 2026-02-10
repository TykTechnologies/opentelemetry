package trace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	otelLib "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
			expectTraceID: "f2cc1abc17099d75e2e8e8d3cd0b885d", // SHA-256 hash of "request-abc-123"
			expectSpanID:  "f2cc1abc17099d75",                  // First 16 chars of trace ID
			expectSampled: true,
		},
		{
			name:        "empty header value",
			headerName:  "X-Correlation-ID",
			headerValue: "",
			expectValid: false,
		},
		{
			name:          "only non-hex characters",
			headerName:    "X-Correlation-ID",
			headerValue:   "xyz-ghi-jkl",
			expectValid:   true,
			expectTraceID: "cb4e6e14245cdda9e83b56db247548a4", // SHA-256 hash of "xyz-ghi-jkl"
			expectSpanID:  "cb4e6e14245cdda9",                  // First 16 chars of trace ID
			expectSampled: true,
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
			expectTraceID:  "f2cc1abc17099d75e2e8e8d3cd0b885d", // SHA-256 hash of "request-abc-123"
			expectSpanID:   "f2cc1abc17099d75",                  // First 16 chars of trace ID
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
			name:     "non-hex characters hashed",
			input:    "abc-123-xyz",
			expected: "af84f1c4d481964877e53196a5621659", // SHA-256 hash of "abc-123-xyz" (contains non-hex after dash removal)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := propagator.normaliseTraceID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomHeaderPropagator_QAScenarios(t *testing.T) {
	propagator := NewCustomHeaderPropagator("X-Correlation-ID", true)

	tests := []struct {
		name          string
		headerValue   string
		expectTraceID string
		expectSpanID  string
	}{
		{
			name:          "QA: stuffabc",
			headerValue:   "stuffabc",
			expectTraceID: "e9463449a54def0d61a12b4df1c2d180",
			expectSpanID:  "e9463449a54def0d",
		},
		{
			name:          "QA: lotsofletters",
			headerValue:   "lotsofletters",
			expectTraceID: "773fd81c8d5ef83aadbecaf1abc8b816",
			expectSpanID:  "773fd81c8d5ef83a",
		},
		{
			name:          "QA: anothertraceid",
			headerValue:   "anothertraceid",
			expectTraceID: "1913d5b305bc7f4cbaa5b288b1a637eb",
			expectSpanID:  "1913d5b305bc7f4c",
		},
		{
			name:          "QA: moretraces",
			headerValue:   "moretraces",
			expectTraceID: "73bb59d7a2742a7268cfbfe58070fe95",
			expectSpanID:  "73bb59d7a2742a72",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carrier := propagation.HeaderCarrier(http.Header{})
			carrier.Set("X-Correlation-ID", tt.headerValue)

			ctx := propagator.Extract(context.Background(), carrier)
			sc := trace.SpanContextFromContext(ctx)

			assert.True(t, sc.IsValid(), "expected valid span context for %q", tt.headerValue)
			assert.Equal(t, tt.expectTraceID, sc.TraceID().String(), "trace ID mismatch")
			assert.Equal(t, tt.expectSpanID, sc.SpanID().String(), "span ID mismatch")
			assert.True(t, sc.IsSampled(), "expected sampled")
			assert.True(t, sc.IsRemote(), "expected remote")
		})
	}
}

func TestCustomHeaderPropagator_CollisionAvoidance(t *testing.T) {
	propagator := NewCustomHeaderPropagator("X-Correlation-ID", true)

	// These inputs previously collided because stripping non-hex chars left the same residue.
	// With SHA-256 hashing, each distinct input must produce a distinct trace ID.
	inputs := []string{
		"stuff",
		"sniff",
		"stuffabc",
		"lotsofletters",
		"anothertraceid",
		"moretraces",
		"request-abc-123",
		"xyz-ghi-jkl",
	}

	seen := make(map[string]string) // traceID -> original input
	for _, input := range inputs {
		carrier := propagation.HeaderCarrier(http.Header{})
		carrier.Set("X-Correlation-ID", input)

		ctx := propagator.Extract(context.Background(), carrier)
		sc := trace.SpanContextFromContext(ctx)

		assert.True(t, sc.IsValid(), "expected valid span context for %q", input)

		traceID := sc.TraceID().String()
		if prev, exists := seen[traceID]; exists {
			t.Errorf("collision: %q and %q both produced trace ID %s", prev, input, traceID)
		}
		seen[traceID] = input
	}
}

func TestCustomHeaderPropagator_Determinism(t *testing.T) {
	propagator := NewCustomHeaderPropagator("X-Correlation-ID", true)

	// Same input must always produce the same trace ID.
	inputs := []string{"stuffabc", "request-abc-123", "xyz-ghi-jkl", "abc123", "550e8400-e29b-41d4-a716-446655440000"}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			var first string
			for i := 0; i < 5; i++ {
				carrier := propagation.HeaderCarrier(http.Header{})
				carrier.Set("X-Correlation-ID", input)

				ctx := propagator.Extract(context.Background(), carrier)
				sc := trace.SpanContextFromContext(ctx)

				assert.True(t, sc.IsValid(), "expected valid span context")

				if i == 0 {
					first = sc.TraceID().String()
				} else {
					assert.Equal(t, first, sc.TraceID().String(), "non-deterministic trace ID for %q on iteration %d", input, i)
				}
			}
		})
	}
}

func TestCustomHeaderPropagator_CompositeMode(t *testing.T) {
	// In composite mode, the custom header propagator and W3C traceparent propagator
	// run together. Verify that the custom header preserves the original value
	// while the normalised trace ID is consistent.
	customProp := NewCustomHeaderPropagator("X-Correlation-ID", true)
	composite := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent
		customProp,
	)

	originalValue := "my-service-correlation-id-42"

	// Simulate inbound request with custom header only
	inbound := propagation.HeaderCarrier(http.Header{})
	inbound.Set("X-Correlation-ID", originalValue)

	ctx := composite.Extract(context.Background(), inbound)
	sc := trace.SpanContextFromContext(ctx)

	assert.True(t, sc.IsValid(), "expected valid span context")

	// Inject into outbound carrier
	outbound := propagation.HeaderCarrier(http.Header{})
	composite.Inject(ctx, outbound)

	// Custom header should preserve original value
	assert.Equal(t, originalValue, outbound.Get("X-Correlation-ID"), "custom header should preserve original value")

	// traceparent should contain the normalised (hashed) trace ID
	traceparent := outbound.Get("traceparent")
	assert.Contains(t, traceparent, sc.TraceID().String(), "traceparent should contain the normalised trace ID")
}

func TestCustomHeaderPropagator_OtelHTTPIntegration(t *testing.T) {
	// This test simulates the real Gateway flow: otelhttp.NewHandler extracts
	// the span context using the custom propagator, creates a server span,
	// and the server span should inherit the trace ID from the extraction.
	//
	// This covers the exact code path: handler.go:42 → otelhttp.WithPropagators → Extract → Start span

	tests := []struct {
		name               string
		contextPropagation string // "custom" or "tracecontext" or "composite"
		headerName         string
		headerValue        string
		expectTraceID      string // SHA-256 hash (first 32 hex chars)
	}{
		{
			name:               "custom mode - non-hex value",
			contextPropagation: "custom",
			headerName:         "customtraceheader",
			headerValue:        "27aa334455667788aabbccddeeff11hh",
			expectTraceID:      "7150e64eb479f53060b2221e81416dcb",
		},
		{
			name:               "custom mode - QA value stuffabc",
			contextPropagation: "custom",
			headerName:         "customtraceheader",
			headerValue:        "stuffabc",
			expectTraceID:      "e9463449a54def0d61a12b4df1c2d180",
		},
		{
			name:               "custom mode - valid hex value",
			contextPropagation: "custom",
			headerName:         "customtraceheader",
			headerValue:        "0102030405060708090a0b0c0d0e0f10",
			expectTraceID:      "0102030405060708090a0b0c0d0e0f10",
		},
		{
			name:               "tracecontext mode - non-hex custom header",
			contextPropagation: "tracecontext",
			headerName:         "customtraceheader",
			headerValue:        "26aa334455667788aabbccddeeff11hh",
			expectTraceID:      "75598d42a9828ef6fbaedf06302b56a4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the tracer provider (in-memory, no exporter needed)
			tp := sdktrace.NewTracerProvider()
			defer func() { _ = tp.Shutdown(context.Background()) }()

			otelLib.SetTracerProvider(tp)

			// Build propagator based on mode (mirrors propagatorFactory logic)
			var prop propagation.TextMapPropagator
			switch tt.contextPropagation {
			case "custom":
				prop = NewCustomHeaderPropagator(tt.headerName, true)
			case "tracecontext":
				prop = propagation.NewCompositeTextMapPropagator(
					NewCustomHeaderPropagator(tt.headerName, false),
					propagation.TraceContext{},
				)
			case "composite":
				prop = propagation.NewCompositeTextMapPropagator(
					NewCustomHeaderPropagator(tt.headerName, true),
					propagation.TraceContext{},
				)
			}
			otelLib.SetTextMapPropagator(prop)

			// Variable to capture the trace ID from inside the handler
			var capturedTraceID string

			// Build the handler using the same function the Gateway uses (handler.go)
			innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				span := trace.SpanFromContext(r.Context())
				capturedTraceID = span.SpanContext().TraceID().String()
				w.WriteHeader(http.StatusOK)
			})
			wrappedHandler := NewHTTPHandler("test", innerHandler, nil)

			// Build a request with the custom header
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(tt.headerName, tt.headerValue)
			rec := httptest.NewRecorder()

			// Serve the request
			wrappedHandler.ServeHTTP(rec, req)

			// Verify the span's trace ID matches the expected value
			assert.Equal(t, tt.expectTraceID, capturedTraceID,
				"otelhttp server span should inherit trace ID from custom propagator extraction")
		})
	}
}
