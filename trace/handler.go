package trace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type responseWriterWithSize struct {
	http.ResponseWriter
	size int
}

func (rw *responseWriterWithSize) Write(p []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(p)
	rw.size += n

	return n, err
}

func NewHTTPHandler(name string, handler http.Handler, tp Provider, attr ...Attribute) http.Handler {
	opts := []otelhttp.Option{
		otelhttp.WithSpanNameFormatter(httpSpanNameFormatter),
	}

	opts = append(opts, otelhttp.WithSpanOptions(
		trace.WithAttributes(attr...),
	))

	return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		// Wrap response writer to capture the response size
		rw := &responseWriterWithSize{
			ResponseWriter: w,
		}

		span.SetAttributes(NewAttribute("http.request.body.size", r.ContentLength))
		handler.ServeHTTP(rw, r)
		span.SetAttributes(NewAttribute("http.response.body.size", rw.size))
	}), name, opts...)
}

var httpSpanNameFormatter = func(operation string, r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// NewHTTPTransport wraps the provided http.RoundTripper with one that
// starts a span and injects the span context into the outbound request headers.
func NewHTTPTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base)
}
