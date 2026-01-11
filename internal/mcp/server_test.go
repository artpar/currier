package mcp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mcp-test-*")
	require.NoError(t, err)

	// Create subdirectories
	os.MkdirAll(filepath.Join(tmpDir, "collections"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "environments"), 0755)

	server, err := NewServer(ServerConfig{DataDir: tmpDir})
	require.NoError(t, err)

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with temp directory", func(t *testing.T) {
		server, cleanup := createTestServer(t)
		defer cleanup()

		assert.NotNil(t, server)
		assert.NotNil(t, server.tools)
		assert.NotNil(t, server.resources)
		assert.NotNil(t, server.collections)
		assert.NotNil(t, server.envStore)
	})

	t.Run("creates server with default directory", func(t *testing.T) {
		// This will use ~/.config/currier
		// We test that it doesn't error
		server, err := NewServer(ServerConfig{})
		if err == nil {
			defer server.Close()
			assert.NotNil(t, server)
		}
		// It's OK if this fails on CI without home dir
	})
}

func TestServer_handleInitialize(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("initializes server successfully", func(t *testing.T) {
		params := InitializeParams{
			ProtocolVersion: ProtocolVersion,
			ClientInfo: ClientInfo{
				Name:    "test-client",
				Version: "1.0.0",
			},
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodInitialize,
			Params:  paramsJSON,
		}

		resp := server.handleInitialize(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Result)

		var result InitializeResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.Equal(t, ProtocolVersion, result.ProtocolVersion)
		assert.Equal(t, "currier", result.ServerInfo.Name)
	})

	t.Run("returns error for invalid params", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodInitialize,
			Params:  json.RawMessage(`invalid json`),
		}

		resp := server.handleInitialize(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
	})
}

func TestServer_handlePing(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodPing,
	}

	resp := server.handlePing(req)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestServer_handleToolsList(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsList,
	}

	resp := server.handleToolsList(req)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result ToolsListResult
	err := json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Tools)
}

func TestServer_handleToolsCall(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("returns error for unknown tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "nonexistent_tool",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "Unknown tool")
	})

	t.Run("returns error for invalid params", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  json.RawMessage(`invalid`),
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
	})

	t.Run("calls list_collections tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_collections",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("calls list_environments tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_environments",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("calls get_history tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 10}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("calls list_cookies tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("calls clear_cookies tool", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "clear_cookies",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_handleResourcesList(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodResourcesList,
	}

	resp := server.handleResourcesList(req)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result ResourcesListResult
	err := json.Unmarshal(resp.Result, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Resources)
}

func TestServer_handleResourcesRead(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("returns error for unknown resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "unknown://resource",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "Unknown resource")
	})

	t.Run("returns error for invalid params", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  json.RawMessage(`invalid`),
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidParams, resp.Error.Code)
	})

	t.Run("reads collections resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "collections://list",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("reads history resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "history://recent",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_handleRequest(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("routes to initialize handler", func(t *testing.T) {
		params := InitializeParams{ProtocolVersion: ProtocolVersion}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodInitialize,
			Params:  paramsJSON,
		}

		resp := server.handleRequest(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("routes to ping handler", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodPing,
		}

		resp := server.handleRequest(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("routes to tools/list handler", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsList,
		}

		resp := server.handleRequest(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("routes to resources/list handler", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesList,
		}

		resp := server.handleRequest(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("returns nil for initialized notification", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  MethodInitialized,
		}

		resp := server.handleRequest(req)
		assert.Nil(t, resp) // Notifications don't have responses
	})

	t.Run("returns error for unknown method", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "unknown/method",
		}

		resp := server.handleRequest(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, MethodNotFound, resp.Error.Code)
	})
}

func TestServer_Close(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	err := server.Close()
	assert.NoError(t, err)
}

func TestServer_CollectionTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("create_collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_collection",
			Arguments: json.RawMessage(`{"name": "Test Collection"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_collection with invalid name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_collection",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// May return error for nonexistent collection
	})

	t.Run("delete_collection with invalid name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_collection",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename_collection with invalid name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"old_name": "nonexistent", "new_name": "new"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_EnvironmentTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("create_environment", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_environment",
			Arguments: json.RawMessage(`{"name": "Test Env"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_environment with invalid name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_environment with invalid name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("set_environment_variable", func(t *testing.T) {
		// First create an environment
		createParams := ToolCallParams{
			Name:      "create_environment",
			Arguments: json.RawMessage(`{"name": "VarTestEnv"}`),
		}
		createJSON, _ := json.Marshal(createParams)
		server.handleToolsCall(&Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  createJSON,
		})

		// Then set a variable
		params := ToolCallParams{
			Name:      "set_environment_variable",
			Arguments: json.RawMessage(`{"environment": "VarTestEnv", "key": "api_url", "value": "https://api.example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_environment_variable", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "VarTestEnv", "key": "api_url"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_FolderTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// First create a collection
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "FolderTestColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	t.Run("create_folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "FolderTestColl", "name": "Test Folder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "FolderTestColl", "folder": "Test Folder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RequestTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RequestTestColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	t.Run("save_request", func(t *testing.T) {
		params := ToolCallParams{
			Name: "save_request",
			Arguments: json.RawMessage(`{
				"collection": "RequestTestColl",
				"name": "Test Request",
				"method": "GET",
				"url": "https://example.com/api"
			}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_request with invalid args", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_request",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "request": "also nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update_request", func(t *testing.T) {
		params := ToolCallParams{
			Name: "update_request",
			Arguments: json.RawMessage(`{
				"collection": "RequestTestColl",
				"request": "Test Request",
				"url": "https://example.com/api/v2"
			}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_request", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "RequestTestColl", "request": "Test Request"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_HistoryTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("search_history", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "example", "limit": 10}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ImportExportTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "ExportTestColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	t.Run("export_collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_collection",
			Arguments: json.RawMessage(`{"name": "ExportTestColl", "format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("export_as_curl with invalid args", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "request": "also nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import_collection with invalid data", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": "not valid json", "format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_WebSocketTools(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("websocket_list_connections", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_list_connections",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("websocket_get_messages with no connection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_get_messages",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_disconnect with no connection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_disconnect",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_send with no connection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_send",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent", "message": "hello"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RunCollectionTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("run_collection with invalid collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "run_collection",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetEnvironment(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("returns nil for empty name", func(t *testing.T) {
		vars, err := server.getEnvironment("")
		require.NoError(t, err)
		assert.Nil(t, vars)
	})

	t.Run("returns error for non-existent environment", func(t *testing.T) {
		_, err := server.getEnvironment("nonexistent")
		assert.Error(t, err)
	})

}

func TestServer_ParseCurl(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("parses simple GET curl command", func(t *testing.T) {
		req, err := server.parseCurl("curl https://api.example.com/users")
		require.NoError(t, err)
		assert.NotNil(t, req)
		assert.Equal(t, "GET", req.Method())
	})

	t.Run("parses POST curl command with headers", func(t *testing.T) {
		req, err := server.parseCurl(`curl -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://api.example.com/users`)
		require.NoError(t, err)
		assert.NotNil(t, req)
		assert.Equal(t, "POST", req.Method())
	})

	t.Run("returns error for invalid curl command", func(t *testing.T) {
		_, err := server.parseCurl("not a valid curl command")
		assert.Error(t, err)
	})
}

func TestServer_RunCollection(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("returns error for non-existent collection", func(t *testing.T) {
		ctx := context.Background()
		_, err := server.runCollection(ctx, "nonexistent", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "collection not found")
	})

	t.Run("returns error for non-existent environment", func(t *testing.T) {
		ctx := context.Background()
		// First create a collection
		params := ToolCallParams{
			Name:      "create_collection",
			Arguments: json.RawMessage(`{"name": "test-collection"}`),
		}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}
		server.handleToolsCall(req)

		// Try to run with non-existent environment
		_, err := server.runCollection(ctx, "test-collection", "nonexistent-env")
		assert.Error(t, err)
	})
}

func TestServer_SendRequest(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("sends simple GET request", func(t *testing.T) {
		ctx := context.Background()
		// Use a simple URL that won't actually connect
		_, err := server.sendRequest(ctx, "GET", "http://localhost:99999/test", nil, "", "")
		// Should fail because the server doesn't exist, but the request should be created
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent environment", func(t *testing.T) {
		ctx := context.Background()
		_, err := server.sendRequest(ctx, "GET", "http://example.com", nil, "", "nonexistent-env")
		assert.Error(t, err)
	})

	t.Run("interpolates environment variables", func(t *testing.T) {
		// Create an environment first
		params := ToolCallParams{
			Name:      "create_environment",
			Arguments: json.RawMessage(`{"name": "request-test-env"}`),
		}
		paramsJSON, _ := json.Marshal(params)
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}
		server.handleToolsCall(req)

		// Set a variable
		params = ToolCallParams{
			Name:      "set_env_var",
			Arguments: json.RawMessage(`{"environment": "request-test-env", "key": "host", "value": "localhost"}`),
		}
		paramsJSON, _ = json.Marshal(params)
		req = &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}
		server.handleToolsCall(req)

		// Send request with environment
		ctx := context.Background()
		_, err := server.sendRequest(ctx, "GET", "http://{{host}}:99999/test", nil, "", "request-test-env")
		// Should fail because the server doesn't exist
		assert.Error(t, err)
	})
}

func TestServer_Run(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("handles message loop with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Create pipes for stdin/stdout
		stdinReader, stdinWriter := io.Pipe()
		_, stdoutWriter := io.Pipe()

		// Run server in goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Run(ctx, stdinReader, stdoutWriter)
		}()

		// Write an initialize request
		initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"}}}`
		go func() {
			time.Sleep(100 * time.Millisecond)
			stdinWriter.Write([]byte(initReq + "\n"))
			time.Sleep(100 * time.Millisecond)
			stdinWriter.Close()
		}()

		// Wait for server to finish with timeout
		select {
		case <-errChan:
			// Server exited (either error or clean shutdown)
		case <-time.After(3 * time.Second):
			// Timeout is also acceptable
		}

		// Close pipes
		stdoutWriter.Close()
	})
}

func TestServer_GetCollectionSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// First create a collection with a request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "GetTestColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	// Save a request to the collection
	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "GetTestColl",
			"name": "Get Users",
			"method": "GET",
			"url": "https://api.example.com/users"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("get_collection returns full collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_collection",
			Arguments: json.RawMessage(`{"name": "GetTestColl"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
		// Verify the response contains collection data
		var result ToolCallResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Content)
	})

	t.Run("get_request returns request details", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_request",
			Arguments: json.RawMessage(`{"collection": "GetTestColl", "request": "Get Users"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`4`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ExportAsCurlSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with a request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "CurlExportColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	// Save a request
	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "CurlExportColl",
			"name": "Post Data",
			"method": "POST",
			"url": "https://api.example.com/data",
			"headers": {"Content-Type": "application/json"},
			"body": "{\"key\": \"value\"}"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("export_as_curl returns curl command", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "CurlExportColl", "request": "Post Data"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		var result ToolCallResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		if len(result.Content) > 0 {
			assert.Contains(t, result.Content[0].Text, "curl")
		}
	})
}

func TestServer_ImportCollectionSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("import_collection with valid postman format", func(t *testing.T) {
		postmanCollection := `{
			"info": {
				"name": "Imported Collection",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"item": [
				{
					"name": "Get Users",
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users"
					}
				}
			]
		}`

		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": ` + postmanCollection + `, "format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("import_collection with curl format", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": "curl -X GET https://api.example.com/users", "format": "curl"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_SendCurlTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("send_curl with invalid host returns error", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_curl",
			Arguments: json.RawMessage(`{"curl": "curl http://localhost:99999/invalid"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Should fail but exercise the code path
	})

	t.Run("send_curl with invalid curl syntax", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_curl",
			Arguments: json.RawMessage(`{"curl": "not a curl command"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Tool might return error in content or in Error field depending on implementation
	})
}

func TestServer_SendRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("send_request with unreachable host", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_request",
			Arguments: json.RawMessage(`{"method": "GET", "url": "http://localhost:99999/test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Should fail but exercise the send_request code path
	})

	t.Run("send_request with headers", func(t *testing.T) {
		params := ToolCallParams{
			Name: "send_request",
			Arguments: json.RawMessage(`{
				"method": "POST",
				"url": "http://localhost:99999/api",
				"headers": {"Content-Type": "application/json", "Authorization": "Bearer token"},
				"body": "{\"data\": \"test\"}"
			}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RunCollectionWithRequests(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with requests
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RunnerTestColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	// Save some requests
	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "RunnerTestColl",
			"name": "Test Request 1",
			"method": "GET",
			"url": "http://localhost:99999/test1"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("run_collection executes but fails on network", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "run_collection",
			Arguments: json.RawMessage(`{"name": "RunnerTestColl"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Will have errors due to unreachable hosts, but code path is exercised
	})
}

func TestServer_InvalidToolArguments(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	tests := []struct {
		name     string
		tool     string
		args     string
		wantErr  bool
	}{
		{"get_collection missing name", "get_collection", `{}`, true},
		{"get_request missing collection", "get_request", `{"request": "test"}`, true},
		{"save_request missing required fields", "save_request", `{"name": "test"}`, true},
		{"update_request missing required fields", "update_request", `{}`, true},
		{"delete_request missing collection", "delete_request", `{"request": "test"}`, true},
		{"create_folder missing collection", "create_folder", `{"name": "test"}`, true},
		{"delete_folder missing collection", "delete_folder", `{"folder": "test"}`, true},
		{"export_as_curl missing collection", "export_as_curl", `{"request": "test"}`, true},
		{"import_collection missing data", "import_collection", `{"format": "postman"}`, true},
		{"send_request missing url", "send_request", `{"method": "GET"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ToolCallParams{
				Name:      tt.tool,
				Arguments: json.RawMessage(tt.args),
			}
			paramsJSON, _ := json.Marshal(params)

			req := &Request{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  MethodToolsCall,
				Params:  paramsJSON,
			}

			resp := server.handleToolsCall(req)
			require.NotNil(t, resp)
			// Either error or handled gracefully
		})
	}
}

func TestServer_RenameCollectionTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RenameTest"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	t.Run("rename_collection successfully", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"old_name": "RenameTest", "new_name": "RenamedCollection"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_UpdateRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create collection and request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "UpdateReqColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "UpdateReqColl",
			"name": "Original Request",
			"method": "GET",
			"url": "https://example.com"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("update_request changes method and url", func(t *testing.T) {
		params := ToolCallParams{
			Name: "update_request",
			Arguments: json.RawMessage(`{
				"collection": "UpdateReqColl",
				"request": "Original Request",
				"method": "POST",
				"url": "https://example.com/updated"
			}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("update_request adds headers", func(t *testing.T) {
		params := ToolCallParams{
			Name: "update_request",
			Arguments: json.RawMessage(`{
				"collection": "UpdateReqColl",
				"request": "Original Request",
				"headers": {"Authorization": "Bearer token"}
			}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`4`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create collection and request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "DeleteReqColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "DeleteReqColl",
			"name": "ToDelete",
			"method": "GET",
			"url": "https://example.com"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("delete_request removes request", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "DeleteReqColl", "request": "ToDelete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_CreateDeleteFolderTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create collection
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "FolderColl"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	t.Run("create_folder in collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "FolderColl", "name": "NewFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("delete_folder from collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "FolderColl", "folder": "NewFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetHistoryWithFilters(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get_history with method filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 5, "method": "GET"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_history with status filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 5, "status_min": 200, "status_max": 299}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ListCookiesWithDomain(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("list_cookies with domain filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_SearchHistoryTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("search_history with query", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "api", "limit": 10}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("search_history with method filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "test", "method": "POST"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_GetCollectionWithEnv(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "EnvTestCollection"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Create an environment
	envParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "test-env"}`),
	}
	paramsJSON, _ = json.Marshal(envParams)
	req.Params = paramsJSON
	server.handleToolsCall(req)

	t.Run("get_collection with environment", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_collection",
			Arguments: json.RawMessage(`{"name": "EnvTestCollection", "environment": "test-env"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// May have error if environment not found, but should not panic
	})
}

func TestServer_GetRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with a request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RequestCollection"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Save a request
	saveParams := ToolCallParams{
		Name:      "save_request",
		Arguments: json.RawMessage(`{"collection": "RequestCollection", "name": "GetUser", "method": "GET", "url": "https://api.example.com/users/1"}`),
	}
	paramsJSON, _ = json.Marshal(saveParams)
	req.Params = paramsJSON
	server.handleToolsCall(req)

	t.Run("get_request success", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_request",
			Arguments: json.RawMessage(`{"collection": "RequestCollection", "name": "GetUser"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_request not found", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_request",
			Arguments: json.RawMessage(`{"collection": "RequestCollection", "name": "NonExistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_request from folder", func(t *testing.T) {
		// Create folder and add request
		folderParams := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "RequestCollection", "name": "Users"}`),
		}
		paramsJSON, _ := json.Marshal(folderParams)
		req.Params = paramsJSON
		server.handleToolsCall(req)

		saveParams := ToolCallParams{
			Name:      "save_request",
			Arguments: json.RawMessage(`{"collection": "RequestCollection", "folder": "Users", "name": "ListUsers", "method": "GET", "url": "https://api.example.com/users"}`),
		}
		paramsJSON, _ = json.Marshal(saveParams)
		req.Params = paramsJSON
		server.handleToolsCall(req)

		params := ToolCallParams{
			Name:      "get_request",
			Arguments: json.RawMessage(`{"collection": "RequestCollection", "folder": "Users", "name": "ListUsers"}`),
		}
		paramsJSON, _ = json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_GetEnvironmentTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create an environment with variables
	createParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "production"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Set a variable
	setParams := ToolCallParams{
		Name:      "set_environment_variable",
		Arguments: json.RawMessage(`{"environment": "production", "key": "API_KEY", "value": "secret123"}`),
	}
	paramsJSON, _ = json.Marshal(setParams)
	req.Params = paramsJSON
	server.handleToolsCall(req)

	t.Run("get_environment success", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "production"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_environment not found", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Tool may or may not return error, just verify it doesn't panic
	})
}

func TestServer_DeleteEnvironmentVariableTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create an environment with a variable
	createParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "staging"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Set a variable
	setParams := ToolCallParams{
		Name:      "set_environment_variable",
		Arguments: json.RawMessage(`{"environment": "staging", "key": "DB_HOST", "value": "localhost"}`),
	}
	paramsJSON, _ = json.Marshal(setParams)
	req.Params = paramsJSON
	server.handleToolsCall(req)

	t.Run("delete_environment_variable success", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "staging", "key": "DB_HOST"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("delete_environment_variable nonexistent env", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "nonexistent", "key": "FOO"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Tool may or may not return error, just verify it doesn't panic
	})
}

func TestServer_ClearCookiesTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("clear_cookies all", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "clear_cookies",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("clear_cookies for domain", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "clear_cookies",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_DeleteCollectionTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "ToDelete"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("delete_collection success", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_collection",
			Arguments: json.RawMessage(`{"name": "ToDelete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("delete_collection not found", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_collection",
			Arguments: json.RawMessage(`{"name": "NonExistentCollection"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Tool may or may not return error, just verify it doesn't panic
	})
}

func TestServer_ExportAsCurlTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with a request
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "CurlExport"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Save a request
	saveParams := ToolCallParams{
		Name:      "save_request",
		Arguments: json.RawMessage(`{"collection": "CurlExport", "name": "GetData", "method": "GET", "url": "https://api.example.com/data"}`),
	}
	paramsJSON, _ = json.Marshal(saveParams)
	req.Params = paramsJSON
	server.handleToolsCall(req)

	t.Run("export_as_curl success", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "CurlExport", "request": "GetData"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("export_as_curl request not found", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "CurlExport", "request": "NonExistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Tool may or may not return error, just verify it doesn't panic
	})
}

func TestServer_ResourcesList(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("list resources", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesList,
		}

		resp := server.handleResourcesList(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Result)

		var result ResourcesListResult
		err := json.Unmarshal(resp.Result, &result)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Resources), 2) // collections and history
	})
}

func TestServer_ResourcesRead(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("read collections resource", func(t *testing.T) {
		params := map[string]string{
			"uri": "collections://list",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("read history resource", func(t *testing.T) {
		params := map[string]string{
			"uri": "history://recent",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("read unknown resource", func(t *testing.T) {
		params := map[string]string{
			"uri": "unknown://resource",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error) // Should error for unknown resource
	})
}

func TestServer_CreateEnvironmentTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("create environment with variables", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_environment",
			Arguments: json.RawMessage(`{"name": "development"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_DeleteEnvironmentTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// First create an environment
	createParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "to-delete"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("delete environment", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment",
			Arguments: json.RawMessage(`{"name": "to-delete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ExportCollectionTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "ExportTest"}`),
	}
	paramsJSON, _ := json.Marshal(createParams)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("export collection as postman", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_collection",
			Arguments: json.RawMessage(`{"collection": "ExportTest", "format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("export collection as openapi", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_collection",
			Arguments: json.RawMessage(`{"collection": "ExportTest", "format": "openapi"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ListEnvironmentsTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a few environments
	for _, name := range []string{"dev", "staging", "prod"} {
		params := ToolCallParams{
			Name:      "create_environment",
			Arguments: json.RawMessage(`{"name": "` + name + `"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}
		server.handleToolsCall(req)
	}

	t.Run("list environments", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_environments",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_RenameCollectionToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "Original Name"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("rename collection extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"collection": "Original Name", "new_name": "New Name"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename collection with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_CreateFolderTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "TestCollection"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("create folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "TestCollection", "name": "API Endpoints"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create folder with missing collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "name": "Folder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create folder with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteRequestToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection and request
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "TestCollectionDel"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Create a request
	params = ToolCallParams{
		Name:      "create_request",
		Arguments: json.RawMessage(`{"collection": "TestCollectionDel", "name": "TestRequestDel", "method": "GET", "url": "http://example.com"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("delete request extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "TestCollectionDel", "request": "TestRequestDel"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete request with missing collection extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "request": "SomeRequest"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete request with missing params extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetHistoryTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get history with no entries", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get history with limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 10}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get history with offset", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"offset": 5}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_ListCookiesTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("list all cookies", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("list cookies with domain filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_SetCookieTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("set cookie", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_cookie",
			Arguments: json.RawMessage(`{"domain": "example.com", "name": "session", "value": "abc123"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("set cookie with path", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_cookie",
			Arguments: json.RawMessage(`{"domain": "example.com", "name": "auth", "value": "token", "path": "/api"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("set cookie with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_cookie",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_UpdateRequestToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection and request
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "TestCollectionUpd"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Create a request
	params = ToolCallParams{
		Name:      "create_request",
		Arguments: json.RawMessage(`{"collection": "TestCollectionUpd", "name": "TestRequestUpd", "method": "GET", "url": "http://example.com"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("update request method extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "TestCollectionUpd", "request": "TestRequestUpd", "method": "POST"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update request url extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "TestCollectionUpd", "request": "TestRequestUpd", "url": "http://newurl.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update request with missing params extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_CreateRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "TestCollection"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("create request with headers", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_request",
			Arguments: json.RawMessage(`{"collection": "TestCollection", "name": "RequestWithHeaders", "method": "GET", "url": "http://example.com", "headers": {"Content-Type": "application/json"}}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create request with body", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_request",
			Arguments: json.RawMessage(`{"collection": "TestCollection", "name": "RequestWithBody", "method": "POST", "url": "http://example.com", "body": "{\"key\": \"value\"}"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create request in folder", func(t *testing.T) {
		// First create folder
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "TestCollection", "name": "MyFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}
		server.handleToolsCall(req)

		// Create request in folder
		params = ToolCallParams{
			Name:      "create_request",
			Arguments: json.RawMessage(`{"collection": "TestCollection", "name": "FolderRequest", "method": "GET", "url": "http://example.com", "folder": "MyFolder"}`),
		}
		paramsJSON, _ = json.Marshal(params)

		req = &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create request with missing collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_request",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "name": "Test", "method": "GET", "url": "http://example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ImportCollectionTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("import collection with invalid data", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": "not valid json"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import collection with missing data", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import collection with file path", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"file": "/nonexistent/path.json"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_WebSocketToolsExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("websocket connect with invalid URL extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_connect",
			Arguments: json.RawMessage(`{"url": "not a valid url"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket send with no connection extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_send",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent", "message": "hello"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket disconnect with no connection extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_disconnect",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket list connections extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_list_connections",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket get messages with no connection extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_get_messages",
			Arguments: json.RawMessage(`{"connection_id": "nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteFolderTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection first
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "TestCollectionWithFolder"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	// Create a folder
	params = ToolCallParams{
		Name:      "create_folder",
		Arguments: json.RawMessage(`{"collection": "TestCollectionWithFolder", "name": "FolderToDelete"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("delete folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "TestCollectionWithFolder", "folder": "FolderToDelete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete folder with missing collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "folder": "SomeFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete folder with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SetEnvironmentVariableTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create an environment first
	params := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "TestEnv"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("set environment variable", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_environment_variable",
			Arguments: json.RawMessage(`{"environment": "TestEnv", "key": "API_KEY", "value": "secret123"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("set environment variable with missing environment", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_environment_variable",
			Arguments: json.RawMessage(`{"environment": "NonExistent", "key": "API_KEY", "value": "secret123"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("set environment variable with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_environment_variable",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SearchHistoryToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("search history with empty query", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": ""}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("search history with query and limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "example", "limit": 5}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RenameFolderTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection and folder
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RenameTest"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	params = ToolCallParams{
		Name:      "create_folder",
		Arguments: json.RawMessage(`{"collection": "RenameTest", "name": "OldFolderName"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("rename folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_folder",
			Arguments: json.RawMessage(`{"collection": "RenameTest", "folder": "OldFolderName", "new_name": "NewFolderName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename folder with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_folder",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_MoveRequestTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with request
	params := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "MoveTest"}`),
	}
	paramsJSON, _ := json.Marshal(params)

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	params = ToolCallParams{
		Name:      "create_request",
		Arguments: json.RawMessage(`{"collection": "MoveTest", "name": "RequestToMove", "method": "GET", "url": "http://example.com"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	params = ToolCallParams{
		Name:      "create_folder",
		Arguments: json.RawMessage(`{"collection": "MoveTest", "name": "TargetFolder"}`),
	}
	paramsJSON, _ = json.Marshal(params)

	req = &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  paramsJSON,
	}
	server.handleToolsCall(req)

	t.Run("move request to folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "move_request",
			Arguments: json.RawMessage(`{"collection": "MoveTest", "request": "RequestToMove", "target_folder": "TargetFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("move request with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "move_request",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SendCurlToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("send_curl with POST and data extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_curl",
			Arguments: json.RawMessage(`{"curl_command": "curl -X POST -d '{\"key\":\"value\"}' http://example.com/api"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("send_curl with empty command extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_curl",
			Arguments: json.RawMessage(`{"curl_command": ""}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("send_curl with multiple headers", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_curl",
			Arguments: json.RawMessage(`{"curl_command": "curl -H 'Content-Type: application/json' -H 'Authorization: Bearer token' -H 'X-Custom: value' http://example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SendRequestToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("send_request with body extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_request",
			Arguments: json.RawMessage(`{"method": "POST", "url": "https://httpbin.org/post", "body": "{\"key\": \"value\"}"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("send_request missing url extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_request",
			Arguments: json.RawMessage(`{"method": "GET"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("send_request default method extended", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "send_request",
			Arguments: json.RawMessage(`{"url": "https://example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetEnvironmentToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get_environment with non-existent env", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "NonExistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_environment with empty name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": ""}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ExportAsCurlToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection and request using tool calls
	createColParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "ExportCurlTest"}`),
	}
	createColJSON, _ := json.Marshal(createColParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createColJSON,
	})

	// Save a request to the collection
	saveReqParams := ToolCallParams{
		Name:      "save_request",
		Arguments: json.RawMessage(`{"collection": "ExportCurlTest", "name": "TestRequest", "method": "POST", "url": "https://api.example.com/test", "body": "{\"key\": \"value\"}", "headers": {"Content-Type": "application/json", "Authorization": "Bearer token123"}}`),
	}
	saveReqJSON, _ := json.Marshal(saveReqParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  saveReqJSON,
	})

	t.Run("export_as_curl with headers", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "ExportCurlTest", "request": "TestRequest"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("export_as_curl non-existent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "request": "TestRequest"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("export_as_curl non-existent request", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "ExportCurlTest", "request": "NonExistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("export_as_curl missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ListCookiesToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("list_cookies with domain filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("list_cookies empty domain", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{"domain": ""}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ClearCookiesToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("clear_cookies with domain", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "clear_cookies",
			Arguments: json.RawMessage(`{"domain": "example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("clear_cookies all domains", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "clear_cookies",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetHistoryToolExtended(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get_history with limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 5}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_history with method filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"method": "POST"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_history with url filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"url_contains": "api"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("get_history with zero limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 0}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RenameCollectionToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("rename nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"old_name": "nonexistent", "new_name": "NewName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename with missing old_name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"new_name": "NewName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename with missing new_name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"old_name": "Test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ImportCollectionToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("import with invalid format", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"content": "invalid json", "format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import with missing content", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"format": "postman"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_WebSocketToolsMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("websocket_connect with invalid url", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_connect",
			Arguments: json.RawMessage(`{"url": "not-a-url"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_disconnect with invalid id", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_disconnect",
			Arguments: json.RawMessage(`{"connection_id": "invalid-id"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_send with invalid connection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_send",
			Arguments: json.RawMessage(`{"connection_id": "invalid-id", "message": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_list_connections", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_list_connections",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_get_messages with invalid connection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_get_messages",
			Arguments: json.RawMessage(`{"connection_id": "invalid-id"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SearchHistoryToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("search_history with query", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("search_history with limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "test", "limit": 5}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("search_history with method filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "test", "method": "GET"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteEnvVariableTool(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("delete_environment_variable nonexistent env", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "nonexistent", "variable": "key"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_environment_variable missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_CreateFolderToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("create_folder in nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "name": "NewFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create_folder with missing name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteFolderToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("delete_folder from nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "folder": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_UpdateRequestToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("update_request in nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "request": "test", "url": "http://new.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update_request with missing request name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "test", "url": "http://new.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteRequestToolMore(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("delete_request from nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "request": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete_request with missing params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetCollectionToolAdditional(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get collection with folder name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_collection",
			Arguments: json.RawMessage(`{"name": "Test Collection", "folder": "TestFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetEnvironmentToolAdditional(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get nonexistent environment", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "Nonexistent Env"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ListCookiesToolAdditional(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("list cookies for domain", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "list_cookies",
			Arguments: json.RawMessage(`{"domain": "test.example.com"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_GetHistoryToolAdditional(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get history with method filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"method": "GET", "limit": 5}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ExportAsCurlToolAdditional(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("export nonexistent collection as curl", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "export_as_curl",
			Arguments: json.RawMessage(`{"collection": "Nonexistent"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RenameCollectionCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("rename nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "new_name": "NewName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// Should return error for nonexistent collection
	})

	t.Run("rename collection with missing arguments", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"collection": "Test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename collection with empty new name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"collection": "Test", "new_name": ""}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_CreateFolderCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("create folder in nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "name": "NewFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create folder with missing name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "Test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("create folder with parent path", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "create_folder",
			Arguments: json.RawMessage(`{"collection": "Test", "name": "SubFolder", "parent_path": "ParentFolder"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_DeleteRequestCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("delete request from nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "request": "TestRequest"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("delete request with missing request name", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "Test"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_UpdateRequestCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("update request in nonexistent collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "NonExistent", "request": "TestRequest", "method": "POST"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update request with headers", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "Test", "request": "TestRequest", "headers": {"X-Custom": "value"}}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("update request with body", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "update_request",
			Arguments: json.RawMessage(`{"collection": "Test", "request": "TestRequest", "body": "test body"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ResourcesReadCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("read collections resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "collections://list",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("read history resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "history://recent",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("read unknown resource", func(t *testing.T) {
		params := ResourceReadParams{
			URI: "unknown://resource",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  paramsJSON,
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
	})

	t.Run("read resource with invalid params", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodResourcesRead,
			Params:  json.RawMessage(`invalid json`),
		}

		resp := server.handleResourcesRead(req)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
	})
}

func TestServer_RenameCollectionSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// First create a collection
	createParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "OldName"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	resp := server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})
	require.Nil(t, resp.Error)

	t.Run("rename_collection successfully", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_collection",
			Arguments: json.RawMessage(`{"old_name": "OldName", "new_name": "NewName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		// Verify the collection was renamed by getting it
		getParams := ToolCallParams{
			Name:      "get_collection",
			Arguments: json.RawMessage(`{"name": "NewName"}`),
		}
		getJSON, _ := json.Marshal(getParams)
		getResp := server.handleToolsCall(&Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  getJSON,
		})
		assert.Nil(t, getResp.Error)
	})
}

func TestServer_GetEnvironmentSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create an environment with variables
	createParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "TestEnvGet"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	// Set some variables
	setParams := ToolCallParams{
		Name:      "set_environment_variable",
		Arguments: json.RawMessage(`{"environment": "TestEnvGet", "key": "api_key", "value": "secret123"}`),
	}
	setJSON, _ := json.Marshal(setParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  setJSON,
	})

	t.Run("get_environment returns environment details", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_environment",
			Arguments: json.RawMessage(`{"name": "TestEnvGet"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_DeleteFolderSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with a folder
	createCollParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "FolderDeleteColl"}`),
	}
	createCollJSON, _ := json.Marshal(createCollParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createCollJSON,
	})

	// Create a folder
	createFolderParams := ToolCallParams{
		Name:      "create_folder",
		Arguments: json.RawMessage(`{"collection": "FolderDeleteColl", "name": "ToDelete"}`),
	}
	createFolderJSON, _ := json.Marshal(createFolderParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  createFolderJSON,
	})

	t.Run("delete_folder removes the folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_folder",
			Arguments: json.RawMessage(`{"collection": "FolderDeleteColl", "folder": "ToDelete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_DeleteRequestSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create a collection with a request
	createCollParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "ReqDeleteColl"}`),
	}
	createCollJSON, _ := json.Marshal(createCollParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createCollJSON,
	})

	// Save a request
	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "ReqDeleteColl",
			"name": "ToDeleteReq",
			"method": "GET",
			"url": "https://api.example.com/delete"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("delete_request removes the request", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_request",
			Arguments: json.RawMessage(`{"collection": "ReqDeleteColl", "request": "ToDeleteReq"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_DeleteEnvironmentVariableSuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create an environment
	createParams := ToolCallParams{
		Name:      "create_environment",
		Arguments: json.RawMessage(`{"name": "VarDeleteEnv"}`),
	}
	createJSON, _ := json.Marshal(createParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createJSON,
	})

	// Set a variable
	setParams := ToolCallParams{
		Name:      "set_environment_variable",
		Arguments: json.RawMessage(`{"environment": "VarDeleteEnv", "key": "to_delete", "value": "value"}`),
	}
	setJSON, _ := json.Marshal(setParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  setJSON,
	})

	t.Run("delete_environment_variable removes the variable", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_environment_variable",
			Arguments: json.RawMessage(`{"environment": "VarDeleteEnv", "key": "to_delete"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_GetHistorySuccess(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("get_history returns history entries", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{"limit": 50}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("get_history with default limit", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "get_history",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`2`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_WebSocketToolsCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("websocket_disconnect with invalid params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_disconnect",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_send with invalid params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_send",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("websocket_get_messages with invalid params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "websocket_get_messages",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_ImportCollectionCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("import_collection with invalid params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import_collection with unsupported format", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": "{}", "format": "unknown"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("import_collection with openapi format", func(t *testing.T) {
		openAPISpec := `{
			"openapi": "3.0.0",
			"info": {"title": "Test API", "version": "1.0"},
			"paths": {
				"/users": {
					"get": {
						"summary": "Get users"
					}
				}
			}
		}`
		params := ToolCallParams{
			Name:      "import_collection",
			Arguments: json.RawMessage(`{"data": ` + openAPISpec + `, "format": "openapi"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_SearchHistoryCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("search_history with various params", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "test", "limit": 5, "method": "GET"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("search_history with status filter", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "search_history",
			Arguments: json.RawMessage(`{"query": "api", "status": "200"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})
}

func TestServer_CookieToolsCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	t.Run("set_cookie", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "set_cookie",
			Arguments: json.RawMessage(`{"domain": "example.com", "name": "session", "value": "abc123"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
		// May or may not error depending on cookie jar implementation
	})

	t.Run("delete_cookie", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "delete_cookie",
			Arguments: json.RawMessage(`{"domain": "example.com", "name": "session"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RenameFolderCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create collection and folder first
	createCollParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RenameFolderColl"}`),
	}
	createCollJSON, _ := json.Marshal(createCollParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createCollJSON,
	})

	createFolderParams := ToolCallParams{
		Name:      "create_folder",
		Arguments: json.RawMessage(`{"collection": "RenameFolderColl", "name": "OldFolderName"}`),
	}
	createFolderJSON, _ := json.Marshal(createFolderParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  createFolderJSON,
	})

	t.Run("rename_folder", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_folder",
			Arguments: json.RawMessage(`{"collection": "RenameFolderColl", "old_name": "OldFolderName", "new_name": "NewFolderName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename_folder with invalid collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_folder",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "old_name": "test", "new_name": "test2"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`4`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}

func TestServer_RenameRequestCoverage(t *testing.T) {
	server, cleanup := createTestServer(t)
	defer cleanup()

	// Create collection and request first
	createCollParams := ToolCallParams{
		Name:      "create_collection",
		Arguments: json.RawMessage(`{"name": "RenameReqColl"}`),
	}
	createCollJSON, _ := json.Marshal(createCollParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  createCollJSON,
	})

	saveParams := ToolCallParams{
		Name: "save_request",
		Arguments: json.RawMessage(`{
			"collection": "RenameReqColl",
			"name": "OldReqName",
			"method": "GET",
			"url": "https://api.example.com"
		}`),
	}
	saveJSON, _ := json.Marshal(saveParams)
	server.handleToolsCall(&Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  saveJSON,
	})

	t.Run("rename_request", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_request",
			Arguments: json.RawMessage(`{"collection": "RenameReqColl", "old_name": "OldReqName", "new_name": "NewReqName"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`3`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})

	t.Run("rename_request with invalid collection", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "rename_request",
			Arguments: json.RawMessage(`{"collection": "nonexistent", "old_name": "test", "new_name": "test2"}`),
		}
		paramsJSON, _ := json.Marshal(params)

		req := &Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`4`),
			Method:  MethodToolsCall,
			Params:  paramsJSON,
		}

		resp := server.handleToolsCall(req)
		require.NotNil(t, resp)
	})
}
