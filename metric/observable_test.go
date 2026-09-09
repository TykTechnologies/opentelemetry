package metric

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/TykTechnologies/opentelemetry/config"
)

// findObservableMetric returns the metric with the given name from rm, failing the test if absent.
func findObservableMetric(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}

func TestObservableCounter_NilReceiver(t *testing.T) {
	var c *ObservableCounter
	assert.False(t, c.Enabled())
	assert.NoError(t, c.Unregister())
	// Twice, to prove idempotence on nil too.
	assert.NoError(t, c.Unregister())
}

func TestNewObservableCounter_Disabled(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{Enabled: ptr(false)}),
	)
	assert.NoError(t, err)

	invoked := atomic.Bool{}
	counter, err := provider.NewObservableCounter("test.observable.counter", "A test observable counter", "1",
		func(_ context.Context, observe Int64Observer) error {
			invoked.Store(true)
			observe(1)
			return nil
		})
	assert.NoError(t, err)
	assert.NotNil(t, counter)
	assert.False(t, counter.Enabled())
	assert.NoError(t, counter.Unregister())
	assert.False(t, invoked.Load(), "disabled provider must never invoke the callback")
}

func TestNewObservableCounter_CollectInvokesCallbackEveryTime(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var source atomic.Int64
	source.Store(5)
	counter, err := provider.NewObservableCounter("test.observable.counter", "A test observable counter", "1",
		func(_ context.Context, observe Int64Observer) error {
			observe(source.Load(), attribute.String("worker", "a"))
			return nil
		})
	assert.NoError(t, err)
	assert.True(t, counter.Enabled())

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	m := findObservableMetric(t, rm, "test.observable.counter")
	sum, ok := m.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "expected Sum[int64], got %T", m.Data)
	assert.True(t, sum.IsMonotonic, "observable counter must be monotonic")
	assert.Len(t, sum.DataPoints, 1)
	assert.Equal(t, int64(5), sum.DataPoints[0].Value)
	val, found := sum.DataPoints[0].Attributes.Value("worker")
	assert.True(t, found)
	assert.Equal(t, "a", val.AsString())

	// Second collection sees the updated source value: callback fired again.
	source.Store(9)
	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	m2 := findObservableMetric(t, rm2, "test.observable.counter")
	sum2 := m2.Data.(metricdata.Sum[int64])
	assert.Equal(t, int64(9), sum2.DataPoints[0].Value)
}

func TestObservableCounter_Unregister(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var calls atomic.Int64
	counter, err := provider.NewObservableCounter("test.observable.unregister", "d", "1",
		func(_ context.Context, observe Int64Observer) error {
			observe(calls.Add(1))
			return nil
		})
	assert.NoError(t, err)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	assert.Equal(t, int64(1), calls.Load())

	assert.NoError(t, counter.Unregister())

	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	assert.Equal(t, int64(1), calls.Load(), "callback must not fire after Unregister")

	// Idempotent: second Unregister is safe and returns no error.
	assert.NoError(t, counter.Unregister())
}

func TestNewObservableCounter_CallbackErrorDoesNotBreakCollection(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var calls atomic.Int64
	_, err = provider.NewObservableCounter("test.observable.err", "d", "1",
		func(_ context.Context, observe Int64Observer) error {
			if calls.Add(1) == 1 {
				return errors.New("boom")
			}
			observe(7)
			return nil
		})
	assert.NoError(t, err)

	// First collection: error flows out of the SDK collection path
	// (PeriodicReader routes this same error to the global error handler in production).
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	assert.ErrorContains(t, err, "boom")

	// Second collection succeeds and carries the observed value.
	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	m := findObservableMetric(t, rm2, "test.observable.err")
	sum := m.Data.(metricdata.Sum[int64])
	assert.Equal(t, int64(7), sum.DataPoints[0].Value)
}

// captureLogger records Error() calls for asserting the error-handler path.
type captureLogger struct{ errs []string }

func (l *captureLogger) Info(_ ...interface{})     {}
func (l *captureLogger) Error(args ...interface{}) { l.errs = append(l.errs, fmt.Sprint(args...)) }

func TestErrHandler_SurfacesCallbackError(t *testing.T) {
	// In production the PeriodicReader passes callback errors to otel.Handle,
	// which reaches this handler (installed at provider init, provider.go:243).
	logger := &captureLogger{}
	h := &errHandler{logger: logger}
	h.Handle(errors.New("observable callback failed: boom"))
	assert.Len(t, logger.errs, 1)
	assert.Contains(t, logger.errs[0], "observable callback failed: boom")
}

func TestNewObservableCounter_InvalidNameError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	// Same parity check as the sync constructors: instrument-name validation errors propagate.
	obs, err := provider.NewObservableCounter("invalid name with spaces!", "d", "1",
		func(_ context.Context, observe Int64Observer) error { observe(1); return nil })
	assert.Error(t, err)
	assert.Nil(t, obs)

	// Sync parity: NewCounter rejects the same name.
	syncCounter, syncErr := provider.NewCounter("invalid name with spaces!", "d", "1")
	assert.Error(t, syncErr)
	assert.Nil(t, syncCounter)
}

func TestNewObservableCounter_NilCallbackError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	obs, err := provider.NewObservableCounter("test.observable.nilcb", "d", "1", nil)
	assert.Error(t, err)
	assert.Nil(t, obs)
}

func TestObservableGauge_NilReceiver(t *testing.T) {
	var g *ObservableGauge
	assert.False(t, g.Enabled())
	assert.NoError(t, g.Unregister())
	assert.NoError(t, g.Unregister())
}

func TestNewObservableGauge_Disabled(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{Enabled: ptr(false)}),
	)
	assert.NoError(t, err)

	invoked := atomic.Bool{}
	gauge, err := provider.NewObservableGauge("test.observable.gauge", "A test observable gauge", "1",
		func(_ context.Context, observe Float64Observer) error {
			invoked.Store(true)
			observe(1.0)
			return nil
		})
	assert.NoError(t, err)
	assert.NotNil(t, gauge)
	assert.False(t, gauge.Enabled())
	assert.NoError(t, gauge.Unregister())
	assert.False(t, invoked.Load())
}

func TestNewObservableGauge_CollectReportsLatestValue(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var source atomic.Int64 // stored as int, observed as float
	source.Store(42)
	gauge, err := provider.NewObservableGauge("test.observable.gauge", "A test observable gauge", "1",
		func(_ context.Context, observe Float64Observer) error {
			observe(float64(source.Load()), attribute.String("pool", "db"))
			return nil
		})
	assert.NoError(t, err)
	assert.True(t, gauge.Enabled())

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	m := findObservableMetric(t, rm, "test.observable.gauge")
	data, ok := m.Data.(metricdata.Gauge[float64])
	assert.True(t, ok, "expected Gauge[float64], got %T", m.Data)
	assert.Len(t, data.DataPoints, 1)
	assert.Equal(t, 42.0, data.DataPoints[0].Value)
	val, found := data.DataPoints[0].Attributes.Value("pool")
	assert.True(t, found)
	assert.Equal(t, "db", val.AsString())

	// Gauge reports the latest value on the next collection (down as well as up).
	source.Store(13)
	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	m2 := findObservableMetric(t, rm2, "test.observable.gauge")
	data2 := m2.Data.(metricdata.Gauge[float64])
	assert.Equal(t, 13.0, data2.DataPoints[0].Value)
}

func TestObservableGauge_Unregister(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var calls atomic.Int64
	gauge, err := provider.NewObservableGauge("test.observable.gauge.unreg", "d", "1",
		func(_ context.Context, observe Float64Observer) error {
			observe(float64(calls.Add(1)))
			return nil
		})
	assert.NoError(t, err)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	assert.Equal(t, int64(1), calls.Load())

	assert.NoError(t, gauge.Unregister())

	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	assert.Equal(t, int64(1), calls.Load(), "callback must not fire after Unregister")
	assert.NoError(t, gauge.Unregister())
}

func TestNewObservableGauge_NilCallbackError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	gauge, err := provider.NewObservableGauge("test.observable.gauge.nilcb", "d", "1", nil)
	assert.Error(t, err)
	assert.Nil(t, gauge)
}

func TestObservableUpDownCounter_NilReceiver(t *testing.T) {
	var u *ObservableUpDownCounter
	assert.False(t, u.Enabled())
	assert.NoError(t, u.Unregister())
	assert.NoError(t, u.Unregister())
}

func TestNewObservableUpDownCounter_Disabled(t *testing.T) {
	provider, err := NewProvider(
		WithContext(context.Background()),
		WithConfig(&config.MetricsConfig{Enabled: ptr(false)}),
	)
	assert.NoError(t, err)

	invoked := atomic.Bool{}
	updown, err := provider.NewObservableUpDownCounter("test.observable.updown", "A test observable up-down counter", "1",
		func(_ context.Context, observe Int64Observer) error {
			invoked.Store(true)
			observe(1)
			return nil
		})
	assert.NoError(t, err)
	assert.NotNil(t, updown)
	assert.False(t, updown.Enabled())
	assert.NoError(t, updown.Unregister())
	assert.False(t, invoked.Load())
}

func TestNewObservableUpDownCounter_LastObservedCumulative(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var source atomic.Int64
	source.Store(10)
	updown, err := provider.NewObservableUpDownCounter("test.observable.updown", "A test observable up-down counter", "1",
		func(_ context.Context, observe Int64Observer) error {
			observe(source.Load(), attribute.String("queue", "jobs"))
			return nil
		})
	assert.NoError(t, err)
	assert.True(t, updown.Enabled())

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	m := findObservableMetric(t, rm, "test.observable.updown")
	sum, ok := m.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "expected Sum[int64], got %T", m.Data)
	assert.False(t, sum.IsMonotonic, "observable up-down counter must be non-monotonic")
	assert.Len(t, sum.DataPoints, 1)
	assert.Equal(t, int64(10), sum.DataPoints[0].Value)

	// The cumulative point reflects the LAST observed value (4),
	// not deltas summed across collections (10 + 4 = 14).
	source.Store(4)
	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	m2 := findObservableMetric(t, rm2, "test.observable.updown")
	sum2 := m2.Data.(metricdata.Sum[int64])
	assert.Equal(t, int64(4), sum2.DataPoints[0].Value)
}

func TestObservableUpDownCounter_Unregister(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	var calls atomic.Int64
	updown, err := provider.NewObservableUpDownCounter("test.observable.updown.unreg", "d", "1",
		func(_ context.Context, observe Int64Observer) error {
			observe(calls.Add(1))
			return nil
		})
	assert.NoError(t, err)

	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))
	assert.Equal(t, int64(1), calls.Load())

	assert.NoError(t, updown.Unregister())

	var rm2 metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm2))
	assert.Equal(t, int64(1), calls.Load(), "callback must not fire after Unregister")
	assert.NoError(t, updown.Unregister())
}

func TestNewObservableUpDownCounter_NilCallbackError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider, err := NewProvider(WithContext(context.Background()), WithReader(reader))
	assert.NoError(t, err)

	updown, err := provider.NewObservableUpDownCounter("test.observable.updown.nilcb", "d", "1", nil)
	assert.Error(t, err)
	assert.Nil(t, updown)
}
