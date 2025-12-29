package cookies

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"

	"golang.org/x/net/publicsuffix"
)

// PersistentJar implements http.CookieJar with SQLite persistence.
type PersistentJar struct {
	mu    sync.RWMutex
	jar   *cookiejar.Jar // In-memory jar for standard behavior
	store Store          // Persistence layer
}

// NewPersistentJar creates a new persistent cookie jar.
func NewPersistentJar(store Store) (*PersistentJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	pj := &PersistentJar{
		jar:   jar,
		store: store,
	}

	// Load existing cookies from store
	if err := pj.loadFromStore(); err != nil {
		return nil, err
	}

	return pj, nil
}

// loadFromStore loads all non-expired cookies into memory.
func (pj *PersistentJar) loadFromStore() error {
	ctx := context.Background()
	cookies, err := pj.store.List(ctx, QueryOptions{
		IncludeExpired: false,
	})
	if err != nil {
		return err
	}

	// Group cookies by domain and set them
	byDomain := make(map[string][]*http.Cookie)
	for _, c := range cookies {
		hc := c.ToHTTPCookie()
		byDomain[c.Domain] = append(byDomain[c.Domain], hc)
	}

	for domain, domainCookies := range byDomain {
		u := &url.URL{
			Scheme: "https",
			Host:   domain,
			Path:   "/",
		}
		pj.jar.SetCookies(u, domainCookies)
	}

	return nil
}

// SetCookies implements http.CookieJar.
func (pj *PersistentJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	pj.mu.Lock()
	defer pj.mu.Unlock()

	// Set in memory jar
	pj.jar.SetCookies(u, cookies)

	// Persist to store
	ctx := context.Background()
	for _, hc := range cookies {
		c := FromHTTPCookie(u, hc)

		// Handle cookie deletion (MaxAge < 0)
		if hc.MaxAge < 0 {
			pj.store.Delete(ctx, c.Domain, c.Path, c.Name)
			continue
		}

		pj.store.Set(ctx, c)
	}
}

// Cookies implements http.CookieJar.
func (pj *PersistentJar) Cookies(u *url.URL) []*http.Cookie {
	pj.mu.RLock()
	defer pj.mu.RUnlock()

	return pj.jar.Cookies(u)
}

// Clear removes all cookies from jar and store.
func (pj *PersistentJar) Clear() error {
	pj.mu.Lock()
	defer pj.mu.Unlock()

	// Create new empty jar
	newJar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
	}
	pj.jar = newJar

	// Clear store
	return pj.store.Clear(context.Background())
}

// ClearDomain removes all cookies for a domain.
func (pj *PersistentJar) ClearDomain(domain string) error {
	pj.mu.Lock()
	defer pj.mu.Unlock()

	ctx := context.Background()

	// Delete from store
	if err := pj.store.DeleteByDomain(ctx, domain); err != nil {
		return err
	}

	// Reload jar from store (recreate without the deleted domain)
	newJar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
	}
	pj.jar = newJar

	// Reload remaining cookies
	cookies, err := pj.store.List(ctx, QueryOptions{IncludeExpired: false})
	if err != nil {
		return err
	}

	byDomain := make(map[string][]*http.Cookie)
	for _, c := range cookies {
		hc := c.ToHTTPCookie()
		byDomain[c.Domain] = append(byDomain[c.Domain], hc)
	}

	for d, domainCookies := range byDomain {
		u := &url.URL{
			Scheme: "https",
			Host:   d,
			Path:   "/",
		}
		pj.jar.SetCookies(u, domainCookies)
	}

	return nil
}

// Cleanup removes expired cookies from the store.
func (pj *PersistentJar) Cleanup() (int64, error) {
	pj.mu.Lock()
	defer pj.mu.Unlock()

	return pj.store.DeleteExpired(context.Background())
}

// Count returns the number of stored cookies.
func (pj *PersistentJar) Count() (int64, error) {
	pj.mu.RLock()
	defer pj.mu.RUnlock()

	return pj.store.Count(context.Background())
}

// ListAll returns all stored cookies.
func (pj *PersistentJar) ListAll() ([]*Cookie, error) {
	pj.mu.RLock()
	defer pj.mu.RUnlock()

	return pj.store.List(context.Background(), QueryOptions{
		IncludeExpired: false,
	})
}

// Store returns the underlying store (for closing).
func (pj *PersistentJar) Store() Store {
	return pj.store
}
