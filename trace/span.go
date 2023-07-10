package trace

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type Span trace.Span

func SpanFromContext(ctx context.Context) Span {
	return trace.SpanFromContext(ctx)
}
