package exporter

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/artpar/currier/internal/core"
)

// OpenAPIExporter exports collections to OpenAPI 3.0 format.
type OpenAPIExporter struct{}

// NewOpenAPIExporter creates a new OpenAPI exporter.
func NewOpenAPIExporter() *OpenAPIExporter {
	return &OpenAPIExporter{}
}

func (o *OpenAPIExporter) Name() string {
	return "OpenAPI 3.0"
}

func (o *OpenAPIExporter) Format() Format {
	return FormatOpenAPI
}

func (o *OpenAPIExporter) FileExtension() string {
	return ".openapi.json"
}

func (o *OpenAPIExporter) Export(ctx context.Context, coll *core.Collection) ([]byte, error) {
	if coll == nil {
		return nil, ErrInvalidCollection
	}

	spec := openAPISpec{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       coll.Name(),
			Description: coll.Description(),
			Version:     coll.Version(),
		},
		Paths: make(map[string]openAPIPathItem),
		Tags:  make([]openAPITag, 0),
	}

	if spec.Info.Version == "" {
		spec.Info.Version = "1.0.0"
	}

	// Add server from base_url variable if available
	if baseURL := coll.GetVariable("base_url"); baseURL != "" {
		spec.Servers = []openAPIServer{{URL: baseURL}}
	}

	// Export collection-level auth
	if coll.Auth().Type != "" {
		spec.Components = &openAPIComponents{
			SecuritySchemes: make(map[string]openAPISecurityScheme),
		}
		schemeName, scheme := o.convertAuth(coll.Auth())
		if schemeName != "" {
			spec.Components.SecuritySchemes[schemeName] = scheme
			spec.Security = []map[string][]string{{schemeName: {}}}
		}
	}

	// Track tags for folders
	tagSet := make(map[string]string) // name -> description

	// Export root-level requests
	for _, req := range coll.Requests() {
		o.addRequest(&spec, req, "")
	}

	// Export folders as tags
	for _, folder := range coll.Folders() {
		tagSet[folder.Name()] = folder.Description()
		o.exportFolder(&spec, folder, folder.Name(), tagSet)
	}

	// Build tags array (sorted)
	tagNames := make([]string, 0, len(tagSet))
	for name := range tagSet {
		tagNames = append(tagNames, name)
	}
	sort.Strings(tagNames)

	for _, name := range tagNames {
		spec.Tags = append(spec.Tags, openAPITag{
			Name:        name,
			Description: tagSet[name],
		})
	}

	return json.MarshalIndent(spec, "", "  ")
}

func (o *OpenAPIExporter) exportFolder(spec *openAPISpec, folder *core.Folder, tag string, tagSet map[string]string) {
	for _, req := range folder.Requests() {
		o.addRequest(spec, req, tag)
	}

	// Recursively export subfolders
	for _, subfolder := range folder.Folders() {
		subTag := tag + "/" + subfolder.Name()
		tagSet[subTag] = subfolder.Description()
		o.exportFolder(spec, subfolder, subTag, tagSet)
	}
}

func (o *OpenAPIExporter) addRequest(spec *openAPISpec, req *core.RequestDefinition, tag string) {
	// Parse URL to extract path and query params
	reqURL := req.URL()

	// Replace variable placeholders {{var}} with OpenAPI path params {var}
	reqURL = convertVariablesToPathParams(reqURL)

	// Extract base path from URL
	path, queryParams := extractPathAndQuery(reqURL)
	if path == "" {
		path = "/"
	}

	// Get or create path item
	pathItem, exists := spec.Paths[path]
	if !exists {
		pathItem = openAPIPathItem{}
	}

	// Create operation
	operation := openAPIOperation{
		Summary:     req.Name(),
		Description: req.Description(),
		OperationID: generateOperationID(req.Method(), path, req.Name()),
		Parameters:  make([]openAPIParameter, 0),
		Responses: map[string]openAPIResponse{
			"200": {Description: "Successful response"},
		},
	}

	if tag != "" {
		operation.Tags = []string{tag}
	}

	// Extract path parameters
	pathParams := extractPathParams(path)
	for _, param := range pathParams {
		operation.Parameters = append(operation.Parameters, openAPIParameter{
			Name:     param,
			In:       "path",
			Required: true,
			Schema:   &openAPISchema{Type: "string"},
		})
	}

	// Add query parameters
	for key, value := range queryParams {
		operation.Parameters = append(operation.Parameters, openAPIParameter{
			Name:    key,
			In:      "query",
			Schema:  &openAPISchema{Type: "string"},
			Example: value,
		})
	}

	// Add header parameters
	for key, value := range req.Headers() {
		// Skip content-type as it's handled by requestBody
		if strings.EqualFold(key, "Content-Type") {
			continue
		}
		// Skip authorization as it's handled by security
		if strings.EqualFold(key, "Authorization") {
			continue
		}
		operation.Parameters = append(operation.Parameters, openAPIParameter{
			Name:    key,
			In:      "header",
			Schema:  &openAPISchema{Type: "string"},
			Example: value,
		})
	}

	// Add request body if present
	body := req.Body()
	if body != "" && (req.Method() == "POST" || req.Method() == "PUT" || req.Method() == "PATCH") {
		contentType := req.GetHeader("Content-Type")
		if contentType == "" {
			// Try to detect JSON
			if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
				contentType = "application/json"
			} else {
				contentType = "text/plain"
			}
		}

		operation.RequestBody = &openAPIRequestBody{
			Content: map[string]openAPIMediaType{
				contentType: {
					Schema:  o.inferSchemaFromBody(body, contentType),
					Example: o.parseExample(body, contentType),
				},
			},
		}
	}

	// Add request-level security
	if req.Auth() != nil && req.Auth().Type != "" {
		if spec.Components == nil {
			spec.Components = &openAPIComponents{
				SecuritySchemes: make(map[string]openAPISecurityScheme),
			}
		}
		schemeName, scheme := o.convertAuth(*req.Auth())
		if schemeName != "" {
			spec.Components.SecuritySchemes[schemeName] = scheme
			operation.Security = []map[string][]string{{schemeName: {}}}
		}
	}

	// Assign operation to path item based on method
	switch strings.ToUpper(req.Method()) {
	case "GET":
		pathItem.Get = &operation
	case "POST":
		pathItem.Post = &operation
	case "PUT":
		pathItem.Put = &operation
	case "DELETE":
		pathItem.Delete = &operation
	case "PATCH":
		pathItem.Patch = &operation
	case "HEAD":
		pathItem.Head = &operation
	case "OPTIONS":
		pathItem.Options = &operation
	}

	spec.Paths[path] = pathItem
}

func (o *OpenAPIExporter) convertAuth(auth core.AuthConfig) (string, openAPISecurityScheme) {
	switch auth.Type {
	case "bearer":
		return "bearerAuth", openAPISecurityScheme{
			Type:   "http",
			Scheme: "bearer",
		}
	case "basic":
		return "basicAuth", openAPISecurityScheme{
			Type:   "http",
			Scheme: "basic",
		}
	case "apikey":
		in := auth.In
		if in == "" {
			in = "header"
		}
		return "apiKey", openAPISecurityScheme{
			Type: "apiKey",
			Name: auth.Key,
			In:   in,
		}
	}
	return "", openAPISecurityScheme{}
}

func (o *OpenAPIExporter) inferSchemaFromBody(body, contentType string) *openAPISchema {
	if strings.Contains(contentType, "application/json") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(body), &parsed); err == nil {
			return o.inferSchemaFromValue(parsed)
		}
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return &openAPISchema{
			Type:       "object",
			Properties: make(map[string]openAPISchema),
		}
	}

	return &openAPISchema{Type: "string"}
}

func (o *OpenAPIExporter) inferSchemaFromValue(v interface{}) *openAPISchema {
	switch val := v.(type) {
	case map[string]interface{}:
		props := make(map[string]openAPISchema)
		for k, v := range val {
			schema := o.inferSchemaFromValue(v)
			if schema != nil {
				props[k] = *schema
			}
		}
		return &openAPISchema{
			Type:       "object",
			Properties: props,
		}
	case []interface{}:
		if len(val) > 0 {
			return &openAPISchema{
				Type:  "array",
				Items: o.inferSchemaFromValue(val[0]),
			}
		}
		return &openAPISchema{Type: "array"}
	case string:
		return &openAPISchema{Type: "string"}
	case float64:
		if val == float64(int64(val)) {
			return &openAPISchema{Type: "integer"}
		}
		return &openAPISchema{Type: "number"}
	case bool:
		return &openAPISchema{Type: "boolean"}
	case nil:
		return &openAPISchema{Type: "string", Nullable: true}
	}
	return &openAPISchema{Type: "string"}
}

func (o *OpenAPIExporter) parseExample(body, contentType string) interface{} {
	if strings.Contains(contentType, "application/json") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(body), &parsed); err == nil {
			return parsed
		}
	}
	return body
}

// Helper functions

func convertVariablesToPathParams(urlStr string) string {
	// Convert {{variable}} to {variable}
	result := urlStr
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		varName := result[start+2 : start+end]
		result = result[:start] + "{" + varName + "}" + result[start+end+2:]
	}
	return result
}

func extractPathAndQuery(urlStr string) (string, map[string]string) {
	queryParams := make(map[string]string)

	// Handle relative URLs
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		// Check if it starts with a path
		if strings.HasPrefix(urlStr, "/") || strings.HasPrefix(urlStr, "{") {
			// Parse query string if present
			parts := strings.SplitN(urlStr, "?", 2)
			path := parts[0]
			if len(parts) > 1 {
				parseQueryString(parts[1], queryParams)
			}
			return path, queryParams
		}
		// Add https:// prefix for parsing
		urlStr = "https://example.com" + urlStr
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, queryParams
	}

	// Extract query parameters
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	return parsed.Path, queryParams
}

func parseQueryString(query string, params map[string]string) {
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key, _ := url.QueryUnescape(kv[0])
			value, _ := url.QueryUnescape(kv[1])
			params[key] = value
		} else if len(kv) == 1 {
			key, _ := url.QueryUnescape(kv[0])
			params[key] = ""
		}
	}
}

func extractPathParams(path string) []string {
	var params []string
	for {
		start := strings.Index(path, "{")
		if start == -1 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end == -1 {
			break
		}
		params = append(params, path[start+1:start+end])
		path = path[start+end+1:]
	}
	return params
}

func generateOperationID(method, path, name string) string {
	// Try to generate a meaningful operation ID
	if name != "" {
		// Convert name to camelCase
		words := strings.Fields(name)
		if len(words) > 0 {
			result := strings.ToLower(words[0])
			for _, word := range words[1:] {
				if len(word) > 0 {
					result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
				}
			}
			return result
		}
	}

	// Fall back to method + path based ID
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	var parts []string
	parts = append(parts, strings.ToLower(method))
	for _, part := range pathParts {
		if part != "" && !strings.HasPrefix(part, "{") {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "_")
}

// OpenAPI 3.0 specification structures for export

type openAPISpec struct {
	OpenAPI    string                     `json:"openapi"`
	Info       openAPIInfo                `json:"info"`
	Servers    []openAPIServer            `json:"servers,omitempty"`
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components *openAPIComponents         `json:"components,omitempty"`
	Security   []map[string][]string      `json:"security,omitempty"`
	Tags       []openAPITag               `json:"tags,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type openAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type openAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type openAPIPathItem struct {
	Get     *openAPIOperation `json:"get,omitempty"`
	Post    *openAPIOperation `json:"post,omitempty"`
	Put     *openAPIOperation `json:"put,omitempty"`
	Delete  *openAPIOperation `json:"delete,omitempty"`
	Patch   *openAPIOperation `json:"patch,omitempty"`
	Head    *openAPIOperation `json:"head,omitempty"`
	Options *openAPIOperation `json:"options,omitempty"`
}

type openAPIOperation struct {
	Tags        []string                   `json:"tags,omitempty"`
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Parameters  []openAPIParameter         `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type openAPIParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"` // query, header, path, cookie
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Schema      *openAPISchema `json:"schema,omitempty"`
	Example     interface{}    `json:"example,omitempty"`
}

type openAPIRequestBody struct {
	Description string                     `json:"description,omitempty"`
	Required    bool                       `json:"required,omitempty"`
	Content     map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema  *openAPISchema `json:"schema,omitempty"`
	Example interface{}    `json:"example,omitempty"`
}

type openAPIResponse struct {
	Description string                     `json:"description"`
	Content     map[string]openAPIMediaType `json:"content,omitempty"`
}

type openAPISchema struct {
	Type       string                   `json:"type,omitempty"`
	Format     string                   `json:"format,omitempty"`
	Properties map[string]openAPISchema `json:"properties,omitempty"`
	Items      *openAPISchema           `json:"items,omitempty"`
	Nullable   bool                     `json:"nullable,omitempty"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"` // bearer, basic (for http type)
	Name   string `json:"name,omitempty"`   // for apiKey
	In     string `json:"in,omitempty"`     // header, query, cookie (for apiKey)
}

// Verify OpenAPIExporter implements Exporter interface
var _ Exporter = (*OpenAPIExporter)(nil)
