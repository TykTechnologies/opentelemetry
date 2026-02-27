package metrictest_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/metric/metrictest"
)

func TestNewProvider(t *testing.T) {
	tp := metrictest.NewProvider(t)
	if !tp.Enabled() {
		t.Fatal("expected provider to be enabled")
	}
	if tp.Type() != "otel" {
		t.Fatalf("expected type otel, got %s", tp.Type())
	}
}

func TestCounter_AssertSum(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("test.counter", "A test counter", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 3)
	counter.Add(ctx, 7)

	m := tp.FindMetric(t, "test.counter")
	metrictest.AssertSum(t, m, int64(10))
}

func TestCounter_AssertSumWithAttrs(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("test.counter.attrs", "A test counter", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 5, attribute.String("method", "GET"))
	counter.Add(ctx, 3, attribute.String("method", "POST"))

	m := tp.FindMetric(t, "test.counter.attrs")
	metrictest.AssertSum(t, m, int64(8))
	metrictest.AssertSumWithAttrs(t, m, int64(5), attribute.String("method", "GET"))
	metrictest.AssertSumWithAttrs(t, m, int64(3), attribute.String("method", "POST"))
}

func TestHistogram_AssertCountAndSum(t *testing.T) {
	tp := metrictest.NewProvider(t)
	hist, err := tp.NewHistogram("test.histogram", "A test histogram", "ms", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	hist.Record(ctx, 50.0)
	hist.Record(ctx, 150.0)
	hist.Record(ctx, 100.0)

	m := tp.FindMetric(t, "test.histogram")
	metrictest.AssertHistogramCount(t, m, uint64(3))
	metrictest.AssertHistogramSum(t, m, 300.0)
}

func TestGauge_AssertGauge(t *testing.T) {
	tp := metrictest.NewProvider(t)
	gauge, err := tp.NewGauge("test.gauge", "A test gauge", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	gauge.Record(ctx, 42.0)

	m := tp.FindMetric(t, "test.gauge")
	metrictest.AssertGauge(t, m, 42.0)
}

func TestUpDownCounter_AssertSum(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewUpDownCounter("test.updown", "A test updown counter", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 5)
	counter.Add(ctx, -2)

	m := tp.FindMetric(t, "test.updown")
	metrictest.AssertSum(t, m, int64(3))
}

func TestAssertHasAttributes(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("test.attrs", "A test counter", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 1,
		attribute.String("key1", "val1"),
		attribute.String("key2", "val2"),
		attribute.Int("key3", 42),
	)

	m := tp.FindMetric(t, "test.attrs")
	// Subset match — only checking key1 and key3.
	metrictest.AssertHasAttributes(t, m,
		attribute.String("key1", "val1"),
		attribute.Int("key3", 42),
	)
}

func TestAssertDataPointCount(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("test.dp.count", "A test counter", "1")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 1, attribute.String("method", "GET"))
	counter.Add(ctx, 2, attribute.String("method", "POST"))
	counter.Add(ctx, 3, attribute.String("method", "DELETE"))

	m := tp.FindMetric(t, "test.dp.count")
	metrictest.AssertDataPointCount(t, m, 3)
}

func TestMetricNames(t *testing.T) {
	tp := metrictest.NewProvider(t)

	ctx := context.Background()

	c1, err := tp.NewCounter("counter.a", "Counter A", "1")
	if err != nil {
		t.Fatal(err)
	}
	c1.Add(ctx, 1)
	c2, err := tp.NewCounter("counter.b", "Counter B", "1")
	if err != nil {
		t.Fatal(err)
	}
	c2.Add(ctx, 1)
	h1, err := tp.NewHistogram("hist.a", "Hist A", "ms", nil)
	if err != nil {
		t.Fatal(err)
	}
	h1.Record(ctx, 1.0)

	names := tp.MetricNames()
	if len(names) < 3 {
		t.Fatalf("expected at least 3 metric names, got %d: %v", len(names), names)
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"counter.a", "counter.b", "hist.a"} {
		if !nameSet[expected] {
			t.Errorf("expected metric %q in names %v", expected, names)
		}
	}
}

func TestParallelProviders(t *testing.T) {
	t.Run("provider_a", func(t *testing.T) {
		t.Parallel()
		tp := metrictest.NewProvider(t)
		counter, err := tp.NewCounter("parallel.a", "Counter A", "1")
		if err != nil {
			t.Fatal(err)
		}
		counter.Add(context.Background(), 10)

		m := tp.FindMetric(t, "parallel.a")
		metrictest.AssertSum(t, m, int64(10))
	})

	t.Run("provider_b", func(t *testing.T) {
		t.Parallel()
		tp := metrictest.NewProvider(t)
		counter, err := tp.NewCounter("parallel.b", "Counter B", "1")
		if err != nil {
			t.Fatal(err)
		}
		counter.Add(context.Background(), 20)

		m := tp.FindMetric(t, "parallel.b")
		metrictest.AssertSum(t, m, int64(20))
	})
}
