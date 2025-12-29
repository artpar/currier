package cookies

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestCookie_IsExpired(t *testing.T) {
	t.Run("returns false for session cookie (zero expiry)", func(t *testing.T) {
		c := &Cookie{
			Name:    "session",
			Value:   "abc123",
			Expires: time.Time{},
		}
		if c.IsExpired() {
			t.Error("expected session cookie to not be expired")
		}
	})

	t.Run("returns false for future expiry", func(t *testing.T) {
		c := &Cookie{
			Name:    "token",
			Value:   "xyz",
			Expires: time.Now().Add(24 * time.Hour),
		}
		if c.IsExpired() {
			t.Error("expected future cookie to not be expired")
		}
	})

	t.Run("returns true for past expiry", func(t *testing.T) {
		c := &Cookie{
			Name:    "old",
			Value:   "stale",
			Expires: time.Now().Add(-24 * time.Hour),
		}
		if !c.IsExpired() {
			t.Error("expected past cookie to be expired")
		}
	})
}

func TestCookie_IsSession(t *testing.T) {
	t.Run("returns true for zero expiry", func(t *testing.T) {
		c := &Cookie{
			Name:    "session",
			Expires: time.Time{},
		}
		if !c.IsSession() {
			t.Error("expected cookie with zero expiry to be session cookie")
		}
	})

	t.Run("returns false for non-zero expiry", func(t *testing.T) {
		c := &Cookie{
			Name:    "persistent",
			Expires: time.Now().Add(time.Hour),
		}
		if c.IsSession() {
			t.Error("expected cookie with expiry to not be session cookie")
		}
	})
}

func TestCookie_ToHTTPCookie(t *testing.T) {
	t.Run("converts basic cookie", func(t *testing.T) {
		expires := time.Now().Add(time.Hour)
		c := &Cookie{
			Name:     "test",
			Value:    "value123",
			Domain:   "example.com",
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			Expires:  expires,
		}

		hc := c.ToHTTPCookie()

		if hc.Name != "test" {
			t.Errorf("expected name 'test', got %s", hc.Name)
		}
		if hc.Value != "value123" {
			t.Errorf("expected value 'value123', got %s", hc.Value)
		}
		if hc.Domain != "example.com" {
			t.Errorf("expected domain 'example.com', got %s", hc.Domain)
		}
		if hc.Path != "/api" {
			t.Errorf("expected path '/api', got %s", hc.Path)
		}
		if !hc.Secure {
			t.Error("expected Secure to be true")
		}
		if !hc.HttpOnly {
			t.Error("expected HttpOnly to be true")
		}
	})

	t.Run("converts SameSite lax", func(t *testing.T) {
		c := &Cookie{Name: "test", SameSite: "lax"}
		hc := c.ToHTTPCookie()
		if hc.SameSite != http.SameSiteLaxMode {
			t.Errorf("expected SameSiteLaxMode, got %v", hc.SameSite)
		}
	})

	t.Run("converts SameSite strict", func(t *testing.T) {
		c := &Cookie{Name: "test", SameSite: "strict"}
		hc := c.ToHTTPCookie()
		if hc.SameSite != http.SameSiteStrictMode {
			t.Errorf("expected SameSiteStrictMode, got %v", hc.SameSite)
		}
	})

	t.Run("converts SameSite none", func(t *testing.T) {
		c := &Cookie{Name: "test", SameSite: "none"}
		hc := c.ToHTTPCookie()
		if hc.SameSite != http.SameSiteNoneMode {
			t.Errorf("expected SameSiteNoneMode, got %v", hc.SameSite)
		}
	})

	t.Run("defaults SameSite to default mode", func(t *testing.T) {
		c := &Cookie{Name: "test", SameSite: ""}
		hc := c.ToHTTPCookie()
		if hc.SameSite != http.SameSiteDefaultMode {
			t.Errorf("expected SameSiteDefaultMode, got %v", hc.SameSite)
		}
	})
}

func TestFromHTTPCookie(t *testing.T) {
	t.Run("creates cookie from http.Cookie", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/api")
		expires := time.Now().Add(time.Hour)
		hc := &http.Cookie{
			Name:     "session",
			Value:    "abc123",
			Path:     "/api",
			Secure:   true,
			HttpOnly: true,
			Expires:  expires,
			SameSite: http.SameSiteLaxMode,
		}

		c := FromHTTPCookie(u, hc)

		if c.Name != "session" {
			t.Errorf("expected name 'session', got %s", c.Name)
		}
		if c.Value != "abc123" {
			t.Errorf("expected value 'abc123', got %s", c.Value)
		}
		if c.Domain != "example.com" {
			t.Errorf("expected domain 'example.com', got %s", c.Domain)
		}
		if c.Path != "/api" {
			t.Errorf("expected path '/api', got %s", c.Path)
		}
		if !c.Secure {
			t.Error("expected Secure to be true")
		}
		if !c.HttpOnly {
			t.Error("expected HttpOnly to be true")
		}
		if c.SameSite != "lax" {
			t.Errorf("expected SameSite 'lax', got %s", c.SameSite)
		}
	})

	t.Run("uses URL hostname when domain not set", func(t *testing.T) {
		u, _ := url.Parse("https://api.example.com/")
		hc := &http.Cookie{
			Name:  "token",
			Value: "xyz",
		}

		c := FromHTTPCookie(u, hc)

		if c.Domain != "api.example.com" {
			t.Errorf("expected domain 'api.example.com', got %s", c.Domain)
		}
	})

	t.Run("removes leading dot from domain", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{
			Name:   "token",
			Value:  "xyz",
			Domain: ".example.com",
		}

		c := FromHTTPCookie(u, hc)

		if c.Domain != "example.com" {
			t.Errorf("expected domain 'example.com', got %s", c.Domain)
		}
	})

	t.Run("defaults path to /", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/api/v1")
		hc := &http.Cookie{
			Name:  "token",
			Value: "xyz",
			Path:  "",
		}

		c := FromHTTPCookie(u, hc)

		if c.Path != "/" {
			t.Errorf("expected path '/', got %s", c.Path)
		}
	})

	t.Run("converts MaxAge to expiry time", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{
			Name:   "token",
			Value:  "xyz",
			MaxAge: 3600, // 1 hour
		}

		before := time.Now()
		c := FromHTTPCookie(u, hc)
		after := time.Now()

		expectedMin := before.Add(3600 * time.Second)
		expectedMax := after.Add(3600 * time.Second)

		if c.Expires.Before(expectedMin) || c.Expires.After(expectedMax) {
			t.Errorf("expected expires around 1 hour from now, got %v", c.Expires)
		}
	})

	t.Run("handles negative MaxAge (delete cookie)", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{
			Name:   "token",
			Value:  "xyz",
			MaxAge: -1,
		}

		c := FromHTTPCookie(u, hc)

		if !c.Expires.Equal(time.Unix(0, 0)) {
			t.Errorf("expected Unix epoch for deleted cookie, got %v", c.Expires)
		}
	})

	t.Run("converts SameSite strict", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{
			Name:     "token",
			Value:    "xyz",
			SameSite: http.SameSiteStrictMode,
		}

		c := FromHTTPCookie(u, hc)

		if c.SameSite != "strict" {
			t.Errorf("expected SameSite 'strict', got %s", c.SameSite)
		}
	})

	t.Run("converts SameSite none", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{
			Name:     "token",
			Value:    "xyz",
			SameSite: http.SameSiteNoneMode,
		}

		c := FromHTTPCookie(u, hc)

		if c.SameSite != "none" {
			t.Errorf("expected SameSite 'none', got %s", c.SameSite)
		}
	})

	t.Run("sets CreatedAt and UpdatedAt", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/")
		hc := &http.Cookie{Name: "token", Value: "xyz"}

		before := time.Now()
		c := FromHTTPCookie(u, hc)
		after := time.Now()

		if c.CreatedAt.Before(before) || c.CreatedAt.After(after) {
			t.Error("expected CreatedAt to be set to current time")
		}
		if c.UpdatedAt.Before(before) || c.UpdatedAt.After(after) {
			t.Error("expected UpdatedAt to be set to current time")
		}
	})
}

func TestQueryOptions(t *testing.T) {
	t.Run("struct initializes with defaults", func(t *testing.T) {
		opts := QueryOptions{}

		if opts.Domain != "" {
			t.Error("expected empty domain")
		}
		if opts.Path != "" {
			t.Error("expected empty path")
		}
		if opts.Name != "" {
			t.Error("expected empty name")
		}
		if opts.IncludeExpired {
			t.Error("expected IncludeExpired to be false")
		}
		if opts.Limit != 0 {
			t.Error("expected Limit to be 0")
		}
	})

	t.Run("struct accepts all fields", func(t *testing.T) {
		opts := QueryOptions{
			Domain:         "example.com",
			Path:           "/api",
			Name:           "session",
			IncludeExpired: true,
			Limit:          100,
		}

		if opts.Domain != "example.com" {
			t.Errorf("expected domain 'example.com', got %s", opts.Domain)
		}
		if opts.Path != "/api" {
			t.Errorf("expected path '/api', got %s", opts.Path)
		}
		if opts.Name != "session" {
			t.Errorf("expected name 'session', got %s", opts.Name)
		}
		if !opts.IncludeExpired {
			t.Error("expected IncludeExpired to be true")
		}
		if opts.Limit != 100 {
			t.Errorf("expected Limit 100, got %d", opts.Limit)
		}
	})
}
