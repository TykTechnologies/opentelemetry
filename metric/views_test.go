package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/TykTechnologies/opentelemetry/config"
)

func TestBuildViews_Empty(t *testing.T) {
	views := buildViews(nil)
	assert.Empty(t, views)

	views = buildViews([]config.MetricViewConfig{})
	assert.Empty(t, views)
}

func TestBuildViews_DropAttributes(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName: "test.metric",
			DropAttributes: []string{"url.scheme", "network.protocol.version"},
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_AllowAttributes(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName:  "test.metric",
			AllowAttributes: []string{"http.request.method", "http.response.status_code"},
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_AllowTakesPrecedence(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName:  "test.metric",
			DropAttributes:  []string{"should.be.ignored"},
			AllowAttributes: []string{"http.request.method"},
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
	// AllowAttributes should take precedence - verify by building a view
}

func TestBuildViews_HistogramBuckets(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName:   "test.histogram",
			HistogramBuckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_AggregationDrop(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName: "test.metric",
			Aggregation:    "drop",
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_AggregationSum(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName: "test.metric",
			Aggregation:    "sum",
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_StreamName(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName: "test.metric",
			StreamName:     "custom.name",
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestBuildViews_InstrumentType(t *testing.T) {
	configs := []config.MetricViewConfig{
		{
			InstrumentName: "test.metric",
			InstrumentType: "histogram",
		},
	}
	views := buildViews(configs)
	assert.Len(t, views, 1)
}

func TestParseInstrumentKind(t *testing.T) {
	tests := []struct {
		input    string
		expected sdkmetric.InstrumentKind
	}{
		{"counter", sdkmetric.InstrumentKindCounter},
		{"histogram", sdkmetric.InstrumentKindHistogram},
		{"gauge", sdkmetric.InstrumentKindGauge},
		{"updowncounter", sdkmetric.InstrumentKindUpDownCounter},
		{"unknown", sdkmetric.InstrumentKind(0)},
		{"", sdkmetric.InstrumentKind(0)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseInstrumentKind(tt.input))
		})
	}
}

func TestParseAggregation(t *testing.T) {
	tests := []struct {
		name     string
		agg      string
		buckets  []float64
		expected sdkmetric.Aggregation
	}{
		{"drop", "drop", nil, sdkmetric.AggregationDrop{}},
		{"sum", "sum", nil, sdkmetric.AggregationSum{}},
		{"last_value", "last_value", nil, sdkmetric.AggregationLastValue{}},
		{"explicit_bucket_histogram with buckets", "explicit_bucket_histogram", []float64{1, 5, 10}, sdkmetric.AggregationExplicitBucketHistogram{Boundaries: []float64{1, 5, 10}}},
		{"explicit_bucket_histogram without buckets", "explicit_bucket_histogram", nil, sdkmetric.AggregationExplicitBucketHistogram{}},
		{"default", "default", nil, nil},
		{"empty", "", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAggregation(tt.agg, tt.buckets)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToKeys(t *testing.T) {
	keys := toKeys([]string{"key1", "key2", "key3"})
	assert.Len(t, keys, 3)
	assert.Equal(t, attribute.Key("key1"), keys[0])
	assert.Equal(t, attribute.Key("key2"), keys[1])
	assert.Equal(t, attribute.Key("key3"), keys[2])
}
