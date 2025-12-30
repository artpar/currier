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
// Searches in: name, method, URL, body, and headers.
func FilterItemsBySearch(items []TreeItem, search string) []TreeItem {
	if search == "" {
		return items
	}
	search = strings.ToLower(search)
	var result []TreeItem
	for _, item := range items {
		if matchesSearch(item, search) {
			result = append(result, item)
		}
	}
	return result
}

// matchesSearch checks if an item matches the search query.
// Searches name, and for requests: method, URL, body, and headers.
func matchesSearch(item TreeItem, search string) bool {
	// Always search by name
	if strings.Contains(strings.ToLower(item.Name), search) {
		return true
	}

	// For requests, search additional fields
	if item.Type == ItemRequest {
		// Search method
		if strings.Contains(strings.ToLower(item.Method), search) {
			return true
		}

		if item.Request != nil {
			// Search URL
			if strings.Contains(strings.ToLower(item.Request.URL()), search) {
				return true
			}

			// Search body
			if strings.Contains(strings.ToLower(item.Request.Body()), search) {
				return true
			}

			// Search headers (keys and values)
			for key, value := range item.Request.Headers() {
				if strings.Contains(strings.ToLower(key), search) ||
					strings.Contains(strings.ToLower(value), search) {
					return true
				}
			}
		}
	}

	// For WebSocket, search endpoint
	if item.Type == ItemWebSocket && item.WebSocket != nil {
		if strings.Contains(strings.ToLower(item.WebSocket.Endpoint), search) {
			return true
		}
	}

	return false
}

// ============================================================================
// Selection Functions
// ============================================================================

// ToggleSelection returns a new selected map with the specified item toggled.
// Pure function: returns new map, never mutates input.
func ToggleSelection(selected map[string]bool, id string) map[string]bool {
	result := make(map[string]bool, len(selected)+1)
	for k, v := range selected {
		result[k] = v
	}
	if result[id] {
		delete(result, id)
	} else {
		result[id] = true
	}
	return result
}

// SetSelection returns a new selected map with the specified item set to a value.
// Pure function: returns new map, never mutates input.
func SetSelection(selected map[string]bool, id string, isSelected bool) map[string]bool {
	result := make(map[string]bool, len(selected)+1)
	for k, v := range selected {
		result[k] = v
	}
	if isSelected {
		result[id] = true
	} else {
		delete(result, id)
	}
	return result
}

// ClearSelection returns an empty selection map.
// Pure function: returns new map.
func ClearSelection() map[string]bool {
	return make(map[string]bool)
}

// SelectRange returns a new selected map with items from startIdx to endIdx selected.
// Items are identified by their IDs from the items slice.
// Pure function: returns new map, never mutates input.
func SelectRange(selected map[string]bool, items []TreeItem, startIdx, endIdx int, additive bool) map[string]bool {
	var result map[string]bool
	if additive {
		result = make(map[string]bool, len(selected)+abs(endIdx-startIdx)+1)
		for k, v := range selected {
			result[k] = v
		}
	} else {
		result = make(map[string]bool, abs(endIdx-startIdx)+1)
	}

	low, high := startIdx, endIdx
	if low > high {
		low, high = high, low
	}

	for i := low; i <= high && i < len(items); i++ {
		if i >= 0 {
			item := items[i]
			// Only requests and folders are selectable
			if item.Type == ItemRequest || item.Type == ItemFolder {
				result[item.ID] = true
			}
		}
	}
	return result
}

// SelectAll returns a new selected map with all selectable items selected.
// Only requests and folders are selectable, not collections.
// Pure function: returns new map.
func SelectAll(items []TreeItem) map[string]bool {
	result := make(map[string]bool)
	for _, item := range items {
		if item.Type == ItemRequest || item.Type == ItemFolder {
			result[item.ID] = true
		}
	}
	return result
}

// GetSelectedItems returns items that are currently selected.
// Pure function: returns new slice.
func GetSelectedItems(items []TreeItem, selected map[string]bool) []TreeItem {
	var result []TreeItem
	for _, item := range items {
		if selected[item.ID] {
			result = append(result, item)
		}
	}
	return result
}

// CountSelected returns the number of selected items.
// Pure function.
func CountSelected(selected map[string]bool) int {
	count := 0
	for _, v := range selected {
		if v {
			count++
		}
	}
	return count
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
