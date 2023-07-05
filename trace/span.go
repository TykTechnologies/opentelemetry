package trace

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
