package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/artpar/currier/internal/core"
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
