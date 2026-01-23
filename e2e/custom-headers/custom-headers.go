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

	// Get configuration from environment or use defaults
	contextPropagation := getEnv("OTEL_CONTEXT_PROPAGATION", config.PROPAGATOR_CUSTOM)
	customTraceHeader := getEnv("OTEL_CUSTOM_TRACE_HEADER", "X-Correlation-ID")

	cfg := config.OpenTelemetry{
		Enabled:            true,
		Exporter:           "grpc",
		Endpoint:           "otel-collector:4317",
		ConnectionTimeout:  10,
		ResourceName:       "e2e-custom-headers",
		ContextPropagation: contextPropagation,
		CustomTraceHeader:  customTraceHeader,
		TLS: config.TLS{
			Enable: false,
		},
	}

	log.Printf("Initialising OpenTelemetry at e2e-custom-headers: %s", cfg.Endpoint)
	log.Printf("Context propagation: %s", cfg.ContextPropagation)
	log.Printf("Custom trace header: %s", cfg.CustomTraceHeader)

	provider, err := trace.NewProvider(
		trace.WithContext(ctx),
		trace.WithConfig(&cfg),
		trace.WithLogger(logrus.New()),
		trace.WithServiceID("custom-headers-service"),
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
		semconv.TykAPIName("custom-headers-test"),
		semconv.TykAPIOrgID("test-org"),
	}

	mux := http.NewServeMux()

	// Endpoint that demonstrates custom header propagation
	mux.Handle("/test", trace.NewHTTPHandler("get_test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := provider.Tracer().Start(r.Context(), "childspan")
		defer span.End()

		// Log incoming headers for debugging
		customHeader := r.Header.Get(customTraceHeader)
		traceparent := r.Header.Get("traceparent")
		log.Printf("Incoming request - %s: %s, traceparent: %s", customTraceHeader, customHeader, traceparent)

		attributes := []trace.Attribute{
			trace.NewAttribute("test-string-attr", "value"),
			trace.NewAttribute("test-int-attr", 1),
			trace.NewAttribute("custom-header-present", customHeader != ""),
			trace.NewAttribute("traceparent-present", traceparent != ""),
		}
		span.SetAttributes(attributes...)

		response := map[string]interface{}{
			"status":                "success",
			"context_propagation":   contextPropagation,
			"custom_trace_header":   customTraceHeader,
			"custom_header_value":   customHeader,
			"traceparent_value":     traceparent,
			"custom_header_present": customHeader != "",
			"traceparent_present":   traceparent != "",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("error on encode response %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}), provider, baseTykAttributes...))

	// Endpoint that makes an upstream request to test propagation
	mux.Handle("/upstream", trace.NewHTTPHandler("get_upstream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := provider.Tracer().Start(r.Context(), "upstream-request")
		defer span.End()

		// Create a request to the /test endpoint
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/test", nil)
		if err != nil {
			log.Printf("error creating request: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Use the HTTP transport with trace propagation
		client := &http.Client{
			Transport: trace.NewHTTPTransport(http.DefaultTransport),
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error making upstream request: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var upstreamResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&upstreamResponse); err != nil {
			log.Printf("error decoding upstream response: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":            "success",
			"upstream_response": upstreamResponse,
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
