package exporter

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/artpar/currier/internal/core"
	"github.com/google/uuid"
)

// PostmanExporter exports collections to Postman format (v2.1).
type PostmanExporter struct{}

// NewPostmanExporter creates a new Postman exporter.
func NewPostmanExporter() *PostmanExporter {
	return &PostmanExporter{}
}

func (p *PostmanExporter) Name() string {
	return "Postman Collection"
}

func (p *PostmanExporter) Format() Format {
	return FormatPostman
}

func (p *PostmanExporter) FileExtension() string {
	return ".postman_collection.json"
}

func (p *PostmanExporter) Export(ctx context.Context, coll *core.Collection) ([]byte, error) {
	if coll == nil {
		return nil, ErrInvalidCollection
	}

	pm := postmanCollection{
		Info: postmanInfo{
			PostmanID:   uuid.New().String(),
			Name:        coll.Name(),
			Description: coll.Description(),
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: make([]postmanItem, 0),
	}

	// Export variables
	for key, value := range coll.Variables() {
		pm.Variable = append(pm.Variable, postmanVar{
			Key:   key,
			Value: value,
			Type:  "string",
		})
	}

	// Export collection-level auth
	if coll.Auth().Type != "" {
		pm.Auth = p.convertAuth(coll.Auth())
	}

	// Export collection-level scripts
	if coll.PreScript() != "" || coll.PostScript() != "" {
		pm.Event = make([]postmanEvent, 0)
		if coll.PreScript() != "" {
			pm.Event = append(pm.Event, postmanEvent{
				Listen: "prerequest",
				Script: &postmanScript{
					Type: "text/javascript",
					Exec: strings.Split(coll.PreScript(), "\n"),
				},
			})
		}
		if coll.PostScript() != "" {
			pm.Event = append(pm.Event, postmanEvent{
				Listen: "test",
				Script: &postmanScript{
					Type: "text/javascript",
					Exec: strings.Split(coll.PostScript(), "\n"),
				},
			})
		}
	}

	// Export root-level requests
	for _, req := range coll.Requests() {
		pm.Item = append(pm.Item, p.convertRequest(req))
	}

	// Export folders
	for _, folder := range coll.Folders() {
		pm.Item = append(pm.Item, p.convertFolder(folder))
	}

	return json.MarshalIndent(pm, "", "  ")
}

func (p *PostmanExporter) convertFolder(folder *core.Folder) postmanItem {
	item := postmanItem{
		Name:        folder.Name(),
		Description: folder.Description(),
		Item:        make([]postmanItem, 0),
	}

	// Add requests
	for _, req := range folder.Requests() {
		item.Item = append(item.Item, p.convertRequest(req))
	}

	// Add subfolders
	for _, sf := range folder.Folders() {
		item.Item = append(item.Item, p.convertFolder(sf))
	}

	return item
}

func (p *PostmanExporter) convertRequest(req *core.RequestDefinition) postmanItem {
	item := postmanItem{
		Name: req.Name(),
		Request: &postmanRequest{
			Method:      req.Method(),
			Description: req.Description(),
			URL:         req.URL(),
			Header:      make([]postmanHeader, 0),
		},
	}

	// Convert headers
	for key, value := range req.Headers() {
		item.Request.Header = append(item.Request.Header, postmanHeader{
			Key:   key,
			Value: value,
		})
	}

	// Convert body
	body := req.Body()
	if body != "" {
		item.Request.Body = &postmanBody{
			Mode: "raw",
			Raw:  body,
		}

		// Detect JSON and set options
		if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
			item.Request.Body.Options = &postmanBodyOptions{
				Raw: struct {
					Language string `json:"language,omitempty"`
				}{
					Language: "json",
				},
			}
		}
	}

	// Convert auth
	if req.Auth() != nil && req.Auth().Type != "" {
		item.Request.Auth = p.convertAuth(*req.Auth())
	}

	// Convert scripts
	if req.PreScript() != "" || req.PostScript() != "" {
		item.Event = make([]postmanEvent, 0)
		if req.PreScript() != "" {
			item.Event = append(item.Event, postmanEvent{
				Listen: "prerequest",
				Script: &postmanScript{
					Type: "text/javascript",
					Exec: strings.Split(req.PreScript(), "\n"),
				},
			})
		}
		if req.PostScript() != "" {
			item.Event = append(item.Event, postmanEvent{
				Listen: "test",
				Script: &postmanScript{
					Type: "text/javascript",
					Exec: strings.Split(req.PostScript(), "\n"),
				},
			})
		}
	}

	return item
}

func (p *PostmanExporter) convertAuth(auth core.AuthConfig) *postmanAuth {
	pm := &postmanAuth{
		Type: auth.Type,
	}

	switch auth.Type {
	case "bearer":
		pm.Bearer = []postmanAuthItem{
			{Key: "token", Value: auth.Token, Type: "string"},
		}
	case "basic":
		pm.Basic = []postmanAuthItem{
			{Key: "username", Value: auth.Username, Type: "string"},
			{Key: "password", Value: auth.Password, Type: "string"},
		}
	case "apikey":
		pm.APIKey = []postmanAuthItem{
			{Key: "key", Value: auth.Key, Type: "string"},
			{Key: "value", Value: auth.Value, Type: "string"},
			{Key: "in", Value: auth.In, Type: "string"},
		}
	}

	return pm
}

// Postman format structures for export

type postmanCollection struct {
	Info     postmanInfo    `json:"info"`
	Item     []postmanItem  `json:"item"`
	Event    []postmanEvent `json:"event,omitempty"`
	Variable []postmanVar   `json:"variable,omitempty"`
	Auth     *postmanAuth   `json:"auth,omitempty"`
}

type postmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

type postmanItem struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Item        []postmanItem   `json:"item,omitempty"`
	Request     *postmanRequest `json:"request,omitempty"`
	Event       []postmanEvent  `json:"event,omitempty"`
}

type postmanRequest struct {
	Method      string          `json:"method"`
	Header      []postmanHeader `json:"header"`
	Body        *postmanBody    `json:"body,omitempty"`
	URL         string          `json:"url"`
	Auth        *postmanAuth    `json:"auth,omitempty"`
	Description string          `json:"description,omitempty"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode    string              `json:"mode"`
	Raw     string              `json:"raw,omitempty"`
	Options *postmanBodyOptions `json:"options,omitempty"`
}

type postmanBodyOptions struct {
	Raw struct {
		Language string `json:"language,omitempty"`
	} `json:"raw,omitempty"`
}

type postmanAuth struct {
	Type   string            `json:"type"`
	Bearer []postmanAuthItem `json:"bearer,omitempty"`
	Basic  []postmanAuthItem `json:"basic,omitempty"`
	APIKey []postmanAuthItem `json:"apikey,omitempty"`
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
	Type string   `json:"type,omitempty"`
	Exec []string `json:"exec,omitempty"`
}

type postmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Verify PostmanExporter implements Exporter interface
var _ Exporter = (*PostmanExporter)(nil)
