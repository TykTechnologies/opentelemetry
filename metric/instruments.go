package metric

import (
	"context"
	"sync"

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

// Gauge is a nil-safe wrapper around an OpenTelemetry Float64Gauge.
// It provides a simple Record method that handles nil and disabled states gracefully.
// Use gauges for values that can go up and down, like pool sizes, queue depths, or temperatures.
type Gauge struct {
	gauge   otelmetric.Float64Gauge
	enabled bool
}

// Record records the current value of the gauge with the provided attributes.
// It is safe to call on a nil or disabled Gauge.
func (g *Gauge) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	if g == nil || !g.enabled {
		return
	}
	g.gauge.Record(ctx, value, otelmetric.WithAttributes(attrs...))
}

// Enabled returns whether the gauge is enabled and recording.
func (g *Gauge) Enabled() bool {
	return g != nil && g.enabled
}

// UpDownCounter is a nil-safe wrapper around an OpenTelemetry Int64UpDownCounter.
// It provides a simple Add method that handles nil and disabled states gracefully.
// Use up-down counters for values that can increase or decrease, like active connections or items in a queue.
type UpDownCounter struct {
	counter otelmetric.Int64UpDownCounter
	enabled bool
}

// Add increments or decrements the counter by the given value with the provided attributes.
// Positive values increment, negative values decrement.
// It is safe to call on a nil or disabled UpDownCounter.
func (u *UpDownCounter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	if u == nil || !u.enabled {
		return
	}
	u.counter.Add(ctx, value, otelmetric.WithAttributes(attrs...))
}

// Enabled returns whether the up-down counter is enabled and recording.
func (u *UpDownCounter) Enabled() bool {
	return u != nil && u.enabled
}

// Int64Observer records a single int64 observation with optional attributes.
// It is passed to Int64Callback on every collection cycle.
type Int64Observer func(value int64, attrs ...attribute.KeyValue)

// Int64Callback is invoked by the SDK once per collection cycle for
// int64-valued observable instruments. Returning an error surfaces it through
// the SDK's error path (the provider's error handler/logger) without breaking
// subsequent collections.
type Int64Callback func(ctx context.Context, observe Int64Observer) error

// ObservableCounter is a nil-safe wrapper around an OpenTelemetry
// Int64ObservableCounter. Values are reported by a callback on every
// collection cycle instead of synchronous Add calls.
// Use it for monotonically increasing values read from an external source,
// like total page faults or total bytes read from the OS.
type ObservableCounter struct {
	observable   otelmetric.Int64ObservableCounter
	registration otelmetric.Registration
	enabled      bool

	unregisterOnce sync.Once
	unregisterErr  error
}

// Enabled returns whether the observable counter is enabled and recording.
func (o *ObservableCounter) Enabled() bool {
	return o != nil && o.enabled
}

// Unregister stops the callback from being invoked on future collections.
// It is safe to call on a nil or disabled ObservableCounter and is idempotent.
func (o *ObservableCounter) Unregister() error {
	if o == nil || !o.enabled || o.registration == nil {
		return nil
	}
	o.unregisterOnce.Do(func() {
		o.unregisterErr = o.registration.Unregister()
	})
	return o.unregisterErr
}

// Float64Observer records a single float64 observation with optional attributes.
// It is passed to Float64Callback on every collection cycle.
type Float64Observer func(value float64, attrs ...attribute.KeyValue)

// Float64Callback is invoked by the SDK once per collection cycle for
// float64-valued observable instruments. Returning an error surfaces it through
// the SDK's error path (the provider's error handler/logger) without breaking
// subsequent collections.
type Float64Callback func(ctx context.Context, observe Float64Observer) error

// ObservableGauge is a nil-safe wrapper around an OpenTelemetry
// Float64ObservableGauge. The current value is reported by a callback on every
// collection cycle. Use it for sampled values like memory usage or CPU load.
type ObservableGauge struct {
	observable   otelmetric.Float64ObservableGauge
	registration otelmetric.Registration
	enabled      bool

	unregisterOnce sync.Once
	unregisterErr  error
}

// Enabled returns whether the observable gauge is enabled and recording.
func (o *ObservableGauge) Enabled() bool {
	return o != nil && o.enabled
}

// Unregister stops the callback from being invoked on future collections.
// It is safe to call on a nil or disabled ObservableGauge and is idempotent.
func (o *ObservableGauge) Unregister() error {
	if o == nil || !o.enabled || o.registration == nil {
		return nil
	}
	o.unregisterOnce.Do(func() {
		o.unregisterErr = o.registration.Unregister()
	})
	return o.unregisterErr
}

// ObservableUpDownCounter is a nil-safe wrapper around an OpenTelemetry
// Int64ObservableUpDownCounter. The current cumulative value is reported by a
// callback on every collection cycle and may go up or down, like the size of a
// queue read from an external source. It exports a non-monotonic Sum equal to
// the last observed value per attribute set.
type ObservableUpDownCounter struct {
	observable   otelmetric.Int64ObservableUpDownCounter
	registration otelmetric.Registration
	enabled      bool

	unregisterOnce sync.Once
	unregisterErr  error
}

// Enabled returns whether the observable up-down counter is enabled and recording.
func (o *ObservableUpDownCounter) Enabled() bool {
	return o != nil && o.enabled
}

// Unregister stops the callback from being invoked on future collections.
// It is safe to call on a nil or disabled ObservableUpDownCounter and is idempotent.
func (o *ObservableUpDownCounter) Unregister() error {
	if o == nil || !o.enabled || o.registration == nil {
		return nil
	}
	o.unregisterOnce.Do(func() {
		o.unregisterErr = o.registration.Unregister()
	})
	return o.unregisterErr
}

// DefaultLatencyBuckets provides default bucket boundaries for latency histograms
// in milliseconds. These buckets are suitable for API gateway latency measurement
// where most requests complete between 1ms and 10s.
// For OTel semantic convention compliance (duration in seconds), use DefaultLatencyBucketsSeconds.
var DefaultLatencyBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

// DefaultLatencyBucketsSeconds provides default bucket boundaries for latency histograms
// in seconds, following OTel HTTP semantic conventions where duration is measured in seconds.
// These are equivalent to DefaultLatencyBuckets converted from milliseconds to seconds.
var DefaultLatencyBucketsSeconds = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
