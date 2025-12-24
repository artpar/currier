package interfaces

import (
	"context"
)

// AgentServer exposes functionality to AI agents via MCP or similar protocols.
type AgentServer interface {
	// Start starts the agent server.
	Start(ctx context.Context) error

	// Stop gracefully stops the server.
	Stop() error

	// RegisterTool registers a tool for agents to use.
	RegisterTool(tool AgentTool) error

	// ListTools returns all available tools.
	ListTools() []ToolInfo

	// ExecuteTool executes a tool by name.
	ExecuteTool(ctx context.Context, name string, params map[string]any) (any, error)
}

// AgentTool represents a capability exposed to AI agents.
type AgentTool interface {
	// Name returns the tool name.
	Name() string

	// Description returns a description for the AI agent.
	Description() string

	// Parameters returns the parameter schema.
	Parameters() ToolParameters

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params map[string]any) (any, error)
}

// ToolInfo provides metadata about a tool.
type ToolInfo struct {
	Name        string
	Description string
	Parameters  ToolParameters
}

// ToolParameters describes tool input parameters.
type ToolParameters struct {
	Type       string                   // "object"
	Properties map[string]ParameterInfo
	Required   []string
}

// ParameterInfo describes a single parameter.
type ParameterInfo struct {
	Type        string   // "string", "number", "boolean", "object", "array"
	Description string
	Enum        []string // Allowed values for string type
	Default     any
	Items       *ParameterInfo // For array types
	Properties  map[string]ParameterInfo // For object types
}

// AgentTransport handles communication with agents.
type AgentTransport interface {
	// Start begins listening for agent connections.
	Start(ctx context.Context) error

	// Stop stops the transport.
	Stop() error

	// SetHandler sets the message handler.
	SetHandler(handler AgentMessageHandler)
}

// AgentMessageHandler processes messages from agents.
type AgentMessageHandler interface {
	// HandleMessage processes an incoming message.
	HandleMessage(ctx context.Context, msg AgentMessage) (AgentMessage, error)
}

// AgentMessage represents a message to/from an agent.
type AgentMessage struct {
	ID      string
	Type    string // "request", "response", "notification"
	Method  string // For requests
	Params  map[string]any
	Result  any
	Error   *AgentError
}

// AgentError represents an error in agent communication.
type AgentError struct {
	Code    int
	Message string
	Data    any
}

// Standard tool names
const (
	ToolSendRequest     = "send_request"
	ToolListCollections = "list_collections"
	ToolGetRequest      = "get_request"
	ToolRunRequest      = "run_request"
	ToolRunCollection   = "run_collection"
	ToolImportCollection = "import_collection"
	ToolExportCollection = "export_collection"
	ToolSetEnvironment  = "set_environment"
	ToolGetEnvironment  = "get_environment"
	ToolGetHistory      = "get_history"
	ToolSetVariable     = "set_variable"
	ToolGetVariable     = "get_variable"
)
