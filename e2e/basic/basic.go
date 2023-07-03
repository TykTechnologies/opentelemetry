package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/TykTechnologies/opentelemetry/config"
	"github.com/TykTechnologies/opentelemetry/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg := config.OpenTelemetry{
		Enabled:           true,
		Exporter:          "http",
		Endpoint:          "otel-collector:4317",
		ConnectionTimeout: 10,
		ResourceName:      "e2e-basic",
	}

	log.Println("Initializing OpenTelemetry at e2e-basic:", cfg.Endpoint)

	provider, err := trace.NewProvider(ctx, cfg)
	if err != nil {
		log.Printf("error on otel provider init %s", err.Error())
		return
	}

	defer func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	tracer := provider.Tracer()

	mux := http.NewServeMux()
	mux.Handle("/test", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := tracer.Start(r.Context(), "main")
		defer span.End()

		span.AddEvent("test event")
		attributes := []attribute.KeyValue{
			attribute.String("test-string-attr", "value"),
			attribute.Int("test-int-attr", 1),
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
	}), "get_test"))

	log.Printf("server listening on port %s", ":8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("error on listen and serve %s", err.Error())
	}
}
