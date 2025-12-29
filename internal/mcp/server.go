package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/artpar/currier/internal/cookies"
	cookiesqlite "github.com/artpar/currier/internal/cookies/sqlite"
	"github.com/artpar/currier/internal/core"
	historysqlite "github.com/artpar/currier/internal/history/sqlite"
	"github.com/artpar/currier/internal/importer"
	"github.com/artpar/currier/internal/interpolate"
	protohttp "github.com/artpar/currier/internal/protocol/http"
	protows "github.com/artpar/currier/internal/protocol/websocket"
	"github.com/artpar/currier/internal/runner"
	"github.com/artpar/currier/internal/storage/filesystem"
)

// Server is the MCP server for Currier
type Server struct {
	transport   Transport
	collections *filesystem.CollectionStore
	envStore    *filesystem.EnvironmentStore
	history     *historysqlite.Store
	cookieJar   *cookies.PersistentJar
	httpClient  *protohttp.Client
	wsClient    *protows.Client

	tools     map[string]*toolDef
	resources map[string]*resourceDef

	// WebSocket message buffers per connection
	wsMessages   map[string][]*protows.Message
	wsMessagesMu sync.RWMutex

	initialized bool
	mu          sync.RWMutex
}

// toolDef defines a tool with its schema and handler
type toolDef struct {
	tool    Tool
	handler func(json.RawMessage) (*ToolCallResult, error)
}

// resourceDef defines a resource with its handler
type resourceDef struct {
	resource Resource
	handler  func() (*ResourceReadResult, error)
}

// ServerConfig holds configuration for the MCP server
type ServerConfig struct {
	DataDir string // Base directory for data (collections, environments, etc.)
}

// NewServer creates a new MCP server
func NewServer(config ServerConfig) (*Server, error) {
	// Determine data directory
	dataDir := config.DataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".config", "currier")
	}

	// Initialize stores
	collectionsPath := filepath.Join(dataDir, "collections")
	collectionStore, err := filesystem.NewCollectionStore(collectionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection store: %w", err)
	}

	envsPath := filepath.Join(dataDir, "environments")
	envStore, err := filesystem.NewEnvironmentStore(envsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment store: %w", err)
	}

	historyPath := filepath.Join(dataDir, "history.db")
	historyStore, err := historysqlite.New(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create history store: %w", err)
	}

	cookiesPath := filepath.Join(dataDir, "cookies.db")
	cookieStore, err := cookiesqlite.New(cookiesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie store: %w", err)
	}
	cookieJar, err := cookies.NewPersistentJar(cookieStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := protohttp.NewClient(
		protohttp.WithCookieJar(cookieJar),
	)

	wsClient := protows.NewClient(nil) // Use default config

	s := &Server{
		collections: collectionStore,
		envStore:    envStore,
		history:     historyStore,
		cookieJar:   cookieJar,
		httpClient:  httpClient,
		wsClient:    wsClient,
		tools:       make(map[string]*toolDef),
		resources:   make(map[string]*resourceDef),
		wsMessages:  make(map[string][]*protows.Message),
	}

	s.registerTools()
	s.registerResources()

	return s, nil
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	s.transport = NewStdioTransport(stdin, stdout)

	return MessageLoop(ctx, s.transport, s.handleRequest)
}

// handleRequest processes an incoming JSON-RPC request
func (s *Server) handleRequest(req *Request) *Response {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodInitialized:
		// Notification, no response needed
		return nil
	case MethodPing:
		return s.handlePing(req)
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(req)
	case MethodResourcesList:
		return s.handleResourcesList(req)
	case MethodResourcesRead:
		return s.handleResourcesRead(req)
	default:
		return &Response{
			Error: &ResponseError{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: "Invalid initialize params: " + err.Error(),
			},
		}
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools:     &ToolsCapability{},
			Resources: &ResourcesCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "currier",
			Version: "0.1.34",
		},
	}

	data, _ := json.Marshal(result)
	return &Response{Result: data}
}

func (s *Server) handlePing(req *Request) *Response {
	data, _ := json.Marshal(map[string]any{})
	return &Response{Result: data}
}

func (s *Server) handleToolsList(req *Request) *Response {
	tools := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, t.tool)
	}

	result := ToolsListResult{Tools: tools}
	data, _ := json.Marshal(result)
	return &Response{Result: data}
}

func (s *Server) handleToolsCall(req *Request) *Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: "Invalid tool call params: " + err.Error(),
			},
		}
	}

	toolDef, ok := s.tools[params.Name]
	if !ok {
		return &Response{
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		}
	}

	result, err := toolDef.handler(params.Arguments)
	if err != nil {
		result = &ToolCallResult{
			Content: []ContentBlock{ErrorContent(err)},
			IsError: true,
		}
	}

	data, _ := json.Marshal(result)
	return &Response{Result: data}
}

func (s *Server) handleResourcesList(req *Request) *Response {
	resources := make([]Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r.resource)
	}

	result := ResourcesListResult{Resources: resources}
	data, _ := json.Marshal(result)
	return &Response{Result: data}
}

func (s *Server) handleResourcesRead(req *Request) *Response {
	var params ResourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: "Invalid resource read params: " + err.Error(),
			},
		}
	}

	resDef, ok := s.resources[params.URI]
	if !ok {
		return &Response{
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: fmt.Sprintf("Unknown resource: %s", params.URI),
			},
		}
	}

	result, err := resDef.handler()
	if err != nil {
		return &Response{
			Error: &ResponseError{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	data, _ := json.Marshal(result)
	return &Response{Result: data}
}

// Close cleans up server resources
func (s *Server) Close() error {
	if s.history != nil {
		s.history.Close()
	}
	if s.wsClient != nil {
		s.wsClient.CloseAll()
	}
	return nil
}

// Helper to get environment for interpolation
func (s *Server) getEnvironment(name string) (map[string]string, error) {
	if name == "" {
		return nil, nil
	}

	ctx := context.Background()
	env, err := s.envStore.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", name, err)
	}

	return env.Variables(), nil
}

// Helper to create and send an HTTP request
func (s *Server) sendRequest(ctx context.Context, method, url string, headers map[string]string, body string, envName string) (*core.Response, error) {
	// Get environment variables if specified
	envVars, err := s.getEnvironment(envName)
	if err != nil {
		return nil, err
	}

	// Interpolate variables in URL
	if envVars != nil {
		engine := interpolate.NewEngine()
		for k, v := range envVars {
			engine.SetVariable(k, v)
		}
		url, _ = engine.Interpolate(url)
		for k, v := range headers {
			headers[k], _ = engine.Interpolate(v)
		}
		body, _ = engine.Interpolate(body)
	}

	// Create request
	req, err := core.NewRequest("http", method, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		req.Headers().Set(k, v)
	}

	// Set body
	if body != "" {
		req.SetBody(core.NewRawBody([]byte(body), ""))
	}

	// Send request
	return s.httpClient.Send(ctx, req)
}

// Helper to run a collection
func (s *Server) runCollection(ctx context.Context, collectionName, envName string) (*runner.RunSummary, error) {
	// Find collection
	collections, err := s.collections.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var coll *core.Collection
	for _, meta := range collections {
		if meta.Name == collectionName {
			coll, err = s.collections.Get(ctx, meta.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get collection: %w", err)
			}
			break
		}
	}

	if coll == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	// Build runner options
	opts := []runner.Option{
		runner.WithCookieJar(s.cookieJar),
	}

	// Get environment
	if envName != "" {
		env, err := s.envStore.Get(ctx, envName)
		if err != nil {
			return nil, fmt.Errorf("failed to get environment: %w", err)
		}
		opts = append(opts, runner.WithEnvironment(env))
	}

	// Create runner
	r := runner.NewRunner(coll, opts...)

	// Run collection
	return r.Run(ctx), nil
}

// Helper to parse curl command
func (s *Server) parseCurl(curlCmd string) (*core.Request, error) {
	// Use the curl importer to parse the command
	curlImporter := importer.NewCurlImporter()
	coll, err := curlImporter.Import(context.Background(), []byte(curlCmd))
	if err != nil {
		return nil, err
	}

	// Get the first request from the collection
	requests := coll.Requests()
	if len(requests) == 0 {
		return nil, fmt.Errorf("no request found in curl command")
	}

	reqDef := requests[0]

	// Convert RequestDefinition to Request
	req, err := core.NewRequest("http", reqDef.Method(), reqDef.FullURL())
	if err != nil {
		return nil, err
	}

	// Copy headers
	for k, v := range reqDef.Headers() {
		req.Headers().Set(k, v)
	}

	// Copy body
	if body := reqDef.BodyContent(); body != "" {
		req.SetBody(core.NewRawBody([]byte(body), ""))
	}

	return req, nil
}
