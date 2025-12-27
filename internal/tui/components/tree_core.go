package components

import "strings"

// This file contains pure functions for tree operations.
// These functions take values and return values - no mutation, no side effects.
// This enables trivial testing (input → output) and makes the shell code explicit.

// MoveCursor computes new cursor position within bounds.
// Pure function: takes current state, returns new position.
func MoveCursor(cursor, delta, itemCount int) int {
	if itemCount == 0 {
		return 0
	}
	newCursor := cursor + delta
	if newCursor < 0 {
		return 0
	}
	if newCursor >= itemCount {
		return itemCount - 1
	}
	return newCursor
}

// AdjustOffset ensures cursor is visible within viewport.
// Pure function: takes scroll state, returns new offset.
func AdjustOffset(cursor, offset, visibleHeight int) int {
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if cursor < offset {
		return cursor
	}
	if cursor >= offset+visibleHeight {
		return cursor - visibleHeight + 1
	}
	return offset
}

// ToggleExpand returns a new expanded map with the specified item toggled.
// Pure function: returns new map, never mutates input.
func ToggleExpand(expanded map[string]bool, id string, expand bool) map[string]bool {
	result := make(map[string]bool, len(expanded)+1)
	for k, v := range expanded {
		result[k] = v
	}
	result[id] = expand
	return result
}

// FilterItemsBySearch returns items matching search query.
// Pure function: returns filtered slice, never mutates input.
func FilterItemsBySearch(items []TreeItem, search string) []TreeItem {
	if search == "" {
		return items
	}
	search = strings.ToLower(search)
	var result []TreeItem
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), search) ||
			(item.Type == ItemRequest && strings.Contains(strings.ToLower(item.Method), search)) {
			result = append(result, item)
		}
	}
	return result
}
