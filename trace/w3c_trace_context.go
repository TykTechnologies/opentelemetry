package trace

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

// W3C Trace Context carrier keys.
const (
	w3cTraceParentKey = "traceparent"
	w3cTraceStateKey  = "tracestate"
)

// ExtractW3CTraceContext installs a W3C traceparent/tracestate (e.g. carried in
// a request body rather than the HTTP header) as the active remote span context,
// so later spans join that trace. Always W3C, ignoring the configured propagator.
func ExtractW3CTraceContext(ctx context.Context, traceParent, traceState string) context.Context {
	carrier := propagation.MapCarrier{w3cTraceParentKey: traceParent}
	if traceState != "" {
		carrier[w3cTraceStateKey] = traceState
	}
	return propagation.TraceContext{}.Extract(ctx, carrier)
}

// CurrentW3CTraceContext serialises the active span context to W3C
// traceparent/tracestate, or empty strings if no span is active. Inverse of
// ExtractW3CTraceContext.
func CurrentW3CTraceContext(ctx context.Context) (traceParent, traceState string) {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	return carrier[w3cTraceParentKey], carrier[w3cTraceStateKey]
}
