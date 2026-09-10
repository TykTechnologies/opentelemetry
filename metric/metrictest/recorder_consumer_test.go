package metrictest_test

// This file deliberately imports nothing from go.opentelemetry.io/otel. It
// proves a consumer can test its own provider initialisation path using only
// metrictest and metric (plus the standard library). Keep the import list
// minimal — that is the point of the file.

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/TykTechnologies/opentelemetry/metric"
	"github.com/TykTechnologies/opentelemetry/metric/metrictest"
)

// initMetrics stands in for a consumer's own initialisation function (for
// example tyk-analytics/internal/otel.InitMetrics). It applies the identity
// options the consumer cares about and accepts extra options so tests can
// inject a metrictest.Recorder.
func initMetrics(ctx context.Context, nodeID, version string, extra ...metric.Option) (metric.Provider, error) {
	opts := []metric.Option{
		metric.WithContext(ctx),
		metric.WithServiceID(nodeID),
		metric.WithServiceVersion(version),
		metric.WithCustomResourceAttributes(metric.StringAttribute("tyk.component", "dashboard")),
	}
	opts = append(opts, extra...)

	return metric.NewProvider(opts...)
}

func TestRecorder_ConsumerInitPath(t *testing.T) {
	ctx := context.Background()
	rec := metrictest.NewRecorder(t)

	provider, err := initMetrics(ctx, "dash-1", "v5.9.0", rec.Option())
	if err != nil {
		t.Fatalf("initMetrics: %v", err)
	}
	if !provider.Enabled() {
		t.Fatal("expected provider to be enabled")
	}

	// Resource identity, as plain strings.
	attrs := rec.ResourceAttributes()
	for key, want := range map[string]string{
		"service.instance.id": "dash-1",
		"service.version":     "v5.9.0",
		"tyk.component":       "dashboard",
	} {
		if attrs[key] != want {
			t.Errorf("resource %q = %q, want %q", key, attrs[key], want)
		}
	}

	// Resource identity, typed.
	metrictest.AssertResourceAttributes(t, rec.Collect(),
		metric.StringAttribute("service.instance.id", "dash-1"),
		metric.StringAttribute("service.version", "v5.9.0"),
		metric.StringAttribute("tyk.component", "dashboard"),
	)

	// Instruments created through the consumer's provider are visible.
	uptime, err := provider.NewGauge("process.uptime", "Process uptime", "s")
	if err != nil {
		t.Fatal(err)
	}
	uptime.Record(ctx, 12.5)

	m := rec.FindMetric(t, "process.uptime")
	metrictest.AssertGauge(t, m, 12.5)

	// The consumer's own shutdown path keeps working.
	if err := provider.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// TestRecorder_ConsumerGaugeAccessors mirrors the Dashboard's process.uptime
// test: an observable gauge that must be positive and strictly increasing
// between two collections, read without touching metricdata types.
func TestRecorder_ConsumerGaugeAccessors(t *testing.T) {
	ctx := context.Background()
	rec := metrictest.NewRecorder(t)

	provider, err := initMetrics(ctx, "dash-1", "v5.9.0", rec.Option())
	if err != nil {
		t.Fatalf("initMetrics: %v", err)
	}

	// Deterministic stand-in for time.Since(start): grows on every collection.
	var ticks atomic.Int64
	_, err = provider.NewObservableGauge("process.uptime", "Process uptime", "s",
		func(_ context.Context, observe metric.Float64Observer) error {
			observe(float64(ticks.Add(1)) * 0.5)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	first := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
	if first <= 0 {
		t.Fatalf("uptime = %v, want > 0", first)
	}

	second := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
	if second <= first {
		t.Fatalf("uptime did not increase between collections: %v then %v", first, second)
	}

	// Per-data-point values for a counter split by attribute.
	requests, err := provider.NewCounter("http.requests", "Requests", "1")
	if err != nil {
		t.Fatal(err)
	}
	requests.Add(ctx, 3, metric.StringAttribute("method", "GET"))
	requests.Add(ctx, 5, metric.StringAttribute("method", "POST"))

	values := metrictest.DataPointValues[int64](t, rec.FindMetric(t, "http.requests"))
	if len(values) != 2 || values[0]+values[1] != 8 {
		t.Fatalf("request data points = %v, want two values summing to 8", values)
	}
}
