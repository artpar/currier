package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/artpar/currier/internal/core"
	"gopkg.in/yaml.v3"
)

// OpenAPIImporter imports OpenAPI 3.x specifications.
type OpenAPIImporter struct{}

// NewOpenAPIImporter creates a new OpenAPI importer.
func NewOpenAPIImporter() *OpenAPIImporter {
	return &OpenAPIImporter{}
}

func (o *OpenAPIImporter) Name() string {
	return "OpenAPI 3.x"
}

func (o *OpenAPIImporter) Format() Format {
	return FormatOpenAPI
}

func (o *OpenAPIImporter) FileExtensions() []string {
	return []string{".yaml", ".yml", ".json"}
}

func (o *OpenAPIImporter) DetectFormat(content []byte) bool {
	// Try JSON first
	var check struct {
		OpenAPI string `json:"openapi" yaml:"openapi"`
	}

	if err := json.Unmarshal(content, &check); err == nil {
		return strings.HasPrefix(check.OpenAPI, "3.")
	}

	// Try YAML
	if err := yaml.Unmarshal(content, &check); err == nil {
		return strings.HasPrefix(check.OpenAPI, "3.")
	}

	return false
}

func (o *OpenAPIImporter) Import(ctx context.Context, content []byte) (*core.Collection, error) {
	var spec openAPISpec

	// Try JSON first, then YAML
	if err := json.Unmarshal(content, &spec); err != nil {
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("%w: failed to parse OpenAPI spec: %v", ErrParseError, err)
		}
	}

	// Validate version
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return nil, fmt.Errorf("%w: expected OpenAPI 3.x, got %s", ErrUnsupportedVersion, spec.OpenAPI)
	}

	// Create collection
	collName := "OpenAPI Import"
	if spec.Info.Title != "" {
		collName = spec.Info.Title
	}

	coll := core.NewCollection(collName)
	coll.SetDescription(spec.Info.Description)
	coll.SetVersion(spec.Info.Version)

	// Determine base URL from servers
	baseURL := o.getBaseURL(spec.Servers)
	if baseURL != "" {
		coll.SetVariable("base_url", baseURL)
	}

	// Group paths by tags
	taggedPaths := make(map[string][]pathOperation)
	untaggedPaths := make([]pathOperation, 0)

	// Process all paths
	for path, pathItem := range spec.Paths {
		operations := o.extractOperations(path, pathItem)
		for _, op := range operations {
			if len(op.Tags) > 0 {
				tag := op.Tags[0] // Use first tag for grouping
				taggedPaths[tag] = append(taggedPaths[tag], op)
			} else {
				untaggedPaths = append(untaggedPaths, op)
			}
		}
	}

	// Create folders for tags (sorted for deterministic output)
	tags := make([]string, 0, len(taggedPaths))
	for tag := range taggedPaths {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		ops := taggedPaths[tag]
		folder := coll.AddFolder(tag)

		// Find tag description if available
		for _, t := range spec.Tags {
			if t.Name == tag {
				folder.SetDescription(t.Description)
				break
			}
		}

		// Sort operations by path then method
		sort.Slice(ops, func(i, j int) bool {
			if ops[i].Path == ops[j].Path {
				return ops[i].Method < ops[j].Method
			}
			return ops[i].Path < ops[j].Path
		})

		for _, op := range ops {
			req := o.createRequest(op, baseURL, spec.Components)
			folder.AddRequest(req)
		}
	}

	// Add untagged requests to root
	sort.Slice(untaggedPaths, func(i, j int) bool {
		if untaggedPaths[i].Path == untaggedPaths[j].Path {
			return untaggedPaths[i].Method < untaggedPaths[j].Method
		}
		return untaggedPaths[i].Path < untaggedPaths[j].Path
	})

	for _, op := range untaggedPaths {
		req := o.createRequest(op, baseURL, spec.Components)
		coll.AddRequest(req)
	}

	// Set collection-level auth from security schemes
	if len(spec.Security) > 0 && spec.Components != nil {
		auth := o.extractAuth(spec.Security, spec.Components.SecuritySchemes)
		if auth.Type != "" {
			coll.SetAuth(auth)
		}
	}

	return coll, nil
}

func (o *OpenAPIImporter) getBaseURL(servers []openAPIServer) string {
	if len(servers) == 0 {
		return ""
	}

	// Use first server as default
	server := servers[0]
	baseURL := server.URL

	// Replace server variables with defaults
	for name, variable := range server.Variables {
		defaultVal := variable.Default
		if defaultVal == "" && len(variable.Enum) > 0 {
			defaultVal = variable.Enum[0]
		}
		baseURL = strings.ReplaceAll(baseURL, "{"+name+"}", defaultVal)
	}

	return baseURL
}

type pathOperation struct {
	Path        string
	Method      string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Parameters  []openAPIParameter
	RequestBody *openAPIRequestBody
	Security    []map[string][]string
	Deprecated  bool
}

func (o *OpenAPIImporter) extractOperations(path string, pathItem openAPIPathItem) []pathOperation {
	var ops []pathOperation

	// Common parameters for all operations on this path
	commonParams := pathItem.Parameters

	methods := map[string]*openAPIOperation{
		"GET":     pathItem.Get,
		"POST":    pathItem.Post,
		"PUT":     pathItem.Put,
		"DELETE":  pathItem.Delete,
		"PATCH":   pathItem.Patch,
		"HEAD":    pathItem.Head,
		"OPTIONS": pathItem.Options,
	}

	for method, operation := range methods {
		if operation == nil {
			continue
		}

		op := pathOperation{
			Path:        path,
			Method:      method,
			OperationID: operation.OperationID,
			Summary:     operation.Summary,
			Description: operation.Description,
			Tags:        operation.Tags,
			RequestBody: operation.RequestBody,
			Security:    operation.Security,
			Deprecated:  operation.Deprecated,
		}

		// Merge path-level and operation-level parameters
		op.Parameters = append(op.Parameters, commonParams...)
		op.Parameters = append(op.Parameters, operation.Parameters...)

		ops = append(ops, op)
	}

	return ops
}

func (o *OpenAPIImporter) createRequest(op pathOperation, baseURL string, components *openAPIComponents) *core.RequestDefinition {
	// Generate request name
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", op.Method, op.Path)
	}

	// Build URL with path parameters replaced by placeholders
	reqURL := baseURL + op.Path
	if baseURL == "" {
		reqURL = "{{base_url}}" + op.Path
	}

	req := core.NewRequestDefinition(name, op.Method, reqURL)

	// Set description
	desc := op.Description
	if desc == "" {
		desc = op.Summary
	}
	if op.Deprecated {
		desc = "[DEPRECATED] " + desc
	}
	req.SetDescription(desc)

	// Process parameters
	for _, param := range op.Parameters {
		// Resolve $ref if present
		param = o.resolveParameterRef(param, components)

		switch param.In {
		case "header":
			example := o.getExampleValue(param.Schema, param.Example)
			req.SetHeader(param.Name, example)
		case "query":
			// Add query parameters to URL
			if strings.Contains(reqURL, "?") {
				reqURL += "&"
			} else {
				reqURL += "?"
			}
			example := o.getExampleValue(param.Schema, param.Example)
			reqURL += url.QueryEscape(param.Name) + "=" + url.QueryEscape(example)
		case "path":
			// Replace path parameter placeholder
			placeholder := "{" + param.Name + "}"
			example := o.getExampleValue(param.Schema, param.Example)
			if example == "" {
				example = "{{" + param.Name + "}}"
			}
			reqURL = strings.ReplaceAll(reqURL, placeholder, example)
		}
	}

	// Update URL with query parameters
	req = core.NewRequestDefinition(name, op.Method, reqURL)
	req.SetDescription(desc)

	// Re-add headers
	for _, param := range op.Parameters {
		param = o.resolveParameterRef(param, components)
		if param.In == "header" {
			example := o.getExampleValue(param.Schema, param.Example)
			req.SetHeader(param.Name, example)
		}
	}

	// Process request body
	if op.RequestBody != nil {
		body := o.resolveRequestBodyRef(op.RequestBody, components)
		if body != nil {
			// Prefer JSON content type
			if content, ok := body.Content["application/json"]; ok {
				req.SetHeader("Content-Type", "application/json")
				example := o.generateBodyExample(content.Schema, content.Example, components)
				req.SetBody(example)
			} else if content, ok := body.Content["application/x-www-form-urlencoded"]; ok {
				req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
				example := o.generateFormExample(content.Schema, components)
				req.SetBody(example)
			} else {
				// Use first available content type
				for contentType, content := range body.Content {
					req.SetHeader("Content-Type", contentType)
					example := o.generateBodyExample(content.Schema, content.Example, components)
					req.SetBody(example)
					break
				}
			}
		}
	}

	// Process security
	if len(op.Security) > 0 && components != nil {
		auth := o.extractAuth(op.Security, components.SecuritySchemes)
		if auth.Type != "" {
			req.SetAuth(auth)
		}
	}

	return req
}

func (o *OpenAPIImporter) resolveParameterRef(param openAPIParameter, components *openAPIComponents) openAPIParameter {
	if param.Ref == "" || components == nil {
		return param
	}

	// Extract reference name from #/components/parameters/Name
	refName := extractRefName(param.Ref)
	if refName != "" && components.Parameters != nil {
		if resolved, ok := components.Parameters[refName]; ok {
			return resolved
		}
	}

	return param
}

func (o *OpenAPIImporter) resolveRequestBodyRef(body *openAPIRequestBody, components *openAPIComponents) *openAPIRequestBody {
	if body == nil || body.Ref == "" || components == nil {
		return body
	}

	refName := extractRefName(body.Ref)
	if refName != "" && components.RequestBodies != nil {
		if resolved, ok := components.RequestBodies[refName]; ok {
			return &resolved
		}
	}

	return body
}

func (o *OpenAPIImporter) resolveSchemaRef(schema *openAPISchema, components *openAPIComponents) *openAPISchema {
	if schema == nil || schema.Ref == "" || components == nil {
		return schema
	}

	refName := extractRefName(schema.Ref)
	if refName != "" && components.Schemas != nil {
		if resolved, ok := components.Schemas[refName]; ok {
			return &resolved
		}
	}

	return schema
}

func extractRefName(ref string) string {
	// Handle $ref like "#/components/schemas/Pet"
	re := regexp.MustCompile(`#/components/\w+/(.+)$`)
	matches := re.FindStringSubmatch(ref)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (o *OpenAPIImporter) getExampleValue(schema *openAPISchema, example interface{}) string {
	if example != nil {
		return fmt.Sprintf("%v", example)
	}

	if schema == nil {
		return ""
	}

	if schema.Example != nil {
		return fmt.Sprintf("%v", schema.Example)
	}

	if schema.Default != nil {
		return fmt.Sprintf("%v", schema.Default)
	}

	// Generate based on type
	switch schema.Type {
	case "string":
		if schema.Format == "date" {
			return "2024-01-01"
		}
		if schema.Format == "date-time" {
			return "2024-01-01T00:00:00Z"
		}
		if schema.Format == "email" {
			return "user@example.com"
		}
		if schema.Format == "uuid" {
			return "550e8400-e29b-41d4-a716-446655440000"
		}
		if len(schema.Enum) > 0 {
			return fmt.Sprintf("%v", schema.Enum[0])
		}
		return "string"
	case "integer":
		return "0"
	case "number":
		return "0.0"
	case "boolean":
		return "true"
	}

	return ""
}

func (o *OpenAPIImporter) generateBodyExample(schema *openAPISchema, example interface{}, components *openAPIComponents) string {
	if example != nil {
		data, _ := json.MarshalIndent(example, "", "  ")
		return string(data)
	}

	if schema == nil {
		return ""
	}

	// Resolve reference
	schema = o.resolveSchemaRef(schema, components)
	if schema == nil {
		return ""
	}

	if schema.Example != nil {
		data, _ := json.MarshalIndent(schema.Example, "", "  ")
		return string(data)
	}

	// Generate example from schema
	obj := o.generateSchemaExample(schema, components, 0)
	data, _ := json.MarshalIndent(obj, "", "  ")
	return string(data)
}

func (o *OpenAPIImporter) generateSchemaExample(schema *openAPISchema, components *openAPIComponents, depth int) interface{} {
	if depth > 5 {
		return nil // Prevent infinite recursion
	}

	if schema == nil {
		return nil
	}

	// Resolve reference
	schema = o.resolveSchemaRef(schema, components)
	if schema == nil {
		return nil
	}

	if schema.Example != nil {
		return schema.Example
	}

	switch schema.Type {
	case "object":
		obj := make(map[string]interface{})
		for propName, propSchema := range schema.Properties {
			obj[propName] = o.generateSchemaExample(&propSchema, components, depth+1)
		}
		return obj
	case "array":
		if schema.Items != nil {
			return []interface{}{o.generateSchemaExample(schema.Items, components, depth+1)}
		}
		return []interface{}{}
	case "string":
		return o.getExampleValue(schema, nil)
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return true
	}

	// Handle allOf, oneOf, anyOf
	if len(schema.AllOf) > 0 {
		merged := make(map[string]interface{})
		for _, s := range schema.AllOf {
			if result := o.generateSchemaExample(&s, components, depth+1); result != nil {
				if m, ok := result.(map[string]interface{}); ok {
					for k, v := range m {
						merged[k] = v
					}
				}
			}
		}
		return merged
	}

	if len(schema.OneOf) > 0 {
		return o.generateSchemaExample(&schema.OneOf[0], components, depth+1)
	}

	if len(schema.AnyOf) > 0 {
		return o.generateSchemaExample(&schema.AnyOf[0], components, depth+1)
	}

	return nil
}

func (o *OpenAPIImporter) generateFormExample(schema *openAPISchema, components *openAPIComponents) string {
	if schema == nil {
		return ""
	}

	schema = o.resolveSchemaRef(schema, components)
	if schema == nil {
		return ""
	}

	var pairs []string
	for propName, propSchema := range schema.Properties {
		value := o.getExampleValue(&propSchema, nil)
		pairs = append(pairs, url.QueryEscape(propName)+"="+url.QueryEscape(value))
	}

	sort.Strings(pairs) // Deterministic order
	return strings.Join(pairs, "&")
}

func (o *OpenAPIImporter) extractAuth(security []map[string][]string, schemes map[string]openAPISecurityScheme) core.AuthConfig {
	if len(security) == 0 || schemes == nil {
		return core.AuthConfig{}
	}

	// Use first security requirement
	for schemeName := range security[0] {
		scheme, ok := schemes[schemeName]
		if !ok {
			continue
		}

		switch scheme.Type {
		case "http":
			if scheme.Scheme == "bearer" {
				return core.AuthConfig{
					Type:  "bearer",
					Token: "{{access_token}}",
				}
			}
			if scheme.Scheme == "basic" {
				return core.AuthConfig{
					Type:     "basic",
					Username: "{{username}}",
					Password: "{{password}}",
				}
			}
		case "apiKey":
			return core.AuthConfig{
				Type:  "apikey",
				Key:   scheme.Name,
				Value: "{{api_key}}",
				In:    scheme.In,
			}
		}
	}

	return core.AuthConfig{}
}

// OpenAPI 3.x specification structures

type openAPISpec struct {
	OpenAPI    string                  `json:"openapi" yaml:"openapi"`
	Info       openAPIInfo             `json:"info" yaml:"info"`
	Servers    []openAPIServer         `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]openAPIPathItem `json:"paths" yaml:"paths"`
	Components *openAPIComponents      `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []map[string][]string   `json:"security,omitempty" yaml:"security,omitempty"`
	Tags       []openAPITag            `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

type openAPIServer struct {
	URL         string                        `json:"url" yaml:"url"`
	Description string                        `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]openAPIServerVar   `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type openAPIServerVar struct {
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
}

type openAPITag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type openAPIPathItem struct {
	Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string             `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Get         *openAPIOperation  `json:"get,omitempty" yaml:"get,omitempty"`
	Post        *openAPIOperation  `json:"post,omitempty" yaml:"post,omitempty"`
	Put         *openAPIOperation  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete      *openAPIOperation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch       *openAPIOperation  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Head        *openAPIOperation  `json:"head,omitempty" yaml:"head,omitempty"`
	Options     *openAPIOperation  `json:"options,omitempty" yaml:"options,omitempty"`
	Parameters  []openAPIParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type openAPIOperation struct {
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []openAPIParameter    `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *openAPIRequestBody   `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses,omitempty" yaml:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

type openAPIParameter struct {
	Ref         string        `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Name        string        `json:"name,omitempty" yaml:"name,omitempty"`
	In          string        `json:"in,omitempty" yaml:"in,omitempty"` // query, header, path, cookie
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *openAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     interface{}   `json:"example,omitempty" yaml:"example,omitempty"`
}

type openAPIRequestBody struct {
	Ref         string                      `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string                      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                        `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]openAPIMediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type openAPIMediaType struct {
	Schema  *openAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example interface{}    `json:"example,omitempty" yaml:"example,omitempty"`
}

type openAPIResponse struct {
	Description string                      `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]openAPIMediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type openAPISchema struct {
	Ref        string                   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type       string                   `json:"type,omitempty" yaml:"type,omitempty"`
	Format     string                   `json:"format,omitempty" yaml:"format,omitempty"`
	Properties map[string]openAPISchema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items      *openAPISchema           `json:"items,omitempty" yaml:"items,omitempty"`
	Required   []string                 `json:"required,omitempty" yaml:"required,omitempty"`
	Enum       []interface{}            `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default    interface{}              `json:"default,omitempty" yaml:"default,omitempty"`
	Example    interface{}              `json:"example,omitempty" yaml:"example,omitempty"`
	AllOf      []openAPISchema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf      []openAPISchema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf      []openAPISchema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
}

type openAPIComponents struct {
	Schemas         map[string]openAPISchema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Parameters      map[string]openAPIParameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBodies   map[string]openAPIRequestBody    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type   string `json:"type" yaml:"type"` // http, apiKey, oauth2, openIdConnect
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"` // bearer, basic (for http type)
	Name   string `json:"name,omitempty" yaml:"name,omitempty"` // for apiKey
	In     string `json:"in,omitempty" yaml:"in,omitempty"` // header, query, cookie (for apiKey)
}

// Verify OpenAPIImporter implements Importer interface
var _ Importer = (*OpenAPIImporter)(nil)
