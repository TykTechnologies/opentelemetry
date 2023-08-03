package semconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestTykGWID(t *testing.T) {
	id := "gateway-123"
	expectedAttribute := attribute.Key(TykGWPrefix + "id").String(id)
	actualAttribute := TykGWID(id)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestTykGWDataplane(t *testing.T) {
	isDataplane := true
	expectedAttribute := attribute.Key(TykGWPrefix + "dataplane").Bool(isDataplane)
	actualAttribute := TykGWDataplane(isDataplane)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestTykDataplaneGWGroupID(t *testing.T) {
	groupID := "group-123"
	expectedAttribute := attribute.Key(TykGWPrefix + "group.id").String(groupID)
	actualAttribute := TykDataplaneGWGroupID(groupID)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestTykGWSegmentTags(t *testing.T) {
	tags := []string{"tag1", "tag2", "tag3"}
	expectedAttribute := attribute.Key(TykGWPrefix + "tags").StringSlice(tags)
	actualAttribute := TykGWSegmentTags(tags...)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}
