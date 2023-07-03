package trace

import (
	"context"
	"testing"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
