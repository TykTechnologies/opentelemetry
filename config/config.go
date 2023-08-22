package config

type OpenTelemetry struct {
	// Enabled is a flag that can be used to enable or disable the trace exporter.
	Enabled bool `json:"enabled"`
	// Exporter is the type of the exporter to sending data in OTLP protocol.
	// This should be set to the same type of the OpenTelemetry collector.
	// Valid values are "grpc", or "http".
	// Defaults to "grpc".
	Exporter string `json:"exporter"`
	// Endpoint is the OpenTelemetry collector endpoint to connect to.
	// Defaults to "localhost:4317".
	Endpoint string `json:"endpoint"`
	// Headers is a map of headers that will be sent with HTTP requests to the collector.
	Headers map[string]string `json:"headers"`
	// Connection_timeout is the timeout for establishing a connection to the collector.
	// Defaults to 1 second.
	ConnectionTimeout int `json:"connection_timeout"`
	// Resource_name is the name of the resource that will be used to identify the resource.
	// Defaults to "tyk".
	ResourceName string `json:"resource_name"`
	// Span_processor_type is the type of the span processor to use.
	// Valid values are "simple" or "batch".
	// Defaults to "batch".
	SpanProcessorType string `json:"span_processor_type"`
	// Context_propagation is the type of the context propagator to use.
	// Valid values are:
	// - "tracecontext": tracecontext is a propagator that supports the W3C
	// Trace Context format (https://www.w3.org/TR/trace-context/).
	// - "b3": b3 is a propagator serializes SpanContext to/from B3 multi Headers format.
	// Defaults to "tracecontext".
	ContextPropagation string `json:"context_propagation"`
	// Tls is the TLS configuration for the exporter.
	TLS TLS `json:"tls"`
	// Sampling defines the configurations to use in the sampler.
	Sampling Sampling `json:"sampling"`
}

type TLS struct {
	// Enable is a flag that can be used to enable TLS.
	// Defaults to false (disabled).
	Enable bool `json:"enable"`
	// Insecure_skip_verify is a flag that can be used to skip TLS verification if TLS is enabled.
	// Defaults to false.
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
	// Ca_file is the path to the CA file.
	CAFile string `json:"ca_file"`
	// Cert_file is the path to the cert file.
	CertFile string `json:"cert_file"`
	// Key_file is the path to the key file.
	KeyFile string `json:"key_file"`
	// Max_version is the maximum TLS version that is supported.
	// Options: ["1.0", "1.1", "1.2", "1.3"].
	// Defaults to "1.3".
	MaxVersion string `json:"max_version"`
	// Min_version is the minimum TLS version that is supported.
	// Options: ["1.0", "1.1", "1.2", "1.3"].
	// Defaults to "1.2".
	MinVersion string `json:"min_version"`
}

type Sampling struct {
	// Type refers to the policy used by OpenTelemetry to determine
	// whether a particular trace should be sampled or not. It's determined at the
	// start of a trace and the decision is propagated down the trace. Valid Values are:
	// AlwaysOn, AlwaysOff and TraceIDRatioBased. It defaults to AlwaysOn.
	Type string `json:"type"`
	// Sampling_rate is a parameter for the TraceIDRatioBased sampler type and represents the percentage
	// of traces to be sampled. The value should fall between 0.0 (0%) and 1.0 (100%). For instance, if
	// the sampling rate is set to 0.5, the sampler will aim to sample approximately 50% of the traces.
	// By default, it's set to 0.5.
	Rate float64 `json:"rate"`
	// Parent_based is a rule that ensures that if we decide to record data for a particular operation,
	// we'll also record data for all the subsequent work that operation causes (its "child spans").
	// This approach helps in keeping the entire story of a transaction together. Typically, ParentBased
	// is used in conjunction with TraceIDRatioBased. Using it with AlwaysOn or AlwaysOff might not be as
	// effective since, in those cases, you're either recording everything or nothing, and there are no
	// intermediary decisions to consider. The default value for this option is false.
	ParentBased bool `json:"parent_based"`
}

const (
	// available exporters types
	HTTPEXPORTER = "http"
	GRPCEXPORTER = "grpc"

	// available context propagators
	PROPAGATOR_TRACECONTEXT = "tracecontext"
	PROPAGATOR_B3           = "b3"

	// available sampler types
	ALWAYSON          = "AlwaysOn"
	ALWAYSOFF         = "AlwaysOff"
	TRACEIDRATIOBASED = "TraceIDRatioBased"
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

	if c.Sampling.Type == "" {
		c.Sampling.Type = ALWAYSON
	}

	if c.Sampling.Type == TRACEIDRATIOBASED && c.Sampling.Rate == 0 {
		c.Sampling.Rate = 0.5
	}
}
