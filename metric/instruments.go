package metric

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// Counter is a nil-safe wrapper around an OpenTelemetry Int64Counter.
// It provides a simple Add method that handles nil and disabled states gracefully.
type Counter struct {
	counter otelmetric.Int64Counter
	enabled bool
}

// Add increments the counter by the given value with the provided attributes.
// It is safe to call on a nil or disabled Counter.
func (c *Counter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	if c == nil || !c.enabled {
		return
	}
	c.counter.Add(ctx, value, otelmetric.WithAttributes(attrs...))
}

// Enabled returns whether the counter is enabled and recording.
func (c *Counter) Enabled() bool {
	return c != nil && c.enabled
}

// Histogram is a nil-safe wrapper around an OpenTelemetry Float64Histogram.
// It provides a simple Record method that handles nil and disabled states gracefully.
type Histogram struct {
	histogram otelmetric.Float64Histogram
	enabled   bool
}

// Record records a value in the histogram with the provided attributes.
// It is safe to call on a nil or disabled Histogram.
func (h *Histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	if h == nil || !h.enabled {
		return
	}
	h.histogram.Record(ctx, value, otelmetric.WithAttributes(attrs...))
}

// Enabled returns whether the histogram is enabled and recording.
func (h *Histogram) Enabled() bool {
	return h != nil && h.enabled
}

// DefaultLatencyBuckets provides sensible default bucket boundaries for latency histograms in milliseconds.
var DefaultLatencyBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
