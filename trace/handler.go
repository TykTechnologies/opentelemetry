package trace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// NewHTTPHandler wraps the provided http.Handler with one that starts a span
// and injects the span context into the outbound request headers.
// You need to initialize the TracerProvider first since it utilizes the underlying
// TracerProvider and propagators.
// It also utilizes a spanNameFormatter to format the span name r.Method + " " + r.URL.Path.
func NewHTTPHandler(name string, handler http.Handler, tp Provider, attr ...Attribute) http.Handler {
	opts := []otelhttp.Option{
		otelhttp.WithSpanNameFormatter(httpSpanNameFormatter),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		// Suppress otelhttp's auto-emitted metrics (http_server_duration_milliseconds etc.)
		// to avoid duplicates — Tyk owns its own metric instrumentation.
		otelhttp.WithMeterProvider(noopmetric.NewMeterProvider()),
	}

	opts = append(opts, otelhttp.WithSpanOptions(
		trace.WithAttributes(attr...),
	))

	return otelhttp.NewHandler(handler, name, opts...)
}

var httpSpanNameFormatter = func(operation string, r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// NewHTTPTransport wraps the provided http.RoundTripper with one that
// starts a span and injects the span context into the outbound request headers.
func NewHTTPTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base, otelhttp.WithMeterProvider(noopmetric.NewMeterProvider()))
}
