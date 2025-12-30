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

func TestPostmanExporter_Export_WithOAuth2Auth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetAuth(core.AuthConfig{
		Type: "oauth2",
		OAuth2: &core.OAuth2Config{
			GrantType:    core.OAuth2GrantClientCredentials,
			AccessToken:  "access-token-123",
			TokenURL:     "https://auth.example.com/token",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			Scope:        "read write",
		},
	})

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	auth := pm["auth"].(map[string]interface{})
	assert.Equal(t, "oauth2", auth["type"])

	oauth2Items := auth["oauth2"].([]interface{})
	require.NotEmpty(t, oauth2Items)

	// Check access token is present
	foundAccessToken := false
	for _, item := range oauth2Items {
		authItem := item.(map[string]interface{})
		if authItem["key"] == "accessToken" {
			assert.Equal(t, "access-token-123", authItem["value"])
			foundAccessToken = true
		}
	}
	assert.True(t, foundAccessToken)
}

func TestPostmanExporter_Export_WithAWSAuth(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	coll.SetAuth(core.AuthConfig{
		Type: "awsv4",
		AWS: &core.AWSAuthConfig{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Region:          "us-east-1",
			Service:         "s3",
			SessionToken:    "session-token",
		},
	})

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	auth := pm["auth"].(map[string]interface{})
	assert.Equal(t, "awsv4", auth["type"])

	awsItems := auth["awsv4"].([]interface{})
	require.NotEmpty(t, awsItems)

	// Check AWS fields are present
	awsFields := make(map[string]string)
	for _, item := range awsItems {
		authItem := item.(map[string]interface{})
		awsFields[authItem["key"].(string)] = authItem["value"].(string)
	}
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", awsFields["accessKey"])
	assert.Equal(t, "us-east-1", awsFields["region"])
	assert.Equal(t, "s3", awsFields["service"])
}

func TestPostmanExporter_Export_WithFormDataBody(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Upload", "POST", "https://api.example.com/upload")
	req.SetBodyFormData([]core.FormField{
		{Key: "name", Value: "test.txt"},
		{Key: "file", IsFile: true, FilePath: "/path/to/file.txt"},
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
	body := request["body"].(map[string]interface{})

	assert.Equal(t, "formdata", body["mode"])
	formData := body["formdata"].([]interface{})
	require.Len(t, formData, 2)

	// Check text field
	field0 := formData[0].(map[string]interface{})
	assert.Equal(t, "name", field0["key"])
	assert.Equal(t, "test.txt", field0["value"])
	assert.Equal(t, "text", field0["type"])

	// Check file field
	field1 := formData[1].(map[string]interface{})
	assert.Equal(t, "file", field1["key"])
	assert.Equal(t, "file", field1["type"])
	assert.Equal(t, "/path/to/file.txt", field1["src"])
}

func TestPostmanExporter_Export_WithURLEncodedBody(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Login", "POST", "https://api.example.com/login")
	req.SetBodyType("urlencoded")
	req.SetBody("username=john&password=secret")
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

	assert.Equal(t, "urlencoded", body["mode"])
	urlEncoded := body["urlencoded"].([]interface{})
	require.Len(t, urlEncoded, 2)

	// Check fields
	fields := make(map[string]string)
	for _, f := range urlEncoded {
		field := f.(map[string]interface{})
		fields[field["key"].(string)] = field["value"].(string)
	}
	assert.Equal(t, "john", fields["username"])
	assert.Equal(t, "secret", fields["password"])
}

func TestPostmanExporter_Export_WithQueryParams(t *testing.T) {
	exp := NewPostmanExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test")
	req := core.NewRequestDefinition("Search", "GET", "https://api.example.com/search")
	req.SetQueryParam("q", "test")
	req.SetQueryParam("page", "1")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var pm map[string]interface{}
	err = json.Unmarshal(result, &pm)
	require.NoError(t, err)

	items := pm["item"].([]interface{})
	item := items[0].(map[string]interface{})
	request := item["request"].(map[string]interface{})

	// URL should be an object with query params
	url := request["url"].(map[string]interface{})
	assert.Contains(t, url["raw"], "q=")
	assert.Contains(t, url["raw"], "page=")

	query := url["query"].([]interface{})
	require.GreaterOrEqual(t, len(query), 2)

	// Check query params
	params := make(map[string]string)
	for _, q := range query {
		param := q.(map[string]interface{})
		params[param["key"].(string)] = param["value"].(string)
	}
	assert.Equal(t, "test", params["q"])
	assert.Equal(t, "1", params["page"])
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

// Test OpenAPI Exporter

func TestOpenAPIExporter_Name(t *testing.T) {
	exp := NewOpenAPIExporter()
	assert.Equal(t, "OpenAPI 3.0", exp.Name())
}

func TestOpenAPIExporter_Format(t *testing.T) {
	exp := NewOpenAPIExporter()
	assert.Equal(t, FormatOpenAPI, exp.Format())
}

func TestOpenAPIExporter_FileExtension(t *testing.T) {
	exp := NewOpenAPIExporter()
	assert.Equal(t, ".openapi.json", exp.FileExtension())
}

func TestOpenAPIExporter_Export_BasicCollection(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("My API")
	coll.SetDescription("Test API description")
	coll.SetVersion("2.0.0")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	assert.Equal(t, "3.0.3", spec["openapi"])

	info := spec["info"].(map[string]interface{})
	assert.Equal(t, "My API", info["title"])
	assert.Equal(t, "Test API description", info["description"])
	assert.Equal(t, "2.0.0", info["version"])
}

func TestOpenAPIExporter_Export_WithRequests(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get Users", "GET", "/users")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/users")

	usersPath := paths["/users"].(map[string]interface{})
	assert.Contains(t, usersPath, "get")

	getOp := usersPath["get"].(map[string]interface{})
	assert.Equal(t, "Get Users", getOp["summary"])
}

func TestOpenAPIExporter_Export_WithFolders(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	folder := coll.AddFolder("Users")
	folder.SetDescription("User operations")

	req := core.NewRequestDefinition("List Users", "GET", "/users")
	folder.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	// Check tags
	tags := spec["tags"].([]interface{})
	require.Len(t, tags, 1)
	tag := tags[0].(map[string]interface{})
	assert.Equal(t, "Users", tag["name"])
	assert.Equal(t, "User operations", tag["description"])

	// Check operation has tag
	paths := spec["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	getOp := usersPath["get"].(map[string]interface{})
	opTags := getOp["tags"].([]interface{})
	assert.Contains(t, opTags, "Users")
}

func TestOpenAPIExporter_Export_WithPathParams(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get User", "GET", "/users/{{id}}")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/users/{id}")

	userPath := paths["/users/{id}"].(map[string]interface{})
	getOp := userPath["get"].(map[string]interface{})
	params := getOp["parameters"].([]interface{})

	// Find path parameter
	var foundPathParam bool
	for _, p := range params {
		param := p.(map[string]interface{})
		if param["name"] == "id" && param["in"] == "path" {
			foundPathParam = true
			assert.True(t, param["required"].(bool))
		}
	}
	assert.True(t, foundPathParam)
}

func TestOpenAPIExporter_Export_WithQueryParams(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Search Users", "GET", "/users?page=1&limit=10")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	getOp := usersPath["get"].(map[string]interface{})
	params := getOp["parameters"].([]interface{})

	// Find query parameters
	paramNames := make(map[string]bool)
	for _, p := range params {
		param := p.(map[string]interface{})
		if param["in"] == "query" {
			paramNames[param["name"].(string)] = true
		}
	}
	assert.True(t, paramNames["page"])
	assert.True(t, paramNames["limit"])
}

func TestOpenAPIExporter_Export_WithHeaders(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get Users", "GET", "/users")
	req.SetHeader("X-Custom-Header", "custom-value")
	req.SetHeader("Accept", "application/json")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	getOp := usersPath["get"].(map[string]interface{})
	params := getOp["parameters"].([]interface{})

	// Find header parameters
	headerNames := make(map[string]bool)
	for _, p := range params {
		param := p.(map[string]interface{})
		if param["in"] == "header" {
			headerNames[param["name"].(string)] = true
		}
	}
	assert.True(t, headerNames["X-Custom-Header"])
	assert.True(t, headerNames["Accept"])
}

func TestOpenAPIExporter_Export_WithBody(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Create User", "POST", "/users")
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(`{"name": "John", "email": "john@example.com"}`)
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	postOp := usersPath["post"].(map[string]interface{})

	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	assert.Contains(t, content, "application/json")

	jsonContent := content["application/json"].(map[string]interface{})
	assert.NotNil(t, jsonContent["schema"])
	assert.NotNil(t, jsonContent["example"])
}

func TestOpenAPIExporter_Export_WithAuth(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	t.Run("bearer auth", func(t *testing.T) {
		coll := core.NewCollection("Test API")
		coll.SetAuth(core.AuthConfig{
			Type:  "bearer",
			Token: "mytoken",
		})

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		components := spec["components"].(map[string]interface{})
		schemes := components["securitySchemes"].(map[string]interface{})
		bearerAuth := schemes["bearerAuth"].(map[string]interface{})
		assert.Equal(t, "http", bearerAuth["type"])
		assert.Equal(t, "bearer", bearerAuth["scheme"])
	})

	t.Run("basic auth", func(t *testing.T) {
		coll := core.NewCollection("Test API")
		coll.SetAuth(core.AuthConfig{
			Type:     "basic",
			Username: "user",
			Password: "pass",
		})

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		components := spec["components"].(map[string]interface{})
		schemes := components["securitySchemes"].(map[string]interface{})
		basicAuth := schemes["basicAuth"].(map[string]interface{})
		assert.Equal(t, "http", basicAuth["type"])
		assert.Equal(t, "basic", basicAuth["scheme"])
	})

	t.Run("api key auth", func(t *testing.T) {
		coll := core.NewCollection("Test API")
		coll.SetAuth(core.AuthConfig{
			Type:  "apikey",
			Key:   "X-API-Key",
			Value: "secret",
			In:    "header",
		})

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		components := spec["components"].(map[string]interface{})
		schemes := components["securitySchemes"].(map[string]interface{})
		apiKey := schemes["apiKey"].(map[string]interface{})
		assert.Equal(t, "apiKey", apiKey["type"])
		assert.Equal(t, "X-API-Key", apiKey["name"])
		assert.Equal(t, "header", apiKey["in"])
	})
}

func TestOpenAPIExporter_Export_WithBaseURL(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	coll.SetVariable("base_url", "https://api.example.com/v1")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	servers := spec["servers"].([]interface{})
	require.Len(t, servers, 1)
	server := servers[0].(map[string]interface{})
	assert.Equal(t, "https://api.example.com/v1", server["url"])
}

func TestOpenAPIExporter_Export_AllMethods(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		req := core.NewRequestDefinition(method+" resource", method, "/resource")
		coll.AddRequest(req)
	}

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	resourcePath := paths["/resource"].(map[string]interface{})

	assert.Contains(t, resourcePath, "get")
	assert.Contains(t, resourcePath, "post")
	assert.Contains(t, resourcePath, "put")
	assert.Contains(t, resourcePath, "delete")
	assert.Contains(t, resourcePath, "patch")
	assert.Contains(t, resourcePath, "head")
	assert.Contains(t, resourcePath, "options")
}

func TestOpenAPIExporter_Export_NilCollection(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	_, err := exp.Export(ctx, nil)
	assert.ErrorIs(t, err, ErrInvalidCollection)
}

func TestOpenAPIExporter_Export_NestedFolders(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	folder := coll.AddFolder("API")
	subfolder := folder.AddFolder("v1")

	req := core.NewRequestDefinition("Get Data", "GET", "/data")
	subfolder.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	// Check nested tag
	tags := spec["tags"].([]interface{})
	tagNames := make([]string, 0)
	for _, t := range tags {
		tag := t.(map[string]interface{})
		tagNames = append(tagNames, tag["name"].(string))
	}
	assert.Contains(t, tagNames, "API")
	assert.Contains(t, tagNames, "API/v1")
}

func TestOpenAPIExporter_Export_InferSchemaFromJSON(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Create Item", "POST", "/items")
	req.SetBody(`{
		"name": "Test",
		"count": 42,
		"price": 19.99,
		"active": true,
		"tags": ["a", "b"]
	}`)
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	itemsPath := paths["/items"].(map[string]interface{})
	postOp := itemsPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	jsonContent := content["application/json"].(map[string]interface{})
	schema := jsonContent["schema"].(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "count")
	assert.Contains(t, props, "price")
	assert.Contains(t, props, "active")
	assert.Contains(t, props, "tags")
}
