package metric

import (
	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/internal/attributes"
)

// NewAttribute builds a resource or instrument attribute without importing the
// OTel attribute package. It supports the basic types (string, bool, int,
// int64, float64), their pointer forms, slices of the basic types, and any
// fmt.Stringer; anything else is rendered with fmt.Sprint and stored as a
// string. It shares its implementation with trace.NewAttribute.
//
// Example:
//
//	provider, err := metric.NewProvider(
//		metric.WithCustomResourceAttributes(
//			metric.NewAttribute("tyk.component", "dashboard"),
//			metric.NewAttribute("shard", 3),
//		),
//	)
func NewAttribute(key string, value interface{}) Attribute {
	return attributes.New(key, value)
}

// StringAttribute builds a string attribute. Prefer it over NewAttribute when
// the value type is known.
//
// Example:
//
//	counter.Add(ctx, 1, metric.StringAttribute("http.request.method", "GET"))
func StringAttribute(key, value string) Attribute {
	return attribute.String(key, value)
}

// IntAttribute builds an integer attribute. Prefer it over NewAttribute when
// the value type is known.
//
// Example:
//
//	counter.Add(ctx, 1, metric.IntAttribute("http.response.status_code", 200))
func IntAttribute(key string, value int) Attribute {
	return attribute.Int(key, value)
}

// BoolAttribute builds a boolean attribute. Prefer it over NewAttribute when
// the value type is known.
//
// Example:
//
//	provider, err := metric.NewProvider(
//		metric.WithCustomResourceAttributes(metric.BoolAttribute("tyk.hybrid", true)),
//	)
func BoolAttribute(key string, value bool) Attribute {
	return attribute.Bool(key, value)
}
