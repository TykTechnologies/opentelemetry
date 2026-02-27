package metrictest

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/metric"
)

// TestProvider is a metric.Provider backed by a ManualReader for use in tests.
// It records real metric data that can be collected and asserted on.
//
// TestProvider registers a t.Cleanup handler that calls Shutdown automatically.
type TestProvider struct {
	metric.Provider
	reader *sdkmetric.ManualReader
}

// NewProvider creates a test provider with a ManualReader. No config, no
// exporter, no network, no global state. Safe for parallel tests.
//
//	tp := metrictest.NewProvider(t)
//	counter, _ := tp.NewCounter("hits", "Total hits", "1")
//	counter.Add(ctx, 5)
func NewProvider(t testing.TB) *TestProvider {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		metric.WithReader(reader),
	)
	if err != nil {
		t.Fatalf("metrictest.NewProvider: %v", err)
	}

	tp := &TestProvider{
		Provider: provider,
		reader:   reader,
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
	var rm metricdata.ResourceMetrics
	// ManualReader.Collect does not fail under normal test conditions.
	//nolint:errcheck // intentional — ManualReader.Collect is infallible in tests
	tp.reader.Collect(context.Background(), &rm)
	return rm
}

// FindMetric collects metrics and returns the one matching name.
// Fails the test if not found.
//
//	m := tp.FindMetric(t, "http.server.request.duration")
func (tp *TestProvider) FindMetric(t testing.TB, name string) metricdata.Metrics {
	t.Helper()
	rm := tp.Collect()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found; recorded: %v", name, tp.MetricNames())
	return metricdata.Metrics{} // unreachable
}

// MetricNames collects metrics and returns all recorded metric names.
// Useful for debugging when FindMetric fails or for snapshot tests.
//
//	names := tp.MetricNames()
//	// ["http.server.request.duration", "http.server.active_requests", ...]
func (tp *TestProvider) MetricNames() []string {
	rm := tp.Collect()
	var names []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}
