package metric

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/TykTechnologies/opentelemetry/config"
)

// blockingReader embeds a ManualReader but blocks ForceFlush until the context
// is done, simulating an exporter that cannot reach its collector.
type blockingReader struct {
	*sdkmetric.ManualReader
}

func (r *blockingReader) ForceFlush(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestForceFlush_BoundedByShutdownTimeoutWhenNoDeadline(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{ShutdownTimeout: 1}),
		WithReader(&blockingReader{ManualReader: sdkmetric.NewManualReader()}),
	)
	assert.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- provider.ForceFlush(context.Background()) }()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		t.Fatal("ForceFlush did not return: a caller context without deadline was not bounded")
	}
}

func TestForceFlush_HonoursCallerDeadline(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{ShutdownTimeout: 30}),
		WithReader(&blockingReader{ManualReader: sdkmetric.NewManualReader()}),
	)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = provider.ForceFlush(ctx)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second, "caller deadline must be honoured")
}

func TestWithDefaultTimeout(t *testing.T) {
	t.Run("no deadline gets the default timeout", func(t *testing.T) {
		ctx, cancel := withDefaultTimeout(context.Background(), 30*time.Second)
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(30*time.Second), deadline, time.Second)
	})

	t.Run("existing deadline is preserved", func(t *testing.T) {
		parent, parentCancel := context.WithTimeout(context.Background(), time.Hour)
		defer parentCancel()
		want, _ := parent.Deadline()

		ctx, cancel := withDefaultTimeout(parent, time.Second)
		defer cancel()

		got, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.Equal(t, want, got, "a caller deadline must not be shortened")
	})
}
