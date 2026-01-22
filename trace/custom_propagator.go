package trace

import (
	"context"
	"encoding/hex"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

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
// This writes the trace context to the custom header if inject is enabled.
func (p *CustomHeaderPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if !p.inject {
		return
	}

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}

	// Format: traceID-spanID-flags
	// This is a simplified format similar to B3 single header
	traceID := sc.TraceID().String()
	spanID := sc.SpanID().String()
	flags := "01" // sampled
	if !sc.IsSampled() {
		flags = "00"
	}

	value := traceID + "-" + spanID + "-" + flags
	carrier.Set(p.traceHeader, value)
}

// Extract reads cross-cutting concerns from the carrier into a Context.
// This reads the trace context from the custom header.
func (p *CustomHeaderPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	value := carrier.Get(p.traceHeader)
	if value == "" {
		return ctx
	}

	sc := p.parseTraceContext(value)
	if !sc.IsValid() {
		return ctx
	}

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
// Supports multiple formats:
// 1. traceID-spanID-flags (our custom format)
// 2. traceID-spanID (without flags, assumes sampled)
// 3. traceID only (generates a new spanID, assumes sampled)
// 4. UUID format (uses as traceID, generates spanID)
func (p *CustomHeaderPropagator) parseTraceContext(value string) trace.SpanContext {
	parts := strings.Split(value, "-")

	var traceIDStr, spanIDStr string
	sampled := true

	switch len(parts) {
	case 1:
		// Just a trace ID (or UUID)
		traceIDStr = p.normaliseTraceID(parts[0])
		spanIDStr = "" // Will generate a new span ID
	case 2:
		// traceID-spanID
		traceIDStr = p.normaliseTraceID(parts[0])
		spanIDStr = p.normaliseSpanID(parts[1])
	case 3, 4, 5:
		// Could be traceID-spanID-flags or UUID format (8-4-4-4-12)
		if len(parts) == 5 && len(parts[0]) == 8 && len(parts[1]) == 4 {
			// UUID format: 8-4-4-4-12
			traceIDStr = p.normaliseTraceID(strings.Join(parts, ""))
			spanIDStr = ""
		} else {
			// traceID-spanID-flags
			traceIDStr = p.normaliseTraceID(parts[0])
			spanIDStr = p.normaliseSpanID(parts[1])
			if parts[2] == "00" || strings.ToLower(parts[2]) == "false" {
				sampled = false
			}
		}
	default:
		return trace.SpanContext{}
	}

	// Parse trace ID
	traceID, err := trace.TraceIDFromHex(traceIDStr)
	if err != nil {
		return trace.SpanContext{}
	}

	// Parse or generate span ID
	var spanID trace.SpanID
	if spanIDStr != "" {
		spanID, err = trace.SpanIDFromHex(spanIDStr)
		if err != nil {
			return trace.SpanContext{}
		}
	} else {
		// Generate a new span ID from the first 16 chars of trace ID
		if len(traceIDStr) >= 16 {
			spanID, _ = trace.SpanIDFromHex(traceIDStr[:16])
		} else {
			return trace.SpanContext{}
		}
	}

	// Build trace flags
	var flags trace.TraceFlags
	if sampled {
		flags = trace.FlagsSampled
	}

	config := trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
		Remote:     true,
	}

	return trace.NewSpanContext(config)
}

// normaliseTraceID normalises a trace ID to 32 hex characters.
// Handles UUIDs by removing dashes and padding/truncating as needed.
func (p *CustomHeaderPropagator) normaliseTraceID(id string) string {
	// Remove dashes (for UUID format)
	id = strings.ReplaceAll(id, "-", "")

	// Remove any non-hex characters
	id = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			return r
		}
		return -1
	}, id)

	id = strings.ToLower(id)

	// Pad or truncate to 32 characters
	if len(id) < 32 {
		id = id + strings.Repeat("0", 32-len(id))
	} else if len(id) > 32 {
		id = id[:32]
	}

	// Validate it's valid hex
	if _, err := hex.DecodeString(id); err != nil {
		return ""
	}

	return id
}

// normaliseSpanID normalises a span ID to 16 hex characters.
func (p *CustomHeaderPropagator) normaliseSpanID(id string) string {
	id = strings.ReplaceAll(id, "-", "")

	// Remove any non-hex characters
	id = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			return r
		}
		return -1
	}, id)

	id = strings.ToLower(id)

	// Pad or truncate to 16 characters
	if len(id) < 16 {
		id = id + strings.Repeat("0", 16-len(id))
	} else if len(id) > 16 {
		id = id[:16]
	}

	// Validate it's valid hex
	if _, err := hex.DecodeString(id); err != nil {
		return ""
	}

	return id
}
