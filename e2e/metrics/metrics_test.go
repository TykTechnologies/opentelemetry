package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	appURL        = "http://localhost:8081"
	prometheusURL = "http://localhost:18889/metrics"

	// startupTimeout is how long to wait for the stack to become healthy.
	startupTimeout = 60 * time.Second
	// exportInterval matches the app's MetricsConfig.ExportInterval.
	exportInterval = 5 * time.Second
)

type healthResponse struct {
	Healthy    bool   `json:"healthy"`
	Enabled    bool   `json:"enabled"`
	Type       string `json:"type"`
	Exports    int64  `json:"exports"`
	Successful int64  `json:"successful"`
	Failed     int64  `json:"failed"`
}

func TestMain(m *testing.M) {
	if os.Getenv("E2E_METRICS") == "" {
		fmt.Println("skipping e2e metrics tests (set E2E_METRICS=1 to run)")
		os.Exit(0)
	}

	// Start stack.
	if err := compose("up", "--build", "-d"); err != nil {
		fmt.Fprintf(os.Stderr, "docker compose up failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Tear down stack.
	_ = compose("down")
	os.Exit(code)
}

func TestHealthEndpoint(t *testing.T) {
	waitForHealthy(t)

	h := getHealth(t)
	if !h.Enabled {
		t.Fatal("expected enabled=true")
	}
	if h.Type != "otel" {
		t.Fatalf("expected type=otel, got %s", h.Type)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	waitForHealthy(t)

	// Send test requests.
	const numRequests = 5
	for i := range numRequests {
		resp, err := http.Get(appURL + "/metrics-test")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d returned status %d", i, resp.StatusCode)
		}
	}

	// Wait for at least one export cycle after the requests.
	time.Sleep(exportInterval + 3*time.Second)

	// Verify health shows successful exports.
	h := getHealth(t)
	if !h.Healthy {
		t.Fatal("expected healthy=true after requests")
	}
	if h.Failed > 0 {
		t.Fatalf("expected 0 failed exports, got %d", h.Failed)
	}
	if h.Successful == 0 {
		t.Fatal("expected successful exports > 0")
	}
}

func TestPrometheusMetrics(t *testing.T) {
	waitForHealthy(t)

	// Send requests to generate all metric types.
	for range 3 {
		resp, err := http.Get(appURL + "/metrics-test")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		resp.Body.Close()
	}

	// Wait for export to Prometheus.
	time.Sleep(exportInterval + 3*time.Second)

	body := fetchPrometheus(t)

	expected := []struct {
		name       string
		metricType string
	}{
		{"e2e_requests_total", "Counter"},
		{"e2e_request_duration_bucket", "Histogram"},
		{"e2e_request_duration_sum", "Histogram"},
		{"e2e_request_duration_count", "Histogram"},
		{"e2e_active_connections", "Gauge"},
		{"e2e_queue_size", "UpDownCounter"},
	}

	for _, exp := range expected {
		if !strings.Contains(body, exp.name) {
			t.Errorf("prometheus output missing %s (%s)", exp.name, exp.metricType)
		}
	}

	// Verify attributes are propagated as labels.
	if !strings.Contains(body, `http_method="GET"`) {
		t.Error("prometheus output missing http_method label")
	}
	if !strings.Contains(body, `http_route="/metrics-test"`) {
		t.Error("prometheus output missing http_route label")
	}
}

func TestCardinalityOverflow(t *testing.T) {
	waitForHealthy(t)

	// The app is configured with CardinalityLimit: 5.
	// Send 10 requests, each with a unique request_id attribute.
	const totalRequests = 10
	for i := range totalRequests {
		resp, err := http.Get(fmt.Sprintf("%s/cardinality-test?id=req-%d", appURL, i))
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d returned status %d", i, resp.StatusCode)
		}
	}

	// Wait for export cycle so metrics appear in Prometheus.
	time.Sleep(exportInterval + 3*time.Second)

	body := fetchPrometheus(t)

	// Count distinct series for e2e_cardinality_requests.
	// Each line like `e2e_cardinality_requests{...} N` is one series.
	var seriesCount int
	var hasOverflow bool
	var totalCount float64
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "e2e_cardinality_requests{") {
			continue
		}
		seriesCount++
		if strings.Contains(line, `otel_metric_overflow="true"`) {
			hasOverflow = true
		}
		// Parse the value (last space-separated field).
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			var val float64
			fmt.Sscanf(parts[len(parts)-1], "%f", &val)
			totalCount += val
		}
	}

	// With limit=5, we expect at most 5 individual series + 1 overflow series.
	maxSeries := 5 + 1
	if seriesCount > maxSeries {
		t.Errorf("expected at most %d series for e2e_cardinality_requests, got %d", maxSeries, seriesCount)
	}

	if !hasOverflow {
		t.Error("expected overflow series with otel_metric_overflow=\"true\" label")
	}

	// Total count across all series (including overflow) should equal totalRequests.
	if int(totalCount) != totalRequests {
		t.Errorf("expected total count=%d across all series, got %v", totalRequests, totalCount)
	}

	t.Logf("cardinality test: %d series, overflow=%v, total_count=%v", seriesCount, hasOverflow, totalCount)
}

// promSeriesValue returns the value of the first series whose name matches
// exactly (label block or bare), e.g. `e2e_observable_collections_total{...} 42`.
func promSeriesValue(t *testing.T, body, name string) (float64, bool) {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, name+"{") && !strings.HasPrefix(line, name+" ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		var val float64
		if _, err := fmt.Sscanf(parts[len(parts)-1], "%f", &val); err != nil {
			continue
		}
		return val, true
	}
	return 0, false
}

func TestPrometheusObservableMetrics(t *testing.T) {
	waitForHealthy(t)

	// Wait for at least one export cycle so every observable has been collected once.
	time.Sleep(exportInterval + 3*time.Second)
	body1 := fetchPrometheus(t)

	// Type assertions: monotonic counter, gauge, non-monotonic sum (rendered as gauge).
	typeChecks := []struct{ line, kind string }{
		{"# TYPE e2e_observable_collections_total counter", "ObservableCounter"},
		{"# TYPE e2e_observable_uptime_seconds gauge", "ObservableGauge"},
		{"# TYPE e2e_observable_queue_depth gauge", "ObservableUpDownCounter (non-monotonic sum)"},
	}
	for _, tc := range typeChecks {
		if !strings.Contains(body1, tc.line) {
			t.Errorf("prometheus output missing %q (%s)", tc.line, tc.kind)
		}
	}

	counter1, ok := promSeriesValue(t, body1, "e2e_observable_collections_total")
	if !ok {
		t.Fatal("e2e_observable_collections_total series not found")
	}
	uptime1, ok := promSeriesValue(t, body1, "e2e_observable_uptime_seconds")
	if !ok {
		t.Fatal("e2e_observable_uptime_seconds series not found")
	}
	if _, ok := promSeriesValue(t, body1, "e2e_observable_queue_depth"); !ok {
		t.Fatal("e2e_observable_queue_depth series not found")
	}

	// Callback attribute must be exported as a label.
	if !strings.Contains(body1, `source="callback"`) {
		t.Error("prometheus output missing source=\"callback\" label on observable series")
	}

	// Values must update across export intervals: the callback fires per collection.
	time.Sleep(2*exportInterval + 3*time.Second)
	body2 := fetchPrometheus(t)

	counter2, ok := promSeriesValue(t, body2, "e2e_observable_collections_total")
	if !ok {
		t.Fatal("e2e_observable_collections_total series not found on second scrape")
	}
	if counter2 <= counter1 {
		t.Errorf("observable counter did not increase across export intervals: %v -> %v", counter1, counter2)
	}

	uptime2, ok := promSeriesValue(t, body2, "e2e_observable_uptime_seconds")
	if !ok {
		t.Fatal("e2e_observable_uptime_seconds series not found on second scrape")
	}
	if uptime2 <= uptime1 {
		t.Errorf("observable gauge did not report a newer value: %v -> %v", uptime1, uptime2)
	}
}

// helpers

func compose(args ...string) error {
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForHealthy(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(appURL + "/health")
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		var h healthResponse
		json.NewDecoder(resp.Body).Decode(&h)
		resp.Body.Close()
		if h.Healthy {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("timed out waiting for metrics app to become healthy")
}

func getHealth(t *testing.T) healthResponse {
	t.Helper()
	resp, err := http.Get(appURL + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	var h healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	return h
}

func fetchPrometheus(t *testing.T) string {
	t.Helper()
	resp, err := http.Get(prometheusURL)
	if err != nil {
		t.Fatalf("prometheus scrape failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read prometheus response: %v", err)
	}
	return string(body)
}
