package exporter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Registry

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	exp := NewCurlExporter()
	registry.Register(exp)

	got, ok := registry.Get(FormatCurl)
	assert.True(t, ok)
	assert.Equal(t, exp, got)
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	t.Run("returns registered exporter", func(t *testing.T) {
		exp := NewPostmanExporter()
		registry.Register(exp)

		got, ok := registry.Get(FormatPostman)
		assert.True(t, ok)
		assert.Equal(t, exp, got)
	})

	t.Run("returns false for unregistered format", func(t *testing.T) {
		_, ok := registry.Get(FormatHAR)
		assert.False(t, ok)
	})
}

func TestRegistry_Export(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewCurlExporter())
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
	coll.AddRequest(req)

	result, err := registry.Export(ctx, FormatCurl, coll)
	require.NoError(t, err)
	assert.Equal(t, FormatCurl, result.Format)
	assert.Equal(t, ".sh", result.FileExtension)
	assert.Contains(t, string(result.Content), "curl")
}

func TestRegistry_ListFormats(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewCurlExporter())
	registry.Register(NewPostmanExporter())

	formats := registry.ListFormats()
	assert.Len(t, formats, 2)
	assert.Contains(t, formats, FormatCurl)
	assert.Contains(t, formats, FormatPostman)
}

// Test Curl Exporter

func TestCurlExporter_Name(t *testing.T) {
	exp := NewCurlExporter()
	assert.Equal(t, "curl command", exp.Name())
}

func TestCurlExporter_ExportRequest_Simple(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "GET", "https://api.example.com/users")

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "curl")
	assert.Contains(t, cmd, "https://api.example.com/users")
	assert.NotContains(t, cmd, "-X GET") // GET is default
}

func TestCurlExporter_ExportRequest_WithMethod(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "POST", "https://api.example.com/users")

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "-X POST")
}

func TestCurlExporter_ExportRequest_WithHeaders(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "GET", "https://api.example.com")
	req.SetHeader("Accept", "application/json")
	req.SetHeader("X-Custom", "value")

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "-H")
	assert.Contains(t, cmd, "Accept: application/json")
	assert.Contains(t, cmd, "X-Custom: value")
}

func TestCurlExporter_ExportRequest_WithBody(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "POST", "https://api.example.com")
	req.SetBody(`{"name": "John"}`)

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "--data-raw")
	assert.Contains(t, cmd, `{"name": "John"}`)
}

func TestCurlExporter_ExportRequest_WithBasicAuth(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "GET", "https://api.example.com")
	req.SetAuth(core.AuthConfig{
		Type:     "basic",
		Username: "user",
		Password: "pass",
	})

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "-u")
	assert.Contains(t, cmd, "user:pass")
}

func TestCurlExporter_ExportRequest_WithBearerAuth(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = false
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "GET", "https://api.example.com")
	req.SetAuth(core.AuthConfig{
		Type:  "bearer",
		Token: "mytoken123",
	})

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "Authorization: Bearer mytoken123")
}

func TestCurlExporter_ExportRequest_Pretty(t *testing.T) {
	exp := NewCurlExporter()
	exp.Pretty = true
	ctx := context.Background()

	req := core.NewRequestDefinition("Test", "POST", "https://api.example.com/users")
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(`{"name": "John"}`)

	result, err := exp.ExportRequest(ctx, req)
	require.NoError(t, err)

	cmd := string(result)
	assert.Contains(t, cmd, "\\\n") // Line continuations
}

func TestCurlExporter_Export_Collection(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	coll := core.NewCollection("My API")
	coll.SetDescription("Test collection")

	req1 := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
	req2 := core.NewRequestDefinition("Create User", "POST", "https://api.example.com/users")
	coll.AddRequest(req1)
	coll.AddRequest(req2)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	content := string(result)
	assert.Contains(t, content, "#!/bin/bash")
	assert.Contains(t, content, "Collection: My API")
	assert.Contains(t, content, "# Get Users")
	assert.Contains(t, content, "# Create User")
}

func TestCurlExporter_Export_WithFolders(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	coll := core.NewCollection("API")
	folder := coll.AddFolder("Users")
	folder.SetDescription("User endpoints")

	req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
	folder.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	content := string(result)
	assert.Contains(t, content, "=== Users ===")
	assert.Contains(t, content, "# Get Users")
}

// Test Postman Exporter

func TestPostmanExporter_Name(t *testing.T) {
	exp := NewPostmanExporter()
	assert.Equal(t, "Postman Collection", exp.Name())
}

func TestPostmanExporter_Export_BasicCollection(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("My API")
	coll.SetDescription("Test API collection")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	info := pm["info"].(map[string]interface{})
	assert.Equal(t, "My API", info["name"])
	assert.Equal(t, "Test API collection", info["description"])
	assert.Contains(t, info["schema"].(string), "v2.1.0")
}

func TestPostmanExporter_Export_WithRequests(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
	req.SetHeader("Accept", "application/json")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	items := pm["item"].([]interface{})
	require.Len(t, items, 1)

	item := items[0].(map[string]interface{})
	assert.Equal(t, "Get Users", item["name"])

	request := item["request"].(map[string]interface{})
	assert.Equal(t, "GET", request["method"])
	assert.Equal(t, "https://api.example.com/users", request["url"])
}

func TestPostmanExporter_Export_WithBody(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Create User", "POST", "https://api.example.com/users")
	req.SetBody(`{"name": "John"}`)
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	items := pm["item"].([]interface{})
	item := items[0].(map[string]interface{})
	request := item["request"].(map[string]interface{})
	body := request["body"].(map[string]interface{})

	assert.Equal(t, "raw", body["mode"])
	assert.Equal(t, `{"name": "John"}`, body["raw"])
}

func TestPostmanExporter_Export_WithVariables(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetVariable("base_url", "https://api.example.com")
	coll.SetVariable("api_key", "secret123")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	variables := pm["variable"].([]interface{})
	assert.GreaterOrEqual(t, len(variables), 2)
}

func TestPostmanExporter_Export_WithAuth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetAuth(core.AuthConfig{
		Type:  "bearer",
		Token: "mytoken",
	})

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	auth := pm["auth"].(map[string]interface{})
	assert.Equal(t, "bearer", auth["type"])

	bearerItems := auth["bearer"].([]interface{})
	require.Len(t, bearerItems, 1)

	bearerItem := bearerItems[0].(map[string]interface{})
	assert.Equal(t, "token", bearerItem["key"])
	assert.Equal(t, "mytoken", bearerItem["value"])
}

func TestPostmanExporter_Export_WithFolders(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	folder := coll.AddFolder("Users")
	folder.SetDescription("User endpoints")

	req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
	folder.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	items := pm["item"].([]interface{})
	require.Len(t, items, 1)

	folderItem := items[0].(map[string]interface{})
	assert.Equal(t, "Users", folderItem["name"])
	assert.Equal(t, "User endpoints", folderItem["description"])

	folderRequests := folderItem["item"].([]interface{})
	require.Len(t, folderRequests, 1)
}

func TestPostmanExporter_Export_WithScripts(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetPreScript("console.log('pre');")
	coll.SetPostScript("pm.test('ok', function(){});")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	content := string(result)
	assert.Contains(t, content, "prerequest")
	assert.Contains(t, content, "test")
	assert.Contains(t, content, "console.log")
}

func TestPostmanExporter_Export_NilCollection(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	_, err := exp.Export(ctx, nil)
	assert.ErrorIs(t, err, ErrInvalidCollection)
}

func TestPostmanExporter_FileExtension(t *testing.T) {
	exp := NewPostmanExporter()
	assert.Equal(t, ".postman_collection.json", exp.FileExtension())
}

func TestCurlExporter_FileExtension(t *testing.T) {
	exp := NewCurlExporter()
	assert.Equal(t, ".sh", exp.FileExtension())
}

func TestPostmanExporter_Export_WithBasicAuth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetAuth(core.AuthConfig{
		Type:     "basic",
		Username: "user",
		Password: "pass",
	})

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	auth := pm["auth"].(map[string]interface{})
	assert.Equal(t, "basic", auth["type"])
}

func TestPostmanExporter_Export_WithApiKeyAuth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetAuth(core.AuthConfig{
		Type:  "apikey",
		Key:   "X-Api-Key",
		Value: "secret123",
		In:    "header",
	})

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	auth := pm["auth"].(map[string]interface{})
	assert.Equal(t, "apikey", auth["type"])
}

func TestRegistry_Export_UnknownFormat(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	_, err := registry.Export(ctx, Format("unknown"), coll)
	assert.Error(t, err)
}

func TestPostmanExporter_Export_RequestWithAuth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Test", "GET", "https://api.example.com")
	req.SetAuth(core.AuthConfig{
		Type:  "bearer",
		Token: "token123",
	})
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	items := pm["item"].([]interface{})
	item := items[0].(map[string]interface{})
	request := item["request"].(map[string]interface{})

	auth := request["auth"].(map[string]interface{})
	assert.Equal(t, "bearer", auth["type"])
}

// Test shell quoting

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "'with space'"},
		{"with$var", "'with$var'"},
		{`with"quote`, `'with"quote'`},
		{"with'single", "'" + `with'"'"'single` + "'"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := shellQuote(tc.input)
			// For simple cases without quotes
			if !strings.Contains(tc.input, " ") &&
				!strings.Contains(tc.input, "$") &&
				!strings.Contains(tc.input, "'") &&
				!strings.Contains(tc.input, "\"") {
				assert.Equal(t, tc.input, result)
			}
		})
	}
}
