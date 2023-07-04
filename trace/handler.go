package trace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewHTTPHandler wraps the provided http.Handler with one that starts a span
// and injects the span context into the outbound request headers.
// You need to initialize the TracerProvider first since it utilizes the underlying
// TracerProvider and propagators.
// It also utilizes a spanNameFormatter to format the span name r.Method + " " + r.URL.Path.
func NewHTTPHandler(name string, handler http.Handler, tp Provider) http.Handler {
	opts := []otelhttp.Option{
		otelhttp.WithSpanNameFormatter(httpSpanNameFormatter),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otelhttp.NewHandler(handler, name, opts...).ServeHTTP(w, r)
	})
}

var httpSpanNameFormatter = func(operation string, r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// NewHTTPTransport wraps the provided http.RoundTripper with one that
// starts a span and injects the span context into the outbound request headers.
func NewHTTPTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base)
}
