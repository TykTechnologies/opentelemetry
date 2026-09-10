package metrictest

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/metric"
)

// Recorder captures metrics from a provider that the test does not build
// itself. Pass Option() into your own initialisation code, then collect and
// assert on what that code produced — no OTel SDK imports required.
//
// Recorder registers a t.Cleanup handler that shuts the underlying reader
// down. Shutting the provider down first (as production code does) is fine.
type Recorder struct {
	reader *sdkmetric.ManualReader
}

// NewRecorder creates a Recorder backed by a ManualReader. No config, no
// exporter, no network, no global state. Safe for parallel tests.
//
//	rec := metrictest.NewRecorder(t)
//	provider, err := myotel.InitMetrics(ctx, logger, cfg, nodeID, rec.Option())
//	require.NoError(t, err)
//
//	attrs := rec.ResourceAttributes()
//	assert.Equal(t, nodeID, attrs["service.instance.id"])
func NewRecorder(t testing.TB) *Recorder {
	t.Helper()

	rec := &Recorder{reader: sdkmetric.NewManualReader()}
	t.Cleanup(func() {
		//nolint:errcheck // best-effort cleanup; the provider may already have shut the reader down
		rec.reader.Shutdown(context.Background())
	})
	return rec
}

// Option returns the metric.Option that wires this recorder into a provider.
// Pass it to the code under test alongside its usual options; it implies the
// provider is enabled and replaces the exporter, so no collector is needed.
//
//	provider, err := metric.NewProvider(metric.WithConfig(cfg), rec.Option())
func (r *Recorder) Option() metric.Option {
	return metric.WithReader(r.reader)
}

// Collect gathers all recorded metrics and returns the raw ResourceMetrics.
// Use this when you need full access to the OTel metricdata types, or when
// combining with metricdatatest.AssertEqual for exact matching.
//
//	rm := rec.Collect()
//	// inspect rm.Resource and rm.ScopeMetrics directly
func (r *Recorder) Collect() metricdata.ResourceMetrics {
	//nolint:errcheck // intentional — an unwired recorder yields empty metrics; FindMetric reports the error
	rm, _ := r.collect()
	return rm
}

// collect is Collect with the reader error preserved for better failure
// messages. The only realistic error is that Option() was never passed to a
// provider, in which case the reader has no producer to collect from.
func (r *Recorder) collect() (metricdata.ResourceMetrics, error) {
	var rm metricdata.ResourceMetrics
	err := r.reader.Collect(context.Background(), &rm)
	return rm, err
}

// FindMetric collects metrics and returns the one matching name.
// Fails the test if not found.
//
//	m := rec.FindMetric(t, "process.uptime")
func (r *Recorder) FindMetric(t testing.TB, name string) metricdata.Metrics {
	t.Helper()
	rm, err := r.collect()
	if err != nil {
		t.Fatalf("metrictest: collect failed: %v (was rec.Option() passed to metric.NewProvider?)", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found; recorded: %v", name, metricNames(rm))
	return metricdata.Metrics{} // unreachable
}

// MetricNames collects metrics and returns all recorded metric names.
// Useful for debugging when FindMetric fails or for snapshot tests.
//
//	names := rec.MetricNames()
func (r *Recorder) MetricNames() []string {
	return metricNames(r.Collect())
}

// ResourceAttributes collects metrics and returns the provider's resource
// attributes as strings keyed by attribute name — the common case for
// asserting service.name, service.instance.id, service.version and custom
// keys without importing any SDK types. See ResourceAttributeMap.
//
//	attrs := rec.ResourceAttributes()
//	assert.Equal(t, "v5.9.0", attrs["service.version"])
func (r *Recorder) ResourceAttributes() map[string]string {
	return ResourceAttributeMap(r.Collect())
}

// metricNames flattens all metric names in rm.
func metricNames(rm metricdata.ResourceMetrics) []string {
	var names []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}
