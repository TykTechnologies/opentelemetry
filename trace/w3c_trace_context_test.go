package trace

import (
	"context"
	"testing"
)

// A W3C traceparent (and tracestate) extracted into the context must serialise
// back out byte-identically, so a body-carried trace round-trips through the
// active span context.
func TestW3CTraceContext_RoundTrip(t *testing.T) {
	const (
		traceParent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
		traceState  = "vendor=1"
	)

	ctx := ExtractW3CTraceContext(context.Background(), traceParent, traceState)
	gotParent, gotState := CurrentW3CTraceContext(ctx)

	if gotParent != traceParent {
		t.Errorf("traceparent: got %q, want %q", gotParent, traceParent)
	}
	if gotState != traceState {
		t.Errorf("tracestate: got %q, want %q", gotState, traceState)
	}
}

// With no active span context, serialisation yields empty strings rather than a
// malformed traceparent.
func TestCurrentW3CTraceContext_EmptyWithoutSpan(t *testing.T) {
	gotParent, gotState := CurrentW3CTraceContext(context.Background())

	if gotParent != "" || gotState != "" {
		t.Errorf("expected empty, got traceparent=%q tracestate=%q", gotParent, gotState)
	}
}
