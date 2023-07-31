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

func TestContextWithSpan(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.OpenTelemetry{Enabled: true}),
	)
	assert.NoError(t, err)

	_, sampleSpan := provider.Tracer().Start(context.Background(), "sample span")

	ctx := ContextWithSpan(context.Background(), sampleSpan)
	if got := SpanFromContext(ctx); got != sampleSpan {
		t.Errorf("ContextWithSpan() = %v, want %v", got, sampleSpan)
	}
}

func TestNewSpanFromContext(t *testing.T) {
	assert := assert.New(t)

	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.OpenTelemetry{Enabled: true}),
	)
	assert.NoError(err, "Failed to create new provider")

	// Creating a sample span
	ctx, originalSpan := provider.Tracer().Start(context.Background(), "sample span")

	// Table for the test cases
	tests := []struct {
		name       string
		ctx        context.Context
		tracerName string
		spanName   string
		isValid    bool
	}{
		{
			name:       "Valid context with Span and Tracer names",
			ctx:        ctx,
			tracerName: "tyk",
			spanName:   "new span",
			isValid:    true,
		},
		{
			name:       "Invalid context",
			ctx:        context.Background(),
			tracerName: "tyk",
			spanName:   "new span",
			isValid:    false,
		},
		{
			name:       "Valid context without Tracer name",
			ctx:        ctx,
			tracerName: "",
			spanName:   "new span",
			isValid:    true,
		},
		{
			name:    "Valid context without Tracer and span names",
			ctx:     ctx,
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCtx, got := NewSpanFromContext(tt.ctx, tt.tracerName, tt.spanName)

			assert.NotEqual(newCtx, tt.ctx, "NewSpanFromContext() newCtx is the same as input ctx")

			assert.NotEqual(got.SpanContext().SpanID(),
				originalSpan.SpanContext().SpanID(),
				"NewSpanFromContext() got is the same as original span")

			assert.Equal(got.SpanContext().IsValid(),
				tt.isValid,
				"NewSpanFromContext() got's span validity does not match expected validity")

			if tt.isValid {
				assert.Equal(got.SpanContext().TraceID(),
					originalSpan.SpanContext().TraceID(),
					"NewSpanFromContext() got's TraceID does not match original span's TraceID")
			}
		})
	}
}
