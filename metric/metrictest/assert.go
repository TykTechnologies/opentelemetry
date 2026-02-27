package metrictest

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// AssertSum asserts that a Sum (counter) metric has the expected total value
// across all data points.
//
//	m := tp.FindMetric(t, "http.server.requests")
//	metrictest.AssertSum(t, m, int64(10))
func AssertSum[N int64 | float64](t testing.TB, m metricdata.Metrics, expected N) {
	t.Helper()
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		var total int64
		for _, dp := range data.DataPoints {
			total += dp.Value
		}
		if int64(expected) != total {
			t.Errorf("metric %q: sum = %d, want %d", m.Name, total, int64(expected))
		}
	case metricdata.Sum[float64]:
		var total float64
		for _, dp := range data.DataPoints {
			total += dp.Value
		}
		if float64(expected) != total {
			t.Errorf("metric %q: sum = %f, want %f", m.Name, total, float64(expected))
		}
	default:
		t.Fatalf("metric %q: expected Sum data, got %T", m.Name, m.Data)
	}
}

// AssertSumWithAttrs asserts that a Sum metric has a data point with the
// expected value AND the given attributes.
//
//	m := tp.FindMetric(t, "http.server.requests")
//	metrictest.AssertSumWithAttrs(t, m, int64(5),
//		attribute.String("http.request.method", "GET"),
//		attribute.Int("http.response.status_code", 200),
//	)
func AssertSumWithAttrs[N int64 | float64](t testing.TB, m metricdata.Metrics, expected N, attrs ...attribute.KeyValue) {
	t.Helper()
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			if hasAttributes(dp.Attributes, attrs) && dp.Value == int64(expected) {
				return
			}
		}
		t.Errorf("metric %q: no data point with value %d and matching attributes", m.Name, int64(expected))
	case metricdata.Sum[float64]:
		for _, dp := range data.DataPoints {
			if hasAttributes(dp.Attributes, attrs) && dp.Value == float64(expected) {
				return
			}
		}
		t.Errorf("metric %q: no data point with value %f and matching attributes", m.Name, float64(expected))
	default:
		t.Fatalf("metric %q: expected Sum data, got %T", m.Name, m.Data)
	}
}

// AssertHistogramCount asserts the total observation count of a histogram.
//
//	m := tp.FindMetric(t, "http.server.request.duration")
//	metrictest.AssertHistogramCount(t, m, uint64(100))
func AssertHistogramCount(t testing.TB, m metricdata.Metrics, expected uint64) {
	t.Helper()
	data, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("metric %q: expected Histogram data, got %T", m.Name, m.Data)
	}
	var total uint64
	for _, dp := range data.DataPoints {
		total += dp.Count
	}
	if expected != total {
		t.Errorf("metric %q: histogram count = %d, want %d", m.Name, total, expected)
	}
}

// AssertHistogramSum asserts the sum of all observed values of a histogram.
//
//	m := tp.FindMetric(t, "http.server.request.duration")
//	metrictest.AssertHistogramSum(t, m, 1500.0)
func AssertHistogramSum(t testing.TB, m metricdata.Metrics, expected float64) {
	t.Helper()
	data, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("metric %q: expected Histogram data, got %T", m.Name, m.Data)
	}
	var total float64
	for _, dp := range data.DataPoints {
		total += dp.Sum
	}
	if expected != total {
		t.Errorf("metric %q: histogram sum = %f, want %f", m.Name, total, expected)
	}
}

// AssertGauge asserts that a gauge metric has the expected value.
//
//	m := tp.FindMetric(t, "system.memory.usage")
//	metrictest.AssertGauge(t, m, 1024.0)
func AssertGauge[N int64 | float64](t testing.TB, m metricdata.Metrics, expected N) {
	t.Helper()
	switch data := m.Data.(type) {
	case metricdata.Gauge[int64]:
		if len(data.DataPoints) == 0 {
			t.Fatalf("metric %q: no data points", m.Name)
		}
		if data.DataPoints[0].Value != int64(expected) {
			t.Errorf("metric %q: gauge = %d, want %d", m.Name, data.DataPoints[0].Value, int64(expected))
		}
	case metricdata.Gauge[float64]:
		if len(data.DataPoints) == 0 {
			t.Fatalf("metric %q: no data points", m.Name)
		}
		if data.DataPoints[0].Value != float64(expected) {
			t.Errorf("metric %q: gauge = %f, want %f", m.Name, data.DataPoints[0].Value, float64(expected))
		}
	default:
		t.Fatalf("metric %q: expected Gauge data, got %T", m.Name, m.Data)
	}
}

// AssertHasAttributes asserts that at least one data point in the metric
// contains all of the given attributes (subset match).
//
//	m := tp.FindMetric(t, "http.server.requests")
//	metrictest.AssertHasAttributes(t, m,
//		attribute.String("http.request.method", "POST"),
//	)
func AssertHasAttributes(t testing.TB, m metricdata.Metrics, attrs ...attribute.KeyValue) {
	t.Helper()
	for _, set := range dataPointAttributeSets(m) {
		if hasAttributes(set, attrs) {
			return
		}
	}
	t.Errorf("metric %q: no data point contains all given attributes", m.Name)
}

// AssertDataPointCount asserts the number of distinct data points (unique
// attribute combinations) recorded for a metric.
//
//	m := tp.FindMetric(t, "http.server.requests")
//	metrictest.AssertDataPointCount(t, m, 3) // 3 unique method+status combos
func AssertDataPointCount(t testing.TB, m metricdata.Metrics, expected int) {
	t.Helper()
	count := dataPointCount(m)
	if expected != count {
		t.Errorf("metric %q: data points = %d, want %d", m.Name, count, expected)
	}
}
