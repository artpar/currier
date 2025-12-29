package cookies

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

// mockStore implements Store for testing
type mockStore struct {
	mu       sync.Mutex
	cookies  map[string]*Cookie // key: domain/path/name
	closed   bool
	setErr   error
	listErr  error
	clearErr error
}

func newMockStore() *mockStore {
	return &mockStore{
		cookies: make(map[string]*Cookie),
	}
}

func (m *mockStore) key(domain, path, name string) string {
	return domain + "|" + path + "|" + name
}

func (m *mockStore) Set(ctx context.Context, cookie *Cookie) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrStoreClosed
	}
	if m.setErr != nil {
		return m.setErr
	}
	m.cookies[m.key(cookie.Domain, cookie.Path, cookie.Name)] = cookie
	return nil
}

func (m *mockStore) Get(ctx context.Context, domain, path, name string) (*Cookie, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrStoreClosed
	}
	c, ok := m.cookies[m.key(domain, path, name)]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

func (m *mockStore) List(ctx context.Context, opts QueryOptions) ([]*Cookie, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrStoreClosed
	}
	if m.listErr != nil {
		return nil, m.listErr
	}

	var result []*Cookie
	for _, c := range m.cookies {
		// Filter by domain
		if opts.Domain != "" && c.Domain != opts.Domain {
			continue
		}
		// Filter by path
		if opts.Path != "" && c.Path != opts.Path {
			continue
		}
		// Filter by name
		if opts.Name != "" && c.Name != opts.Name {
			continue
		}
		// Filter expired
		if !opts.IncludeExpired && c.IsExpired() {
			continue
		}
		result = append(result, c)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}

func (m *mockStore) Delete(ctx context.Context, domain, path, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrStoreClosed
	}
	delete(m.cookies, m.key(domain, path, name))
	return nil
}

func (m *mockStore) DeleteByDomain(ctx context.Context, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrStoreClosed
	}
	for k, c := range m.cookies {
		if c.Domain == domain {
			delete(m.cookies, k)
		}
	}
	return nil
}

func (m *mockStore) DeleteExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, ErrStoreClosed
	}
	var count int64
	for k, c := range m.cookies {
		if c.IsExpired() {
			delete(m.cookies, k)
			count++
		}
	}
	return count, nil
}

func (m *mockStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrStoreClosed
	}
	if m.clearErr != nil {
		return m.clearErr
	}
	m.cookies = make(map[string]*Cookie)
	return nil
}

func (m *mockStore) Count(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, ErrStoreClosed
	}
	return int64(len(m.cookies)), nil
}

func (m *mockStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func TestNewPersistentJar(t *testing.T) {
	t.Run("creates jar with empty store", func(t *testing.T) {
		store := newMockStore()
		jar, err := NewPersistentJar(store)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if jar == nil {
			t.Fatal("expected jar to be created")
		}
		if jar.store != store {
			t.Error("expected store to be set")
		}
	})

	t.Run("loads existing cookies from store", func(t *testing.T) {
		store := newMockStore()
		store.cookies["example.com|/|session"] = &Cookie{
			Domain:  "example.com",
			Path:    "/",
			Name:    "session",
			Value:   "abc123",
			Expires: time.Now().Add(time.Hour),
		}

		jar, err := NewPersistentJar(store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u, _ := url.Parse("https://example.com/")
		cookies := jar.Cookies(u)

		if len(cookies) != 1 {
			t.Errorf("expected 1 cookie, got %d", len(cookies))
		}
		if cookies[0].Name != "session" {
			t.Errorf("expected cookie name 'session', got %s", cookies[0].Name)
		}
	})
}

func TestPersistentJar_SetCookies(t *testing.T) {
	t.Run("sets cookies in memory and store", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/api")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "token", Value: "xyz123", Path: "/api"},
		})

		// Check in-memory jar
		cookies := jar.Cookies(u)
		if len(cookies) != 1 {
			t.Errorf("expected 1 cookie in memory, got %d", len(cookies))
		}

		// Check persistence store
		count, _ := store.Count(context.Background())
		if count != 1 {
			t.Errorf("expected 1 cookie in store, got %d", count)
		}
	})

	t.Run("handles cookie deletion via MaxAge", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/")

		// First set a cookie
		jar.SetCookies(u, []*http.Cookie{
			{Name: "session", Value: "abc", Path: "/"},
		})

		count, _ := store.Count(context.Background())
		if count != 1 {
			t.Errorf("expected 1 cookie after set, got %d", count)
		}

		// Delete via MaxAge < 0
		jar.SetCookies(u, []*http.Cookie{
			{Name: "session", Value: "", Path: "/", MaxAge: -1},
		})

		count, _ = store.Count(context.Background())
		if count != 0 {
			t.Errorf("expected 0 cookies after delete, got %d", count)
		}
	})
}

func TestPersistentJar_Cookies(t *testing.T) {
	t.Run("returns cookies for URL", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/api")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "token", Value: "abc"},
			{Name: "session", Value: "xyz"},
		})

		cookies := jar.Cookies(u)
		if len(cookies) != 2 {
			t.Errorf("expected 2 cookies, got %d", len(cookies))
		}
	})

	t.Run("returns empty for unknown domain", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://unknown.com/")
		cookies := jar.Cookies(u)

		if len(cookies) != 0 {
			t.Errorf("expected 0 cookies, got %d", len(cookies))
		}
	})
}

func TestPersistentJar_Clear(t *testing.T) {
	t.Run("clears all cookies", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "token", Value: "abc"},
		})

		err := jar.Clear()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cookies := jar.Cookies(u)
		if len(cookies) != 0 {
			t.Errorf("expected 0 cookies after clear, got %d", len(cookies))
		}

		count, _ := store.Count(context.Background())
		if count != 0 {
			t.Errorf("expected 0 cookies in store after clear, got %d", count)
		}
	})
}

func TestPersistentJar_ClearDomain(t *testing.T) {
	t.Run("clears cookies for specific domain", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u1, _ := url.Parse("https://example.com/")
		u2, _ := url.Parse("https://other.com/")

		jar.SetCookies(u1, []*http.Cookie{{Name: "token", Value: "abc"}})
		jar.SetCookies(u2, []*http.Cookie{{Name: "session", Value: "xyz"}})

		err := jar.ClearDomain("example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// example.com cookies should be gone
		cookies1 := jar.Cookies(u1)
		if len(cookies1) != 0 {
			t.Errorf("expected 0 cookies for example.com, got %d", len(cookies1))
		}

		// other.com cookies should remain
		cookies2 := jar.Cookies(u2)
		if len(cookies2) != 1 {
			t.Errorf("expected 1 cookie for other.com, got %d", len(cookies2))
		}
	})
}

func TestPersistentJar_Cleanup(t *testing.T) {
	t.Run("removes expired cookies", func(t *testing.T) {
		store := newMockStore()

		// Add an expired cookie directly to store
		store.cookies["example.com|/|expired"] = &Cookie{
			Domain:  "example.com",
			Path:    "/",
			Name:    "expired",
			Value:   "old",
			Expires: time.Now().Add(-time.Hour),
		}
		// Add a valid cookie
		store.cookies["example.com|/|valid"] = &Cookie{
			Domain:  "example.com",
			Path:    "/",
			Name:    "valid",
			Value:   "new",
			Expires: time.Now().Add(time.Hour),
		}

		jar, _ := NewPersistentJar(store)
		count, err := jar.Cleanup()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 expired cookie removed, got %d", count)
		}

		storeCount, _ := store.Count(context.Background())
		if storeCount != 1 {
			t.Errorf("expected 1 cookie remaining in store, got %d", storeCount)
		}
	})
}

func TestPersistentJar_Count(t *testing.T) {
	t.Run("returns cookie count", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "a", Value: "1"},
			{Name: "b", Value: "2"},
			{Name: "c", Value: "3"},
		})

		count, err := jar.Count()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})
}

func TestPersistentJar_ListAll(t *testing.T) {
	t.Run("returns all cookies", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/")
		jar.SetCookies(u, []*http.Cookie{
			{Name: "token", Value: "abc"},
			{Name: "session", Value: "xyz"},
		})

		cookies, err := jar.ListAll()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cookies) != 2 {
			t.Errorf("expected 2 cookies, got %d", len(cookies))
		}
	})
}

func TestPersistentJar_Store(t *testing.T) {
	t.Run("returns underlying store", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		if jar.Store() != store {
			t.Error("expected Store() to return the underlying store")
		}
	})
}

func TestPersistentJar_Concurrency(t *testing.T) {
	t.Run("handles concurrent access", func(t *testing.T) {
		store := newMockStore()
		jar, _ := NewPersistentJar(store)

		u, _ := url.Parse("https://example.com/")

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				jar.SetCookies(u, []*http.Cookie{
					{Name: "token", Value: "value"},
				})
				jar.Cookies(u)
			}(i)
		}
		wg.Wait()

		// Should not panic and jar should still work
		cookies := jar.Cookies(u)
		if len(cookies) == 0 {
			t.Error("expected cookies after concurrent access")
		}
	})
}
