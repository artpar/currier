package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/exporter"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/importer"
)

// Constants for pagination and content limits
const (
	DefaultPageSize     = 50
	MaxPageSize         = 200
	MaxResponseBodySize = 100 * 1024 // 100KB max response body in results
	TruncationMessage   = "\n... [TRUNCATED - response too large, showing first %d bytes of %d total]"
)

// truncateBody truncates response body if it exceeds the max size
func truncateBody(body string, maxSize int) (string, bool) {
	if len(body) <= maxSize {
		return body, false
	}
	return body[:maxSize] + fmt.Sprintf(TruncationMessage, maxSize, len(body)), true
}

// paginationParams are common pagination parameters
type paginationParams struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

// paginationResult contains pagination metadata
type paginationResult struct {
	Offset     int  `json:"offset"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	HasMore    bool `json:"has_more"`
	TotalPages int  `json:"total_pages,omitempty"`
}

// getFolderByName finds a subfolder by name within a folder
func getFolderByName(f *core.Folder, name string) (*core.Folder, bool) {
	for _, sub := range f.Folders() {
		if sub.Name() == name {
			return sub, true
		}
	}
	return nil, false
}

// applyPagination applies offset and limit to a slice
func applyPagination[T any](items []T, offset, limit int) ([]T, paginationResult) {
	total := len(items)

	// Apply defaults
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	if offset < 0 {
		offset = 0
	}

	// Apply offset
	if offset >= total {
		return []T{}, paginationResult{
			Offset:     offset,
			Limit:      limit,
			Total:      total,
			HasMore:    false,
			TotalPages: (total + limit - 1) / limit,
		}
	}

	items = items[offset:]

	// Apply limit
	hasMore := len(items) > limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items, paginationResult{
		Offset:     offset,
		Limit:      limit,
		Total:      total,
		HasMore:    hasMore,
		TotalPages: (total + limit - 1) / limit,
	}
}

// registerTools registers all MCP tools
func (s *Server) registerTools() {
	// Request tools
	s.registerSendRequest()
	s.registerSendCurl()

	// Collection CRUD tools
	s.registerListCollections()
	s.registerGetCollection()
	s.registerCreateCollection()
	s.registerDeleteCollection()
	s.registerRenameCollection()

	// Request CRUD tools
	s.registerGetRequest()
	s.registerSaveRequest()
	s.registerDeleteRequest()
	s.registerUpdateRequest()

	// Folder CRUD tools
	s.registerCreateFolder()
	s.registerDeleteFolder()

	// Collection runner
	s.registerRunCollection()

	// Environment tools
	s.registerListEnvironments()
	s.registerGetEnvironment()
	s.registerSetEnvironmentVariable()
	s.registerCreateEnvironment()
	s.registerDeleteEnvironment()
	s.registerDeleteEnvironmentVariable()

	// History tools
	s.registerGetHistory()
	s.registerSearchHistory()

	// Cookie tools
	s.registerListCookies()
	s.registerClearCookies()

	// Import/Export tools
	s.registerExportAsCurl()
	s.registerImportCollection()
	s.registerExportCollection()
}

// ============================================================================
// Request Tools
// ============================================================================

type sendRequestArgs struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Environment string            `json:"environment,omitempty"`
}

type sendRequestResult struct {
	Status      int               `json:"status"`
	StatusText  string            `json:"statusText"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	DurationMs  int64             `json:"duration_ms"`
	SizeBytes   int               `json:"size_bytes"`
	IsTruncated bool              `json:"is_truncated,omitempty"`
}

func (s *Server) registerSendRequest() {
	schema := `{
		"type": "object",
		"properties": {
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
				"description": "HTTP method"
			},
			"url": {
				"type": "string",
				"description": "Full URL or URL with {{variables}}"
			},
			"headers": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Request headers as key-value pairs"
			},
			"body": {
				"type": "string",
				"description": "Request body (JSON string for JSON content)"
			},
			"environment": {
				"type": "string",
				"description": "Environment name to use for variable interpolation"
			}
		},
		"required": ["method", "url"]
	}`

	s.tools["send_request"] = &toolDef{
		tool: Tool{
			Name:        "send_request",
			Description: "Send an HTTP request to an API endpoint and return the response",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params sendRequestArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.Method == "" {
				return nil, fmt.Errorf("method is required")
			}
			if params.URL == "" {
				return nil, fmt.Errorf("url is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := s.sendRequest(ctx, params.Method, params.URL, params.Headers, params.Body, params.Environment)
			if err != nil {
				return nil, err
			}

			// Convert response headers
			headers := make(map[string]string)
			for _, key := range resp.Headers().Keys() {
				headers[key] = resp.Headers().Get(key)
			}

			timing := resp.Timing()
			duration := timing.EndTime.Sub(timing.StartTime)

			// Truncate large response bodies
			bodyStr := resp.Body().String()
			bodyStr, isTruncated := truncateBody(bodyStr, MaxResponseBodySize)

			result := sendRequestResult{
				Status:      resp.Status().Code(),
				StatusText:  resp.Status().Text(),
				Headers:     headers,
				Body:        bodyStr,
				DurationMs:  duration.Milliseconds(),
				SizeBytes:   int(resp.Body().Size()),
				IsTruncated: isTruncated,
			}

			content, err := JSONContent(result)
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type sendCurlArgs struct {
	CurlCommand string `json:"curl_command"`
}

func (s *Server) registerSendCurl() {
	schema := `{
		"type": "object",
		"properties": {
			"curl_command": {
				"type": "string",
				"description": "Full curl command (e.g., curl -X POST -H 'Content-Type: application/json' ...)"
			}
		},
		"required": ["curl_command"]
	}`

	s.tools["send_curl"] = &toolDef{
		tool: Tool{
			Name:        "send_curl",
			Description: "Parse and execute a curl command, returning the response",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params sendCurlArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.CurlCommand == "" {
				return nil, fmt.Errorf("curl_command is required")
			}

			// Parse curl command
			req, err := s.parseCurl(params.CurlCommand)
			if err != nil {
				return nil, fmt.Errorf("failed to parse curl command: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Send request
			resp, err := s.httpClient.Send(ctx, req)
			if err != nil {
				return nil, err
			}

			// Convert response headers
			headers := make(map[string]string)
			for _, key := range resp.Headers().Keys() {
				headers[key] = resp.Headers().Get(key)
			}

			timing := resp.Timing()
			duration := timing.EndTime.Sub(timing.StartTime)

			// Truncate large response bodies
			bodyStr := resp.Body().String()
			bodyStr, isTruncated := truncateBody(bodyStr, MaxResponseBodySize)

			result := sendRequestResult{
				Status:      resp.Status().Code(),
				StatusText:  resp.Status().Text(),
				Headers:     headers,
				Body:        bodyStr,
				DurationMs:  duration.Milliseconds(),
				SizeBytes:   int(resp.Body().Size()),
				IsTruncated: isTruncated,
			}

			content, err := JSONContent(result)
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

// ============================================================================
// Collection Tools
// ============================================================================

type collectionInfo struct {
	Name         string `json:"name"`
	ID           string `json:"id"`
	Description  string `json:"description,omitempty"`
	RequestCount int    `json:"request_count"`
}

func (s *Server) registerListCollections() {
	schema := `{
		"type": "object",
		"properties": {}
	}`

	s.tools["list_collections"] = &toolDef{
		tool: Tool{
			Name:        "list_collections",
			Description: "List all saved API collections",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]collectionInfo, 0, len(collections))
			for _, c := range collections {
				result = append(result, collectionInfo{
					Name:         c.Name,
					ID:           c.ID,
					Description:  c.Description,
					RequestCount: c.RequestCount,
				})
			}

			content, err := JSONContent(map[string]any{"collections": result})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type getCollectionArgs struct {
	Name string `json:"name"`
}

type requestInfo struct {
	Name    string `json:"name"`
	Method  string `json:"method"`
	URL     string `json:"url"`
	Path    string `json:"path,omitempty"`
}

func (s *Server) registerGetCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Collection name"
			}
		},
		"required": ["name"]
	}`

	s.tools["get_collection"] = &toolDef{
		tool: Tool{
			Name:        "get_collection",
			Description: "Get all requests in a collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params getCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Name {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Name)
			}

			// Extract all requests
			requests := make([]requestInfo, 0)

			// Helper to walk folders recursively
			var walkFolder func(folder *core.Folder, path string)
			walkFolder = func(folder *core.Folder, path string) {
				// Add requests in this folder
				for _, req := range folder.Requests() {
					reqPath := path
					if reqPath != "" {
						reqPath += "/"
					}
					reqPath += req.Name()
					requests = append(requests, requestInfo{
						Name:   req.Name(),
						Method: req.Method(),
						URL:    req.URL(),
						Path:   reqPath,
					})
				}
				// Walk subfolders
				for _, subfolder := range folder.Folders() {
					newPath := path
					if newPath != "" {
						newPath += "/"
					}
					newPath += subfolder.Name()
					walkFolder(subfolder, newPath)
				}
			}

			// Add top-level requests
			for _, req := range coll.Requests() {
				requests = append(requests, requestInfo{
					Name:   req.Name(),
					Method: req.Method(),
					URL:    req.URL(),
					Path:   req.Name(),
				})
			}

			// Walk folders
			for _, folder := range coll.Folders() {
				walkFolder(folder, folder.Name())
			}

			content, err := JSONContent(map[string]any{
				"name":        coll.Name(),
				"description": coll.Description(),
				"requests":    requests,
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type getRequestArgs struct {
	Collection string `json:"collection"`
	Request    string `json:"request"`
}

func (s *Server) registerGetRequest() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"request": {
				"type": "string",
				"description": "Request name or path (e.g., 'Get Users' or 'Users/Get All')"
			}
		},
		"required": ["collection", "request"]
	}`

	s.tools["get_request"] = &toolDef{
		tool: Tool{
			Name:        "get_request",
			Description: "Get a specific request from a collection by name or path",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params getRequestArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			// Find request by name or path
			var found *core.RequestDefinition

			// Helper to search in folder
			var findInFolder func(folder *core.Folder, path string) bool
			findInFolder = func(folder *core.Folder, path string) bool {
				for _, req := range folder.Requests() {
					reqPath := path
					if reqPath != "" {
						reqPath += "/"
					}
					reqPath += req.Name()
					if req.Name() == params.Request || reqPath == params.Request {
						found = req
						return true
					}
				}
				for _, subfolder := range folder.Folders() {
					newPath := path
					if newPath != "" {
						newPath += "/"
					}
					newPath += subfolder.Name()
					if findInFolder(subfolder, newPath) {
						return true
					}
				}
				return false
			}

			// Search top-level requests
			for _, req := range coll.Requests() {
				if req.Name() == params.Request {
					found = req
					break
				}
			}

			// Search in folders
			if found == nil {
				for _, folder := range coll.Folders() {
					if findInFolder(folder, folder.Name()) {
						break
					}
				}
			}

			if found == nil {
				return nil, fmt.Errorf("request not found: %s", params.Request)
			}

			// Build headers map
			headers := found.Headers()

			// Build query params
			queryParams := found.QueryParams()

			result := map[string]any{
				"name":         found.Name(),
				"method":       found.Method(),
				"url":          found.URL(),
				"headers":      headers,
				"query_params": queryParams,
				"body":         found.BodyContent(),
			}

			content, err := JSONContent(result)
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type saveRequestArgs struct {
	Collection string            `json:"collection"`
	Folder     string            `json:"folder,omitempty"`
	Name       string            `json:"name"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
}

func (s *Server) registerSaveRequest() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name (created if doesn't exist)"
			},
			"folder": {
				"type": "string",
				"description": "Optional folder path within collection"
			},
			"name": {
				"type": "string",
				"description": "Request name"
			},
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
			},
			"url": {
				"type": "string",
				"description": "Request URL"
			},
			"headers": {
				"type": "object",
				"additionalProperties": {"type": "string"}
			},
			"body": {
				"type": "string",
				"description": "Request body"
			}
		},
		"required": ["collection", "name", "method", "url"]
	}`

	s.tools["save_request"] = &toolDef{
		tool: Tool{
			Name:        "save_request",
			Description: "Save a request to a collection (creates collection if it doesn't exist)",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params saveRequestArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()

			// Find or create collection
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				coll = core.NewCollection(params.Collection)
			}

			// Create request definition
			reqDef := core.NewRequestDefinition(params.Name, params.Method, params.URL)
			for k, v := range params.Headers {
				reqDef.SetHeader(k, v)
			}
			if params.Body != "" {
				reqDef.SetBodyRaw(params.Body, "")
			}

			// Add to collection (or folder)
			if params.Folder != "" {
				// Find or create folder
				folder, found := coll.GetFolderByName(params.Folder)
				if !found {
					folder = coll.AddFolder(params.Folder)
				}
				folder.AddRequest(reqDef)
			} else {
				coll.AddRequest(reqDef)
			}

			// Save collection
			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			content, err := JSONContent(map[string]any{
				"success":    true,
				"collection": params.Collection,
				"request":    params.Name,
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type runCollectionArgs struct {
	Name        string `json:"name"`
	Environment string `json:"environment,omitempty"`
}

func (s *Server) registerRunCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Collection name"
			},
			"environment": {
				"type": "string",
				"description": "Environment to use"
			}
		},
		"required": ["name"]
	}`

	s.tools["run_collection"] = &toolDef{
		tool: Tool{
			Name:        "run_collection",
			Description: "Execute all requests in a collection and return results with test outcomes",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params runCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			summary, err := s.runCollection(ctx, params.Name, params.Environment)
			if err != nil {
				return nil, err
			}

			// Format results
			results := make([]map[string]any, 0, len(summary.Results))
			for _, r := range summary.Results {
				tests := make([]map[string]any, 0, len(r.TestResults))
				for _, t := range r.TestResults {
					tests = append(tests, map[string]any{
						"name":   t.Name,
						"passed": t.Passed,
						"error":  t.Error,
					})
				}

				result := map[string]any{
					"name":        r.RequestName,
					"status":      r.Status,
					"passed":      r.IsSuccess(),
					"duration_ms": r.Duration.Milliseconds(),
					"tests":       tests,
				}
				if r.Error != nil {
					result["error"] = r.Error.Error()
				}
				results = append(results, result)
			}

			output := map[string]any{
				"summary": map[string]any{
					"total_requests": summary.TotalRequests,
					"passed":         summary.Passed,
					"failed":         summary.Failed,
					"total_tests":    summary.TotalTests,
					"tests_passed":   summary.TestsPassed,
					"tests_failed":   summary.TestsFailed,
					"duration_ms":    summary.TotalDuration.Milliseconds(),
				},
				"results": results,
			}

			content, err := JSONContent(output)
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

// ============================================================================
// Environment Tools
// ============================================================================

func (s *Server) registerListEnvironments() {
	schema := `{
		"type": "object",
		"properties": {}
	}`

	s.tools["list_environments"] = &toolDef{
		tool: Tool{
			Name:        "list_environments",
			Description: "List all available environments",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			ctx := context.Background()
			envs, err := s.envStore.List(ctx)
			if err != nil {
				return nil, err
			}

			result := make([]map[string]any, 0, len(envs))
			for _, e := range envs {
				result = append(result, map[string]any{
					"name": e.Name,
					"id":   e.ID,
				})
			}

			content, err := JSONContent(map[string]any{"environments": result})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type getEnvironmentArgs struct {
	Name string `json:"name"`
}

func (s *Server) registerGetEnvironment() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Environment name"
			}
		},
		"required": ["name"]
	}`

	s.tools["get_environment"] = &toolDef{
		tool: Tool{
			Name:        "get_environment",
			Description: "Get all variables in an environment",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params getEnvironmentArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			env, err := s.envStore.Get(ctx, params.Name)
			if err != nil {
				return nil, fmt.Errorf("environment not found: %s", params.Name)
			}

			content, err := JSONContent(map[string]any{
				"name":      env.Name(),
				"variables": env.Variables(),
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type setEnvVarArgs struct {
	Environment string `json:"environment"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

func (s *Server) registerSetEnvironmentVariable() {
	schema := `{
		"type": "object",
		"properties": {
			"environment": {
				"type": "string",
				"description": "Environment name"
			},
			"key": {
				"type": "string",
				"description": "Variable name"
			},
			"value": {
				"type": "string",
				"description": "Variable value"
			}
		},
		"required": ["environment", "key", "value"]
	}`

	s.tools["set_environment_variable"] = &toolDef{
		tool: Tool{
			Name:        "set_environment_variable",
			Description: "Set or update a variable in an environment",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params setEnvVarArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			env, err := s.envStore.Get(ctx, params.Environment)
			if err != nil {
				// Create new environment
				env = core.NewEnvironment(params.Environment)
			}

			env.SetVariable(params.Key, params.Value)

			if err := s.envStore.Save(ctx, env); err != nil {
				return nil, fmt.Errorf("failed to save environment: %w", err)
			}

			content, err := JSONContent(map[string]any{
				"success":     true,
				"environment": params.Environment,
				"key":         params.Key,
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

// ============================================================================
// History Tools
// ============================================================================

type getHistoryArgs struct {
	Limit  int    `json:"limit,omitempty"`
	Filter string `json:"filter,omitempty"`
}

func (s *Server) registerGetHistory() {
	schema := `{
		"type": "object",
		"properties": {
			"limit": {
				"type": "integer",
				"description": "Maximum number of entries (default 20)"
			},
			"filter": {
				"type": "string",
				"description": "Filter by URL pattern"
			}
		}
	}`

	s.tools["get_history"] = &toolDef{
		tool: Tool{
			Name:        "get_history",
			Description: "Get recent request history",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params getHistoryArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			limit := params.Limit
			if limit <= 0 {
				limit = 20
			}

			ctx := context.Background()
			entries, err := s.history.List(ctx, history.QueryOptions{Limit: limit})
			if err != nil {
				return nil, err
			}

			result := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				// Apply filter if specified
				if params.Filter != "" {
					// Simple substring match
					if !contains(e.RequestURL, params.Filter) {
						continue
					}
				}

				result = append(result, map[string]any{
					"id":          e.ID,
					"method":      e.RequestMethod,
					"url":         e.RequestURL,
					"status":      e.ResponseStatus,
					"duration_ms": e.ResponseTime,
					"timestamp":   e.Timestamp.Format(time.RFC3339),
				})
			}

			content, err := JSONContent(map[string]any{"history": result})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Cookie Tools
// ============================================================================

type listCookiesArgs struct {
	Domain string `json:"domain,omitempty"`
}

func (s *Server) registerListCookies() {
	schema := `{
		"type": "object",
		"properties": {
			"domain": {
				"type": "string",
				"description": "Filter by domain"
			}
		}
	}`

	s.tools["list_cookies"] = &toolDef{
		tool: Tool{
			Name:        "list_cookies",
			Description: "List all cookies in the cookie jar",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params listCookiesArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			allCookies, err := s.cookieJar.ListAll()
			if err != nil {
				return nil, fmt.Errorf("failed to list cookies: %w", err)
			}

			result := make([]map[string]any, 0)
			for _, c := range allCookies {
				if params.Domain != "" && c.Domain != params.Domain {
					continue
				}
				result = append(result, map[string]any{
					"name":    c.Name,
					"value":   c.Value,
					"domain":  c.Domain,
					"path":    c.Path,
					"expires": c.Expires.Format(time.RFC3339),
					"secure":  c.Secure,
				})
			}

			content, err := JSONContent(map[string]any{"cookies": result})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type clearCookiesArgs struct {
	Domain string `json:"domain,omitempty"`
}

func (s *Server) registerClearCookies() {
	schema := `{
		"type": "object",
		"properties": {
			"domain": {
				"type": "string",
				"description": "Domain to clear (omit for all)"
			}
		}
	}`

	s.tools["clear_cookies"] = &toolDef{
		tool: Tool{
			Name:        "clear_cookies",
			Description: "Clear all cookies or cookies for a specific domain",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params clearCookiesArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.Domain != "" {
				s.cookieJar.ClearDomain(params.Domain)
			} else {
				s.cookieJar.Clear()
			}

			content, err := JSONContent(map[string]any{
				"success": true,
				"domain":  params.Domain,
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

// ============================================================================
// Export Tools
// ============================================================================

type exportAsCurlArgs struct {
	Collection string `json:"collection"`
	Request    string `json:"request"`
}

func (s *Server) registerExportAsCurl() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"request": {
				"type": "string",
				"description": "Request name"
			}
		},
		"required": ["collection", "request"]
	}`

	s.tools["export_as_curl"] = &toolDef{
		tool: Tool{
			Name:        "export_as_curl",
			Description: "Export a request as a curl command",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params exportAsCurlArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			// Find request
			var found *core.RequestDefinition

			// Helper to search in folder
			var findInFolder func(folder *core.Folder) bool
			findInFolder = func(folder *core.Folder) bool {
				for _, req := range folder.Requests() {
					if req.Name() == params.Request {
						found = req
						return true
					}
				}
				for _, subfolder := range folder.Folders() {
					if findInFolder(subfolder) {
						return true
					}
				}
				return false
			}

			// Search top-level requests
			for _, req := range coll.Requests() {
				if req.Name() == params.Request {
					found = req
					break
				}
			}

			// Search in folders
			if found == nil {
				for _, folder := range coll.Folders() {
					if findInFolder(folder) {
						break
					}
				}
			}

			if found == nil {
				return nil, fmt.Errorf("request not found: %s", params.Request)
			}

			// Export to curl
			curlExporter := exporter.NewCurlExporter()
			curlCmd, err := curlExporter.ExportRequest(context.Background(), found)
			if err != nil {
				return nil, fmt.Errorf("failed to export as curl: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(string(curlCmd))},
			}, nil
		},
	}
}

// ============================================================================
// Collection CRUD Tools
// ============================================================================

type createCollectionArgs struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (s *Server) registerCreateCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Collection name"
			},
			"description": {
				"type": "string",
				"description": "Collection description"
			}
		},
		"required": ["name"]
	}`

	s.tools["create_collection"] = &toolDef{
		tool: Tool{
			Name:        "create_collection",
			Description: "Create a new API collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params createCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.Name == "" {
				return nil, fmt.Errorf("name is required")
			}

			ctx := context.Background()
			coll := core.NewCollection(params.Name)
			if params.Description != "" {
				coll.SetDescription(params.Description)
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to create collection: %w", err)
			}

			content, err := JSONContent(map[string]any{
				"id":      coll.ID(),
				"name":    coll.Name(),
				"message": "Collection created successfully",
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

type deleteCollectionArgs struct {
	Name string `json:"name"`
}

func (s *Server) registerDeleteCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Collection name to delete"
			}
		},
		"required": ["name"]
	}`

	s.tools["delete_collection"] = &toolDef{
		tool: Tool{
			Name:        "delete_collection",
			Description: "Delete an API collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params deleteCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var collID string
			for _, meta := range collections {
				if meta.Name == params.Name {
					collID = meta.ID
					break
				}
			}

			if collID == "" {
				return nil, fmt.Errorf("collection not found: %s", params.Name)
			}

			if err := s.collections.Delete(ctx, collID); err != nil {
				return nil, fmt.Errorf("failed to delete collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Collection '%s' deleted successfully", params.Name))},
			}, nil
		},
	}
}

type renameCollectionArgs struct {
	Name    string `json:"name"`
	NewName string `json:"new_name"`
}

func (s *Server) registerRenameCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Current collection name"
			},
			"new_name": {
				"type": "string",
				"description": "New collection name"
			}
		},
		"required": ["name", "new_name"]
	}`

	s.tools["rename_collection"] = &toolDef{
		tool: Tool{
			Name:        "rename_collection",
			Description: "Rename an API collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params renameCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Name {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Name)
			}

			coll.SetName(params.NewName)
			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to rename collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Collection renamed from '%s' to '%s'", params.Name, params.NewName))},
			}, nil
		},
	}
}

// ============================================================================
// Request CRUD Tools
// ============================================================================

type deleteRequestArgs struct {
	Collection string `json:"collection"`
	Request    string `json:"request"`
}

func (s *Server) registerDeleteRequest() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"request": {
				"type": "string",
				"description": "Request name or path (e.g., 'Get Users' or 'Users/Get All')"
			}
		},
		"required": ["collection", "request"]
	}`

	s.tools["delete_request"] = &toolDef{
		tool: Tool{
			Name:        "delete_request",
			Description: "Delete a request from a collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params deleteRequestArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			// Parse path if provided
			parts := strings.Split(params.Request, "/")
			requestName := parts[len(parts)-1]

			// Try to delete from root
			deleted := false
			if len(parts) == 1 {
				for _, req := range coll.Requests() {
					if req.Name() == requestName {
						coll.RemoveRequest(req.ID())
						deleted = true
						break
					}
				}
			} else {
				// Navigate to folder
				folderPath := parts[:len(parts)-1]
				currentFolder, _ := coll.GetFolderByName(folderPath[0])
				for i := 1; i < len(folderPath) && currentFolder != nil; i++ {
					currentFolder, _ = getFolderByName(currentFolder, folderPath[i])
				}
				if currentFolder != nil {
					for _, req := range currentFolder.Requests() {
						if req.Name() == requestName {
							currentFolder.RemoveRequest(req.ID())
							deleted = true
							break
						}
					}
				}
			}

			if !deleted {
				return nil, fmt.Errorf("request not found: %s", params.Request)
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Request '%s' deleted successfully", params.Request))},
			}, nil
		},
	}
}

type updateRequestArgs struct {
	Collection string            `json:"collection"`
	Request    string            `json:"request"`
	Method     string            `json:"method,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	NewName    string            `json:"new_name,omitempty"`
}

func (s *Server) registerUpdateRequest() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"request": {
				"type": "string",
				"description": "Request name or path"
			},
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
				"description": "New HTTP method"
			},
			"url": {
				"type": "string",
				"description": "New URL"
			},
			"headers": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "New headers (replaces existing)"
			},
			"body": {
				"type": "string",
				"description": "New request body"
			},
			"new_name": {
				"type": "string",
				"description": "Rename the request"
			}
		},
		"required": ["collection", "request"]
	}`

	s.tools["update_request"] = &toolDef{
		tool: Tool{
			Name:        "update_request",
			Description: "Update an existing request in a collection (method, URL, headers, body, or name)",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params updateRequestArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			// Find the request
			var found *core.RequestDefinition
			var findInFolder func(folder *core.Folder) bool
			findInFolder = func(folder *core.Folder) bool {
				for _, req := range folder.Requests() {
					if req.Name() == params.Request {
						found = req
						return true
					}
				}
				for _, subfolder := range folder.Folders() {
					if findInFolder(subfolder) {
						return true
					}
				}
				return false
			}

			for _, req := range coll.Requests() {
				if req.Name() == params.Request {
					found = req
					break
				}
			}
			if found == nil {
				for _, folder := range coll.Folders() {
					if findInFolder(folder) {
						break
					}
				}
			}

			if found == nil {
				return nil, fmt.Errorf("request not found: %s", params.Request)
			}

			// Apply updates
			if params.Method != "" {
				found.SetMethod(params.Method)
			}
			if params.URL != "" {
				found.SetURL(params.URL)
			}
			if params.Headers != nil {
				// Clear existing headers first
				for k := range found.Headers() {
					found.RemoveHeader(k)
				}
				for k, v := range params.Headers {
					found.SetHeader(k, v)
				}
			}
			if params.Body != "" {
				found.SetBodyRaw(params.Body, "")
			}
			if params.NewName != "" {
				found.SetName(params.NewName)
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Request '%s' updated successfully", params.Request))},
			}, nil
		},
	}
}

// ============================================================================
// Folder Tools
// ============================================================================

type createFolderArgs struct {
	Collection  string `json:"collection"`
	Name        string `json:"name"`
	Parent      string `json:"parent,omitempty"`
	Description string `json:"description,omitempty"`
}

func (s *Server) registerCreateFolder() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"name": {
				"type": "string",
				"description": "Folder name"
			},
			"parent": {
				"type": "string",
				"description": "Parent folder path (optional, creates at root if not specified)"
			},
			"description": {
				"type": "string",
				"description": "Folder description"
			}
		},
		"required": ["collection", "name"]
	}`

	s.tools["create_folder"] = &toolDef{
		tool: Tool{
			Name:        "create_folder",
			Description: "Create a new folder in a collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params createFolderArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			var folder *core.Folder
			if params.Parent == "" {
				folder = coll.AddFolder(params.Name)
			} else {
				// Navigate to parent folder
				parts := strings.Split(params.Parent, "/")
				parentFolder, _ := coll.GetFolderByName(parts[0])
				for i := 1; i < len(parts) && parentFolder != nil; i++ {
					parentFolder, _ = getFolderByName(parentFolder, parts[i])
				}
				if parentFolder == nil {
					return nil, fmt.Errorf("parent folder not found: %s", params.Parent)
				}
				folder = parentFolder.AddFolder(params.Name)
			}
			if params.Description != "" {
				folder.SetDescription(params.Description)
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Folder '%s' created successfully", params.Name))},
			}, nil
		},
	}
}

type deleteFolderArgs struct {
	Collection string `json:"collection"`
	Folder     string `json:"folder"`
}

func (s *Server) registerDeleteFolder() {
	schema := `{
		"type": "object",
		"properties": {
			"collection": {
				"type": "string",
				"description": "Collection name"
			},
			"folder": {
				"type": "string",
				"description": "Folder path (e.g., 'Users' or 'Users/Admin')"
			}
		},
		"required": ["collection", "folder"]
	}`

	s.tools["delete_folder"] = &toolDef{
		tool: Tool{
			Name:        "delete_folder",
			Description: "Delete a folder and its contents from a collection",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params deleteFolderArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Collection {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Collection)
			}

			parts := strings.Split(params.Folder, "/")
			deleted := false

			if len(parts) == 1 {
				// Delete from root
				for _, f := range coll.Folders() {
					if f.Name() == parts[0] {
						coll.RemoveFolder(f.ID())
						deleted = true
						break
					}
				}
			} else {
				// Navigate to parent and delete
				parentFolder, _ := coll.GetFolderByName(parts[0])
				for i := 1; i < len(parts)-1 && parentFolder != nil; i++ {
					parentFolder, _ = getFolderByName(parentFolder, parts[i])
				}
				if parentFolder != nil {
					folderName := parts[len(parts)-1]
					for _, f := range parentFolder.Folders() {
						if f.Name() == folderName {
							parentFolder.RemoveFolder(f.ID())
							deleted = true
							break
						}
					}
				}
			}

			if !deleted {
				return nil, fmt.Errorf("folder not found: %s", params.Folder)
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Folder '%s' deleted successfully", params.Folder))},
			}, nil
		},
	}
}

// ============================================================================
// Additional Environment Tools
// ============================================================================

type createEnvironmentArgs struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (s *Server) registerCreateEnvironment() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Environment name"
			},
			"variables": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Initial variables"
			}
		},
		"required": ["name"]
	}`

	s.tools["create_environment"] = &toolDef{
		tool: Tool{
			Name:        "create_environment",
			Description: "Create a new environment",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params createEnvironmentArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.Name == "" {
				return nil, fmt.Errorf("name is required")
			}

			ctx := context.Background()
			env := core.NewEnvironment(params.Name)
			for k, v := range params.Variables {
				env.SetVariable(k, v)
			}

			if err := s.envStore.Save(ctx, env); err != nil {
				return nil, fmt.Errorf("failed to create environment: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Environment '%s' created successfully", params.Name))},
			}, nil
		},
	}
}

type deleteEnvironmentArgs struct {
	Name string `json:"name"`
}

func (s *Server) registerDeleteEnvironment() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Environment name to delete"
			}
		},
		"required": ["name"]
	}`

	s.tools["delete_environment"] = &toolDef{
		tool: Tool{
			Name:        "delete_environment",
			Description: "Delete an environment",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params deleteEnvironmentArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			if err := s.envStore.Delete(ctx, params.Name); err != nil {
				return nil, fmt.Errorf("failed to delete environment: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Environment '%s' deleted successfully", params.Name))},
			}, nil
		},
	}
}

type deleteEnvironmentVariableArgs struct {
	Environment string `json:"environment"`
	Key         string `json:"key"`
}

func (s *Server) registerDeleteEnvironmentVariable() {
	schema := `{
		"type": "object",
		"properties": {
			"environment": {
				"type": "string",
				"description": "Environment name"
			},
			"key": {
				"type": "string",
				"description": "Variable name to delete"
			}
		},
		"required": ["environment", "key"]
	}`

	s.tools["delete_environment_variable"] = &toolDef{
		tool: Tool{
			Name:        "delete_environment_variable",
			Description: "Delete a variable from an environment",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params deleteEnvironmentVariableArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			env, err := s.envStore.Get(ctx, params.Environment)
			if err != nil {
				return nil, fmt.Errorf("environment not found: %s", params.Environment)
			}

			env.DeleteVariable(params.Key)

			if err := s.envStore.Save(ctx, env); err != nil {
				return nil, fmt.Errorf("failed to save environment: %w", err)
			}

			return &ToolCallResult{
				Content: []ContentBlock{TextContent(fmt.Sprintf("Variable '%s' deleted from environment '%s'", params.Key, params.Environment))},
			}, nil
		},
	}
}

// ============================================================================
// Search History Tool
// ============================================================================

type searchHistoryArgs struct {
	Query  string `json:"query,omitempty"`
	Method string `json:"method,omitempty"`
	Status int    `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

func (s *Server) registerSearchHistory() {
	schema := `{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query (matches URL)"
			},
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
				"description": "Filter by HTTP method"
			},
			"status": {
				"type": "integer",
				"description": "Filter by response status code"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of entries (default 50, max 200)"
			},
			"offset": {
				"type": "integer",
				"description": "Offset for pagination"
			}
		}
	}`

	s.tools["search_history"] = &toolDef{
		tool: Tool{
			Name:        "search_history",
			Description: "Search request history with filters and pagination",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params searchHistoryArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			limit := params.Limit
			if limit <= 0 {
				limit = DefaultPageSize
			}
			if limit > MaxPageSize {
				limit = MaxPageSize
			}

			ctx := context.Background()
			// Get more entries to allow for filtering
			entries, err := s.history.List(ctx, history.QueryOptions{Limit: limit * 2})
			if err != nil {
				return nil, err
			}

			// Apply filters
			filtered := make([]history.Entry, 0)
			for _, e := range entries {
				if params.Query != "" && !strings.Contains(e.RequestURL, params.Query) {
					continue
				}
				if params.Method != "" && e.RequestMethod != params.Method {
					continue
				}
				if params.Status != 0 && e.ResponseStatus != params.Status {
					continue
				}
				filtered = append(filtered, e)
			}

			// Apply pagination
			paged, pagination := applyPagination(filtered, params.Offset, limit)

			result := make([]map[string]any, 0, len(paged))
			for _, e := range paged {
				result = append(result, map[string]any{
					"id":          e.ID,
					"method":      e.RequestMethod,
					"url":         e.RequestURL,
					"status":      e.ResponseStatus,
					"duration_ms": e.ResponseTime,
					"timestamp":   e.Timestamp.Format(time.RFC3339),
				})
			}

			content, err := JSONContent(map[string]any{
				"history":    result,
				"pagination": pagination,
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}

// ============================================================================
// Import/Export Tools
// ============================================================================

type importCollectionArgs struct {
	Content string `json:"content"`
	Format  string `json:"format,omitempty"`
}

func (s *Server) registerImportCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"content": {
				"type": "string",
				"description": "Collection content (JSON for Postman, YAML/JSON for OpenAPI, curl command)"
			},
			"format": {
				"type": "string",
				"enum": ["postman", "openapi", "curl", "har", "auto"],
				"description": "Import format (auto-detected if not specified)"
			}
		},
		"required": ["content"]
	}`

	s.tools["import_collection"] = &toolDef{
		tool: Tool{
			Name:        "import_collection",
			Description: "Import a collection from Postman, OpenAPI, cURL, or HAR format",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params importCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			if params.Content == "" {
				return nil, fmt.Errorf("content is required")
			}

			ctx := context.Background()
			var coll *core.Collection
			var format string

			content := []byte(params.Content)

			if params.Format == "" || params.Format == "auto" {
				// Auto-detect format
				registry := importer.NewRegistry()
				registry.Register(importer.NewPostmanImporter())
				registry.Register(importer.NewOpenAPIImporter())
				registry.Register(importer.NewCurlImporter())
				registry.Register(importer.NewHARImporter())

				result, err := registry.DetectAndImport(ctx, content)
				if err != nil {
					return nil, fmt.Errorf("failed to import: %w", err)
				}
				coll = result.Collection
				format = string(result.SourceFormat)
			} else {
				var imp importer.Importer
				switch params.Format {
				case "postman":
					imp = importer.NewPostmanImporter()
				case "openapi":
					imp = importer.NewOpenAPIImporter()
				case "curl":
					imp = importer.NewCurlImporter()
				case "har":
					imp = importer.NewHARImporter()
				default:
					return nil, fmt.Errorf("unknown format: %s", params.Format)
				}

				var err error
				coll, err = imp.Import(ctx, content)
				if err != nil {
					return nil, fmt.Errorf("failed to import: %w", err)
				}
				format = params.Format
			}

			if err := s.collections.Save(ctx, coll); err != nil {
				return nil, fmt.Errorf("failed to save collection: %w", err)
			}

			content2, err := JSONContent(map[string]any{
				"name":          coll.Name(),
				"id":            coll.ID(),
				"format":        format,
				"request_count": len(coll.Requests()),
				"message":       "Collection imported successfully",
			})
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content2},
			}, nil
		},
	}
}

type exportCollectionArgs struct {
	Name   string `json:"name"`
	Format string `json:"format,omitempty"`
}

func (s *Server) registerExportCollection() {
	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Collection name to export"
			},
			"format": {
				"type": "string",
				"enum": ["postman", "curl"],
				"description": "Export format (default: postman)"
			}
		},
		"required": ["name"]
	}`

	s.tools["export_collection"] = &toolDef{
		tool: Tool{
			Name:        "export_collection",
			Description: "Export a collection to Postman JSON or curl format",
			InputSchema: json.RawMessage(schema),
		},
		handler: func(args json.RawMessage) (*ToolCallResult, error) {
			var params exportCollectionArgs
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			ctx := context.Background()
			collections, err := s.collections.List(ctx)
			if err != nil {
				return nil, err
			}

			var coll *core.Collection
			for _, meta := range collections {
				if meta.Name == params.Name {
					coll, err = s.collections.Get(ctx, meta.ID)
					if err != nil {
						return nil, err
					}
					break
				}
			}

			if coll == nil {
				return nil, fmt.Errorf("collection not found: %s", params.Name)
			}

			format := params.Format
			if format == "" {
				format = "postman"
			}

			var exp exporter.Exporter
			switch format {
			case "postman":
				exp = exporter.NewPostmanExporter()
			case "curl":
				exp = exporter.NewCurlExporter()
			default:
				return nil, fmt.Errorf("unknown format: %s", format)
			}

			data, err := exp.Export(ctx, coll)
			if err != nil {
				return nil, fmt.Errorf("failed to export: %w", err)
			}

			// Truncate if too large
			dataStr := string(data)
			dataStr, isTruncated := truncateBody(dataStr, MaxResponseBodySize)

			result := map[string]any{
				"format":       format,
				"content":      dataStr,
				"is_truncated": isTruncated,
			}

			content, err := JSONContent(result)
			if err != nil {
				return nil, err
			}

			return &ToolCallResult{
				Content: []ContentBlock{content},
			}, nil
		},
	}
}
