package semconv

import (
	"github.com/TykTechnologies/opentelemetry/trace"
	"go.opentelemetry.io/otel/attribute"
)

const (
	TykAPIPrefix = "tyk.api."

	TykAPIIDKey = attribute.Key(TykAPIPrefix + "id")

	TykAPINameKey = attribute.Key(TykAPIPrefix + "name")

	TykAPIVersionKey = attribute.Key(TykAPIPrefix + "version")

	TykAPIOrgIDKey = attribute.Key(TykAPIPrefix + "orgid")

	TykAPITagsKey = attribute.Key(TykAPIPrefix + "tags")
)

// TykAPIID returns an attribute KeyValue conforming to the
// "tyk.api.id" semantic convention. It represents the id
// of the Tyk API.
func TykAPIID(id string) trace.Attribute {
	return TykAPIIDKey.String(id)
}

// TykAPIName returns an attribute KeyValue conforming to the
// "tyk.api.name" semantic convention. It represents the name
// of the Tyk API.
func TykAPIName(name string) trace.Attribute {
	return TykAPINameKey.String(name)
}

// TykAPIVersion returns an attribute KeyValue conforming to the
// "tyk.api.version" semantic convention. It represents the version
// of the Tyk API.
func TykAPIVersion(version string) trace.Attribute {
	return TykAPIVersionKey.String(version)
}

// TykAPIOrgIDKey returns an attribute KeyValue conforming to the
// "tyk.api.orgid" semantic convention. It represents the org_id
// of the Tyk API.
func TykAPIOrgID(orgid string) trace.Attribute {
	return TykAPIOrgIDKey.String(orgid)
}

// TykAPITags returns an attribute KeyValue conforming to the
// "tyk.api.tags" semantic convention. It represents the tags
// of the Tyk API concatenated by a space.
func TykAPITags(tags ...string) trace.Attribute {
	return TykAPITagsKey.StringSlice(tags)
}
