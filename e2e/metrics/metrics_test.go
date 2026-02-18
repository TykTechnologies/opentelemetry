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
	prometheusURL = "http://localhost:8889/metrics"

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
