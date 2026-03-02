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
// Each TestProvider is fully isolated — no global state is set.
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
