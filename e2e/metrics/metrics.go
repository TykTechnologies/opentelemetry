package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/TykTechnologies/opentelemetry/metric"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	metricsEnabled := true
	cfg := config.OpenTelemetry{
		Enabled:           true,
		Exporter:          "grpc",
		Endpoint:          "otel-collector:4317",
		ConnectionTimeout: 10,
		ResourceName:      "e2e-metrics",
		TLS: config.TLS{
			Enable: false,
		},
		Metrics: config.MetricsConfig{
			Enabled:        &metricsEnabled,
			ExportInterval: 5, // short interval for fast e2e feedback
			Temporality:    "cumulative",
		},
	}

	log.Println("Initializing OpenTelemetry metrics at e2e-metrics:", cfg.Endpoint)

	provider, err := metric.NewProvider(
		metric.WithContext(ctx),
		metric.WithConfig(&cfg),
		metric.WithLogger(logrus.New()),
		metric.WithServiceID("e2e-metrics-1"),
		metric.WithServiceVersion("v1"),
		metric.WithHostDetector(),
		metric.WithContainerDetector(),
		metric.WithProcessDetector(),
	)
	if err != nil {
		log.Printf("error on otel metric provider init: %s", err.Error())
		return
	}

	// Create all four instrument types.
	reqCounter, err := provider.NewCounter("e2e.requests.total", "Total requests", "1")
	if err != nil {
		log.Printf("error creating counter: %s", err.Error())
		return
	}

	latencyHist, err := provider.NewHistogram("e2e.request.duration", "Request duration", "ms", nil)
	if err != nil {
		log.Printf("error creating histogram: %s", err.Error())
		return
	}

	activeGauge, err := provider.NewGauge("e2e.active_connections", "Active connections", "1")
	if err != nil {
		log.Printf("error creating gauge: %s", err.Error())
		return
	}

	queueSize, err := provider.NewUpDownCounter("e2e.queue.size", "Queue depth", "1")
	if err != nil {
		log.Printf("error creating up-down counter: %s", err.Error())
		return
	}

	mux := http.NewServeMux()

	// Metrics test endpoint - records all 4 instrument types on each request.
	mux.Handle("/metrics-test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", "/metrics-test"),
		}

		reqCounter.Add(r.Context(), 1, attrs...)
		activeGauge.Record(r.Context(), float64(10+rand.Intn(90)), attrs...)
		queueSize.Add(r.Context(), 1, attrs...)

		// Simulate some work.
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

		duration := float64(time.Since(start).Milliseconds())
		latencyHist.Record(r.Context(), duration, attrs...)

		// Decrement queue after "processing".
		queueSize.Add(r.Context(), -1, attrs...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"duration_ms": duration,
		})
	}))

	// Health endpoint for e2e assertions.
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats := provider.GetExportStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"healthy":    provider.Healthy(),
			"enabled":    provider.Enabled(),
			"type":       provider.Type(),
			"exports":    stats.TotalExports,
			"successful": stats.SuccessfulExports,
			"failed":     stats.FailedExports,
		})
	}))

	srv := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	go func() {
		log.Printf("e2e-metrics server listening on :8081")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	<-ctx.Done()
	newCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer shutdownCancel()

	// Force flush before shutdown to ensure all pending metrics are exported.
	if err := provider.ForceFlush(newCtx); err != nil {
		log.Printf("failed to force flush metric provider: %v", err)
	}

	if err := provider.Shutdown(newCtx); err != nil {
		log.Printf("failed to shutdown metric provider: %v", err)
	}

	if err := srv.Shutdown(newCtx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	log.Println("e2e-metrics shut down cleanly")
}
