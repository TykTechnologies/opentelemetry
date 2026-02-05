package metric

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// Metric names following OpenTelemetry semantic conventions.
	metricRequestTotal     = "http.server.request.total"
	metricRequestErrors    = "http.server.request.errors"
	metricRequestDuration  = "http.server.request.duration"
	metricGatewayLatency   = "tyk.gateway.latency"
	metricUpstreamLatency  = "tyk.upstream.latency"

	// Unit definitions.
	unitDimensionless = "1"
	unitMilliseconds  = "ms"
)

// Latency holds timing breakdown for a request in milliseconds.
type Latency struct {
	// Total is the end-to-end latency from request receipt to response completion.
	Total float64
	// Gateway is the time spent in gateway processing (Total - Upstream).
	Gateway float64
	// Upstream is the time spent waiting for the upstream response.
	Upstream float64
}

// Attributes holds metric labels for RED metrics.
type Attributes struct {
	// APIID is the unique identifier for the API.
	APIID string
	// APIName is the human-readable name of the API.
	APIName string
	// OrgID is the organization identifier.
	OrgID string
	// Method is the HTTP method (GET, POST, etc.).
	Method string
	// Path is the API listen path.
	Path string
	// ResponseCode is the HTTP response status code.
	ResponseCode int
}

// Recorder is the common interface for recording RED metrics.
// Handlers call Record() with timing data - this is the single integration point.
type Recorder struct {
	requestCounter  metric.Int64Counter
	errorCounter    metric.Int64Counter
	totalLatency    metric.Float64Histogram
	gatewayLatency  metric.Float64Histogram
	upstreamLatency metric.Float64Histogram
	enabled         bool
}

// newRecorder creates a new Recorder with the given meter.
func newRecorder(meter metric.Meter) (*Recorder, error) {
	requestCounter, err := meter.Int64Counter(
		metricRequestTotal,
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit(unitDimensionless),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		metricRequestErrors,
		metric.WithDescription("Total number of HTTP requests that resulted in an error (status >= 400)"),
		metric.WithUnit(unitDimensionless),
	)
	if err != nil {
		return nil, err
	}

	totalLatency, err := meter.Float64Histogram(
		metricRequestDuration,
		metric.WithDescription("Total end-to-end request latency in milliseconds"),
		metric.WithUnit(unitMilliseconds),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return nil, err
	}

	gatewayLatency, err := meter.Float64Histogram(
		metricGatewayLatency,
		metric.WithDescription("Gateway processing time in milliseconds"),
		metric.WithUnit(unitMilliseconds),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return nil, err
	}

	upstreamLatency, err := meter.Float64Histogram(
		metricUpstreamLatency,
		metric.WithDescription("Upstream response time in milliseconds"),
		metric.WithUnit(unitMilliseconds),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return nil, err
	}

	return &Recorder{
		requestCounter:  requestCounter,
		errorCounter:    errorCounter,
		totalLatency:    totalLatency,
		gatewayLatency:  gatewayLatency,
		upstreamLatency: upstreamLatency,
		enabled:         true,
	}, nil
}

// newNoopRecorder creates a recorder that does nothing.
// Used when metrics are disabled.
func newNoopRecorder() *Recorder {
	return &Recorder{
		enabled: false,
	}
}

// Record records a single request's RED metrics.
// This is the ONLY method handlers need to call.
func (r *Recorder) Record(ctx context.Context, attrs Attributes, latency Latency) {
	if r == nil || !r.enabled {
		return
	}

	// Build attributes set.
	attrSet := []attribute.KeyValue{
		attribute.String("tyk.api.id", attrs.APIID),
		attribute.String("tyk.api.name", attrs.APIName),
		attribute.String("tyk.api.org_id", attrs.OrgID),
		attribute.String("http.request.method", attrs.Method),
		attribute.String("http.route", attrs.Path),
		attribute.Int("http.response.status_code", attrs.ResponseCode),
	}

	// Record request count (Rate).
	r.requestCounter.Add(ctx, 1, metric.WithAttributes(attrSet...))

	// Record error count (Errors) if status >= 400.
	if attrs.ResponseCode >= 400 {
		r.errorCounter.Add(ctx, 1, metric.WithAttributes(attrSet...))
	}

	// Record duration metrics (Duration).
	r.totalLatency.Record(ctx, latency.Total, metric.WithAttributes(attrSet...))
	r.gatewayLatency.Record(ctx, latency.Gateway, metric.WithAttributes(attrSet...))
	r.upstreamLatency.Record(ctx, latency.Upstream, metric.WithAttributes(attrSet...))
}

// Enabled returns whether the recorder is enabled.
func (r *Recorder) Enabled() bool {
	return r != nil && r.enabled
}
