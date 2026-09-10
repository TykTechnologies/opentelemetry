package metrictest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// ResourceAttributeMap returns the resource attributes of rm as strings keyed
// by attribute name. Strings are returned as-is; booleans and numbers use
// their literal form (attribute.Int("shard", 3) becomes "3"); slices are
// rendered as JSON arrays. Returns an empty map when rm carries no resource.
//
//	attrs := metrictest.ResourceAttributeMap(rec.Collect())
//	assert.Equal(t, "tyk-dashboard", attrs["service.name"])
func ResourceAttributeMap(rm metricdata.ResourceMetrics) map[string]string {
	out := make(map[string]string)
	if rm.Resource == nil {
		return out
	}
	for _, kv := range rm.Resource.Attributes() {
		out[string(kv.Key)] = valueString(kv.Value)
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
			t.Errorf("resource: attribute %q = %s, want %s", want.Key, valueString(got), valueString(want.Value))
		}
	}
}

// valueString renders an attribute value the same way attribute.Value.String
// does in otel >= 1.44 (and Value.Emit did before it): strings as-is, bools
// and numbers as literals, slices as JSON arrays. It is implemented locally so
// the library neither depends on the newer String method nor calls the
// deprecated Emit.
func valueString(v attribute.Value) string {
	switch v.Type() {
	case attribute.STRING:
		return v.AsString()
	case attribute.BOOL:
		return strconv.FormatBool(v.AsBool())
	case attribute.INT64:
		return strconv.FormatInt(v.AsInt64(), 10)
	case attribute.FLOAT64:
		return strconv.FormatFloat(v.AsFloat64(), 'g', -1, 64)
	case attribute.BOOLSLICE, attribute.INT64SLICE, attribute.FLOAT64SLICE, attribute.STRINGSLICE:
		if b, err := json.Marshal(v.AsInterface()); err == nil {
			return string(b)
		}
		return fmt.Sprint(v.AsInterface())
	default:
		return fmt.Sprint(v.AsInterface())
	}
}
