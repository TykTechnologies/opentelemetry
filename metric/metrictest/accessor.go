package metrictest

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// GaugeValue returns the value of a gauge metric's single data point, so tests
// can apply their own comparison (greater than zero, strictly increasing
// between two collections, within a tolerance) without touching metricdata
// types. Fails the test if m is not a Gauge[N] or does not have exactly one
// data point. Gauges created through this library are Gauge[float64].
//
//	first := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
//	second := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
//	assert.Greater(t, second, first)
func GaugeValue[N int64 | float64](t testing.TB, m metricdata.Metrics) N {
	t.Helper()
	var zero N

	data, ok := m.Data.(metricdata.Gauge[N])
	if !ok {
		t.Fatalf("metric %q: expected Gauge[%T] data, got %T", m.Name, zero, m.Data)
		return zero // unreachable with a real testing.TB
	}
	if len(data.DataPoints) != 1 {
		t.Fatalf("metric %q: expected exactly 1 gauge data point, got %d (use DataPointValues for several)",
			m.Name, len(data.DataPoints))
		return zero // unreachable with a real testing.TB
	}
	return data.DataPoints[0].Value
}

// DataPointValues returns the value of every data point of a Sum (counter,
// up-down counter) or Gauge metric, one per attribute combination, in the
// order the SDK reports them. Fails the test if m is not a Sum[N] or Gauge[N].
// Histograms have no single value per point; use AssertHistogramCount and
// AssertHistogramSum instead.
//
//	values := metrictest.DataPointValues[int64](t, tp.FindMetric(t, "http.requests"))
//	// one value per method label
func DataPointValues[N int64 | float64](t testing.TB, m metricdata.Metrics) []N {
	t.Helper()
	var zero N

	switch data := m.Data.(type) {
	case metricdata.Sum[N]:
		return dataPointValues(data.DataPoints)
	case metricdata.Gauge[N]:
		return dataPointValues(data.DataPoints)
	default:
		t.Fatalf("metric %q: expected Sum[%T] or Gauge[%T] data, got %T", m.Name, zero, zero, m.Data)
		return nil // unreachable with a real testing.TB
	}
}

// dataPointValues extracts the values from a slice of data points.
func dataPointValues[N int64 | float64](dps []metricdata.DataPoint[N]) []N {
	values := make([]N, len(dps))
	for i, dp := range dps {
		values[i] = dp.Value
	}
	return values
}
