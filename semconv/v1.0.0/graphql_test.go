package semconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestGraphQLOperationName(t *testing.T) {
	name := "MyQuery"
	expectedAttribute := attribute.Key(GraphQLOperationPrefix + "name").String(name)
	actualAttribute := GraphQLOperationName(name)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGraphQLOperationType(t *testing.T) {
	operationType := "mutation"
	expectedAttribute := attribute.Key(GraphQLOperationPrefix + "type").String(operationType)
	actualAttribute := GraphQLOperationType(operationType)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGraphQLDocument(t *testing.T) {
	document := "query{}"
	expectedAttribute := GraphQLDocumentKey.String(document)
	actualAttribute := GraphQLDocument(document)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}
