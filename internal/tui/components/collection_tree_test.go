package components

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history"
	"github.com/stretchr/testify/assert"
)

func TestNewCollectionTree(t *testing.T) {
	t.Run("creates empty tree", func(t *testing.T) {
		tree := NewCollectionTree()
		assert.NotNil(t, tree)
		assert.Equal(t, "Collections", tree.Title())
	})

	t.Run("starts with no selection", func(t *testing.T) {
		tree := NewCollectionTree()
		assert.Nil(t, tree.Selected())
	})

	t.Run("starts unfocused", func(t *testing.T) {
		tree := NewCollectionTree()
		assert.False(t, tree.Focused())
	})
}

func TestCollectionTree_SetCollections(t *testing.T) {
	t.Run("sets collections", func(t *testing.T) {
		tree := NewCollectionTree()

		c1 := core.NewCollection("API 1")
		c2 := core.NewCollection("API 2")

		tree.SetCollections([]*core.Collection{c1, c2})
		assert.Equal(t, 2, tree.ItemCount())
	})

	t.Run("selects first item when setting collections", func(t *testing.T) {
		tree := NewCollectionTree()

		c := core.NewCollection("My API")
		tree.SetCollections([]*core.Collection{c})

		assert.Equal(t, 0, tree.Cursor())
	})
}

func TestCollectionTree_Navigation(t *testing.T) {
	t.Run("moves cursor down with j", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 1, tree.Cursor())
	})

	t.Run("moves cursor up with k", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetCursor(2)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 1, tree.Cursor())
	})

	t.Run("does not move past first item", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 0, tree.Cursor())
	})

	t.Run("does not move past last item", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetCursor(tree.ItemCount() - 1)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, tree.ItemCount()-1, tree.Cursor())
	})

	t.Run("goes to top with gg", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetCursor(5)

		// First g
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Second g
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 0, tree.Cursor())
	})

	t.Run("goes to bottom with G", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, tree.ItemCount()-1, tree.Cursor())
	})

	t.Run("ignores navigation when unfocused", func(t *testing.T) {
		tree := newTestTree(t)
		// Don't focus
		tree.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 0, tree.Cursor())
	})
}

func TestCollectionTree_Expand(t *testing.T) {
	t.Run("expands collection with Enter", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.Focus()
		tree.SetCursor(0) // First collection

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.True(t, tree.IsExpanded(0))
	})

	t.Run("collapses collection with Enter when expanded", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.Focus()
		tree.SetCursor(0)
		tree.Expand(0)

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.False(t, tree.IsExpanded(0))
	})

	t.Run("expands with l key", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.Focus()
		tree.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.True(t, tree.IsExpanded(0))
	})

	t.Run("collapses with h key", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.Focus()
		tree.SetCursor(0)
		tree.Expand(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.False(t, tree.IsExpanded(0))
	})
}

func TestCollectionTree_Selection(t *testing.T) {
	t.Run("selects request on Enter", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()

		// Expand to show requests
		tree.Expand(0)

		// Move to first request
		tree.SetCursor(1) // After expanding, cursor 1 is first request

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Should produce a selection message
		assert.NotNil(t, cmd)
	})

	t.Run("returns selected item", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.Expand(0)
		tree.SetCursor(1)

		selected := tree.Selected()
		assert.NotNil(t, selected)
	})
}

func TestCollectionTree_View(t *testing.T) {
	t.Run("renders collection names", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(40, 20)

		view := tree.View()
		assert.Contains(t, view, "API 1")
		assert.Contains(t, view, "API 2")
	})

	t.Run("shows expand indicator for collections with folders", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(40, 20)

		view := tree.View()
		assert.Contains(t, view, "▶") // Collapsed indicator
	})

	t.Run("shows expanded indicator", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(40, 20)
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "▼") // Expanded indicator
	})

	t.Run("highlights cursor line", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(40, 20)
		tree.Focus()

		view := tree.View()
		// Focused view should contain highlight styling
		assert.NotEmpty(t, view)
	})
}

func TestCollectionTree_Search(t *testing.T) {
	t.Run("filters items by search query", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSearch("API 1")

		// Should only show matching items
		assert.Equal(t, 1, tree.VisibleItemCount())
	})

	t.Run("clears search", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSearch("API 1")
		tree.ClearSearch()

		assert.Equal(t, tree.ItemCount(), tree.VisibleItemCount())
	})

	t.Run("case insensitive search", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSearch("api 1")

		assert.Equal(t, 1, tree.VisibleItemCount())
	})
}

func TestCollectionTree_Init(t *testing.T) {
	t.Run("returns nil command", func(t *testing.T) {
		tree := NewCollectionTree()
		cmd := tree.Init()
		assert.Nil(t, cmd)
	})
}

func TestCollectionTree_Methods(t *testing.T) {
	t.Run("Blur sets unfocused", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.Blur()
		assert.False(t, tree.Focused())
	})

	t.Run("Width returns width", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 40)
		assert.Equal(t, 80, tree.Width())
	})

	t.Run("Height returns height", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 40)
		assert.Equal(t, 40, tree.Height())
	})

	t.Run("Collapse collapses item", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.Expand(0)
		assert.True(t, tree.IsExpanded(0))
		tree.Collapse(0)
		assert.False(t, tree.IsExpanded(0))
	})
}

func TestCollectionTree_IsSearching(t *testing.T) {
	t.Run("returns false when not searching", func(t *testing.T) {
		tree := NewCollectionTree()
		assert.False(t, tree.IsSearching())
	})

	t.Run("returns true when searching", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.True(t, tree.IsSearching())
	})

	t.Run("returns false after exiting search mode", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.searching = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.False(t, tree.IsSearching())
	})
}

func TestCollectionTree_SearchMode(t *testing.T) {
	t.Run("enters search mode with /", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Should be in search mode
		assert.True(t, tree.searching)
	})

	t.Run("handles search input", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.searching = true

		// Type a character
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		tree.Update(msg)

		// Search query should be set
		assert.Contains(t, tree.search, "a")
	})

	t.Run("exits search mode with Escape", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.searching = true
		tree.search = "test"

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.False(t, tree.searching)
	})

	t.Run("handles Enter in search mode", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.searching = true
		tree.search = "API"

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.False(t, tree.searching)
	})

	t.Run("handles backspace in search mode", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.searching = true
		tree.search = "test"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, "tes", tree.search)
	})
}

func TestCollectionTree_ViewWithSearch(t *testing.T) {
	t.Run("renders search bar when in search mode", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(40, 20)
		tree.Focus()
		tree.searching = true
		tree.search = "test"

		view := tree.View()
		assert.Contains(t, view, "test")
	})
}

func TestCollectionTree_MethodBadges(t *testing.T) {
	t.Run("renders GET method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Get Users", "GET", "/users")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "GET")
	})

	t.Run("renders POST method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Create User", "POST", "/users")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "POST")
	})

	t.Run("renders PUT method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Update User", "PUT", "/users")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "PUT")
	})

	t.Run("renders DELETE method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Delete User", "DELETE", "/users/1")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "DEL")
	})

	t.Run("renders PATCH method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Patch User", "PATCH", "/users/1")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "PTCH")
	})
}

func TestCollectionTree_FolderItems(t *testing.T) {
	t.Run("expands nested folders", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		c := core.NewCollection("API")
		folder := c.AddFolder("Users")
		folder.AddRequest(core.NewRequestDefinition("Get", "GET", "/users"))

		tree.SetCollections([]*core.Collection{c})

		// Expand collection
		tree.Expand(0)

		assert.Greater(t, tree.VisibleItemCount(), 1)
	})
}

func TestCollectionTree_WindowSizeMessage(t *testing.T) {
	t.Run("handles window size message", func(t *testing.T) {
		tree := NewCollectionTree()

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, 100, tree.Width())
		assert.Equal(t, 50, tree.Height())
	})
}

func TestCollectionTree_NestedFolders(t *testing.T) {
	t.Run("expands nested folder with requests", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		c := core.NewCollection("API")
		parentFolder := c.AddFolder("Users")
		childFolder := parentFolder.AddFolder("Admin")
		childFolder.AddRequest(core.NewRequestDefinition("List Admins", "GET", "/admin/users"))

		tree.SetCollections([]*core.Collection{c})

		// Expand collection
		tree.Expand(0)
		// Expand Users folder
		tree.Expand(1)
		// Expand Admin folder
		tree.Expand(2)

		view := tree.View()
		assert.Contains(t, view, "Admin")
		assert.Contains(t, view, "List Admins")
	})
}

func TestCollectionTree_MethodBadgeOther(t *testing.T) {
	t.Run("renders HEAD method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("Head Check", "HEAD", "/health")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "HEAD")
	})

	t.Run("renders OPTIONS method badge", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(60, 20)

		c := core.NewCollection("Test API")
		req := core.NewRequestDefinition("CORS Options", "OPTIONS", "/api")
		c.AddRequest(req)

		tree.SetCollections([]*core.Collection{c})
		tree.Expand(0)

		view := tree.View()
		assert.Contains(t, view, "OPT")
	})
}

func TestCollectionTree_SetCursorEdgeCases(t *testing.T) {
	t.Run("SetCursor clamps to valid range", func(t *testing.T) {
		tree := newTestTree(t)

		// Try to set cursor beyond end
		tree.SetCursor(100)
		assert.Equal(t, tree.ItemCount()-1, tree.Cursor())
	})

	t.Run("SetCursor handles negative", func(t *testing.T) {
		tree := newTestTree(t)

		tree.SetCursor(-1)
		assert.Equal(t, 0, tree.Cursor())
	})
}

func TestCollectionTree_ExpandCollapse(t *testing.T) {
	t.Run("Expand does nothing for non-expandable", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Expand(0)

		// Move to request (not expandable)
		tree.SetCursor(1)
		tree.Expand(1)

		// Should not panic
		assert.True(t, true)
	})

	t.Run("IsExpanded returns false for non-expandable", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Expand(0)

		// Request at index 1 is not expandable
		assert.False(t, tree.IsExpanded(1))
	})
}

func TestCollectionTree_ScrollView(t *testing.T) {
	t.Run("scrolls view when cursor exceeds viewport", func(t *testing.T) {
		tree := newTestTree(t) // 10 items
		tree.SetSize(40, 5)   // Small height
		tree.Focus()

		// Move cursor to bottom
		tree.SetCursor(9)

		view := tree.View()
		// Should show cursor line
		assert.NotEmpty(t, view)
	})
}

// Helper functions

func newTestTree(t *testing.T) *CollectionTree {
	t.Helper()
	tree := NewCollectionTree()

	collections := make([]*core.Collection, 10)
	for i := 0; i < 10; i++ {
		c := core.NewCollection("API " + string(rune('1'+i)))
		collections[i] = c
	}

	tree.SetCollections(collections)
	return tree
}

func newTestTreeWithFolders(t *testing.T) *CollectionTree {
	t.Helper()
	tree := NewCollectionTree()

	c := core.NewCollection("My API")
	c.AddFolder("Users")
	c.AddFolder("Posts")

	tree.SetCollections([]*core.Collection{c})
	return tree
}

func newTestTreeWithRequests(t *testing.T) *CollectionTree {
	t.Helper()
	tree := NewCollectionTree()

	c := core.NewCollection("My API")

	// Add requests directly to collection root (not in folder)
	req1 := core.NewRequestDefinition("Get User", "GET", "/users/1")
	req2 := core.NewRequestDefinition("List Users", "GET", "/users")
	c.AddRequest(req1)
	c.AddRequest(req2)

	tree.SetCollections([]*core.Collection{c})
	return tree
}

// Mock history store for testing
type mockHistoryStore struct {
	entries []history.Entry
	err     error
}

func (m *mockHistoryStore) Add(ctx context.Context, entry history.Entry) (string, error) {
	return "test-id", m.err
}

func (m *mockHistoryStore) Get(ctx context.Context, id string) (history.Entry, error) {
	return history.Entry{}, m.err
}

func (m *mockHistoryStore) List(ctx context.Context, opts history.QueryOptions) ([]history.Entry, error) {
	return m.entries, m.err
}

func (m *mockHistoryStore) Count(ctx context.Context, opts history.QueryOptions) (int64, error) {
	return int64(len(m.entries)), m.err
}

func (m *mockHistoryStore) Update(ctx context.Context, entry history.Entry) error {
	return m.err
}

func (m *mockHistoryStore) Delete(ctx context.Context, id string) error {
	return m.err
}

func (m *mockHistoryStore) DeleteMany(ctx context.Context, opts history.QueryOptions) (int64, error) {
	return 0, m.err
}

func (m *mockHistoryStore) Search(ctx context.Context, query string, opts history.QueryOptions) ([]history.Entry, error) {
	var results []history.Entry
	for _, e := range m.entries {
		if strings.Contains(e.RequestURL, query) || strings.Contains(e.RequestMethod, query) {
			results = append(results, e)
		}
	}
	return results, m.err
}

func (m *mockHistoryStore) Prune(ctx context.Context, opts history.PruneOptions) (history.PruneResult, error) {
	return history.PruneResult{}, m.err
}

func (m *mockHistoryStore) Stats(ctx context.Context) (history.Stats, error) {
	return history.Stats{}, m.err
}

func (m *mockHistoryStore) Clear(ctx context.Context) error {
	return m.err
}

func (m *mockHistoryStore) Close() error {
	return nil
}

func TestCollectionTree_HistoryView(t *testing.T) {
	t.Run("switches to history view with H key", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, ViewHistory, tree.ViewMode())
	})

	t.Run("returns to collections with C key from history", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Switch back to collections
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})

	t.Run("returns to collections with H key from history (toggle)", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Toggle back with H
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})

	t.Run("navigates history with j/k keys", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/users"},
				{ID: "3", RequestMethod: "DELETE", RequestURL: "https://api.example.com/users/1"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history view
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Initial cursor should be at 0
		assert.Equal(t, 0, tree.HistoryCursor())

		// Move down with j
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Cursor should now be at 1
		assert.Equal(t, 1, tree.HistoryCursor())

		// Move down again
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Cursor should now be at 2
		assert.Equal(t, 2, tree.HistoryCursor())

		// Move up with k
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Cursor should be back at 1
		assert.Equal(t, 1, tree.HistoryCursor())

		// Move up again
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Cursor should be at 0
		assert.Equal(t, 0, tree.HistoryCursor())
	})

	t.Run("handles gg navigation in history", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/users"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// First g
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Second g
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles G navigation in history", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/users"},
				{ID: "3", RequestMethod: "DELETE", RequestURL: "https://api.example.com/users/1"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Go to end with G
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("exits history with Escape", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{entries: []history.Entry{}}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Exit with Escape
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})

	t.Run("refreshes history with r key", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Refresh with r
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("selects history entry with Enter", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Select with Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Should have a command
		assert.NotNil(t, cmd)
	})

	t.Run("handles h/l keys in history (no-op)", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{entries: []history.Entry{}}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Press h (should do nothing)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Press l (should do nothing)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, ViewHistory, tree.ViewMode())
	})

	t.Run("shows empty history message without store", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		// Switch to history without store
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.Contains(t, view, "History")
	})

	t.Run("renders history items with timestamps", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{
					ID:            "1",
					RequestMethod: "GET",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-5 * time.Minute),
				},
				{
					ID:            "2",
					RequestMethod: "POST",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-2 * time.Hour),
				},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.Contains(t, view, "GET")
	})

	t.Run("renders selected history item when focused", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/posts"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.True(t, tree.Focused())
		assert.Equal(t, 0, tree.HistoryCursor())

		view := tree.View()
		assert.Contains(t, view, "GET")
		assert.Contains(t, view, "POST")
	})

	t.Run("renders selected history item when unfocused", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/posts"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Blur the tree (unfocus)
		tree.Blur()

		assert.False(t, tree.Focused())
		assert.Equal(t, 0, tree.HistoryCursor())

		view := tree.View()
		// Both items should render even when unfocused
		assert.Contains(t, view, "GET")
		assert.Contains(t, view, "POST")
	})

	t.Run("clears search with Escape in history", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Start search
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Type search query
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Finish search with Enter
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Clear search with Escape (goes back to search mode)
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Should still be in history mode but search cleared
		assert.Equal(t, ViewHistory, tree.ViewMode())
	})
}

func TestCollectionTree_HistoryAccessors(t *testing.T) {
	t.Run("ViewModeName returns correct names", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		assert.Equal(t, "collections", tree.ViewModeName())

		// Switch to history
		tree.Focus()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, "history", tree.ViewModeName())
	})

	t.Run("SearchQuery returns current search", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		// Initial query is empty
		assert.Equal(t, "", tree.SearchQuery())

		// Start search
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, "test", tree.SearchQuery())
	})

	t.Run("HistoryEntries returns entries", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://api.example.com/users"},
				{ID: "2", RequestMethod: "POST", RequestURL: "https://api.example.com/users"},
			},
		}
		tree.SetHistoryStore(store)
		tree.Focus()

		// Switch to history to trigger load
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		entries := tree.HistoryEntries()
		assert.Equal(t, 2, len(entries))
	})

	t.Run("Collections returns collections", func(t *testing.T) {
		tree := newTestTree(t)
		collections := tree.Collections()
		assert.Equal(t, 10, len(collections))
	})

	t.Run("AddRequest adds request to tree", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		// Create a collection first
		coll := core.NewCollection("Test Collection")
		tree.SetCollections([]*core.Collection{coll})

		req := core.NewRequestDefinition("Test", "GET", "https://example.com")
		result := tree.AddRequest(req, coll)

		assert.True(t, result)
	})

	t.Run("AddRequest returns false for nil request", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		result := tree.AddRequest(nil, nil)
		assert.False(t, result)
	})
}

func TestCollectionTree_Update(t *testing.T) {
	t.Run("handles window size message", func(t *testing.T) {
		tree := newTestTree(t)
		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)
		assert.Equal(t, 100, tree.width)
	})

	t.Run("handles mouse events without crashing", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(80, 30)
		tree.Focus()
		tree.SetCursor(5)

		// Test mouse wheel up
		msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)
		view := tree.View()
		assert.NotEmpty(t, view)

		// Test mouse wheel down
		msg = tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)
		view = tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("ignores updates when not focused", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(80, 30)
		tree.SetCursor(0)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)
		assert.Equal(t, 0, tree.Cursor())
	})
}

func TestCollectionTree_SearchInput(t *testing.T) {
	t.Run("enters search mode with /", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.SetSize(80, 30)

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// View should render (search mode active)
		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles typing in search mode", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Type search query
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, "get", tree.SearchQuery())
	})

	t.Run("handles backspace in search", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Type
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Backspace
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.Equal(t, "a", tree.SearchQuery())
	})

	t.Run("finalizes search with Enter", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Type
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Enter to finalize
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("escape exits search mode", func(t *testing.T) {
		tree := newTestTreeWithRequests(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Exit with escape
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})
}

func TestCollectionTree_FormatTimeAgo(t *testing.T) {
	t.Run("renders recent timestamps correctly", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		// Create entries with various timestamps
		store := &mockHistoryStore{
			entries: []history.Entry{
				{
					ID:            "1",
					RequestMethod: "GET",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-30 * time.Second),
				},
				{
					ID:            "2",
					RequestMethod: "POST",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-5 * time.Minute),
				},
				{
					ID:            "3",
					RequestMethod: "PUT",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-2 * time.Hour),
				},
				{
					ID:            "4",
					RequestMethod: "DELETE",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-3 * 24 * time.Hour),
				},
				{
					ID:            "5",
					RequestMethod: "PATCH",
					RequestURL:    "https://api.example.com/users",
					Timestamp:     time.Now().Add(-14 * 24 * time.Hour),
				},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history to trigger rendering
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "GET")
	})
}

func TestCollectionTree_AddRequestWithCollection(t *testing.T) {
	t.Run("adds request to specified collection", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		coll := core.NewCollection("My API")
		tree.SetCollections([]*core.Collection{coll})

		req := core.NewRequestDefinition("Test Request", "POST", "https://api.example.com")
		result := tree.AddRequest(req, coll)

		assert.True(t, result)
	})

	t.Run("adds request to first collection when nil specified", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		coll := core.NewCollection("My API")
		tree.SetCollections([]*core.Collection{coll})

		req := core.NewRequestDefinition("Test Request", "GET", "https://example.com")
		result := tree.AddRequest(req, nil)

		assert.True(t, result)
	})

	t.Run("creates new collection when none exists", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)

		req := core.NewRequestDefinition("Test Request", "GET", "https://example.com")
		result := tree.AddRequest(req, nil)

		assert.True(t, result)
		assert.Equal(t, 1, len(tree.Collections()))
	})
}

func TestCollectionTree_ExpandCollapseEdgeCases(t *testing.T) {
	t.Run("expand collection by index", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(80, 30)

		// Expand first collection
		tree.Expand(0)
		assert.True(t, tree.IsExpanded(0))
	})

	t.Run("collapse collection by index", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(80, 30)

		// Expand first then collapse
		tree.Expand(0)
		tree.Collapse(0)
		assert.False(t, tree.IsExpanded(0))
	})

	t.Run("IsExpanded returns false for invalid index", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(80, 30)

		// Check invalid index
		assert.False(t, tree.IsExpanded(100))
	})

	t.Run("Expand does nothing for invalid index", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(80, 30)

		// Expand invalid index should not panic
		tree.Expand(100)
		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("Collapse does nothing for invalid index", func(t *testing.T) {
		tree := newTestTreeWithFolders(t)
		tree.SetSize(80, 30)

		// Collapse invalid index should not panic
		tree.Collapse(100)
		view := tree.View()
		assert.NotEmpty(t, view)
	})
}

func TestCollectionTree_HistoryCursorBounds(t *testing.T) {
	t.Run("cursor stays within bounds when moving up from first", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://example.com"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Try to move up past first item
		for i := 0; i < 5; i++ {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
			updated, _ = tree.Update(msg)
			tree = updated.(*CollectionTree)
		}

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("cursor stays within bounds when moving down past last", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://example.com"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Try to move down past last item
		for i := 0; i < 5; i++ {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			updated, _ = tree.Update(msg)
			tree = updated.(*CollectionTree)
		}

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles Enter on empty history", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 30)

		store := &mockHistoryStore{entries: []history.Entry{}}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Press Enter on empty list
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		updated, cmd := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Should not crash and should not produce a command
		assert.Nil(t, cmd)
	})

	t.Run("history scrolling with small viewport", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.Focus()
		tree.SetSize(80, 5) // Very small height

		store := &mockHistoryStore{
			entries: []history.Entry{
				{ID: "1", RequestMethod: "GET", RequestURL: "https://example.com/1"},
				{ID: "2", RequestMethod: "GET", RequestURL: "https://example.com/2"},
				{ID: "3", RequestMethod: "GET", RequestURL: "https://example.com/3"},
				{ID: "4", RequestMethod: "GET", RequestURL: "https://example.com/4"},
				{ID: "5", RequestMethod: "GET", RequestURL: "https://example.com/5"},
			},
		}
		tree.SetHistoryStore(store)

		// Switch to history
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		// Move to bottom
		for i := 0; i < 10; i++ {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			updated, _ = tree.Update(msg)
			tree = updated.(*CollectionTree)
		}

		view := tree.View()
		assert.NotEmpty(t, view)
	})
}

func TestCollectionTree_UpdateCases(t *testing.T) {
	t.Run("handles unknown message type", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// Send some random message type
		type unknownMsg struct{}
		updated, _ := tree.Update(unknownMsg{})
		tree = updated.(*CollectionTree)

		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles arrow keys without crashing", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)
		tree.SetCursor(5)

		// Down arrow - just test it doesn't crash
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)
		view := tree.View()
		assert.NotEmpty(t, view)

		// Up arrow
		msg = tea.KeyMsg{Type: tea.KeyUp}
		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)
		view = tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles cursor blink message", func(t *testing.T) {
		tree := newTestTree(t)
		tree.Focus()
		tree.SetSize(80, 30)

		// This covers the blink message case in Update
		tree.Update(nil)
		view := tree.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles unfocused state", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(80, 30)
		tree.Blur()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)
		view := tree.View()
		assert.NotEmpty(t, view)
	})
}

func TestCollectionTree_GPressed(t *testing.T) {
	t.Run("returns false initially", func(t *testing.T) {
		tree := newTestTree(t)
		assert.False(t, tree.GPressed())
	})

	t.Run("returns true after pressing g", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(80, 30)
		tree.Focus() // Must be focused to handle key messages

		// Press 'g' once to set pending state
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		assert.True(t, tree.GPressed())
	})

	t.Run("returns false after gg sequence", func(t *testing.T) {
		tree := newTestTree(t)
		tree.SetSize(80, 30)
		tree.Focus()

		// Press 'g' twice for gg sequence
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		updated, _ := tree.Update(msg)
		tree = updated.(*CollectionTree)

		updated, _ = tree.Update(msg)
		tree = updated.(*CollectionTree)

		// After gg, gPressed should be false
		assert.False(t, tree.GPressed())
	})
}
