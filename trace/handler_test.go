package trace

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Test_httpSpanNameFormatter(t *testing.T) {
	type args struct {
		operation string
		r         *http.Request
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "httpSpanNameFormatter",
			args: args{
				operation: "test",
				r: &http.Request{
					Method: "GET",
					URL:    &url.URL{Path: "/test"},
				},
			},
			want: "GET /test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httpSpanNameFormatter(tt.args.operation, tt.args.r); got != tt.want {
				t.Errorf("httpSpanNameFormatter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NewHTTPTransport(t *testing.T) {
	// create context propagator
	prop := propagation.TraceContext{}
	// set context propagator
	otel.SetTextMapPropagator(prop)

	content := []byte("Hello, world!")

	// set the new span with mocked data
	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	// create a new server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract context from request
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// get span from context
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), sc.SpanID())
		}

		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	// create a new request
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	assert.Nil(t, err)

	// create the transport
	tr := NewHTTPTransport(http.DefaultTransport)

	// create a new client with the transport
	c := http.Client{Transport: tr}

	// do the request
	res, err := c.Do(r)
	assert.Nil(t, err)

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	// check if the response is the same as the content
	assert.Equal(t, body, content)
}

func Test_NewHTTPHandler(t *testing.T) {
	provider, err := NewProvider()
	assert.Nil(t, err)
	assert.NotNil(t, provider)

	content := []byte("Hello, world!")

	// set the new span with mocked data
	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	// create a new handler with the provider
	handler := NewHTTPHandler("test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract context from request
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// get span from context
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("check if SpanID is propagated: got %s, expected %s", span.SpanID(), sc.SpanID())
		}

		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))

	// create a new server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// create a new request
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	assert.Nil(t, err)

	// create the transport
	tr := NewHTTPTransport(http.DefaultTransport)

	// create a new client with the transport
	c := http.Client{Transport: tr}

	// do the request
	res, err := c.Do(r)
	assert.Nil(t, err)

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	// check if the response is the same as the content
	assert.Equal(t, body, content)
}
