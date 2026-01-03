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

func TestCurlExporter_Export_NilCollection(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	_, err := exp.Export(ctx, nil)
	assert.ErrorIs(t, err, ErrInvalidCollection)
}

func TestCurlExporter_ExportRequest_NilRequest(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	_, err := exp.ExportRequest(ctx, nil)
	assert.ErrorIs(t, err, ErrInvalidCollection)
}

func TestCurlExporter_Export_NestedFolders(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	coll := core.NewCollection("API")
	folder := coll.AddFolder("V1")
	subfolder := folder.AddFolder("Users")
	subfolder.SetDescription("User management")

	req := core.NewRequestDefinition("Get User", "GET", "https://api.example.com/v1/users/1")
	subfolder.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	content := string(result)
	assert.Contains(t, content, "=== V1 ===")
	assert.Contains(t, content, "=== Users ===")
	assert.Contains(t, content, "# Get User")
}

func TestCurlExporter_Export_CollectionWithDescription(t *testing.T) {
	exp := NewCurlExporter()
	ctx := context.Background()

	coll := core.NewCollection("My Detailed API")
	coll.SetDescription("This is a comprehensive API for testing")

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	content := string(result)
	assert.Contains(t, content, "# Collection: My Detailed API")
	assert.Contains(t, content, "# This is a comprehensive API for testing")
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

func TestOpenAPIExporter_Export_RelativeURL(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get Items", "GET", "/api/items?page=1&limit=10")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/api/items")
}

func TestOpenAPIExporter_Export_TemplatedPath(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get User", "GET", "/{userId}/profile")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/{userId}/profile")
}

func TestOpenAPIExporter_Export_MultipleRequestsSamePathDifferentMethod(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req1 := core.NewRequestDefinition("Get Users", "GET", "/users")
	req2 := core.NewRequestDefinition("Create User", "POST", "/users")
	req3 := core.NewRequestDefinition("Delete All Users", "DELETE", "/users")
	coll.AddRequest(req1)
	coll.AddRequest(req2)
	coll.AddRequest(req3)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	usersPath := paths["/users"].(map[string]interface{})
	assert.Contains(t, usersPath, "get")
	assert.Contains(t, usersPath, "post")
	assert.Contains(t, usersPath, "delete")
}

func TestOpenAPIExporter_Export_InferSchemaFromArray(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Create Items", "POST", "/items")
	req.SetBody(`[{"name": "item1"}, {"name": "item2"}]`)
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

	assert.Equal(t, "array", schema["type"])
}

func TestOpenAPIExporter_Export_InferSchemaFromEmptyBody(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Create Item", "POST", "/items")
	req.SetBody("")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/items")
}

func TestOpenAPIExporter_Export_InferSchemaFromNestedObject(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Test", "POST", "/test")
	req.SetBody(`{
		"user": {
			"name": "John",
			"address": {
				"city": "NYC",
				"zip": 10001
			}
		}
	}`)
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	testPath := paths["/test"].(map[string]interface{})
	postOp := testPath["post"].(map[string]interface{})
	requestBody := postOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	jsonContent := content["application/json"].(map[string]interface{})
	schema := jsonContent["schema"].(map[string]interface{})

	assert.Equal(t, "object", schema["type"])
	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "user")
}

func TestOpenAPIExporter_Export_GeneratesOperationID(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Get User Profile", "GET", "/users/{id}/profile")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	userPath := paths["/users/{id}/profile"].(map[string]interface{})
	getOp := userPath["get"].(map[string]interface{})
	assert.Contains(t, getOp, "operationId")
}

func TestOpenAPIExporter_Export_URLWithPort(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	coll := core.NewCollection("Test API")
	req := core.NewRequestDefinition("Test", "GET", "http://localhost:8080/api/users")
	coll.AddRequest(req)

	result, err := exp.Export(ctx, coll)
	require.NoError(t, err)

	var spec map[string]interface{}
	err = json.Unmarshal(result, &spec)
	require.NoError(t, err)

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/api/users")
}

// Test helper functions directly

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		reqName  string
		expected string
	}{
		{
			name:     "generates from request name",
			method:   "GET",
			path:     "/users",
			reqName:  "Get All Users",
			expected: "getAllUsers",
		},
		{
			name:     "generates from path when no name",
			method:   "POST",
			path:     "/users/profile",
			reqName:  "",
			expected: "post_users_profile",
		},
		{
			name:     "skips path parameters",
			method:   "GET",
			path:     "/users/{id}/posts",
			reqName:  "",
			expected: "get_users_posts",
		},
		{
			name:     "handles single word name",
			method:   "GET",
			path:     "/users",
			reqName:  "Users",
			expected: "users",
		},
		{
			name:     "handles root path",
			method:   "GET",
			path:     "/",
			reqName:  "",
			expected: "get",
		},
		{
			name:     "handles empty path parts",
			method:   "DELETE",
			path:     "//users//",
			reqName:  "",
			expected: "delete_users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateOperationID(tt.method, tt.path, tt.reqName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathAndQuery(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantPath    string
		wantParams  map[string]string
	}{
		{
			name:       "simple path",
			url:        "/users",
			wantPath:   "/users",
			wantParams: map[string]string{},
		},
		{
			name:       "path with query",
			url:        "/users?page=1&limit=10",
			wantPath:   "/users",
			wantParams: map[string]string{"page": "1", "limit": "10"},
		},
		{
			name:       "full URL with path and query",
			url:        "https://api.example.com/users?name=john",
			wantPath:   "/users",
			wantParams: map[string]string{"name": "john"},
		},
		{
			name:       "relative path starting with brace",
			url:        "{baseUrl}/users",
			wantPath:   "{baseUrl}/users",
			wantParams: map[string]string{},
		},
		{
			name:       "URL without protocol gets prefixed",
			url:        "api/users",
			wantPath:   "/users",
			wantParams: map[string]string{},
		},
		{
			name:       "encoded query params",
			url:        "/search?q=hello%20world&tag=a%2Bb",
			wantPath:   "/search",
			wantParams: map[string]string{"q": "hello world", "tag": "a+b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, params := extractPathAndQuery(tt.url)
			assert.Equal(t, tt.wantPath, path)
			for k, v := range tt.wantParams {
				assert.Equal(t, v, params[k], "param %s mismatch", k)
			}
		})
	}
}

func TestParseQueryString(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		want   map[string]string
	}{
		{
			name:  "simple params",
			query: "a=1&b=2",
			want:  map[string]string{"a": "1", "b": "2"},
		},
		{
			name:  "encoded values",
			query: "name=John%20Doe&city=New%20York",
			want:  map[string]string{"name": "John Doe", "city": "New York"},
		},
		{
			name:  "param without value",
			query: "flag&name=test",
			want:  map[string]string{"flag": "", "name": "test"},
		},
		{
			name:  "empty query",
			query: "",
			want:  map[string]string{"": ""},
		},
		{
			name:  "value with equals sign",
			query: "expr=a=b",
			want:  map[string]string{"expr": "a=b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(map[string]string)
			parseQueryString(tt.query, params)
			for k, v := range tt.want {
				assert.Equal(t, v, params[k], "param %s mismatch", k)
			}
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		want   []string
	}{
		{
			name: "single param",
			path: "/users/{id}",
			want: []string{"id"},
		},
		{
			name: "multiple params",
			path: "/users/{userId}/posts/{postId}",
			want: []string{"userId", "postId"},
		},
		{
			name: "no params",
			path: "/users",
			want: nil,
		},
		{
			name: "param at start",
			path: "{version}/users",
			want: []string{"version"},
		},
		{
			name: "malformed braces - missing end",
			path: "/users/{id",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPathParams(tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestConvertVariablesToPathParams(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "single variable",
			url:  "/users/{{id}}",
			want: "/users/{id}",
		},
		{
			name: "multiple variables",
			url:  "/{{version}}/users/{{userId}}",
			want: "/{version}/users/{userId}",
		},
		{
			name: "no variables",
			url:  "/users/123",
			want: "/users/123",
		},
		{
			name: "malformed - missing end braces",
			url:  "/users/{{id",
			want: "/users/{{id",
		},
		{
			name: "already converted",
			url:  "/users/{id}",
			want: "/users/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVariablesToPathParams(tt.url)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOpenAPIExporter_InferSchemaFromBody(t *testing.T) {
	exp := NewOpenAPIExporter()

	t.Run("JSON body with object", func(t *testing.T) {
		schema := exp.inferSchemaFromBody(`{"name": "John"}`, "application/json")
		assert.Equal(t, "object", schema.Type)
		assert.Contains(t, schema.Properties, "name")
	})

	t.Run("JSON body with array", func(t *testing.T) {
		schema := exp.inferSchemaFromBody(`[1, 2, 3]`, "application/json")
		assert.Equal(t, "array", schema.Type)
	})

	t.Run("form urlencoded body", func(t *testing.T) {
		schema := exp.inferSchemaFromBody("name=john", "application/x-www-form-urlencoded")
		assert.Equal(t, "object", schema.Type)
	})

	t.Run("plain text body", func(t *testing.T) {
		schema := exp.inferSchemaFromBody("hello world", "text/plain")
		assert.Equal(t, "string", schema.Type)
	})

	t.Run("invalid JSON returns string schema", func(t *testing.T) {
		schema := exp.inferSchemaFromBody("{invalid json", "application/json")
		assert.Equal(t, "string", schema.Type)
	})

	t.Run("JSON with charset", func(t *testing.T) {
		schema := exp.inferSchemaFromBody(`{"ok": true}`, "application/json; charset=utf-8")
		assert.Equal(t, "object", schema.Type)
	})
}

func TestOpenAPIExporter_InferSchemaFromValue(t *testing.T) {
	exp := NewOpenAPIExporter()

	t.Run("string value", func(t *testing.T) {
		schema := exp.inferSchemaFromValue("hello")
		assert.Equal(t, "string", schema.Type)
	})

	t.Run("integer value", func(t *testing.T) {
		schema := exp.inferSchemaFromValue(float64(42))
		assert.Equal(t, "integer", schema.Type)
	})

	t.Run("float value", func(t *testing.T) {
		schema := exp.inferSchemaFromValue(float64(3.14))
		assert.Equal(t, "number", schema.Type)
	})

	t.Run("boolean value", func(t *testing.T) {
		schema := exp.inferSchemaFromValue(true)
		assert.Equal(t, "boolean", schema.Type)
	})

	t.Run("null value", func(t *testing.T) {
		schema := exp.inferSchemaFromValue(nil)
		assert.Equal(t, "string", schema.Type)
		assert.True(t, schema.Nullable)
	})

	t.Run("empty array", func(t *testing.T) {
		schema := exp.inferSchemaFromValue([]interface{}{})
		assert.Equal(t, "array", schema.Type)
	})

	t.Run("array with items", func(t *testing.T) {
		schema := exp.inferSchemaFromValue([]interface{}{"a", "b"})
		assert.Equal(t, "array", schema.Type)
		assert.NotNil(t, schema.Items)
		assert.Equal(t, "string", schema.Items.Type)
	})

	t.Run("nested object", func(t *testing.T) {
		val := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
			},
		}
		schema := exp.inferSchemaFromValue(val)
		assert.Equal(t, "object", schema.Type)
		userProp := schema.Properties["user"]
		assert.Equal(t, "object", userProp.Type)
	})
}

func TestOpenAPIExporter_ParseExample(t *testing.T) {
	exp := NewOpenAPIExporter()

	t.Run("JSON body", func(t *testing.T) {
		result := exp.parseExample(`{"name": "John"}`, "application/json")
		m, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "John", m["name"])
	})

	t.Run("invalid JSON returns string", func(t *testing.T) {
		result := exp.parseExample("{invalid", "application/json")
		assert.Equal(t, "{invalid", result)
	})

	t.Run("non-JSON content type returns string", func(t *testing.T) {
		result := exp.parseExample("hello world", "text/plain")
		assert.Equal(t, "hello world", result)
	})
}

func TestOpenAPIExporter_ConvertAuth(t *testing.T) {
	exp := NewOpenAPIExporter()

	t.Run("bearer auth", func(t *testing.T) {
		auth := core.AuthConfig{Type: "bearer"}
		name, scheme := exp.convertAuth(auth)
		assert.Equal(t, "bearerAuth", name)
		assert.Equal(t, "http", scheme.Type)
		assert.Equal(t, "bearer", scheme.Scheme)
	})

	t.Run("basic auth", func(t *testing.T) {
		auth := core.AuthConfig{Type: "basic"}
		name, scheme := exp.convertAuth(auth)
		assert.Equal(t, "basicAuth", name)
		assert.Equal(t, "http", scheme.Type)
		assert.Equal(t, "basic", scheme.Scheme)
	})

	t.Run("apikey auth with in specified", func(t *testing.T) {
		auth := core.AuthConfig{Type: "apikey", Key: "X-API-Key", In: "query"}
		name, scheme := exp.convertAuth(auth)
		assert.Equal(t, "apiKey", name)
		assert.Equal(t, "apiKey", scheme.Type)
		assert.Equal(t, "X-API-Key", scheme.Name)
		assert.Equal(t, "query", scheme.In)
	})

	t.Run("apikey auth defaults to header", func(t *testing.T) {
		auth := core.AuthConfig{Type: "apikey", Key: "X-API-Key"}
		name, scheme := exp.convertAuth(auth)
		assert.Equal(t, "apiKey", name)
		assert.Equal(t, "apiKey", scheme.Type)
		assert.Equal(t, "X-API-Key", scheme.Name)
		assert.Equal(t, "header", scheme.In)
	})

	t.Run("unknown auth type returns empty", func(t *testing.T) {
		auth := core.AuthConfig{Type: "oauth2"}
		name, scheme := exp.convertAuth(auth)
		assert.Equal(t, "", name)
		assert.Equal(t, "", scheme.Type)
	})
}

func TestOpenAPIExporter_AddRequest(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := context.Background()

	t.Run("request with auth", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		req.SetAuth(core.AuthConfig{Type: "bearer", Token: "test-token"})
		coll.AddRequest(req)

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		// Check security schemes exist
		components := spec["components"].(map[string]interface{})
		secSchemes := components["securitySchemes"].(map[string]interface{})
		assert.Contains(t, secSchemes, "bearerAuth")
	})

	t.Run("request with path params", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Get User", "GET", "https://api.example.com/users/{id}")
		coll.AddRequest(req)

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		assert.Contains(t, paths, "/users/{id}")
	})

	t.Run("request with query params in URL", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Search", "GET", "https://api.example.com/search?q=test&limit=10")
		coll.AddRequest(req)

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		searchPath := paths["/search"].(map[string]interface{})
		getOp := searchPath["get"].(map[string]interface{})
		params := getOp["parameters"].([]interface{})

		// Check query params are included
		assert.GreaterOrEqual(t, len(params), 2)
	})

	t.Run("POST request with JSON body", func(t *testing.T) {
		coll := core.NewCollection("Test")
		req := core.NewRequestDefinition("Create User", "POST", "https://api.example.com/users")
		req.SetBody(`{"name": "John", "email": "john@example.com"}`)
		req.SetHeader("Content-Type", "application/json")
		coll.AddRequest(req)

		result, err := exp.Export(ctx, coll)
		require.NoError(t, err)

		var spec map[string]interface{}
		err = json.Unmarshal(result, &spec)
		require.NoError(t, err)

		paths := spec["paths"].(map[string]interface{})
		usersPath := paths["/users"].(map[string]interface{})
		postOp := usersPath["post"].(map[string]interface{})
		reqBody := postOp["requestBody"].(map[string]interface{})
		content := reqBody["content"].(map[string]interface{})
		assert.Contains(t, content, "application/json")
	})
}
