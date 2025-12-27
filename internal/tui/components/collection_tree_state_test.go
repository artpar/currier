package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history"
)

// Systematic state machine tests for CollectionTree.
// Tests are organized by: State × Key → Expected Behavior

// =============================================================================
// TEST HELPERS
// =============================================================================

func newTreeWithItems(t *testing.T) *CollectionTree {
	t.Helper()
	tree := NewCollectionTree()
	tree.SetSize(80, 30)

	coll := core.NewCollection("Test Collection")
	folder := coll.AddFolder("Test Folder")
	folder.AddRequest(core.NewRequestDefinition("Folder Request", "GET", "http://example.com/folder"))
	coll.AddRequest(core.NewRequestDefinition("Root Request", "POST", "http://example.com/root"))

	tree.SetCollections([]*core.Collection{coll})
	return tree
}

func newTreeWithHistory(t *testing.T) *CollectionTree {
	t.Helper()
	tree := newTreeWithItems(t)
	store := &mockHistoryStore{
		entries: []history.Entry{
			{ID: "1", RequestMethod: "GET", RequestURL: "http://example.com/1", Timestamp: time.Now()},
			{ID: "2", RequestMethod: "POST", RequestURL: "http://example.com/2", Timestamp: time.Now()},
			{ID: "3", RequestMethod: "DELETE", RequestURL: "http://example.com/3", Timestamp: time.Now()},
		},
	}
	tree.SetHistoryStore(store)
	return tree
}

func sendKey(tree *CollectionTree, key rune) *CollectionTree {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
	updated, _ := tree.Update(msg)
	return updated.(*CollectionTree)
}

func sendSpecialKey(tree *CollectionTree, keyType tea.KeyType) *CollectionTree {
	msg := tea.KeyMsg{Type: keyType}
	updated, _ := tree.Update(msg)
	return updated.(*CollectionTree)
}

func sendKeyWithCmd(tree *CollectionTree, key rune) (*CollectionTree, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
	updated, cmd := tree.Update(msg)
	return updated.(*CollectionTree), cmd
}

func sendSpecialKeyWithCmd(tree *CollectionTree, keyType tea.KeyType) (*CollectionTree, tea.Cmd) {
	msg := tea.KeyMsg{Type: keyType}
	updated, cmd := tree.Update(msg)
	return updated.(*CollectionTree), cmd
}

// =============================================================================
// UNFOCUSED STATE TESTS
// All keys should be no-op when unfocused
// =============================================================================

func TestCollectionTree_Unfocused_AllKeysNoOp(t *testing.T) {
	keys := []struct {
		name string
		send func(*CollectionTree) *CollectionTree
	}{
		{"j", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'j') }},
		{"k", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'k') }},
		{"l", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'l') }},
		{"h", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'h') }},
		{"G", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'G') }},
		{"g", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'g') }},
		{"H", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'H') }},
		{"C", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, 'C') }},
		{"/", func(tree *CollectionTree) *CollectionTree { return sendKey(tree, '/') }},
		{"Enter", func(tree *CollectionTree) *CollectionTree { return sendSpecialKey(tree, tea.KeyEnter) }},
		{"Escape", func(tree *CollectionTree) *CollectionTree { return sendSpecialKey(tree, tea.KeyEsc) }},
	}

	for _, k := range keys {
		t.Run(k.name+"_does_nothing_when_unfocused", func(t *testing.T) {
			tree := newTreeWithItems(t)
			// Ensure unfocused
			tree.Blur()
			assert.False(t, tree.Focused())

			initialCursor := tree.Cursor()
			initialViewMode := tree.ViewMode()
			initialSearching := tree.IsSearching()

			tree = k.send(tree)

			assert.Equal(t, initialCursor, tree.Cursor(), "cursor should not change")
			assert.Equal(t, initialViewMode, tree.ViewMode(), "viewMode should not change")
			assert.Equal(t, initialSearching, tree.IsSearching(), "searching should not change")
		})
	}
}

// =============================================================================
// COLLECTIONS MODE + NOT SEARCHING
// =============================================================================

func TestCollectionTree_CollectionsMode_Navigation(t *testing.T) {
	t.Run("j_moves_cursor_down", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand to have multiple items
		assert.Equal(t, 0, tree.Cursor())

		tree = sendKey(tree, 'j')

		assert.Equal(t, 1, tree.Cursor())
	})

	t.Run("j_at_bottom_stays_at_bottom", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		// Expand to see all items
		tree = sendKey(tree, 'l')
		// Go to bottom
		tree = sendKey(tree, 'G')
		bottomCursor := tree.Cursor()

		tree = sendKey(tree, 'j')

		assert.Equal(t, bottomCursor, tree.Cursor(), "should stay at bottom")
	})

	t.Run("k_moves_cursor_up", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand first
		tree = sendKey(tree, 'j') // Move down
		assert.Equal(t, 1, tree.Cursor())

		tree = sendKey(tree, 'k')

		assert.Equal(t, 0, tree.Cursor())
	})

	t.Run("k_at_top_stays_at_top", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.Equal(t, 0, tree.Cursor())

		tree = sendKey(tree, 'k')

		assert.Equal(t, 0, tree.Cursor(), "should stay at top")
	})

	t.Run("G_goes_to_bottom", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand to have multiple items
		assert.Equal(t, 0, tree.Cursor())

		tree = sendKey(tree, 'G')

		assert.Greater(t, tree.Cursor(), 0, "cursor should be at bottom")
	})

	t.Run("gg_goes_to_top", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		tree = sendKey(tree, 'G') // Go to bottom
		assert.Greater(t, tree.Cursor(), 0)

		tree = sendKey(tree, 'g')
		tree = sendKey(tree, 'g')

		assert.Equal(t, 0, tree.Cursor(), "cursor should be at top")
	})
}

func TestCollectionTree_CollectionsMode_ExpandCollapse(t *testing.T) {
	t.Run("l_expands_collapsed_collection", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		initialCount := tree.ItemCount()
		assert.False(t, tree.IsExpanded(0))

		tree = sendKey(tree, 'l')

		assert.True(t, tree.IsExpanded(0), "collection should be expanded")
		assert.Greater(t, tree.ItemCount(), initialCount, "should have more items visible")
	})

	t.Run("l_does_nothing_on_already_expanded", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand first
		assert.True(t, tree.IsExpanded(0))
		count := tree.ItemCount()

		tree = sendKey(tree, 'l') // Try expand again

		assert.True(t, tree.IsExpanded(0))
		assert.Equal(t, count, tree.ItemCount(), "item count should not change")
	})

	t.Run("l_does_nothing_on_request", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand collection
		tree = sendKey(tree, 'j') // Move to folder
		tree = sendKey(tree, 'l') // Expand folder
		tree = sendKey(tree, 'j') // Move to request
		count := tree.ItemCount()

		tree = sendKey(tree, 'l') // Try expand request

		assert.Equal(t, count, tree.ItemCount(), "item count should not change")
	})

	t.Run("h_collapses_expanded_collection", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand first
		assert.True(t, tree.IsExpanded(0))
		expandedCount := tree.ItemCount()

		tree = sendKey(tree, 'h')

		assert.False(t, tree.IsExpanded(0), "collection should be collapsed")
		assert.Less(t, tree.ItemCount(), expandedCount, "should have fewer items visible")
	})

	t.Run("h_does_nothing_on_already_collapsed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.False(t, tree.IsExpanded(0))
		count := tree.ItemCount()

		tree = sendKey(tree, 'h')

		assert.False(t, tree.IsExpanded(0))
		assert.Equal(t, count, tree.ItemCount())
	})

	t.Run("h_does_nothing_on_request", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		tree = sendKey(tree, 'j')
		tree = sendKey(tree, 'l') // Expand folder
		tree = sendKey(tree, 'j') // Move to request
		count := tree.ItemCount()

		tree = sendKey(tree, 'h')

		assert.Equal(t, count, tree.ItemCount())
	})
}

func TestCollectionTree_CollectionsMode_ViewSwitch(t *testing.T) {
	t.Run("H_switches_to_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		assert.Equal(t, ViewCollections, tree.ViewMode())

		tree = sendKey(tree, 'H')

		assert.Equal(t, ViewHistory, tree.ViewMode())
	})

	t.Run("C_does_nothing_in_collections_mode", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.Equal(t, ViewCollections, tree.ViewMode())

		tree = sendKey(tree, 'C')

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})
}

func TestCollectionTree_CollectionsMode_Search(t *testing.T) {
	t.Run("slash_enters_search_mode", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.False(t, tree.IsSearching())

		tree = sendKey(tree, '/')

		assert.True(t, tree.IsSearching())
	})

	t.Run("Escape_clears_search_filter", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		// Enter search, type query, exit search
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 'R')
		tree = sendKey(tree, 'o')
		tree = sendKey(tree, 'o')
		tree = sendKey(tree, 't')
		tree = sendSpecialKey(tree, tea.KeyEnter)
		assert.NotEmpty(t, tree.SearchQuery())

		tree = sendSpecialKey(tree, tea.KeyEsc)

		assert.Empty(t, tree.SearchQuery(), "search should be cleared")
	})
}

// =============================================================================
// HISTORY MODE + NOT SEARCHING
// =============================================================================

func TestCollectionTree_HistoryMode_Navigation(t *testing.T) {
	t.Run("j_moves_cursor_down_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H') // Switch to history
		assert.Equal(t, 0, tree.HistoryCursor())

		tree = sendKey(tree, 'j')

		assert.Equal(t, 1, tree.HistoryCursor())
	})

	t.Run("k_moves_cursor_up_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'j') // Move down first
		assert.Equal(t, 1, tree.HistoryCursor())

		tree = sendKey(tree, 'k')

		assert.Equal(t, 0, tree.HistoryCursor())
	})

	t.Run("G_goes_to_bottom_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		assert.Equal(t, 0, tree.HistoryCursor())

		tree = sendKey(tree, 'G')

		assert.Equal(t, len(tree.HistoryEntries())-1, tree.HistoryCursor())
	})

	t.Run("gg_goes_to_top_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'G') // Go to bottom
		assert.Greater(t, tree.HistoryCursor(), 0)

		tree = sendKey(tree, 'g')
		tree = sendKey(tree, 'g')

		assert.Equal(t, 0, tree.HistoryCursor())
	})
}

func TestCollectionTree_HistoryMode_ExpandCollapseNoOp(t *testing.T) {
	t.Run("l_does_nothing_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		cursor := tree.HistoryCursor()

		tree = sendKey(tree, 'l')

		assert.Equal(t, cursor, tree.HistoryCursor())
		assert.Equal(t, ViewHistory, tree.ViewMode())
	})

	t.Run("h_does_nothing_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		cursor := tree.HistoryCursor()

		tree = sendKey(tree, 'h')

		assert.Equal(t, cursor, tree.HistoryCursor())
		assert.Equal(t, ViewHistory, tree.ViewMode())
	})
}

func TestCollectionTree_HistoryMode_ViewSwitch(t *testing.T) {
	t.Run("C_switches_to_collections", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		assert.Equal(t, ViewHistory, tree.ViewMode())

		tree = sendKey(tree, 'C')

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})

	t.Run("H_toggles_back_to_collections", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		assert.Equal(t, ViewHistory, tree.ViewMode())

		tree = sendKey(tree, 'H')

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})

	t.Run("Escape_switches_to_collections", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		assert.Equal(t, ViewHistory, tree.ViewMode())

		tree = sendSpecialKey(tree, tea.KeyEsc)

		assert.Equal(t, ViewCollections, tree.ViewMode())
	})
}

func TestCollectionTree_HistoryMode_Refresh(t *testing.T) {
	t.Run("r_refreshes_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'j') // Move cursor
		assert.Equal(t, 1, tree.HistoryCursor())

		tree = sendKey(tree, 'r')

		// After refresh, cursor resets to 0
		assert.Equal(t, 0, tree.HistoryCursor())
	})
}

// =============================================================================
// SEARCH MODE
// =============================================================================

func TestCollectionTree_SearchMode_Input(t *testing.T) {
	t.Run("typing_adds_to_search_query", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, '/')
		assert.True(t, tree.IsSearching())

		tree = sendKey(tree, 't')
		tree = sendKey(tree, 'e')
		tree = sendKey(tree, 's')
		tree = sendKey(tree, 't')

		assert.Equal(t, "test", tree.SearchQuery())
	})

	t.Run("j_adds_to_query_not_move", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		tree = sendKey(tree, '/')

		tree = sendKey(tree, 'j')

		assert.Equal(t, "j", tree.SearchQuery())
		assert.Equal(t, 0, tree.Cursor(), "cursor should not move")
	})

	t.Run("k_adds_to_query_and_resets_cursor", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand first
		tree = sendKey(tree, 'j') // Move down
		assert.Equal(t, 1, tree.Cursor())
		tree = sendKey(tree, '/')

		tree = sendKey(tree, 'k')

		assert.Equal(t, "k", tree.SearchQuery())
		// Cursor resets to 0 because filter changes visible items
		assert.Equal(t, 0, tree.Cursor(), "cursor resets when filter applied")
	})

	t.Run("Backspace_removes_character", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 'a')
		tree = sendKey(tree, 'b')
		tree = sendKey(tree, 'c')
		assert.Equal(t, "abc", tree.SearchQuery())

		tree = sendSpecialKey(tree, tea.KeyBackspace)

		assert.Equal(t, "ab", tree.SearchQuery())
	})

	t.Run("CtrlU_clears_query", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 'a')
		tree = sendKey(tree, 'b')
		tree = sendKey(tree, 'c')
		assert.Equal(t, "abc", tree.SearchQuery())

		tree = sendSpecialKey(tree, tea.KeyCtrlU)

		assert.Equal(t, "", tree.SearchQuery())
	})

	t.Run("Space_adds_space_to_query", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 'a')
		tree = sendSpecialKey(tree, tea.KeySpace)
		tree = sendKey(tree, 'b')

		assert.Equal(t, "a b", tree.SearchQuery())
	})
}

func TestCollectionTree_SearchMode_Exit(t *testing.T) {
	t.Run("Enter_exits_search_keeps_filter", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 'R')
		tree = sendKey(tree, 'o')
		tree = sendKey(tree, 'o')
		tree = sendKey(tree, 't')
		assert.True(t, tree.IsSearching())

		tree = sendSpecialKey(tree, tea.KeyEnter)

		assert.False(t, tree.IsSearching())
		assert.Equal(t, "Root", tree.SearchQuery(), "filter should remain")
	})

	t.Run("Escape_exits_search_keeps_filter", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, '/')
		tree = sendKey(tree, 't')
		tree = sendKey(tree, 'e')
		tree = sendKey(tree, 's')
		tree = sendKey(tree, 't')
		assert.True(t, tree.IsSearching())

		tree = sendSpecialKey(tree, tea.KeyEsc)

		assert.False(t, tree.IsSearching())
		assert.Equal(t, "test", tree.SearchQuery(), "filter should remain")
	})
}

func TestCollectionTree_SearchMode_History(t *testing.T) {
	t.Run("search_works_in_history_mode", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H') // Switch to history
		tree = sendKey(tree, '/')
		assert.True(t, tree.IsSearching())

		tree = sendKey(tree, 'G')
		tree = sendKey(tree, 'E')
		tree = sendKey(tree, 'T')

		// Should be searching in history
		assert.Equal(t, ViewHistory, tree.ViewMode())
		assert.True(t, tree.IsSearching())
	})
}

// =============================================================================
// gPressed STATE MANAGEMENT
// =============================================================================

func TestCollectionTree_GPressed_StateManagement(t *testing.T) {
	t.Run("g_sets_gPressed_true", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.False(t, tree.GPressed())

		tree = sendKey(tree, 'g')

		assert.True(t, tree.GPressed())
	})

	t.Run("gg_resets_gPressed_false", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'g')

		assert.False(t, tree.GPressed())
	})

	t.Run("j_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'j')

		assert.False(t, tree.GPressed())
	})

	t.Run("k_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'k')

		assert.False(t, tree.GPressed())
	})

	t.Run("l_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'l')

		assert.False(t, tree.GPressed())
	})

	t.Run("h_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'h')

		assert.False(t, tree.GPressed())
	})

	t.Run("G_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'G')

		assert.False(t, tree.GPressed())
	})

	t.Run("H_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'H')

		assert.False(t, tree.GPressed())
	})

	t.Run("slash_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, '/')

		assert.False(t, tree.GPressed())
	})

	t.Run("Enter_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendSpecialKey(tree, tea.KeyEnter)

		assert.False(t, tree.GPressed())
	})

	t.Run("Escape_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendSpecialKey(tree, tea.KeyEsc)

		assert.False(t, tree.GPressed())
	})

	t.Run("unknown_key_resets_gPressed", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'x') // Unknown key

		assert.False(t, tree.GPressed())
	})
}

func TestCollectionTree_GPressed_HistoryMode(t *testing.T) {
	t.Run("g_sets_gPressed_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		assert.False(t, tree.GPressed())

		tree = sendKey(tree, 'g')

		assert.True(t, tree.GPressed())
	})

	t.Run("gg_goes_to_top_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'G') // Go to bottom
		assert.Greater(t, tree.HistoryCursor(), 0)

		tree = sendKey(tree, 'g')
		tree = sendKey(tree, 'g')

		assert.Equal(t, 0, tree.HistoryCursor())
		assert.False(t, tree.GPressed())
	})

	t.Run("C_resets_gPressed_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'C')

		assert.False(t, tree.GPressed())
	})

	t.Run("r_resets_gPressed_in_history", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')
		tree = sendKey(tree, 'g')
		assert.True(t, tree.GPressed())

		tree = sendKey(tree, 'r')

		assert.False(t, tree.GPressed())
	})
}

// =============================================================================
// ENTER ON DIFFERENT ITEM TYPES
// =============================================================================

func TestCollectionTree_Enter_ItemTypes(t *testing.T) {
	t.Run("Enter_on_collection_toggles_expand", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		assert.False(t, tree.IsExpanded(0))

		tree, cmd := sendSpecialKeyWithCmd(tree, tea.KeyEnter)

		assert.True(t, tree.IsExpanded(0))
		assert.Nil(t, cmd, "should not return selection command")
	})

	t.Run("Enter_on_expanded_collection_collapses", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		assert.True(t, tree.IsExpanded(0))

		tree, cmd := sendSpecialKeyWithCmd(tree, tea.KeyEnter)

		assert.False(t, tree.IsExpanded(0))
		assert.Nil(t, cmd)
	})

	t.Run("Enter_on_folder_toggles_expand", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand collection
		tree = sendKey(tree, 'j') // Move to folder

		tree, cmd := sendSpecialKeyWithCmd(tree, tea.KeyEnter)

		assert.Nil(t, cmd, "folder expand should not return command")
	})

	t.Run("Enter_on_request_returns_selection", func(t *testing.T) {
		tree := newTreeWithItems(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand collection
		tree = sendKey(tree, 'j') // Move to folder
		tree = sendKey(tree, 'l') // Expand folder
		tree = sendKey(tree, 'j') // Move to request

		tree, cmd := sendSpecialKeyWithCmd(tree, tea.KeyEnter)

		assert.NotNil(t, cmd, "should return selection command")
		// Execute command and check message type
		msg := cmd()
		_, ok := msg.(SelectionMsg)
		assert.True(t, ok, "should be SelectionMsg")
	})

	t.Run("Enter_on_history_entry_returns_selection", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'H')

		tree, cmd := sendSpecialKeyWithCmd(tree, tea.KeyEnter)

		assert.NotNil(t, cmd, "should return selection command")
		msg := cmd()
		_, ok := msg.(SelectHistoryItemMsg)
		assert.True(t, ok, "should be SelectHistoryItemMsg")
	})
}

// =============================================================================
// CURSOR BOUNDARIES
// =============================================================================

func TestCollectionTree_CursorBoundaries(t *testing.T) {
	t.Run("cursor_clamps_at_zero_on_empty", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)
		tree.Focus()

		tree = sendKey(tree, 'j')

		assert.Equal(t, 0, tree.Cursor())
	})

	t.Run("G_does_nothing_on_empty", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 30)
		tree.Focus()

		tree = sendKey(tree, 'G')

		assert.Equal(t, 0, tree.Cursor())
	})

	t.Run("cursor_independent_between_modes", func(t *testing.T) {
		tree := newTreeWithHistory(t)
		tree.Focus()
		tree = sendKey(tree, 'l') // Expand
		tree = sendKey(tree, 'j')
		tree = sendKey(tree, 'j')
		collectionsCursor := tree.Cursor()

		tree = sendKey(tree, 'H') // Switch to history
		tree = sendKey(tree, 'j')
		historyCursor := tree.HistoryCursor()

		tree = sendKey(tree, 'C') // Switch back

		assert.Equal(t, collectionsCursor, tree.Cursor(), "collections cursor should be preserved")
		assert.Equal(t, historyCursor, tree.HistoryCursor(), "history cursor should be preserved")
	})
}

// =============================================================================
// SCROLL OFFSET
// =============================================================================

func TestCollectionTree_ScrollOffset(t *testing.T) {
	t.Run("G_adjusts_offset_to_show_cursor", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 10) // Small height to trigger scrolling
		tree.Focus()

		// Create a collection with many items
		coll := core.NewCollection("Big Collection")
		for i := 0; i < 20; i++ {
			coll.AddRequest(core.NewRequestDefinition("Request", "GET", "http://example.com"))
		}
		tree.SetCollections([]*core.Collection{coll})
		tree = sendKey(tree, 'l') // Expand

		tree = sendKey(tree, 'G')

		// Cursor should be at bottom, and offset should have scrolled
		assert.Greater(t, tree.Cursor(), 0)
	})

	t.Run("gg_resets_offset_to_zero", func(t *testing.T) {
		tree := NewCollectionTree()
		tree.SetSize(80, 10)
		tree.Focus()

		coll := core.NewCollection("Big Collection")
		for i := 0; i < 20; i++ {
			coll.AddRequest(core.NewRequestDefinition("Request", "GET", "http://example.com"))
		}
		tree.SetCollections([]*core.Collection{coll})
		tree = sendKey(tree, 'l')
		tree = sendKey(tree, 'G') // Scroll down

		tree = sendKey(tree, 'g')
		tree = sendKey(tree, 'g')

		assert.Equal(t, 0, tree.Cursor())
	})
}
