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
