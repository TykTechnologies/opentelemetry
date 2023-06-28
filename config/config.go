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
}

const (
	HTTPEXPORTER = "http"
	GRPCEXPORTER = "grpc"
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
}
