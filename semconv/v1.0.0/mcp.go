// Package semconv provides semantic convention attributes for OpenTelemetry.
// This file implements MCP (Model Context Protocol) semantic conventions
// as defined in: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/
package semconv

import (
	"github.com/TykTechnologies/opentelemetry/trace"
	"go.opentelemetry.io/otel/attribute"
)

// MCP attribute prefixes as defined in the OpenTelemetry semantic conventions.
const (
	// MCPPrefix is the base prefix for all MCP attributes.
	MCPPrefix = "mcp."
	// GenAIPrefix is the base prefix for all GenAI attributes.
	GenAIPrefix = "gen_ai."
	// JSONRPCPrefix is the base prefix for all JSON-RPC attributes.
	JSONRPCPrefix = "jsonrpc."
	// RPCPrefix is the base prefix for RPC attributes.
	RPCPrefix = "rpc."
)

// MCP attribute keys as defined in the OpenTelemetry semantic conventions.
// Reference: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/
const (
	// MCPMethodNameKey identifies the MCP request or notification method.
	// Required attribute for all MCP spans.
	// Examples: "tools/call", "resources/read", "prompts/get", "initialize"
	MCPMethodNameKey = attribute.Key(MCPPrefix + "method.name")

	// MCPProtocolVersionKey is the MCP protocol version used.
	// Recommended attribute.
	MCPProtocolVersionKey = attribute.Key(MCPPrefix + "protocol.version")

	// MCPSessionIDKey is the session identifier for the MCP connection.
	// Recommended attribute when part of a session.
	MCPSessionIDKey = attribute.Key(MCPPrefix + "session.id")

	// MCPResourceURIKey is the URI of the resource being accessed.
	// Conditionally required when request includes a resource URI.
	MCPResourceURIKey = attribute.Key(MCPPrefix + "resource.uri")
)

// GenAI attribute keys for MCP tool and prompt operations.
const (
	// GenAIToolNameKey is the name of the tool being called.
	// Conditionally required when related to a specific tool.
	GenAIToolNameKey = attribute.Key(GenAIPrefix + "tool.name")

	// GenAIPromptNameKey is the name of the prompt being accessed.
	// Conditionally required when related to a specific prompt.
	GenAIPromptNameKey = attribute.Key(GenAIPrefix + "prompt.name")

	// GenAIOperationNameKey describes the operation being performed.
	// Recommended attribute. Set to "execute_tool" for tool calls.
	GenAIOperationNameKey = attribute.Key(GenAIPrefix + "operation.name")

	// GenAIToolCallArgumentsKey contains the tool call arguments.
	// Opt-in attribute - may contain sensitive data.
	GenAIToolCallArgumentsKey = attribute.Key(GenAIPrefix + "tool.call.arguments")

	// GenAIToolCallResultKey contains the tool execution result.
	// Opt-in attribute - may contain sensitive data.
	GenAIToolCallResultKey = attribute.Key(GenAIPrefix + "tool.call.result")
)

// JSON-RPC attribute keys for MCP protocol communication.
const (
	// JSONRPCRequestIDKey is the JSON-RPC request identifier.
	// Conditionally required when client executes a request.
	JSONRPCRequestIDKey = attribute.Key(JSONRPCPrefix + "request.id")

	// JSONRPCProtocolVersionKey is the JSON-RPC protocol version.
	// Recommended attribute. Include if not "2.0".
	JSONRPCProtocolVersionKey = attribute.Key(JSONRPCPrefix + "protocol.version")
)

// RPC attribute keys for response status.
const (
	// RPCResponseStatusCodeKey is the JSON-RPC error code if present.
	// Conditionally required if response contains error code.
	RPCResponseStatusCodeKey = attribute.Key(RPCPrefix + "response.status_code")
)

// GenAI operation name constants.
const (
	// GenAIOperationExecuteTool is the operation name for tool execution.
	GenAIOperationExecuteTool = "execute_tool"
)

// Well-known MCP method names as defined in the MCP specification.
const (
	MCPMethodInitialize       = "initialize"
	MCPMethodToolsCall        = "tools/call"
	MCPMethodToolsList        = "tools/list"
	MCPMethodResourcesRead    = "resources/read"
	MCPMethodResourcesList    = "resources/list"
	MCPMethodPromptsGet       = "prompts/get"
	MCPMethodPromptsList      = "prompts/list"
	MCPMethodPing             = "ping"
	MCPMethodCompletionComplete = "completion/complete"
)

// MCPMethodName returns an attribute KeyValue conforming to the
// "mcp.method.name" semantic convention. This is REQUIRED for all MCP spans.
// It identifies the MCP request or notification method being invoked.
func MCPMethodName(method string) trace.Attribute {
	return MCPMethodNameKey.String(method)
}

// MCPProtocolVersion returns an attribute KeyValue conforming to the
// "mcp.protocol.version" semantic convention.
// It represents the MCP protocol version used for the operation.
func MCPProtocolVersion(version string) trace.Attribute {
	return MCPProtocolVersionKey.String(version)
}

// MCPSessionID returns an attribute KeyValue conforming to the
// "mcp.session.id" semantic convention.
// It represents the session identifier for the MCP connection.
func MCPSessionID(sessionID string) trace.Attribute {
	return MCPSessionIDKey.String(sessionID)
}

// MCPResourceURI returns an attribute KeyValue conforming to the
// "mcp.resource.uri" semantic convention.
// It represents the URI of the resource being accessed.
func MCPResourceURI(uri string) trace.Attribute {
	return MCPResourceURIKey.String(uri)
}

// GenAIToolName returns an attribute KeyValue conforming to the
// "gen_ai.tool.name" semantic convention.
// It represents the name of the tool being called.
func GenAIToolName(name string) trace.Attribute {
	return GenAIToolNameKey.String(name)
}

// GenAIPromptName returns an attribute KeyValue conforming to the
// "gen_ai.prompt.name" semantic convention.
// It represents the name of the prompt being accessed.
func GenAIPromptName(name string) trace.Attribute {
	return GenAIPromptNameKey.String(name)
}

// GenAIOperationName returns an attribute KeyValue conforming to the
// "gen_ai.operation.name" semantic convention.
// It describes the GenAI operation being performed.
// Use GenAIOperationExecuteTool constant for tool calls.
func GenAIOperationName(operation string) trace.Attribute {
	return GenAIOperationNameKey.String(operation)
}

// GenAIToolCallArguments returns an attribute KeyValue conforming to the
// "gen_ai.tool.call.arguments" semantic convention.
// WARNING: This may contain sensitive data. Use with caution.
func GenAIToolCallArguments(arguments string) trace.Attribute {
	return GenAIToolCallArgumentsKey.String(arguments)
}

// GenAIToolCallResult returns an attribute KeyValue conforming to the
// "gen_ai.tool.call.result" semantic convention.
// WARNING: This may contain sensitive data. Use with caution.
func GenAIToolCallResult(result string) trace.Attribute {
	return GenAIToolCallResultKey.String(result)
}

// JSONRPCRequestID returns an attribute KeyValue conforming to the
// "jsonrpc.request.id" semantic convention.
// It represents the JSON-RPC request identifier.
func JSONRPCRequestID(id string) trace.Attribute {
	return JSONRPCRequestIDKey.String(id)
}

// JSONRPCRequestIDInt returns an attribute KeyValue conforming to the
// "jsonrpc.request.id" semantic convention for integer IDs.
// It represents the JSON-RPC request identifier as an integer.
func JSONRPCRequestIDInt(id int64) trace.Attribute {
	return JSONRPCRequestIDKey.Int64(id)
}

// JSONRPCProtocolVersion returns an attribute KeyValue conforming to the
// "jsonrpc.protocol.version" semantic convention.
// It represents the JSON-RPC protocol version (typically "2.0").
func JSONRPCProtocolVersion(version string) trace.Attribute {
	return JSONRPCProtocolVersionKey.String(version)
}

// RPCResponseStatusCode returns an attribute KeyValue conforming to the
// "rpc.response.status_code" semantic convention.
// It represents the JSON-RPC error code if present in the response.
func RPCResponseStatusCode(code int64) trace.Attribute {
	return RPCResponseStatusCodeKey.Int64(code)
}
