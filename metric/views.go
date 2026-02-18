package metric

import (
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/TykTechnologies/opentelemetry/config"
)

// buildViews converts config-driven MetricViewConfig entries into OTel SDK Views.
func buildViews(configs []config.MetricViewConfig) []sdkmetric.View {
	var views []sdkmetric.View

	for _, cfg := range configs {
		inst := sdkmetric.Instrument{
			Name: cfg.InstrumentName,
		}

		if cfg.InstrumentType != "" {
			inst.Kind = parseInstrumentKind(cfg.InstrumentType)
		}

		stream := sdkmetric.Stream{}

		if cfg.StreamName != "" {
			stream.Name = cfg.StreamName
		}

		if len(cfg.AllowAttributes) > 0 {
			stream.AttributeFilter = attribute.NewAllowKeysFilter(toKeys(cfg.AllowAttributes)...)
		} else if len(cfg.DropAttributes) > 0 {
			stream.AttributeFilter = attribute.NewDenyKeysFilter(toKeys(cfg.DropAttributes)...)
		}

		if cfg.Aggregation != "" {
			stream.Aggregation = parseAggregation(cfg.Aggregation, cfg.HistogramBuckets)
		} else if len(cfg.HistogramBuckets) > 0 {
			stream.Aggregation = sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: cfg.HistogramBuckets,
			}
		}

		views = append(views, sdkmetric.NewView(inst, stream))
	}

	return views
}

func toKeys(keys []string) []attribute.Key {
	result := make([]attribute.Key, len(keys))
	for i, k := range keys {
		result[i] = attribute.Key(k)
	}
	return result
}

func parseInstrumentKind(t string) sdkmetric.InstrumentKind {
	switch t {
	case "counter":
		return sdkmetric.InstrumentKindCounter
	case "histogram":
		return sdkmetric.InstrumentKindHistogram
	case "gauge":
		return sdkmetric.InstrumentKindGauge
	case "updowncounter":
		return sdkmetric.InstrumentKindUpDownCounter
	default:
		return 0
	}
}

func parseAggregation(agg string, buckets []float64) sdkmetric.Aggregation {
	switch agg {
	case "drop":
		return sdkmetric.AggregationDrop{}
	case "sum":
		return sdkmetric.AggregationSum{}
	case "last_value":
		return sdkmetric.AggregationLastValue{}
	case "explicit_bucket_histogram":
		if len(buckets) > 0 {
			return sdkmetric.AggregationExplicitBucketHistogram{Boundaries: buckets}
		}
		return sdkmetric.AggregationExplicitBucketHistogram{}
	default:
		return nil
	}
}
