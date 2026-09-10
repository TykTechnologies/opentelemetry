package trace

import (
	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/internal/attributes"
)

type Attribute = attribute.KeyValue

// NewAttribute creates a new attribute.KeyValue pair based on the provided key and value.
// The function supports multiple types for the value parameter including
// basic types (string, bool, int, int64, float64), their pointer types, slices of basic types,
// and any type implementing the fmt.Stringer interface.
//
// Usage:
//
//	attr := trace.NewAttribute("key1", "value1")
//	fmt.Println(attr) // Output: "key1":"value1"
func NewAttribute(key string, value interface{}) Attribute {
	return attributes.New(key, value)
}
