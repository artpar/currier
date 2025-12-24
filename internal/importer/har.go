package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/artpar/currier/internal/core"
)

// HARImporter imports HTTP Archive (HAR) format files.
type HARImporter struct{}

// NewHARImporter creates a new HAR importer.
func NewHARImporter() *HARImporter {
	return &HARImporter{}
}

func (h *HARImporter) Name() string {
	return "HTTP Archive (HAR)"
}

func (h *HARImporter) Format() Format {
	return FormatHAR
}

func (h *HARImporter) FileExtensions() []string {
	return []string{".har"}
}

func (h *HARImporter) DetectFormat(content []byte) bool {
	var check struct {
		Log struct {
			Version string `json:"version"`
			Creator struct {
				Name string `json:"name"`
			} `json:"creator"`
		} `json:"log"`
	}

	if err := json.Unmarshal(content, &check); err != nil {
		return false
	}

	// HAR files have a log object with version and creator
	return check.Log.Version != "" || check.Log.Creator.Name != ""
}

func (h *HARImporter) Import(ctx context.Context, content []byte) (*core.Collection, error) {
	var har harFile
	if err := json.Unmarshal(content, &har); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseError, err)
	}

	// Determine collection name from creator
	collName := "HAR Import"
	if har.Log.Creator.Name != "" {
		collName = fmt.Sprintf("HAR from %s", har.Log.Creator.Name)
	}

	coll := core.NewCollection(collName)
	coll.SetVersion(har.Log.Version)

	// Group requests by domain
	domainFolders := make(map[string]*core.Folder)

	for i, entry := range har.Log.Entries {
		req := entry.Request

		// Parse URL to get domain
		parsedURL, err := url.Parse(req.URL)
		if err != nil {
			continue
		}
		domain := parsedURL.Host

		// Get or create folder for domain
		folder, exists := domainFolders[domain]
		if !exists {
			folder = coll.AddFolder(domain)
			domainFolders[domain] = folder
		}

		// Create request name from path
		name := generateHARRequestName(parsedURL, i)

		// Create request definition
		reqDef := core.NewRequestDefinition(name, req.Method, req.URL)

		// Add headers (skip pseudo-headers like :authority, :method, etc.)
		for _, header := range req.Headers {
			if !strings.HasPrefix(header.Name, ":") {
				reqDef.SetHeader(header.Name, header.Value)
			}
		}

		// Add cookies as header if present
		if len(req.Cookies) > 0 {
			var cookieParts []string
			for _, cookie := range req.Cookies {
				cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
			}
			reqDef.SetHeader("Cookie", strings.Join(cookieParts, "; "))
		}

		// Add body
		if req.PostData != nil && req.PostData.Text != "" {
			reqDef.SetBody(req.PostData.Text)
		}

		folder.AddRequest(reqDef)
	}

	return coll, nil
}

func generateHARRequestName(u *url.URL, index int) string {
	path := u.Path
	if path == "" || path == "/" {
		return fmt.Sprintf("Request %d", index+1)
	}

	// Get last segment of path
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		// Remove query params or extension
		if idx := strings.Index(lastSegment, "?"); idx > 0 {
			lastSegment = lastSegment[:idx]
		}
		if lastSegment != "" {
			return lastSegment
		}
	}

	return fmt.Sprintf("Request %d", index+1)
}

// HAR format structures (HTTP Archive 1.2)

type harFile struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Creator harCreator `json:"creator"`
	Browser *harBrowser `json:"browser,omitempty"`
	Pages   []harPage  `json:"pages,omitempty"`
	Entries []harEntry `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harBrowser struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harPage struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	StartedDateTime string `json:"startedDateTime"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
	Cache           harCache    `json:"cache"`
	Timings         harTimings  `json:"timings"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
	Connection      string      `json:"connection,omitempty"`
	Pageref         string      `json:"pageref,omitempty"`
}

type harRequest struct {
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	HTTPVersion string          `json:"httpVersion"`
	Headers     []harNameValue  `json:"headers"`
	QueryString []harNameValue  `json:"queryString"`
	Cookies     []harCookie     `json:"cookies"`
	HeadersSize int             `json:"headersSize"`
	BodySize    int             `json:"bodySize"`
	PostData    *harPostData    `json:"postData,omitempty"`
}

type harResponse struct {
	Status      int            `json:"status"`
	StatusText  string         `json:"statusText"`
	HTTPVersion string         `json:"httpVersion"`
	Headers     []harNameValue `json:"headers"`
	Cookies     []harCookie    `json:"cookies"`
	Content     harContent     `json:"content"`
	RedirectURL string         `json:"redirectURL"`
	HeadersSize int            `json:"headersSize"`
	BodySize    int            `json:"bodySize"`
}

type harNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Expires  string `json:"expires,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
}

type harPostData struct {
	MimeType string         `json:"mimeType"`
	Text     string         `json:"text,omitempty"`
	Params   []harPostParam `json:"params,omitempty"`
}

type harPostParam struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

type harContent struct {
	Size        int    `json:"size"`
	Compression int    `json:"compression,omitempty"`
	MimeType    string `json:"mimeType"`
	Text        string `json:"text,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

type harCache struct {
	BeforeRequest *harCacheEntry `json:"beforeRequest,omitempty"`
	AfterRequest  *harCacheEntry `json:"afterRequest,omitempty"`
}

type harCacheEntry struct {
	Expires    string `json:"expires,omitempty"`
	LastAccess string `json:"lastAccess"`
	ETag       string `json:"eTag,omitempty"`
	HitCount   int    `json:"hitCount"`
}

type harTimings struct {
	Blocked float64 `json:"blocked"`
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
	SSL     float64 `json:"ssl"`
}

// Verify HARImporter implements Importer interface
var _ Importer = (*HARImporter)(nil)
