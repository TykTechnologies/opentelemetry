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
	// Sampling defines the configurations to use in the sampler
	Sampling Sampling `json:"sampling"`
}

type Sampling struct {
	// sampler_type refers to the policy used by OpenTelemetry to determine
	// whether a particular trace should be sampled or not. It's determined at the
	// start of a trace and the decision is propagated down the trace. Valid Values are:
	// AlwaysOn, AlwaysOff and TraceIDRatioBased. It defaults to AlwaysOn
	SamplerType string `json:"sampler_type"`
	// sampling_rate is a parameter for the TraceIDRatioBased sampler type. It represents
	// the percentage of traces to be sampled. The value should be a float between 0.0 (0%) and 1.0 (100%).
	// If the sampling rate is 0.5, the sampler will aim to sample approximately 50% of traces.
	// it defaults to 0.5
	SamplingRate float64 `json:"sampling_rate"`
	// parent_based_sampling is a rule that makes sure that if we decide to record data
	// for a particular operation, we'll also record data for all the work that operation
	// causes (its "child spans"). This helps keep the whole story of a transaction together.
	// You usually use ParentBased with TraceIDRatioBased, because with AlwaysOn or AlwaysOff,
	// you're either recording everything or nothing, so there are no decisions to respect.
	// It defaults to false
	ParentBasedSampling bool `json:"parent_based_sampling"`
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

	if c.Sampling.SamplerType == "" {
		c.Sampling.SamplerType = ALWAYSON
	}

	if c.Sampling.SamplingRate == 0 {
		c.Sampling.SamplingRate = 0.5
	}
}
