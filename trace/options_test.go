package trace

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/TykTechnologies/opentelemetry/config"
)

func Test_WithLogger(t *testing.T) {
	tcs := []struct {
		name   string
		logger Logger
	}{
		{
			name:   "noop logger",
			logger: &noopLogger{},
		},
		{
			name:   "logrus logger",
			logger: logrus.New(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tp := &traceProvider{}
			WithLogger(tc.logger).apply(tp)

			assert.NotNil(t, tp.logger)
			assert.IsType(t, tc.logger, tp.logger)
		})
	}
}

func Test_WithContext(t *testing.T) {
	ctx := context.Background()
	tp := &traceProvider{}
	WithContext(ctx).apply(tp)

	assert.NotNil(t, tp.ctx)
	assert.IsType(t, ctx, tp.ctx)
}

func Test_WithConfig(t *testing.T) {
	cfg := config.OpenTelemetry{
		Exporter: "http",
		Enabled:  true,
		Endpoint: "localhost:4317",
	}
	tp := &traceProvider{}

	WithConfig(&cfg).apply(tp)

	assert.NotNil(t, tp.cfg)
	assert.IsType(t, cfg, *tp.cfg)
}

func Test_WithServiceID(t *testing.T) {
	tp := &traceProvider{}
	WithServiceID("id1").apply(tp)

	assert.Equal(t, "id1", tp.resources.id)
}

func Test_WithServiceVersion(t *testing.T) {
	tp := &traceProvider{}
	WithServiceVersion("v1").apply(tp)

	assert.Equal(t, "v1", tp.resources.version)
}

func Test_WithHostDetector(t *testing.T) {
	tp := &traceProvider{}
	WithHostDetector().apply(tp)

	assert.Equal(t, true, tp.resources.withHost)
}

func Test_WithContainerDetector(t *testing.T) {
	tp := &traceProvider{}
	WithContainerDetector().apply(tp)

	assert.Equal(t, true, tp.resources.withContainer)
}

func Test_WithProcessDetector(t *testing.T) {
	tp := &traceProvider{}
	WithProcessDetector().apply(tp)

	assert.Equal(t, true, tp.resources.withProcess)
}

func Test_WithCustomResourceAttributes(t *testing.T) {
	tp := &traceProvider{}
	attrs := []Attribute{NewAttribute("key", "value")}

	WithCustomResourceAttributes(attrs...).apply(tp)

	assert.Len(t, tp.resources.customAttrs, 1)
}
