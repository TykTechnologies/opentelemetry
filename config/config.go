package config

// ExporterConfig holds shared transport fields for OTLP exporters.
// It is embedded by both OpenTelemetry (traces) and MetricsConfig (metrics)
// so each can target a different collector independently.
type ExporterConfig struct {
	// The type of the exporter to sending data in OTLP protocol.
	// This should be set to the same type of the OpenTelemetry collector.
	// Valid values are "grpc", or "http".
	// Defaults to "grpc".
	Exporter string `json:"exporter"`
	// OpenTelemetry collector endpoint to connect to.
	// Defaults to "localhost:4317".
	Endpoint string `json:"endpoint"`
	// A map of headers that will be sent with HTTP requests to the collector.
	Headers map[string]string `json:"headers"`
	// Timeout for establishing a connection to the collector.
	// Defaults to 1 second.
	ConnectionTimeout int `json:"connection_timeout"`
	// Name of the resource that will be used to identify the resource.
	// Defaults to "tyk".
	ResourceName string `json:"resource_name"`
	// TLS configuration for the exporter.
	TLS TLS `json:"tls"`
}

type OpenTelemetry struct {
	// A flag that can be used to enable or disable the trace exporter.
	Enabled bool `json:"enabled"`
	// Shared exporter/transport configuration.
	ExporterConfig `json:",inline"`
	// Type of the span processor to use. Valid values are "simple" or "batch".
	// Defaults to "batch".
	SpanProcessorType string `json:"span_processor_type"`
	// Configuration for the batch span processor.
	// Only applies when SpanProcessorType is "batch".
	SpanBatchConfig SpanBatchConfig `json:"span_batch_config"`
	// Type of the context propagator to use. Valid values are:
	// - "tracecontext": tracecontext is a propagator that supports the W3C
	// Trace Context format (https://www.w3.org/TR/trace-context/).
	// - "b3": b3 is a propagator serializes SpanContext to/from B3 multi Headers format.
	// - "custom": custom propagator reads from and writes to a custom header only.
	// - "composite": composite propagator reads from custom header (priority) or standard headers,
	//   and writes to both custom and standard headers.
	// Defaults to "tracecontext".
	ContextPropagation string `json:"context_propagation"`
	// Name of custom header to use for trace ID instead of standard "traceparent".
	// When set with context_propagation="custom", the gateway will extract trace context
	// from this header and propagate it using the same custom header.
	// When set with context_propagation="tracecontext" or "b3", the gateway will extract
	// trace context from this header (priority) or standard headers (fallback), and propagate
	// using standard headers only.
	// When set with context_propagation="composite", the gateway will extract trace context
	// from this header (priority) or standard headers (fallback), and propagate using both
	// custom and standard headers.
	// Example: "X-Correlation-ID", "X-Request-ID", "X-Trace-ID"
	//
	// The header value should be a valid OpenTelemetry Trace ID: a 32-character (16-byte)
	// lowercase hex string with at least one non-zero byte
	// (e.g. "0102030405060708090a0b0c0d0e0f10"). UUIDs with dashes are also accepted
	// (e.g. "550e8400-e29b-41d4-a716-446655440000") — dashes are removed automatically.
	// See: https://opentelemetry.io/docs/specs/otel/trace/api/
	//
	// If the value contains non-hex characters, those characters will be stripped and the
	// remaining hex characters will be zero-padded to 32 characters. This means arbitrary
	// strings like "my-request-id" will NOT produce a predictable trace ID. To ensure
	// trace ID consistency between the custom header and the reported trace, always send
	// a valid OpenTelemetry Trace ID or UUID.
	CustomTraceHeader string `json:"custom_trace_header"`
	// Defines the configurations to use in the sampler.
	Sampling Sampling `json:"sampling"`
}

// MetricsConfig holds the configuration for OpenTelemetry metrics.
// It embeds ExporterConfig so metrics can target a different collector
// than traces, with its own endpoint, headers, TLS, etc.
type MetricsConfig struct {
	// A flag that can be used to enable or disable metrics export.
	// Must be explicitly set to true to enable metrics. If omitted (nil), metrics are disabled.
	Enabled *bool `json:"enabled,omitempty"`
	// Shared exporter/transport configuration.
	ExporterConfig `json:",inline"`
	// The interval in seconds at which metrics are exported.
	// Defaults to 60 seconds.
	ExportInterval int `json:"export_interval"`

	// Temporality defines the aggregation temporality for metrics.
	// Valid values are "cumulative" (default) or "delta".
	// Cumulative is preferred for Prometheus-style backends.
	// Delta is preferred for some cloud backends like AWS CloudWatch.
	Temporality string `json:"temporality"`

	// ShutdownTimeout is the timeout in seconds for graceful shutdown of the exporter.
	// Defaults to 30 seconds.
	ShutdownTimeout int `json:"shutdown_timeout"`

	// Retry configuration for exporter failures.
	Retry MetricsRetryConfig `json:"retry"`
}

// MetricsRetryConfig configures retry behavior for metric export failures.
type MetricsRetryConfig struct {
	// Enabled enables retry on export failures.
	// Defaults to true.
	Enabled *bool `json:"enabled,omitempty"`
	// InitialInterval is the initial backoff interval in milliseconds.
	// Defaults to 5000 (5 seconds).
	InitialInterval int `json:"initial_interval"`
	// MaxInterval is the maximum backoff interval in milliseconds.
	// Defaults to 30000 (30 seconds).
	MaxInterval int `json:"max_interval"`
	// MaxElapsedTime is the maximum total time in milliseconds to keep retrying.
	// After this duration, the export is abandoned.
	// Defaults to 60000 (1 minute).
	MaxElapsedTime int `json:"max_elapsed_time"`
}

type TLS struct {
	// Flag that can be used to enable TLS. Defaults to false (disabled).
	Enable bool `json:"enable"`
	// Flag that can be used to skip TLS verification if TLS is enabled.
	// Defaults to false.
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
	// Path to the CA file.
	CAFile string `json:"ca_file"`
	// Path to the cert file.
	CertFile string `json:"cert_file"`
	// Path to the key file.
	KeyFile string `json:"key_file"`
	// Maximum TLS version that is supported.
	// Options: ["1.0", "1.1", "1.2", "1.3"].
	// Defaults to "1.3".
	MaxVersion string `json:"max_version"`
	// Minimum TLS version that is supported.
	// Options: ["1.0", "1.1", "1.2", "1.3"].
	// Defaults to "1.2".
	MinVersion string `json:"min_version"`
}

// SpanBatchConfig configures the batch span processor.
type SpanBatchConfig struct {
	// MaxQueueSize is the maximum queue size to buffer spans for delayed processing.
	// If the queue gets full it drops the spans.
	// The default value is 2048.
	MaxQueueSize int `json:"max_queue_size"`
	// MaxExportBatchSize is the maximum number of spans to process in a single batch.
	// If there are more than one batch worth of spans then it processes multiple batches
	// of spans one batch after the other without any delay.
	// The default value is 512.
	MaxExportBatchSize int `json:"max_export_batch_size"`
	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available spans when timeout is reached.
	// The default value is 5 seconds.
	BatchTimeout int `json:"batch_timeout"`
}

type Sampling struct {
	// Refers to the policy used by OpenTelemetry to determine
	// whether a particular trace should be sampled or not. It's determined at the
	// start of a trace and the decision is propagated down the trace. Valid Values are:
	// AlwaysOn, AlwaysOff and TraceIDRatioBased. It defaults to AlwaysOn.
	Type string `json:"type"`
	// Parameter for the TraceIDRatioBased sampler type and represents the percentage
	// of traces to be sampled. The value should fall between 0.0 (0%) and 1.0 (100%). For instance, if
	// the sampling rate is set to 0.5, the sampler will aim to sample approximately 50% of the traces.
	// By default, it's set to 0.5.
	Rate float64 `json:"rate"`
	// Rule that ensures that if we decide to record data for a particular operation,
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
	PROPAGATOR_CUSTOM       = "custom"
	PROPAGATOR_COMPOSITE    = "composite"

	// available sampler types
	ALWAYSON          = "AlwaysOn"
	ALWAYSOFF         = "AlwaysOff"
	TRACEIDRATIOBASED = "TraceIDRatioBased"

	// available metric temporality types
	TEMPORALITY_CUMULATIVE = "cumulative"
	TEMPORALITY_DELTA      = "delta"
)

// SetDefaults sets the default values for the shared exporter config.
func (c *ExporterConfig) SetDefaults() {
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
}

// SetDefaults sets the default values for the OpenTelemetry trace config.
func (c *OpenTelemetry) SetDefaults() {
	if !c.Enabled {
		return
	}

	c.ExporterConfig.SetDefaults()

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

	if c.SpanProcessorType == "batch" {
		if c.SpanBatchConfig.MaxQueueSize == 0 {
			c.SpanBatchConfig.MaxQueueSize = 2048
		}

		if c.SpanBatchConfig.MaxExportBatchSize == 0 {
			c.SpanBatchConfig.MaxExportBatchSize = 512
		}

		if c.SpanBatchConfig.BatchTimeout == 0 {
			c.SpanBatchConfig.BatchTimeout = 5
		}
	}
}

// SetDefaults sets the default values for the metrics retry config.
func (c *MetricsRetryConfig) SetDefaults() {
	if c.Enabled == nil {
		enabled := true
		c.Enabled = &enabled
	}

	if c.InitialInterval == 0 {
		c.InitialInterval = 5000 // 5 seconds
	}

	if c.MaxInterval == 0 {
		c.MaxInterval = 30000 // 30 seconds
	}

	if c.MaxElapsedTime == 0 {
		c.MaxElapsedTime = 60000 // 1 minute
	}
}

// SetDefaults sets the default values for the metrics config.
// Note: Enabled has no default — if omitted (nil), metrics are disabled.
// Consumers must explicitly set enabled=true to enable metrics.
func (c *MetricsConfig) SetDefaults() {
	c.ExporterConfig.SetDefaults()

	if c.ExportInterval == 0 {
		c.ExportInterval = 60
	}

	if c.Temporality == "" {
		c.Temporality = TEMPORALITY_CUMULATIVE
	}

	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 30
	}

	c.Retry.SetDefaults()
}
