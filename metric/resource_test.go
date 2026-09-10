package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

func assertResourceAttr(t *testing.T, res *resource.Resource, key, want string) {
	t.Helper()
	val, found := res.Set().Value(attribute.Key(key))
	if !found {
		t.Errorf("resource attribute %q not found; got %v", key, res.Attributes())
		return
	}
	assert.Equal(t, want, val.AsString(), "resource attribute %q", key)
}

func TestResourceFactory_IgnoresEnvByDefault(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=prod")

	res, err := resourceFactory(context.Background(), "tyk", resourceConfig{})
	assert.NoError(t, err)

	_, found := res.Set().Value("deployment.environment")
	assert.False(t, found, "env attributes must be opt-in")
}

func TestResourceFactory_FromEnv_AddsEnvAttributes(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=prod,team=platform")

	res, err := resourceFactory(context.Background(), "tyk", resourceConfig{fromEnv: true})
	assert.NoError(t, err)

	assertResourceAttr(t, res, "deployment.environment", "prod")
	assertResourceAttr(t, res, "team", "platform")
	assertResourceAttr(t, res, "service.name", "tyk")
}

func TestResourceFactory_FromEnv_ExplicitAttributesWin(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-env")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES",
		"service.version=env-version,service.instance.id=env-id,custom.key=env-value")

	res, err := resourceFactory(context.Background(), "explicit-name", resourceConfig{
		fromEnv:     true,
		id:          "explicit-id",
		version:     "explicit-version",
		customAttrs: []Attribute{attribute.String("custom.key", "explicit-value")},
	})
	assert.NoError(t, err)

	assertResourceAttr(t, res, "service.name", "explicit-name")
	assertResourceAttr(t, res, "service.instance.id", "explicit-id")
	assertResourceAttr(t, res, "service.version", "explicit-version")
	assertResourceAttr(t, res, "custom.key", "explicit-value")
}

func TestNewProvider_WithResourceFromEnv(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=staging")

	reader := sdkmetric.NewManualReader()
	_, err := NewProvider(
		WithContext(context.Background()),
		WithReader(reader),
		WithResourceFromEnv(),
	)
	assert.NoError(t, err)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))

	assertResourceAttr(t, rm.Resource, "deployment.environment", "staging")
}
