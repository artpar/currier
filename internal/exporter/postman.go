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
			Header:      make([]postmanHeader, 0),
		},
	}

	// Convert URL with query parameters
	item.Request.URL = p.convertURL(req)

	// Convert headers
	for key, value := range req.Headers() {
		item.Request.Header = append(item.Request.Header, postmanHeader{
			Key:   key,
			Value: value,
		})
	}

	// Convert body based on body type
	bodyType := req.BodyType()
	bodyContent := req.Body()

	switch bodyType {
	case "form":
		// Form-data body
		formFields := req.FormFields()
		if len(formFields) > 0 {
			item.Request.Body = &postmanBody{
				Mode:     "formdata",
				FormData: make([]postmanFormData, 0, len(formFields)),
			}
			for _, field := range formFields {
				fd := postmanFormData{
					Key:   field.Key,
					Value: field.Value,
				}
				if field.IsFile {
					fd.Type = "file"
					fd.Src = field.FilePath
				} else {
					fd.Type = "text"
				}
				item.Request.Body.FormData = append(item.Request.Body.FormData, fd)
			}
		}
	case "urlencoded":
		// URL-encoded body - parse from body content
		if bodyContent != "" {
			item.Request.Body = &postmanBody{
				Mode:       "urlencoded",
				URLEncoded: make([]postmanURLEncoded, 0),
			}
			pairs := strings.Split(bodyContent, "&")
			for _, pair := range pairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					item.Request.Body.URLEncoded = append(item.Request.Body.URLEncoded, postmanURLEncoded{
						Key:   kv[0],
						Value: kv[1],
					})
				} else if len(kv) == 1 && kv[0] != "" {
					item.Request.Body.URLEncoded = append(item.Request.Body.URLEncoded, postmanURLEncoded{
						Key:   kv[0],
						Value: "",
					})
				}
			}
		}
	default:
		// Raw body (default)
		if bodyContent != "" {
			item.Request.Body = &postmanBody{
				Mode: "raw",
				Raw:  bodyContent,
			}

			// Detect JSON and set options
			if strings.HasPrefix(strings.TrimSpace(bodyContent), "{") || strings.HasPrefix(strings.TrimSpace(bodyContent), "[") {
				item.Request.Body.Options = &postmanBodyOptions{
					Raw: struct {
						Language string `json:"language,omitempty"`
					}{
						Language: "json",
					},
				}
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

func (p *PostmanExporter) convertURL(req *core.RequestDefinition) interface{} {
	queryParams := req.QueryParams()

	// If no query params, return simple string URL
	if len(queryParams) == 0 {
		return req.URL()
	}

	// Build URL object with query parameters
	urlObj := postmanURLObject{
		Raw:   req.FullURL(),
		Query: make([]postmanQueryParam, 0, len(queryParams)),
	}

	for key, value := range queryParams {
		urlObj.Query = append(urlObj.Query, postmanQueryParam{
			Key:   key,
			Value: value,
		})
	}

	return urlObj
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
	case "oauth2":
		if auth.OAuth2 != nil {
			pm.OAuth2 = []postmanAuthItem{
				{Key: "grant_type", Value: string(auth.OAuth2.GrantType), Type: "string"},
				{Key: "accessToken", Value: auth.OAuth2.AccessToken, Type: "string"},
				{Key: "tokenType", Value: auth.OAuth2.TokenType, Type: "string"},
				{Key: "addTokenTo", Value: auth.OAuth2.AddTokenTo, Type: "string"},
			}
			if auth.OAuth2.AuthURL != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "authUrl", Value: auth.OAuth2.AuthURL, Type: "string"})
			}
			if auth.OAuth2.TokenURL != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "accessTokenUrl", Value: auth.OAuth2.TokenURL, Type: "string"})
			}
			if auth.OAuth2.ClientID != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "clientId", Value: auth.OAuth2.ClientID, Type: "string"})
			}
			if auth.OAuth2.ClientSecret != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "clientSecret", Value: auth.OAuth2.ClientSecret, Type: "string"})
			}
			if auth.OAuth2.Scope != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "scope", Value: auth.OAuth2.Scope, Type: "string"})
			}
			if auth.OAuth2.RedirectURI != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "redirect_uri", Value: auth.OAuth2.RedirectURI, Type: "string"})
			}
			if auth.OAuth2.RefreshToken != "" {
				pm.OAuth2 = append(pm.OAuth2, postmanAuthItem{Key: "refreshToken", Value: auth.OAuth2.RefreshToken, Type: "string"})
			}
		}
	case "awsv4":
		if auth.AWS != nil {
			pm.AWSv4 = []postmanAuthItem{
				{Key: "accessKey", Value: auth.AWS.AccessKeyID, Type: "string"},
				{Key: "secretKey", Value: auth.AWS.SecretAccessKey, Type: "string"},
				{Key: "region", Value: auth.AWS.Region, Type: "string"},
				{Key: "service", Value: auth.AWS.Service, Type: "string"},
			}
			if auth.AWS.SessionToken != "" {
				pm.AWSv4 = append(pm.AWSv4, postmanAuthItem{Key: "sessionToken", Value: auth.AWS.SessionToken, Type: "string"})
			}
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
	URL         interface{}     `json:"url"` // Can be string or postmanURLObject
	Auth        *postmanAuth    `json:"auth,omitempty"`
	Description string          `json:"description,omitempty"`
}

type postmanURLObject struct {
	Raw   string              `json:"raw"`
	Query []postmanQueryParam `json:"query,omitempty"`
}

type postmanQueryParam struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode       string              `json:"mode"`
	Raw        string              `json:"raw,omitempty"`
	Options    *postmanBodyOptions `json:"options,omitempty"`
	FormData   []postmanFormData   `json:"formdata,omitempty"`
	URLEncoded []postmanURLEncoded `json:"urlencoded,omitempty"`
}

type postmanFormData struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Type     string `json:"type,omitempty"` // text, file
	Src      string `json:"src,omitempty"`  // for files
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanURLEncoded struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
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
	OAuth2 []postmanAuthItem `json:"oauth2,omitempty"`
	AWSv4  []postmanAuthItem `json:"awsv4,omitempty"`
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
