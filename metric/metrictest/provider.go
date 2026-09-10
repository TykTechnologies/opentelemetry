package metrictest

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/metric"
)

// TestProvider is a metric.Provider backed by a ManualReader for use in tests.
// It records real metric data that can be collected and asserted on.
//
// TestProvider registers a t.Cleanup handler that calls Shutdown automatically.
// To test your own initialisation code instead of a bare provider, use
// NewRecorder.
type TestProvider struct {
	metric.Provider
	rec *Recorder
}

// NewProvider creates a test provider with a ManualReader. No config, no
// exporter, no network, no global state. Safe for parallel tests.
//
//	tp := metrictest.NewProvider(t)
//	counter, _ := tp.NewCounter("hits", "Total hits", "1")
//	counter.Add(ctx, 5)
func NewProvider(t testing.TB) *TestProvider {
	t.Helper()

	rec := NewRecorder(t)
	provider, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		rec.Option(),
	)
	if err != nil {
		t.Fatalf("metrictest.NewProvider: %v", err)
	}

	tp := &TestProvider{
		Provider: provider,
		rec:      rec,
	}
	t.Cleanup(func() {
		//nolint:errcheck // best-effort cleanup in tests
		tp.Shutdown(context.Background())
	})
	return tp
}

// Collect gathers all recorded metrics and returns the raw ResourceMetrics.
// Use this when you need full access to the OTel metricdata types, or when
// combining with metricdatatest.AssertEqual for exact matching.
//
//	rm := tp.Collect()
//	// inspect rm.ScopeMetrics directly
func (tp *TestProvider) Collect() metricdata.ResourceMetrics {
	return tp.rec.Collect()
}

// FindMetric collects metrics and returns the one matching name.
// Fails the test if not found.
//
//	m := tp.FindMetric(t, "http.server.request.duration")
func (tp *TestProvider) FindMetric(t testing.TB, name string) metricdata.Metrics {
	t.Helper()
	return tp.rec.FindMetric(t, name)
}

// MetricNames collects metrics and returns all recorded metric names.
// Useful for debugging when FindMetric fails or for snapshot tests.
//
//	names := tp.MetricNames()
//	// ["http.server.request.duration", "http.server.active_requests", ...]
func (tp *TestProvider) MetricNames() []string {
	return tp.rec.MetricNames()
}

// ResourceAttributes returns the provider's resource attributes as strings
// keyed by attribute name. See ResourceAttributeMap.
//
//	attrs := tp.ResourceAttributes()
//	// attrs["service.name"] == "tyk"
func (tp *TestProvider) ResourceAttributes() map[string]string {
	return tp.rec.ResourceAttributes()
}
