package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func Test_ResourceFactory(t *testing.T) {
	ctx := context.Background()
	resourceName := "test-service"

	res, err := resourceFactory(ctx, resourceName)

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
