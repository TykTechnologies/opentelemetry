package trace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// customHeaderContextKey is used to store the original header value in context
type customHeaderContextKey struct{}

// CustomHeaderPropagator implements the OpenTelemetry TextMapPropagator interface
// to handle custom trace headers (e.g., X-Correlation-ID, X-Request-ID).
type CustomHeaderPropagator struct {
	traceHeader string // Custom header name (e.g., "X-Correlation-ID")
	inject      bool   // Whether to inject the custom header on outbound requests
}

// NewCustomHeaderPropagator creates a new custom header propagator.
func NewCustomHeaderPropagator(traceHeader string, inject bool) *CustomHeaderPropagator {
	return &CustomHeaderPropagator{
		traceHeader: traceHeader,
		inject:      inject,
	}
}

// Inject sets cross-cutting concerns from the Context into the carrier.
// This writes the original header value back to the custom header if inject is enabled.
// The original value is preserved to maintain compatibility with legacy correlation ID systems
// and to ensure log correlation works across services.
func (p *CustomHeaderPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if !p.inject {
		return
	}

	// Try to get the original header value from context
	if originalValue, ok := ctx.Value(customHeaderContextKey{}).(string); ok && originalValue != "" {
		// Inject the original value unchanged to preserve correlation IDs for logging
		carrier.Set(p.traceHeader, originalValue)
	}
}

// Extract reads cross-cutting concerns from the carrier into a Context.
// This reads the trace context from the custom header and stores the original value
// in the context so it can be injected unchanged to downstream services.
func (p *CustomHeaderPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	originalValue := carrier.Get(p.traceHeader)
	if originalValue == "" {
		return ctx
	}

	// Parse the value and normalise it for OpenTelemetry
	sc := p.parseTraceContext(originalValue)
	if !sc.IsValid() {
		return ctx
	}

	// Store the original value in context for later injection
	ctx = context.WithValue(ctx, customHeaderContextKey{}, originalValue)

	// Store the normalised span context
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

// Fields returns the keys whose values are set with Inject.
func (p *CustomHeaderPropagator) Fields() []string {
	if !p.inject {
		return []string{}
	}
	return []string{p.traceHeader}
}

// parseTraceContext parses the custom header value into a SpanContext.
// Accepts any correlation ID format and normalises it to a valid OpenTelemetry trace ID.
func (p *CustomHeaderPropagator) parseTraceContext(value string) trace.SpanContext {
	// Normalise the value to a valid 32-character hex trace ID
	traceIDStr := p.normaliseTraceID(value)
	if traceIDStr == "" {
		return trace.SpanContext{}
	}

	// Parse trace ID
	traceID, err := trace.TraceIDFromHex(traceIDStr)
	if err != nil {
		return trace.SpanContext{}
	}

	// Generate span ID from the first 16 characters of the trace ID
	spanID, err := trace.SpanIDFromHex(traceIDStr[:16])
	if err != nil {
		return trace.SpanContext{}
	}

	// Create span context (always sampled, always remote)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
}

// normaliseTraceID normalises a trace ID to 32 hex characters.
// Handles UUIDs by removing dashes and padding/truncating as needed.
func (p *CustomHeaderPropagator) normaliseTraceID(id string) string {
	return p.normaliseHexID(id, 32)
}

// isValidHex reports whether s is a non-empty string containing only hex characters.
func isValidHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// normaliseHexID normalises an ID to the specified length of hex characters.
// For valid hex strings (including UUIDs after dash removal): lowercase, pad/truncate.
// For non-hex strings: SHA-256 hash the original input to produce a deterministic hex ID.
func (p *CustomHeaderPropagator) normaliseHexID(id string, targetLen int) string {
	if id == "" {
		return ""
	}

	// Preserve original input for hashing before any mutation
	original := id

	// Remove dashes (for UUID format)
	stripped := strings.ReplaceAll(id, "-", "")

	if isValidHex(stripped) {
		// Valid hex path: lowercase, pad or truncate to target length
		stripped = strings.ToLower(stripped)
		if len(stripped) < targetLen {
			stripped = stripped + strings.Repeat("0", targetLen-len(stripped))
		} else if len(stripped) > targetLen {
			stripped = stripped[:targetLen]
		}
		return stripped
	}

	// Non-hex path: SHA-256 hash the original input for a deterministic, collision-resistant ID
	hash := sha256.Sum256([]byte(original))
	// Take the first targetLen/2 bytes and hex-encode to get targetLen hex chars
	numBytes := targetLen / 2
	if numBytes > len(hash) {
		numBytes = len(hash)
	}
	return hex.EncodeToString(hash[:numBytes])
}
