package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/artpar/currier/internal/core"
)

// PostmanImporter imports Postman collection format (v2.0 and v2.1).
type PostmanImporter struct{}

// NewPostmanImporter creates a new Postman importer.
func NewPostmanImporter() *PostmanImporter {
	return &PostmanImporter{}
}

func (p *PostmanImporter) Name() string {
	return "Postman Collection"
}

func (p *PostmanImporter) Format() Format {
	return FormatPostman
}

func (p *PostmanImporter) FileExtensions() []string {
	return []string{".json", ".postman_collection.json"}
}

func (p *PostmanImporter) DetectFormat(content []byte) bool {
	var check struct {
		Info struct {
			Schema string `json:"schema"`
		} `json:"info"`
	}

	if err := json.Unmarshal(content, &check); err != nil {
		return false
	}

	// Check for Postman collection schema
	return strings.Contains(check.Info.Schema, "schema.getpostman.com/json/collection")
}

func (p *PostmanImporter) Import(ctx context.Context, content []byte) (*core.Collection, error) {
	var pm postmanCollection
	if err := json.Unmarshal(content, &pm); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseError, err)
	}

	// Determine version
	version := "2.0"
	if strings.Contains(pm.Info.Schema, "v2.1") {
		version = "2.1"
	}

	// Create collection
	coll := core.NewCollection(pm.Info.Name)
	coll.SetDescription(pm.Info.Description)
	coll.SetVersion(version)

	// Import variables
	for _, v := range pm.Variable {
		coll.SetVariable(v.Key, v.Value)
	}

	// Import auth if present
	if pm.Auth != nil {
		auth := convertPostmanAuth(pm.Auth)
		coll.SetAuth(auth)
	}

	// Import items (requests and folders)
	for _, item := range pm.Item {
		if err := p.importItem(coll, nil, item); err != nil {
			return nil, err
		}
	}

	// Import pre-request and test scripts from events
	for _, event := range pm.Event {
		if event.Listen == "prerequest" && event.Script != nil {
			coll.SetPreScript(strings.Join(event.Script.Exec, "\n"))
		}
		if event.Listen == "test" && event.Script != nil {
			coll.SetPostScript(strings.Join(event.Script.Exec, "\n"))
		}
	}

	return coll, nil
}

func (p *PostmanImporter) importItem(coll *core.Collection, folder *core.Folder, item postmanItem) error {
	// Check if this is a folder (has sub-items) or a request
	if len(item.Item) > 0 {
		// It's a folder
		var newFolder *core.Folder
		if folder == nil {
			newFolder = coll.AddFolder(item.Name)
		} else {
			newFolder = folder.AddFolder(item.Name)
		}
		newFolder.SetDescription(item.Description)

		// Recursively import sub-items
		for _, subItem := range item.Item {
			if err := p.importItem(coll, newFolder, subItem); err != nil {
				return err
			}
		}
	} else if item.Request != nil {
		// It's a request
		req := p.convertRequest(item)
		if folder == nil {
			coll.AddRequest(req)
		} else {
			folder.AddRequest(req)
		}
	}

	return nil
}

func (p *PostmanImporter) convertRequest(item postmanItem) *core.RequestDefinition {
	pm := item.Request

	// Get method and URL
	method := "GET"
	if pm.Method != "" {
		method = pm.Method
	}

	url := extractURL(pm.URL)

	req := core.NewRequestDefinition(item.Name, method, url)
	req.SetDescription(pm.Description)

	// Import headers
	for _, h := range pm.Header {
		if !h.Disabled {
			req.SetHeader(h.Key, h.Value)
		}
	}

	// Import body
	if pm.Body != nil {
		switch pm.Body.Mode {
		case "raw":
			req.SetBody(pm.Body.Raw)
		case "urlencoded":
			var pairs []string
			for _, p := range pm.Body.URLEncoded {
				if !p.Disabled {
					pairs = append(pairs, fmt.Sprintf("%s=%s", p.Key, p.Value))
				}
			}
			req.SetBody(strings.Join(pairs, "&"))
		case "formdata":
			// Store as JSON representation for now
			data, _ := json.Marshal(pm.Body.FormData)
			req.SetBody(string(data))
		case "graphql":
			if pm.Body.GraphQL != nil {
				body := map[string]interface{}{
					"query": pm.Body.GraphQL.Query,
				}
				if pm.Body.GraphQL.Variables != "" {
					var vars interface{}
					if err := json.Unmarshal([]byte(pm.Body.GraphQL.Variables), &vars); err == nil {
						body["variables"] = vars
					}
				}
				data, _ := json.Marshal(body)
				req.SetBody(string(data))
			}
		}
	}

	// Import auth
	if pm.Auth != nil {
		auth := convertPostmanAuth(pm.Auth)
		req.SetAuth(auth)
	}

	// Import scripts from events
	for _, event := range item.Event {
		if event.Listen == "prerequest" && event.Script != nil {
			req.SetPreScript(strings.Join(event.Script.Exec, "\n"))
		}
		if event.Listen == "test" && event.Script != nil {
			req.SetPostScript(strings.Join(event.Script.Exec, "\n"))
		}
	}

	return req
}

func extractURL(url interface{}) string {
	switch v := url.(type) {
	case string:
		return v
	case map[string]interface{}:
		if raw, ok := v["raw"].(string); ok {
			return raw
		}
		// Build URL from parts
		var result strings.Builder
		if protocol, ok := v["protocol"].(string); ok {
			result.WriteString(protocol)
			result.WriteString("://")
		}
		if host, ok := v["host"].([]interface{}); ok {
			var hostParts []string
			for _, h := range host {
				if s, ok := h.(string); ok {
					hostParts = append(hostParts, s)
				}
			}
			result.WriteString(strings.Join(hostParts, "."))
		}
		if port, ok := v["port"].(string); ok {
			result.WriteString(":")
			result.WriteString(port)
		}
		if path, ok := v["path"].([]interface{}); ok {
			for _, p := range path {
				if s, ok := p.(string); ok {
					result.WriteString("/")
					result.WriteString(s)
				}
			}
		}
		return result.String()
	}
	return ""
}

func convertPostmanAuth(auth *postmanAuth) core.AuthConfig {
	config := core.AuthConfig{Type: auth.Type}

	switch auth.Type {
	case "bearer":
		for _, item := range auth.Bearer {
			if item.Key == "token" {
				config.Token = item.Value
			}
		}
	case "basic":
		for _, item := range auth.Basic {
			if item.Key == "username" {
				config.Username = item.Value
			}
			if item.Key == "password" {
				config.Password = item.Value
			}
		}
	case "apikey":
		for _, item := range auth.APIKey {
			if item.Key == "key" {
				config.Key = item.Value
			}
			if item.Key == "value" {
				config.Value = item.Value
			}
			if item.Key == "in" {
				config.In = item.Value
			}
		}
	}

	return config
}

// Postman collection format structures

type postmanCollection struct {
	Info     postmanInfo     `json:"info"`
	Item     []postmanItem   `json:"item"`
	Event    []postmanEvent  `json:"event,omitempty"`
	Variable []postmanVar    `json:"variable,omitempty"`
	Auth     *postmanAuth    `json:"auth,omitempty"`
}

type postmanInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
	Version     string `json:"version,omitempty"`
}

type postmanItem struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Item        []postmanItem   `json:"item,omitempty"`
	Request     *postmanRequest `json:"request,omitempty"`
	Response    []interface{}   `json:"response,omitempty"`
	Event       []postmanEvent  `json:"event,omitempty"`
}

type postmanRequest struct {
	Method      string          `json:"method"`
	Header      []postmanHeader `json:"header,omitempty"`
	Body        *postmanBody    `json:"body,omitempty"`
	URL         interface{}     `json:"url"` // Can be string or object
	Auth        *postmanAuth    `json:"auth,omitempty"`
	Description string          `json:"description,omitempty"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode       string               `json:"mode"`
	Raw        string               `json:"raw,omitempty"`
	URLEncoded []postmanURLEncoded  `json:"urlencoded,omitempty"`
	FormData   []postmanFormData    `json:"formdata,omitempty"`
	GraphQL    *postmanGraphQL      `json:"graphql,omitempty"`
	Options    *postmanBodyOptions  `json:"options,omitempty"`
}

type postmanURLEncoded struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanFormData struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Type     string `json:"type,omitempty"` // text, file
	Src      string `json:"src,omitempty"`  // for files
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanGraphQL struct {
	Query     string `json:"query"`
	Variables string `json:"variables,omitempty"`
}

type postmanBodyOptions struct {
	Raw struct {
		Language string `json:"language,omitempty"`
	} `json:"raw,omitempty"`
}

type postmanAuth struct {
	Type   string              `json:"type"`
	Bearer []postmanAuthItem   `json:"bearer,omitempty"`
	Basic  []postmanAuthItem   `json:"basic,omitempty"`
	APIKey []postmanAuthItem   `json:"apikey,omitempty"`
	OAuth2 []postmanAuthItem   `json:"oauth2,omitempty"`
}

type postmanAuthItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

type postmanEvent struct {
	Listen string         `json:"listen"`
	Script *postmanScript `json:"script,omitempty"`
}

type postmanScript struct {
	ID   string   `json:"id,omitempty"`
	Type string   `json:"type,omitempty"`
	Exec []string `json:"exec,omitempty"`
}

type postmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Verify PostmanImporter implements Importer interface
var _ Importer = (*PostmanImporter)(nil)
