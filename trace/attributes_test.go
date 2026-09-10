package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

// The full type matrix is covered in internal/attributes; this guards the
// public wrapper's behaviour for a representative set of inputs.
func TestNewAttribute(t *testing.T) {
	value := "value2"
	assert.Equal(t, attribute.Key("key1").String("value1"), NewAttribute("key1", "value1"))
	assert.Equal(t, attribute.Key("key2").String("value2"), NewAttribute("key2", &value))
	assert.Equal(t, attribute.Key("key3").Int(1), NewAttribute("key3", 1))
	assert.Equal(t, attribute.Key("key4").StringSlice([]string{"a", "b"}), NewAttribute("key4", []string{"a", "b"}))
	assert.Equal(t, attribute.Key("key5").String("{}"), NewAttribute("key5", struct{}{}))
}
