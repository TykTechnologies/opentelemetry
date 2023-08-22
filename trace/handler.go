package trace

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	contentLength int
	onWrite       func(int) // callback for when Write is called
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.contentLength += n
	rw.onWrite(rw.contentLength)
	return n, err
}

func NewHTTPHandler(name string, handler http.Handler, tp Provider, attr ...Attribute) http.Handler {
	opts := []otelhttp.Option{
		otelhttp.WithSpanNameFormatter(httpSpanNameFormatter),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attr = append(attr, NewAttribute("http.request_content_length", r.ContentLength))

		opts = append(opts, otelhttp.WithSpanOptions(
			trace.WithAttributes(attr...),
		))

		rwWrapper := &responseWriterWrapper{
			ResponseWriter: w,
			onWrite: func(length int) {
				span := trace.SpanFromContext(r.Context())
				if span != nil {
					span.SetAttributes(NewAttribute("http.response_content_length", int64(length)))
				}
			},
		}

		otelhttp.NewHandler(handler, name, opts...).ServeHTTP(rwWrapper, r)
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
