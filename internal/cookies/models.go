package cookies

import (
	"net/http"
	"net/url"
	"time"
)

// Cookie represents a stored cookie with all attributes.
type Cookie struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	Secure    bool      `json:"secure"`
	HttpOnly  bool      `json:"http_only"`
	SameSite  string    `json:"same_site"`
	Expires   time.Time `json:"expires"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsExpired returns true if the cookie has expired.
func (c *Cookie) IsExpired() bool {
	if c.Expires.IsZero() {
		return false // Session cookie, never expires
	}
	return time.Now().After(c.Expires)
}

// IsSession returns true if this is a session cookie (no expiration).
func (c *Cookie) IsSession() bool {
	return c.Expires.IsZero()
}

// ToHTTPCookie converts to standard http.Cookie.
func (c *Cookie) ToHTTPCookie() *http.Cookie {
	sameSite := http.SameSiteDefaultMode
	switch c.SameSite {
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	return &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HttpOnly,
		SameSite: sameSite,
		Expires:  c.Expires,
	}
}

// FromHTTPCookie creates a Cookie from http.Cookie and URL.
func FromHTTPCookie(u *url.URL, hc *http.Cookie) *Cookie {
	domain := hc.Domain
	if domain == "" {
		domain = u.Hostname()
	}
	// Remove leading dot if present (normalize)
	if len(domain) > 0 && domain[0] == '.' {
		domain = domain[1:]
	}

	path := hc.Path
	if path == "" {
		path = "/"
	}

	sameSite := ""
	switch hc.SameSite {
	case http.SameSiteLaxMode:
		sameSite = "lax"
	case http.SameSiteStrictMode:
		sameSite = "strict"
	case http.SameSiteNoneMode:
		sameSite = "none"
	}

	// Calculate expiration
	expires := hc.Expires
	if hc.MaxAge > 0 {
		expires = time.Now().Add(time.Duration(hc.MaxAge) * time.Second)
	} else if hc.MaxAge < 0 {
		// MaxAge < 0 means delete cookie immediately
		expires = time.Unix(0, 0)
	}

	now := time.Now()
	return &Cookie{
		Domain:    domain,
		Path:      path,
		Name:      hc.Name,
		Value:     hc.Value,
		Secure:    hc.Secure,
		HttpOnly:  hc.HttpOnly,
		SameSite:  sameSite,
		Expires:   expires,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// QueryOptions for filtering cookies.
type QueryOptions struct {
	Domain         string // Filter by domain
	Path           string // Filter by path
	Name           string // Filter by cookie name
	IncludeExpired bool   // Include expired cookies
	Limit          int    // Max results (0 = no limit)
}
