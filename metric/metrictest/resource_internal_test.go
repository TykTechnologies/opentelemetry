package metrictest

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

// valueString must render every attribute type the way consumers expect from
// a string map, without relying on attribute.Value.Emit (deprecated in newer
// otel releases) or Value.String (absent in older ones).
func TestValueString(t *testing.T) {
	cases := []struct {
		name string
		kv   attribute.KeyValue
		want string
	}{
		{"string", attribute.String("k", "tyk"), "tyk"},
		{"int", attribute.Int("k", 3), "3"},
		{"int64", attribute.Int64("k", 9000000000), "9000000000"},
		{"float", attribute.Float64("k", 12.5), "12.5"},
		{"bool", attribute.Bool("k", true), "true"},
		{"string slice", attribute.StringSlice("k", []string{"a", "b"}), `["a","b"]`},
		{"int slice", attribute.IntSlice("k", []int{1, 2}), "[1,2]"},
		{"bool slice", attribute.BoolSlice("k", []bool{true, false}), "[true,false]"},
		{"float slice", attribute.Float64Slice("k", []float64{1.5, 2}), "[1.5,2]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := valueString(tc.kv.Value); got != tc.want {
				t.Fatalf("valueString(%v) = %q, want %q", tc.kv.Value, got, tc.want)
			}
		})
	}
}
