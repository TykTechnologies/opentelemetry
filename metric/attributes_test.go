package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewAttribute_DelegatesToSharedImplementation(t *testing.T) {
	// Representative cases; the full type matrix is covered in internal/attributes.
	assert.Equal(t, attribute.String("k", "v"), NewAttribute("k", "v"))
	assert.Equal(t, attribute.String("k", "v"), NewAttribute("k", ptr("v")))
	assert.Equal(t, attribute.Int("k", 3), NewAttribute("k", 3))
	assert.Equal(t, attribute.Bool("k", true), NewAttribute("k", true))
	assert.Equal(t, attribute.StringSlice("k", []string{"a", "b"}), NewAttribute("k", []string{"a", "b"}))
	assert.Equal(t, attribute.String("k", "{}"), NewAttribute("k", struct{}{}))
}

func TestTypedAttributeConstructors(t *testing.T) {
	assert.Equal(t, attribute.String("tyk.component", "dashboard"), StringAttribute("tyk.component", "dashboard"))
	assert.Equal(t, attribute.Int("shard", 3), IntAttribute("shard", 3))
	assert.Equal(t, attribute.Bool("hybrid", true), BoolAttribute("hybrid", true))
}

func TestAttributeConstructors_UsableAsResourceAttributes(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	_, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
		WithCustomResourceAttributes(
			StringAttribute("tyk.component", "dashboard"),
			IntAttribute("shard", 3),
			BoolAttribute("hybrid", true),
			NewAttribute("region", "eu"),
		),
	)
	assert.NoError(t, err)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))

	assertResourceAttr(t, rm.Resource, "tyk.component", "dashboard")
	assertResourceAttr(t, rm.Resource, "shard", "3")
	assertResourceAttr(t, rm.Resource, "hybrid", "true")
	assertResourceAttr(t, rm.Resource, "region", "eu")
}
