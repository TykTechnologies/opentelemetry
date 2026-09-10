// Package attributes holds the attribute construction logic shared by the
// trace and metric packages, so consumers can build attribute.KeyValue pairs
// through either public package without importing the OTel attribute package
// themselves.
package attributes

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// New creates an attribute.KeyValue from key and value. It supports the basic
// types (string, bool, int, int64, float64), their pointer forms, slices of
// the basic types, and any fmt.Stringer. Any other value is rendered with
// fmt.Sprint and stored as a string.
func New(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.Key(key).String(v)
	case *string:
		return attribute.Key(key).String(*v)
	case bool:
		return attribute.Key(key).Bool(v)
	case *bool:
		return attribute.Key(key).Bool(*v)
	case int:
		return attribute.Key(key).Int(v)
	case *int:
		return attribute.Key(key).Int(*v)
	case int64:
		return attribute.Key(key).Int64(v)
	case *int64:
		return attribute.Key(key).Int64(*v)
	case float64:
		return attribute.Key(key).Float64(v)
	case *float64:
		return attribute.Key(key).Float64(*v)
	case []string:
		return attribute.Key(key).StringSlice(v)
	case []bool:
		return attribute.Key(key).BoolSlice(v)
	case []int:
		return attribute.Key(key).IntSlice(v)
	case []int64:
		return attribute.Key(key).Int64Slice(v)
	case []float64:
		return attribute.Key(key).Float64Slice(v)
	case fmt.Stringer:
		return attribute.Key(key).String(v.String())
	default:
		return attribute.Key(key).String(fmt.Sprint(v))
	}
}
