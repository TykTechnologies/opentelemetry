package metrictest_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/metric/metrictest"
)

func TestGaugeValue_ReturnsSingleValue(t *testing.T) {
	tp := metrictest.NewProvider(t)
	gauge, err := tp.NewGauge("acc.gauge", "A gauge", "1")
	if err != nil {
		t.Fatal(err)
	}
	gauge.Record(context.Background(), 12.5)

	got := metrictest.GaugeValue[float64](t, tp.FindMetric(t, "acc.gauge"))
	if got != 12.5 {
		t.Fatalf("GaugeValue = %v, want 12.5", got)
	}
}

func TestGaugeValue_FailsOnWrongInstrumentKind(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("acc.counter", "A counter", "1")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(context.Background(), 1)
	m := tp.FindMetric(t, "acc.counter")

	fake := &recordingTB{TB: t}
	metrictest.GaugeValue[int64](fake, m)
	if len(fake.failures) != 1 {
		t.Fatalf("expected one failure for a non-gauge metric, got %v", fake.failures)
	}
}

func TestGaugeValue_FailsOnNumericTypeMismatch(t *testing.T) {
	tp := metrictest.NewProvider(t)
	gauge, err := tp.NewGauge("acc.gauge.float", "A float gauge", "1")
	if err != nil {
		t.Fatal(err)
	}
	gauge.Record(context.Background(), 1.0)
	m := tp.FindMetric(t, "acc.gauge.float")

	fake := &recordingTB{TB: t}
	metrictest.GaugeValue[int64](fake, m) // library gauges are float64; do not silently truncate
	if len(fake.failures) != 1 {
		t.Fatalf("expected one failure for int64 on a float64 gauge, got %v", fake.failures)
	}
}

func TestGaugeValue_FailsOnMultipleDataPoints(t *testing.T) {
	tp := metrictest.NewProvider(t)
	gauge, err := tp.NewGauge("acc.gauge.multi", "A gauge", "1")
	if err != nil {
		t.Fatal(err)
	}
	gauge.Record(context.Background(), 1.0, attribute.String("pool", "a"))
	gauge.Record(context.Background(), 2.0, attribute.String("pool", "b"))
	m := tp.FindMetric(t, "acc.gauge.multi")

	fake := &recordingTB{TB: t}
	metrictest.GaugeValue[float64](fake, m)
	if len(fake.failures) != 1 {
		t.Fatalf("expected one failure for two data points, got %v", fake.failures)
	}
}

func TestDataPointValues_Sum(t *testing.T) {
	tp := metrictest.NewProvider(t)
	counter, err := tp.NewCounter("acc.sum", "A counter", "1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	counter.Add(ctx, 1, attribute.String("method", "GET"))
	counter.Add(ctx, 2, attribute.String("method", "POST"))
	counter.Add(ctx, 4, attribute.String("method", "DELETE"))

	values := metrictest.DataPointValues[int64](t, tp.FindMetric(t, "acc.sum"))
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %v", values)
	}
	var total int64
	for _, v := range values {
		total += v
	}
	if total != 7 {
		t.Fatalf("sum of values = %d, want 7 (%v)", total, values)
	}
}

func TestDataPointValues_Gauge(t *testing.T) {
	tp := metrictest.NewProvider(t)
	gauge, err := tp.NewGauge("acc.gauge.values", "A gauge", "1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	gauge.Record(ctx, 1.5, attribute.String("pool", "a"))
	gauge.Record(ctx, 2.5, attribute.String("pool", "b"))

	values := metrictest.DataPointValues[float64](t, tp.FindMetric(t, "acc.gauge.values"))
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %v", values)
	}
	if values[0]+values[1] != 4.0 {
		t.Fatalf("values = %v, want {1.5, 2.5} in any order", values)
	}
}

func TestDataPointValues_FailsOnHistogram(t *testing.T) {
	tp := metrictest.NewProvider(t)
	hist, err := tp.NewHistogram("acc.hist", "A histogram", "ms", nil)
	if err != nil {
		t.Fatal(err)
	}
	hist.Record(context.Background(), 10)
	m := tp.FindMetric(t, "acc.hist")

	fake := &recordingTB{TB: t}
	values := metrictest.DataPointValues[float64](fake, m)
	if len(fake.failures) != 1 || values != nil {
		t.Fatalf("expected one failure and nil values for a histogram, got %v / %v", fake.failures, values)
	}
}
