package metrictest

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// ResourceAttributeMap returns the resource attributes of rm as strings keyed
// by attribute name. Non-string values are rendered with attribute.Value.Emit
// (for example attribute.Int("shard", 3) becomes "3"). Returns an empty map
// when rm carries no resource.
//
//	attrs := metrictest.ResourceAttributeMap(rec.Collect())
//	assert.Equal(t, "tyk-dashboard", attrs["service.name"])
func ResourceAttributeMap(rm metricdata.ResourceMetrics) map[string]string {
	out := make(map[string]string)
	if rm.Resource == nil {
		return out
	}
	for _, kv := range rm.Resource.Attributes() {
		out[string(kv.Key)] = kv.Value.Emit()
	}
	return out
}

// AssertResourceAttributes asserts that the resource in rm contains every
// given attribute with an equal value. Extra resource attributes are ignored
// (subset match). Works with both TestProvider.Collect and Recorder.Collect.
//
//	metrictest.AssertResourceAttributes(t, rec.Collect(),
//		attribute.String("service.instance.id", "node-1"),
//		attribute.String("service.version", "v5.9.0"),
//	)
func AssertResourceAttributes(t testing.TB, rm metricdata.ResourceMetrics, attrs ...attribute.KeyValue) {
	t.Helper()
	if rm.Resource == nil {
		t.Errorf("resource: no resource in collected metrics (was the recorder passed to the provider?)")
		return
	}
	set := rm.Resource.Set()
	for _, want := range attrs {
		got, found := set.Value(want.Key)
		if !found {
			t.Errorf("resource: attribute %q not found; have %v", want.Key, ResourceAttributeMap(rm))
			continue
		}
		if got != want.Value {
			t.Errorf("resource: attribute %q = %s, want %s", want.Key, got.Emit(), want.Value.Emit())
		}
	}
}
