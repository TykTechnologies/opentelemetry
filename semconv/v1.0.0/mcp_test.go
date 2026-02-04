package semconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestMCPMethodName(t *testing.T) {
	method := "tools/call"
	expectedAttribute := attribute.Key(MCPPrefix + "method.name").String(method)
	actualAttribute := MCPMethodName(method)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestMCPProtocolVersion(t *testing.T) {
	version := "2024-11-05"
	expectedAttribute := attribute.Key(MCPPrefix + "protocol.version").String(version)
	actualAttribute := MCPProtocolVersion(version)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestMCPSessionID(t *testing.T) {
	sessionID := "session-abc-123"
	expectedAttribute := attribute.Key(MCPPrefix + "session.id").String(sessionID)
	actualAttribute := MCPSessionID(sessionID)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestMCPResourceURI(t *testing.T) {
	uri := "file:///data/config.json"
	expectedAttribute := attribute.Key(MCPPrefix + "resource.uri").String(uri)
	actualAttribute := MCPResourceURI(uri)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGenAIToolName(t *testing.T) {
	name := "get_weather"
	expectedAttribute := attribute.Key(GenAIPrefix + "tool.name").String(name)
	actualAttribute := GenAIToolName(name)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGenAIPromptName(t *testing.T) {
	name := "code_review"
	expectedAttribute := attribute.Key(GenAIPrefix + "prompt.name").String(name)
	actualAttribute := GenAIPromptName(name)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGenAIOperationName(t *testing.T) {
	operation := GenAIOperationExecuteTool
	expectedAttribute := attribute.Key(GenAIPrefix + "operation.name").String(operation)
	actualAttribute := GenAIOperationName(operation)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGenAIToolCallArguments(t *testing.T) {
	arguments := `{"location": "San Francisco"}`
	expectedAttribute := attribute.Key(GenAIPrefix + "tool.call.arguments").String(arguments)
	actualAttribute := GenAIToolCallArguments(arguments)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestGenAIToolCallResult(t *testing.T) {
	result := `{"temperature": 72, "unit": "fahrenheit"}`
	expectedAttribute := attribute.Key(GenAIPrefix + "tool.call.result").String(result)
	actualAttribute := GenAIToolCallResult(result)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestJSONRPCRequestID(t *testing.T) {
	id := "req-123"
	expectedAttribute := attribute.Key(JSONRPCPrefix + "request.id").String(id)
	actualAttribute := JSONRPCRequestID(id)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestJSONRPCRequestIDInt(t *testing.T) {
	id := int64(42)
	expectedAttribute := attribute.Key(JSONRPCPrefix + "request.id").Int64(id)
	actualAttribute := JSONRPCRequestIDInt(id)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestJSONRPCProtocolVersion(t *testing.T) {
	version := "2.0"
	expectedAttribute := attribute.Key(JSONRPCPrefix + "protocol.version").String(version)
	actualAttribute := JSONRPCProtocolVersion(version)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestRPCResponseStatusCode(t *testing.T) {
	code := int64(-32600)
	expectedAttribute := attribute.Key(RPCPrefix + "response.status_code").Int64(code)
	actualAttribute := RPCResponseStatusCode(code)
	assert.Equal(t, expectedAttribute, actualAttribute, "The attributes should be equal")
}

func TestMCPMethodConstants(t *testing.T) {
	// Verify the method constants match MCP specification
	assert.Equal(t, "initialize", MCPMethodInitialize)
	assert.Equal(t, "tools/call", MCPMethodToolsCall)
	assert.Equal(t, "tools/list", MCPMethodToolsList)
	assert.Equal(t, "resources/read", MCPMethodResourcesRead)
	assert.Equal(t, "resources/list", MCPMethodResourcesList)
	assert.Equal(t, "prompts/get", MCPMethodPromptsGet)
	assert.Equal(t, "prompts/list", MCPMethodPromptsList)
	assert.Equal(t, "ping", MCPMethodPing)
	assert.Equal(t, "completion/complete", MCPMethodCompletionComplete)
}

func TestGenAIOperationConstants(t *testing.T) {
	assert.Equal(t, "execute_tool", GenAIOperationExecuteTool)
}
