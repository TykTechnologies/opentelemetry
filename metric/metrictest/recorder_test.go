package metrictest_test

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/metric"
	"github.com/TykTechnologies/opentelemetry/metric/metrictest"
)

func TestRecorder_OptionInjectsReader(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	provider, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		rec.Option(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !provider.Enabled() {
		t.Fatal("expected provider built with the recorder to be enabled")
	}

	counter, err := provider.NewCounter("rec.counter", "A counter", "1")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(context.Background(), 4)

	m := rec.FindMetric(t, "rec.counter")
	metrictest.AssertSum(t, m, int64(4))
}

func TestRecorder_ResourceAttributes(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	_, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		metric.WithServiceID("node-1"),
		metric.WithServiceVersion("v5.0.0"),
		metric.WithCustomResourceAttributes(
			attribute.String("tyk.component", "dashboard"),
			attribute.Int("shard", 3),
		),
		rec.Option(),
	)
	if err != nil {
		t.Fatal(err)
	}

	attrs := rec.ResourceAttributes()
	want := map[string]string{
		"service.name":        "tyk", // config default
		"service.instance.id": "node-1",
		"service.version":     "v5.0.0",
		"tyk.component":       "dashboard",
		"shard":               "3",
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("resource attribute %q = %q, want %q (all: %v)", k, attrs[k], v, attrs)
		}
	}
}

func TestRecorder_MetricNames(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	provider, err := metric.NewProvider(metric.WithContext(context.Background()), rec.Option())
	if err != nil {
		t.Fatal(err)
	}

	c, err := provider.NewCounter("rec.names.a", "A", "1")
	if err != nil {
		t.Fatal(err)
	}
	c.Add(context.Background(), 1)

	names := rec.MetricNames()
	if len(names) != 1 || names[0] != "rec.names.a" {
		t.Fatalf("expected [rec.names.a], got %v", names)
	}
}

func TestRecorder_CollectExposesResource(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	_, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		metric.WithServiceID("node-2"),
		rec.Option(),
	)
	if err != nil {
		t.Fatal(err)
	}

	rm := rec.Collect()
	if rm.Resource == nil {
		t.Fatal("expected Collect to return a resource")
	}
	if got := metrictest.ResourceAttributeMap(rm)["service.instance.id"]; got != "node-2" {
		t.Fatalf("service.instance.id = %q, want node-2", got)
	}
}

func TestTestProvider_ResourceAttributes(t *testing.T) {
	tp := metrictest.NewProvider(t)

	attrs := tp.ResourceAttributes()
	if attrs["service.name"] != "tyk" {
		t.Fatalf("service.name = %q, want tyk (all: %v)", attrs["service.name"], attrs)
	}
}

func TestAssertResourceAttributes(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	_, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		metric.WithServiceID("node-1"),
		metric.WithCustomResourceAttributes(attribute.Int("shard", 3)),
		rec.Option(),
	)
	if err != nil {
		t.Fatal(err)
	}

	metrictest.AssertResourceAttributes(t, rec.Collect(),
		attribute.String("service.instance.id", "node-1"),
		attribute.Int("shard", 3),
	)
}

// recordingTB captures failures so the assertion helpers themselves can be
// tested for their negative path.
type recordingTB struct {
	testing.TB
	failures []string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recordingTB) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
}

func TestAssertResourceAttributes_FailsOnMismatch(t *testing.T) {
	rec := metrictest.NewRecorder(t)

	_, err := metric.NewProvider(
		metric.WithContext(context.Background()),
		metric.WithServiceID("node-1"),
		rec.Option(),
	)
	if err != nil {
		t.Fatal(err)
	}

	rm := rec.Collect()

	t.Run("wrong value", func(t *testing.T) {
		fake := &recordingTB{TB: t}
		metrictest.AssertResourceAttributes(fake, rm, attribute.String("service.instance.id", "other"))
		if len(fake.failures) != 1 {
			t.Fatalf("expected exactly one failure, got %v", fake.failures)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		fake := &recordingTB{TB: t}
		metrictest.AssertResourceAttributes(fake, rm, attribute.String("does.not.exist", "x"))
		if len(fake.failures) != 1 {
			t.Fatalf("expected exactly one failure, got %v", fake.failures)
		}
	})

	t.Run("match reports nothing", func(t *testing.T) {
		fake := &recordingTB{TB: t}
		metrictest.AssertResourceAttributes(fake, rm, attribute.String("service.instance.id", "node-1"))
		if len(fake.failures) != 0 {
			t.Fatalf("expected no failures, got %v", fake.failures)
		}
	})
}
