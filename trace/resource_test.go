package trace

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func Test_ResourceFactory_base(t *testing.T) {
	ctx := context.Background()
	resourceName := "test-service"

	res, err := resourceFactory(ctx, resourceName, resourceConfig{})

	assert.Nil(t, err)

	attrs := res.Attributes()

	assert.Equal(t, res.Len(), 1)

	found := false

	for _, attr := range attrs {
		if attr.Key == semconv.ServiceNameKey {
			found = true

			assert.Equal(t, resourceName, attr.Value.AsString())

			break
		}
	}

	assert.True(t, found)
}

func TestResourceFactory(t *testing.T) {

	currentHost, err := os.Hostname()
	assert.Nil(t, err)

	testCases := []struct {
		name          string
		resourceName  string
		cfg           resourceConfig
		expectedAttrs []attribute.KeyValue
		expectedErr   error
	}{
		{
			name:         "Test with default config",
			resourceName: "testResource",
			cfg:          resourceConfig{},
			expectedAttrs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("testResource"),
			},
			expectedErr: nil,
		},
		{
			name:         "Test with id and version",
			resourceName: "testResource",
			cfg: resourceConfig{
				id:      "123",
				version: "1.0.0",
			},
			expectedAttrs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("testResource"),
				semconv.ServiceInstanceID("123"),
				semconv.ServiceVersion("1.0.0"),
			},
			expectedErr: nil,
		},
		{
			name:         "Test with host",
			resourceName: "testResource",
			cfg: resourceConfig{
				withHost: true,
			},
			expectedAttrs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("testResource"),
				semconv.HostName(currentHost),
			},
			expectedErr: nil,
		},
		{
			//special scenario to unit test - we cannot see the container attrs here
			name:         "Test with container",
			resourceName: "testResource",
			cfg: resourceConfig{
				withContainer: true,
			},
			expectedAttrs: []attribute.KeyValue{
				semconv.ServiceNameKey.String("testResource"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			res, err := resourceFactory(ctx, tc.resourceName, tc.cfg)

			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}

			if res != nil {
				attrs := res.Attributes()

				for _, expectedAttr := range tc.expectedAttrs {
					assert.Contains(t, attrs, expectedAttr)
				}
			}
		})
	}
}
