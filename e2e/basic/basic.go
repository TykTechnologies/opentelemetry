package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TykTechnologies/opentelemetry/config"
	semconv "github.com/TykTechnologies/opentelemetry/semconv/v1.0.0"
	"github.com/TykTechnologies/opentelemetry/trace"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg := config.OpenTelemetry{
		Enabled:           true,
		Exporter:          "grpc",
		Endpoint:          "otel-collector:4317",
		ConnectionTimeout: 10,
		ResourceName:      "e2e-basic",
		TLS: config.TLS{
			Enable: false,
		},
	}

	log.Println("Initializing OpenTelemetry at e2e-basic:", cfg.Endpoint)

	provider, err := trace.NewProvider(
		trace.WithContext(ctx),
		trace.WithConfig(&cfg),
		trace.WithLogger(logrus.New()),
		trace.WithServiceID("service-id-1"),
		trace.WithServiceVersion("v1"),
		trace.WithHostDetector(),
		trace.WithContainerDetector(),
		trace.WithProcessDetector(),
	)
	if err != nil {
		log.Printf("error on otel provider init %s", err.Error())
		return
	}

	baseTykAttributes := []trace.Attribute{
		semconv.TykAPIName("test"),
		semconv.TykAPIOrgID("fakeorg"),
	}

	mux := http.NewServeMux()
	mux.Handle("/test", trace.NewHTTPHandler("get_test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := provider.Tracer().Start(r.Context(), "childspan")
		defer span.End()

		attributes := []trace.Attribute{
			trace.NewAttribute("test-string-attr", "value"),
			trace.NewAttribute("test-int-attr", 1),
		}
		span.SetAttributes(attributes...)

		response := map[string]interface{}{
			"status": "success",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("error on encode response %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}), provider, baseTykAttributes...))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Printf("server listening on port %s", ":8080")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	<-ctx.Done() // Blocks here until ctx is cancelled
	newCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ConnectionTimeout)*time.Second)
	defer cancel()
	// Shutdown provider (with a new context)
	if err := provider.Shutdown(newCtx); err != nil {
		log.Fatal("failed to shutdown TracerProvider: %w", err)
	}

	if err := srv.Shutdown(newCtx); err != nil {
		// Error from closing listeners, or context timeout:
		log.Printf("HTTP server Shutdown: %v", err)
	}
}
