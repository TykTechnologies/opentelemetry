package metrictest

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// hasAttributes checks if attrSet contains all of the wanted attributes.
func hasAttributes(attrSet attribute.Set, wanted []attribute.KeyValue) bool {
	for _, w := range wanted {
		val, found := attrSet.Value(w.Key)
		if !found || val != w.Value {
			return false
		}
	}
	return true
}

// dataPointAttributeSets extracts attribute sets from all data points.
func dataPointAttributeSets(m metricdata.Metrics) []attribute.Set {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		sets := make([]attribute.Set, len(data.DataPoints))
		for i, dp := range data.DataPoints {
			sets[i] = dp.Attributes
		}
		return sets
	case metricdata.Sum[float64]:
		sets := make([]attribute.Set, len(data.DataPoints))
		for i, dp := range data.DataPoints {
			sets[i] = dp.Attributes
		}
		return sets
	case metricdata.Histogram[float64]:
		sets := make([]attribute.Set, len(data.DataPoints))
		for i, dp := range data.DataPoints {
			sets[i] = dp.Attributes
		}
		return sets
	case metricdata.Gauge[float64]:
		sets := make([]attribute.Set, len(data.DataPoints))
		for i, dp := range data.DataPoints {
			sets[i] = dp.Attributes
		}
		return sets
	case metricdata.Gauge[int64]:
		sets := make([]attribute.Set, len(data.DataPoints))
		for i, dp := range data.DataPoints {
			sets[i] = dp.Attributes
		}
		return sets
	}
	return nil
}

// dataPointCount returns the number of data points in a metric.
func dataPointCount(m metricdata.Metrics) int {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		return len(data.DataPoints)
	case metricdata.Sum[float64]:
		return len(data.DataPoints)
	case metricdata.Histogram[float64]:
		return len(data.DataPoints)
	case metricdata.Gauge[float64]:
		return len(data.DataPoints)
	case metricdata.Gauge[int64]:
		return len(data.DataPoints)
	}
	return 0
}
