// Package metrictest provides test utilities for the metric package.
//
// It allows unit tests to create a real (non-noop) metric provider that
// records actual values, without requiring any network, config, or OTLP
// collector. Tests can then collect recorded data and assert on metric
// values, attributes, and counts.
//
// # Quick Start
//
// Create a test provider, record metrics, and assert:
//
//	func TestRequestCounter(t *testing.T) {
//		tp := metrictest.NewProvider(t)
//
//		counter, err := tp.NewCounter("http.requests", "Total requests", "1")
//		require.NoError(t, err)
//
//		ctx := context.Background()
//		counter.Add(ctx, 1, attribute.String("method", "GET"))
//		counter.Add(ctx, 3, attribute.String("method", "POST"))
//
//		// Assert counter values.
//		m := tp.FindMetric(t, "http.requests")
//		metrictest.AssertSum(t, m, int64(4))
//		metrictest.AssertDataPointCount(t, m, 2) // GET and POST
//	}
//
// # Testing Histograms
//
// Histogram assertions support count, sum, and attributes:
//
//	func TestLatencyHistogram(t *testing.T) {
//		tp := metrictest.NewProvider(t)
//
//		hist, err := tp.NewHistogram(
//			"http.duration", "Request duration", "ms", nil,
//		)
//		require.NoError(t, err)
//
//		ctx := context.Background()
//		hist.Record(ctx, 50.0, attribute.String("route", "/api/v1"))
//		hist.Record(ctx, 150.0, attribute.String("route", "/api/v1"))
//
//		m := tp.FindMetric(t, "http.duration")
//		metrictest.AssertHistogramCount(t, m, uint64(2))
//		metrictest.AssertHistogramSum(t, m, 200.0)
//	}
//
// # Testing Gauges
//
// Gauges record the latest value:
//
//	func TestPoolSize(t *testing.T) {
//		tp := metrictest.NewProvider(t)
//
//		gauge, err := tp.NewGauge("pool.size", "Connection pool size", "1")
//		require.NoError(t, err)
//
//		ctx := context.Background()
//		gauge.Record(ctx, 42.0)
//
//		m := tp.FindMetric(t, "pool.size")
//		metrictest.AssertGauge(t, m, 42.0)
//	}
//
// # Attribute Assertions
//
// Verify that metrics are recorded with the correct attributes:
//
//	func TestCounterAttributes(t *testing.T) {
//		tp := metrictest.NewProvider(t)
//
//		counter, _ := tp.NewCounter("api.calls", "API calls", "1")
//
//		ctx := context.Background()
//		counter.Add(ctx, 1,
//			attribute.String("api.id", "api-123"),
//			attribute.String("http.request.method", "GET"),
//			attribute.Int("http.response.status_code", 200),
//		)
//
//		m := tp.FindMetric(t, "api.calls")
//		metrictest.AssertSumWithAttrs(t, m, int64(1),
//			attribute.String("api.id", "api-123"),
//			attribute.String("http.request.method", "GET"),
//		)
//	}
//
// # Testing Provider Initialization
//
// TestProvider builds a bare provider, which is enough to unit-test
// instruments but not the code that configures the provider in your own
// service (identity options, detectors, defaults, error handling). For that,
// inject a Recorder into your initialisation path and inspect what it built.
// The consumer test needs only this package and metric — no direct dependency
// on any go.opentelemetry.io module (build attributes with
// metric.StringAttribute, metric.IntAttribute, metric.BoolAttribute or
// metric.NewAttribute):
//
//	func TestInitMetrics(t *testing.T) {
//		rec := metrictest.NewRecorder(t) // wraps a ManualReader; t.Cleanup shuts it down
//
//		// rec.Option() is a metric.Option (metric.WithReader) — pass it through
//		// your own init function alongside its usual options.
//		provider, err := myotel.InitMetrics(ctx, logger, cfg, nodeID, rec.Option())
//		require.NoError(t, err)
//		require.True(t, provider.Enabled())
//
//		// Resource identity as plain strings.
//		attrs := rec.ResourceAttributes()
//		assert.Equal(t, "tyk-dashboard", attrs["service.name"])
//		assert.Equal(t, nodeID, attrs["service.instance.id"])
//		assert.Equal(t, version, attrs["service.version"])
//
//		// Or typed, subset match.
//		metrictest.AssertResourceAttributes(t, rec.Collect(),
//			metric.StringAttribute("service.instance.id", nodeID),
//			metric.StringAttribute("tyk.component", "dashboard"),
//		)
//
//		// Metrics recorded through the provider your init code returned.
//		m := rec.FindMetric(t, "process.uptime")
//		metrictest.AssertGauge(t, m, 12.5)
//
//		// Raw metricdata for advanced cases.
//		rm := rec.Collect()
//		_ = rm.ScopeMetrics
//	}
//
// Because the reader replaces the exporter, no collector or network is
// involved, and the reader implies the provider is enabled regardless of
// cfg.Enabled.
//
// # Reading Values
//
// When an exact match is the wrong assertion (uptime, timestamps, counts that
// only need to grow), read the value out and compare it yourself. Neither
// accessor requires importing metricdata:
//
//	first := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
//	second := metrictest.GaugeValue[float64](t, rec.FindMetric(t, "process.uptime"))
//	assert.Greater(t, first, 0.0)
//	assert.Greater(t, second, first)
//
//	// One value per attribute combination for counters and multi-point gauges.
//	values := metrictest.DataPointValues[int64](t, tp.FindMetric(t, "http.requests"))
//
// # Resource Assertions
//
// Both TestProvider and Recorder expose the provider's resource:
//
//	attrs := tp.ResourceAttributes() // map[string]string; numbers and bools as literals, slices as JSON arrays
//	metrictest.AssertResourceAttributes(t, tp.Collect(), attribute.String("service.name", "tyk"))
//	m := metrictest.ResourceAttributeMap(tp.Collect())
//
// # Advanced: Raw metricdata Access
//
// For complex assertions, use Collect() to get the raw OTel metricdata
// types and combine with metricdatatest.AssertEqual:
//
//	func TestAdvanced(t *testing.T) {
//		tp := metrictest.NewProvider(t)
//		// ... record metrics ...
//
//		rm := tp.Collect()
//		for _, sm := range rm.ScopeMetrics {
//			for _, m := range sm.Metrics {
//				// Full access to metricdata types.
//			}
//		}
//	}
//
// # Debugging
//
// Use MetricNames() to see what was recorded:
//
//	names := tp.MetricNames()
//	t.Logf("recorded metrics: %v", names)
//
// # Parallel Tests
//
// Each TestProvider and Recorder is fully isolated — no global state is set.
// Safe for use with t.Parallel():
//
//	func TestA(t *testing.T) {
//		t.Parallel()
//		tp := metrictest.NewProvider(t)
//		// ...
//	}
//
//	func TestB(t *testing.T) {
//		t.Parallel()
//		tp := metrictest.NewProvider(t)
//		// ...
//	}
package metrictest
