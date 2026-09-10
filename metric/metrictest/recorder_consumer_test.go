package metrictest_test

// This file deliberately imports nothing from go.opentelemetry.io/otel/sdk.
// It proves a consumer can test its own provider initialisation path using
// only metrictest, metric and the attribute package. Keep the import list
// minimal; TestRecorder_ConsumerImports guards it.

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"

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
		metric.WithCustomResourceAttributes(attribute.String("tyk.component", "dashboard")),
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
		attribute.String("service.instance.id", "dash-1"),
		attribute.String("service.version", "v5.9.0"),
		attribute.String("tyk.component", "dashboard"),
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
