package config

type OpenTelemetry struct {
	// enabled is a flag that can be used to enable or disable the trace exporter.
	Enabled bool `json:"enabled"`
	// exporter is the type of the exporter to sending data in OTLP protocol.
	// This should be set to the same type of the OpenTelemetry collector.
	// Valid values are "grpc", or "http".
	// Defaults to "grpc"
	Exporter string `json:"exporter"`
	// endpoint is the OpenTelemetry collector endpoint to connect to.
	// Defaults to "localhost:4317"
	Endpoint string `json:"endpoint"`
	// headers is a map of headers that will be sent with HTTP requests to the collector.
	Headers map[string]string `json:"headers"`
	// connection_timeout is the timeout for establishing a connection to the collector.
	// Defaults to 1 second.
	ConnectionTimeout int `json:"connection_timeout"`
	// resource_name is the name of the resource that will be used to identify the resource.
	// Defaults to "tyk"
	ResourceName string `json:"resource_name"`
	// span_processor_type is the type of the span processor to use.
	// Valid values are "simple" or "batch".
	// Defaults to "batch"
	SpanProcessorType string `json:"span_processor_type"`
	// context_propagation is the type of the context propagator to use.
	// Valid values are:
	// - "tracecontext": tracecontext is a propagator that supports the W3C
	// Trace Context format (https://www.w3.org/TR/trace-context/).
	// - "b3": b3 is a propagator serializes SpanContext to/from B3 multi Headers format.
	// Defaults to "tracecontext"
	ContextPropagation string `json:"context_propagation"`
	// tls_config is the TLS configuration for the exporter.
	TLSConfig TLSConfig `json:"tls_config"`
}

type TLSConfig struct {
	// Insecure is a flag that can be used to enable or disable TLS.
	// Defaults to false (secure).
	Insecure bool `json:"insecure"`
	// InsecureSkipVerify is a flag that can be used to skip TLS verification if TLS is enabled.
	// Defaults to false.
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

const (
	// available exporters types
	HTTPEXPORTER = "http"
	GRPCEXPORTER = "grpc"

	// available context propagators
	PROPAGATOR_TRACECONTEXT = "tracecontext"
	PROPAGATOR_B3           = "b3"
)

// SetDefaults sets the default values for the OpenTelemetry config.
func (c *OpenTelemetry) SetDefaults() {
	if !c.Enabled {
		return
	}

	if c.Exporter == "" {
		c.Exporter = GRPCEXPORTER
	}

	if c.Endpoint == "" {
		c.Endpoint = "localhost:4317"
	}

	if c.ConnectionTimeout == 0 {
		c.ConnectionTimeout = 1
	}

	if c.ResourceName == "" {
		c.ResourceName = "tyk"
	}

	if c.SpanProcessorType == "" {
		c.SpanProcessorType = "batch"
	}

	if c.ContextPropagation == "" {
		c.ContextPropagation = PROPAGATOR_TRACECONTEXT
	}
}
