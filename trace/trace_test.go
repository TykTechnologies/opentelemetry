package trace

import (
	"context"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanFromContext(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.OpenTelemetry{Enabled: true}),
	)
	assert.NoError(t, err)

	// Creating a sample span
	_, sampleSpan := provider.Tracer().Start(context.Background(), "sample span")

	// Table for the test cases
	tests := []struct {
		name string
		ctx  context.Context
		want trace.Span
	}{
		{
			"With Span",
			trace.ContextWithSpan(context.Background(), sampleSpan),
			sampleSpan,
		},
		{
			"Without Span",
			context.Background(),
			trace.SpanFromContext(context.Background()), // This will be an invalid span as no span is attached to the context
		},
		{
			"Nil Context",
			nil,
			trace.SpanFromContext(context.Background()), // This will be an invalid span as no span is attached to the context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SpanFromContext(tt.ctx); got != tt.want {
				t.Errorf("SpanFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
