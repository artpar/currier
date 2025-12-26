package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/tui"
)

// ViewMode represents what the collection tree is displaying.
type ViewMode int

const (
	ViewCollections ViewMode = iota
	ViewHistory
)

// TreeItem represents an item in the collection tree.
type TreeItem struct {
	ID          string
	Name        string
	Type        TreeItemType
	Level       int
	Expandable  bool
	Expanded    bool
	Collection  *core.Collection
	Folder      *core.Folder
	Request     *core.RequestDefinition
	WebSocket   *core.WebSocketDefinition
	Method      string
}

// TreeItemType identifies the type of tree item.
type TreeItemType int

const (
	ItemCollection TreeItemType = iota
	ItemFolder
	ItemRequest
	ItemWebSocket
)

// SelectionMsg is sent when a request is selected.
type SelectionMsg struct {
	Request *core.RequestDefinition
}

// SelectWebSocketMsg is sent when a WebSocket is selected.
type SelectWebSocketMsg struct {
	WebSocket *core.WebSocketDefinition
}

// SelectHistoryItemMsg is sent when a history item is selected.
type SelectHistoryItemMsg struct {
	Entry history.Entry
}

// CollectionTree displays a tree of collections, folders, and requests.
type CollectionTree struct {
	title         string
	focused       bool
	width         int
	height        int
	cursor        int
	offset        int // For scrolling
	collections   []*core.Collection
	items         []TreeItem
	filteredItems []TreeItem // Items after search filter
	expanded      map[string]bool
	search        string
	searching     bool // True when in search mode
	gPressed      bool // For gg sequence

	// View mode (Collections or History)
	viewMode ViewMode

	// History support
	historyStore   history.Store
	historyEntries []history.Entry
	historyCursor  int
	historyOffset  int
	historySearch  string
}

// NewCollectionTree creates a new collection tree component.
func NewCollectionTree() *CollectionTree {
	return &CollectionTree{
		title:    "Collections",
		expanded: make(map[string]bool),
	}
}

// Init initializes the component.
func (c *CollectionTree) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (c *CollectionTree) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
	if !c.focused {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			c.width = msg.Width
			c.height = msg.Height
		case tui.FocusMsg:
			c.focused = true
		}
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case tui.FocusMsg:
		c.focused = true

	case tui.BlurMsg:
		c.focused = false

	case tea.KeyMsg:
		return c.handleKeyMsg(msg)
	}

	return c, nil
}

func (c *CollectionTree) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Handle search mode input
	if c.searching {
		return c.handleSearchInput(msg)
	}

	// Handle history view mode
	if c.viewMode == ViewHistory {
		return c.handleHistoryKeyMsg(msg)
	}

	switch msg.Type {
	case tea.KeyEsc:
		// Clear search filter when not in search mode
		if c.search != "" {
			c.search = ""
			c.filteredItems = nil
			c.cursor = 0
			c.offset = 0
		}
		return c, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			c.searching = true
			c.search = ""
			return c, nil
		case "j":
			c.moveCursor(1)
		case "k":
			c.moveCursor(-1)
		case "l":
			c.expandCurrent()
		case "h":
			c.collapseCurrent()
		case "H":
			// Switch to History view
			c.viewMode = ViewHistory
			c.loadHistory()
			return c, nil
		case "G":
			c.cursor = len(c.getDisplayItems()) - 1
			c.gPressed = false
		case "g":
			if c.gPressed {
				c.cursor = 0
				c.gPressed = false
			} else {
				c.gPressed = true
			}
			return c, nil
		default:
			c.gPressed = false
		}

	case tea.KeyEnter:
		return c.handleEnter()
	}

	c.gPressed = false
	return c, nil
}

func (c *CollectionTree) handleHistoryKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// If searching, clear search filter first
		if c.historySearch != "" {
			c.historySearch = ""
			c.loadHistory()
			return c, nil
		}
		// Otherwise, exit History view and return to Collections
		c.viewMode = ViewCollections
		return c, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			c.searching = true
			c.historySearch = ""
			return c, nil
		case "j":
			c.moveHistoryCursor(1)
		case "k":
			c.moveHistoryCursor(-1)
		case "C", "H":
			// Switch back to Collections view (H toggles, C explicit)
			c.viewMode = ViewCollections
			return c, nil
		case "h", "l":
			// No-op in history view (no expand/collapse) but handle gracefully
			return c, nil
		case "G":
			if len(c.historyEntries) > 0 {
				c.historyCursor = len(c.historyEntries) - 1
			}
			c.gPressed = false
		case "g":
			if c.gPressed {
				c.historyCursor = 0
				c.historyOffset = 0
				c.gPressed = false
			} else {
				c.gPressed = true
			}
			return c, nil
		case "r":
			// Refresh history
			c.loadHistory()
			return c, nil
		default:
			c.gPressed = false
		}

	case tea.KeyEnter:
		return c.handleHistoryEnter()
	}

	c.gPressed = false
	return c, nil
}

func (c *CollectionTree) moveHistoryCursor(delta int) {
	c.historyCursor += delta
	if c.historyCursor < 0 {
		c.historyCursor = 0
	}
	maxCursor := len(c.historyEntries) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}
	if c.historyCursor > maxCursor {
		c.historyCursor = maxCursor
	}

	// Adjust scroll offset
	visibleHeight := c.contentHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if c.historyCursor < c.historyOffset {
		c.historyOffset = c.historyCursor
	}
	if c.historyCursor >= c.historyOffset+visibleHeight {
		c.historyOffset = c.historyCursor - visibleHeight + 1
	}
}

func (c *CollectionTree) handleHistoryEnter() (tui.Component, tea.Cmd) {
	if c.historyCursor < 0 || c.historyCursor >= len(c.historyEntries) {
		return c, nil
	}

	entry := c.historyEntries[c.historyCursor]
	return c, func() tea.Msg {
		return SelectHistoryItemMsg{Entry: entry}
	}
}

func (c *CollectionTree) loadHistory() {
	if c.historyStore == nil {
		c.historyEntries = nil
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := history.QueryOptions{
		Limit:     100, // Load last 100 entries
		SortBy:    "timestamp",
		SortOrder: "DESC",
	}

	if c.historySearch != "" {
		entries, err := c.historyStore.Search(ctx, c.historySearch, opts)
		if err == nil {
			c.historyEntries = entries
		}
	} else {
		entries, err := c.historyStore.List(ctx, opts)
		if err == nil {
			c.historyEntries = entries
		}
	}

	c.historyCursor = 0
	c.historyOffset = 0
}

func (c *CollectionTree) handleSearchInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Determine which search string to modify based on view mode
	getSearch := func() string {
		if c.viewMode == ViewHistory {
			return c.historySearch
		}
		return c.search
	}
	setSearch := func(s string) {
		if c.viewMode == ViewHistory {
			c.historySearch = s
		} else {
			c.search = s
		}
	}

	switch msg.Type {
	case tea.KeyEsc:
		// Exit search mode but keep filter
		c.searching = false
		return c, nil

	case tea.KeyEnter:
		// Exit search mode and apply filter
		c.searching = false
		if c.viewMode == ViewHistory {
			c.loadHistory()
		}
		return c, nil

	case tea.KeyBackspace:
		s := getSearch()
		if len(s) > 0 {
			setSearch(s[:len(s)-1])
			if c.viewMode == ViewHistory {
				c.loadHistory()
			} else {
				c.applyFilter()
			}
		}
		return c, nil

	case tea.KeyCtrlU:
		// Clear search
		setSearch("")
		if c.viewMode == ViewHistory {
			c.loadHistory()
		} else {
			c.applyFilter()
		}
		return c, nil

	case tea.KeySpace:
		// Insert space character
		setSearch(getSearch() + " ")
		if c.viewMode == ViewHistory {
			c.loadHistory()
		} else {
			c.applyFilter()
		}
		return c, nil

	case tea.KeyRunes:
		setSearch(getSearch() + string(msg.Runes))
		if c.viewMode == ViewHistory {
			c.loadHistory()
		} else {
			c.applyFilter()
		}
		return c, nil
	}

	return c, nil
}

func (c *CollectionTree) handleEnter() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	switch item.Type {
	case ItemCollection, ItemFolder:
		// Toggle expand/collapse
		if item.Expanded {
			c.expanded[item.ID] = false
		} else {
			c.expanded[item.ID] = true
		}
		c.rebuildItems()
		c.applyFilter()
	case ItemRequest:
		// Select request
		return c, func() tea.Msg {
			return SelectionMsg{Request: item.Request}
		}
	case ItemWebSocket:
		// Select WebSocket
		return c, func() tea.Msg {
			return SelectWebSocketMsg{WebSocket: item.WebSocket}
		}
	}

	return c, nil
}

func (c *CollectionTree) moveCursor(delta int) {
	displayItems := c.getDisplayItems()
	c.cursor += delta
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(displayItems) {
		c.cursor = len(displayItems) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}

	// Adjust scroll offset
	visibleHeight := c.contentHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if c.cursor < c.offset {
		c.offset = c.cursor
	}
	if c.cursor >= c.offset+visibleHeight {
		c.offset = c.cursor - visibleHeight + 1
	}
}

// getDisplayItems returns the items to display (filtered or all).
func (c *CollectionTree) getDisplayItems() []TreeItem {
	if c.search == "" {
		return c.items
	}
	return c.filteredItems
}

// applyFilter filters items based on search query.
func (c *CollectionTree) applyFilter() {
	if c.search == "" {
		c.filteredItems = nil
		c.cursor = 0
		c.offset = 0
		return
	}

	search := strings.ToLower(c.search)
	c.filteredItems = nil

	// Collect matching items and their parent paths
	matchedIDs := make(map[string]bool)

	// First pass: find all matching items
	for _, item := range c.items {
		if strings.Contains(strings.ToLower(item.Name), search) ||
			(item.Type == ItemRequest && strings.Contains(strings.ToLower(item.Method), search)) {
			matchedIDs[item.ID] = true
		}
	}

	// Add all matching items to filtered list
	for _, item := range c.items {
		if matchedIDs[item.ID] {
			c.filteredItems = append(c.filteredItems, item)
		}
	}

	// Reset cursor
	c.cursor = 0
	c.offset = 0
}

// contentHeight returns the height available for content.
func (c *CollectionTree) contentHeight() int {
	// height - 2 (borders) - 1 (title) - 1 (mode indicator) - search bar if present
	height := c.height - 4
	if c.searching || c.search != "" {
		height-- // Search bar
	}
	if height < 1 {
		height = 1
	}
	return height
}

func (c *CollectionTree) expandCurrent() {
	displayItems := c.getDisplayItems()
	if c.cursor >= 0 && c.cursor < len(displayItems) {
		item := displayItems[c.cursor]
		if item.Expandable && !item.Expanded {
			c.expanded[item.ID] = true
			c.rebuildItems()
			c.applyFilter()
		}
	}
}

func (c *CollectionTree) collapseCurrent() {
	displayItems := c.getDisplayItems()
	if c.cursor >= 0 && c.cursor < len(displayItems) {
		item := displayItems[c.cursor]
		if item.Expandable && item.Expanded {
			c.expanded[item.ID] = false
			c.rebuildItems()
			c.applyFilter()
		}
	}
}

// View renders the component.
func (c *CollectionTree) View() string {
	if c.width == 0 || c.height == 0 {
		return ""
	}

	// Account for borders (2 chars width, 2 lines height)
	innerWidth := c.width - 2
	innerHeight := c.height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Determine title based on view mode
	displayTitle := c.title
	if c.viewMode == ViewHistory {
		displayTitle = "History"
	}

	// Title takes 1 line
	titleStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Bold(true)

	if c.focused {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62"))
	} else {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238"))
	}

	title := titleStyle.Render(displayTitle)

	// Search bar (if searching or has active filter) - takes 1 line
	var searchBar string
	searchLines := 0
	searchQuery := c.search
	if c.viewMode == ViewHistory {
		searchQuery = c.historySearch
	}
	if c.searching || searchQuery != "" {
		searchBar = c.renderSearchBar()
		searchLines = 1
	}

	// Mode indicator (shows H for history, C for collections switch hint)
	var modeIndicator string
	if c.viewMode == ViewHistory {
		modeIndicator = c.renderModeIndicator("C→Collections", innerWidth)
	} else {
		modeIndicator = c.renderModeIndicator("H→History", innerWidth)
	}
	modeLines := 1

	// Content height = inner height - title (1) - search bar (0 or 1) - mode indicator (1)
	contentHeight := innerHeight - 1 - searchLines - modeLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	if c.viewMode == ViewHistory {
		content = c.renderHistoryContent(innerWidth, contentHeight)
	} else {
		content = c.renderCollectionContent(innerWidth, contentHeight)
	}

	// Combine all parts
	var parts []string
	parts = append(parts, title)
	parts = append(parts, modeIndicator)
	if searchBar != "" {
		parts = append(parts, searchBar)
	}
	parts = append(parts, content)

	// Border style without explicit dimensions
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	if c.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		// Use brighter color (244 instead of 240) for better visibility
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("244"))
	}

	return borderStyle.Render(strings.Join(parts, "\n"))
}

func (c *CollectionTree) renderModeIndicator(hint string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Width(width)
	return style.Render(hint)
}

func (c *CollectionTree) renderCollectionContent(innerWidth, contentHeight int) string {
	displayItems := c.getDisplayItems()
	var lines []string
	for i := c.offset; i < len(displayItems) && len(lines) < contentHeight; i++ {
		item := displayItems[i]
		line := c.renderItem(item, i == c.cursor)
		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	emptyLine := strings.Repeat(" ", innerWidth)
	for len(lines) < contentHeight {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines, "\n")
}

func (c *CollectionTree) renderHistoryContent(innerWidth, contentHeight int) string {
	var lines []string

	if len(c.historyEntries) == 0 {
		emptyMsg := "No history entries"
		if c.historyStore == nil {
			emptyMsg = "History not available"
		}
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(innerWidth).
			Align(lipgloss.Center)
		lines = append(lines, emptyStyle.Render(emptyMsg))
	} else {
		for i := c.historyOffset; i < len(c.historyEntries) && len(lines) < contentHeight; i++ {
			entry := c.historyEntries[i]
			line := c.renderHistoryItem(entry, i == c.historyCursor, innerWidth)
			lines = append(lines, line)
		}
	}

	// Pad with empty lines if needed
	emptyLine := strings.Repeat(" ", innerWidth)
	for len(lines) < contentHeight {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines, "\n")
}

func (c *CollectionTree) renderHistoryItem(entry history.Entry, selected bool, width int) string {
	// Format: [METHOD] URL - status - time ago
	methodBadge := c.methodBadge(entry.RequestMethod)

	// Truncate URL to fit
	url := entry.RequestURL
	// Remove protocol for display
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Status badge
	statusStyle := lipgloss.NewStyle().Bold(true)
	switch {
	case entry.ResponseStatus >= 200 && entry.ResponseStatus < 300:
		statusStyle = statusStyle.Foreground(lipgloss.Color("34")) // Green
	case entry.ResponseStatus >= 300 && entry.ResponseStatus < 400:
		statusStyle = statusStyle.Foreground(lipgloss.Color("214")) // Orange
	case entry.ResponseStatus >= 400:
		statusStyle = statusStyle.Foreground(lipgloss.Color("160")) // Red
	}
	statusStr := statusStyle.Render(fmt.Sprintf("%d", entry.ResponseStatus))

	// Time ago
	timeAgo := c.formatTimeAgo(entry.Timestamp)
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	timeStr := timeStyle.Render(timeAgo)

	// Calculate available width for URL
	// methodBadge is ~6 chars, status ~3 chars, time ~10 chars, spaces ~5 chars
	availableWidth := width - 25
	if availableWidth < 10 {
		availableWidth = 10
	}
	if len(url) > availableWidth {
		url = url[:availableWidth-3] + "..."
	}

	line := fmt.Sprintf("%s %s %s %s", methodBadge, url, statusStr, timeStr)

	// Pad to full width
	if len(line) < width {
		line += strings.Repeat(" ", width-lipgloss.Width(line))
	}

	// Apply selection styling
	style := lipgloss.NewStyle()
	if selected {
		if c.focused {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		} else {
			// Dimmer highlight when unfocused but still selected
			style = style.
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("252"))
		}
	}

	return style.Render(line)
}

func (c *CollectionTree) formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

func (c *CollectionTree) renderSearchBar() string {
	width := c.width - 2 // Account for borders only

	// Search icon and input
	searchIcon := "🔍 "
	query := c.search
	if c.viewMode == ViewHistory {
		query = c.historySearch
	}

	// Cursor indicator when in search mode
	cursor := ""
	if c.searching {
		cursor = "▌"
	}

	// Calculate result count for feedback
	var resultCount int
	if c.viewMode == ViewHistory {
		resultCount = len(c.historyEntries)
	} else {
		if c.search != "" {
			resultCount = len(c.filteredItems)
		} else {
			resultCount = len(c.items)
		}
	}

	// Build result feedback string
	var resultFeedback string
	if query != "" && !c.searching {
		// Show results count after search is done (not while typing)
		if resultCount == 0 {
			resultFeedback = " (No matches)"
		} else {
			resultFeedback = fmt.Sprintf(" (%d result", resultCount)
			if resultCount != 1 {
				resultFeedback += "s"
			}
			resultFeedback += ")"
		}
	}

	// Style based on search state
	var style lipgloss.Style
	if c.searching {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
	} else {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	}

	// Build search bar content
	content := searchIcon + query + cursor

	// Add result feedback with different color
	if resultFeedback != "" {
		// Calculate available space
		contentWidth := lipgloss.Width(content)
		feedbackWidth := len(resultFeedback)

		if contentWidth+feedbackWidth <= width {
			// Style the feedback differently
			var feedbackStyle lipgloss.Style
			if resultCount == 0 {
				feedbackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("160")) // Red for no matches
			} else {
				feedbackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34")) // Green for results
			}
			content += feedbackStyle.Render(resultFeedback)
		}
	}

	// Truncate if too long
	contentWidth := lipgloss.Width(content)
	if contentWidth > width {
		content = content[:width-3] + "..."
	}

	// Pad to full width
	contentWidth = lipgloss.Width(content)
	if contentWidth < width {
		content += strings.Repeat(" ", width-contentWidth)
	}

	return style.Render(content)
}

func (c *CollectionTree) renderItem(item TreeItem, selected bool) string {
	width := c.width - 2 // Account for borders only

	// Indentation
	indent := strings.Repeat("  ", item.Level)

	// Expand indicator
	var indicator string
	if item.Expandable {
		if item.Expanded {
			indicator = "▼ "
		} else {
			indicator = "▶ "
		}
	} else {
		indicator = "  "
	}

	// Icon based on type
	var icon string
	switch item.Type {
	case ItemCollection:
		icon = "📁 "
	case ItemFolder:
		icon = "📂 "
	case ItemRequest:
		icon = c.methodBadge(item.Method) + " "
	}

	// Name
	name := item.Name
	availableWidth := width - len(indent) - len(indicator) - len(icon) - 2
	if availableWidth <= 0 {
		// Not enough space for name at all
		name = ""
	} else if availableWidth < 4 {
		// Not enough space for truncation with "...", just cut
		if len(name) > availableWidth {
			name = name[:availableWidth]
		}
	} else if len(name) > availableWidth {
		name = name[:availableWidth-3] + "..."
	}

	line := indent + indicator + icon + name

	// Pad to full width
	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	}

	// Apply selection styling
	style := lipgloss.NewStyle()
	if selected {
		if c.focused {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		} else {
			// Dimmer highlight when unfocused but still selected
			style = style.
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("252"))
		}
	}

	return style.Render(line)
}

func (c *CollectionTree) methodBadge(method string) string {
	// Compact colored method badges
	style := lipgloss.NewStyle().Bold(true)

	switch strings.ToUpper(method) {
	case "GET":
		return style.
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("255")).
			Render(" GET ")
	case "POST":
		return style.
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Render(" POST")
	case "PUT":
		return style.
			Background(lipgloss.Color("33")).
			Foreground(lipgloss.Color("255")).
			Render(" PUT ")
	case "PATCH":
		return style.
			Background(lipgloss.Color("141")).
			Foreground(lipgloss.Color("255")).
			Render(" PTCH")
	case "DELETE":
		return style.
			Background(lipgloss.Color("160")).
			Foreground(lipgloss.Color("255")).
			Render(" DEL ")
	case "HEAD":
		return style.
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Render(" HEAD")
	case "OPTIONS":
		return style.
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Render(" OPT ")
	default:
		return style.
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Render(fmt.Sprintf(" %-4s", method))
	}
}

// Title returns the component title.
func (c *CollectionTree) Title() string {
	return c.title
}

// Focused returns true if focused.
func (c *CollectionTree) Focused() bool {
	return c.focused
}

// Focus sets the component as focused.
func (c *CollectionTree) Focus() {
	c.focused = true
}

// Blur removes focus.
func (c *CollectionTree) Blur() {
	c.focused = false
}

// IsSearching returns true if the tree is in search mode.
func (c *CollectionTree) IsSearching() bool {
	return c.searching
}

// SetSize sets dimensions.
func (c *CollectionTree) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Width returns the width.
func (c *CollectionTree) Width() int {
	return c.width
}

// Height returns the height.
func (c *CollectionTree) Height() int {
	return c.height
}

// SetCollections sets the collections to display.
func (c *CollectionTree) SetCollections(collections []*core.Collection) {
	c.collections = collections
	c.rebuildItems()
	c.cursor = 0
}

// Collections returns the current collections.
func (c *CollectionTree) Collections() []*core.Collection {
	return c.collections
}

// AddRequest adds a new request to the specified collection (or first collection if nil).
// Returns true if the request was added successfully.
func (c *CollectionTree) AddRequest(req *core.RequestDefinition, collection *core.Collection) bool {
	if req == nil {
		return false
	}

	// Use specified collection or find a suitable one
	var targetCollection *core.Collection
	if collection != nil {
		targetCollection = collection
	} else if len(c.collections) > 0 {
		targetCollection = c.collections[0]
	} else {
		// No collections exist - create a default one
		targetCollection = core.NewCollection("Default")
		c.collections = append(c.collections, targetCollection)
	}

	// Add request to collection
	targetCollection.AddRequest(req)

	// Expand the collection so the new request is visible
	c.expanded[targetCollection.ID()] = true

	// Rebuild tree to show the new request
	c.rebuildItems()

	// Find and select the new request
	for i, item := range c.items {
		if item.Type == ItemRequest && item.Request != nil && item.Request.ID() == req.ID() {
			c.cursor = i
			break
		}
	}

	return true
}

// SetHistoryStore sets the history store for browsing request history.
func (c *CollectionTree) SetHistoryStore(store history.Store) {
	c.historyStore = store
}

// ViewMode returns the current view mode.
func (c *CollectionTree) ViewMode() ViewMode {
	return c.viewMode
}

// ItemCount returns the total number of items.
func (c *CollectionTree) ItemCount() int {
	return len(c.items)
}

// VisibleItemCount returns the number of visible items after filtering.
func (c *CollectionTree) VisibleItemCount() int {
	if c.search == "" {
		return len(c.items)
	}

	count := 0
	search := strings.ToLower(c.search)
	for _, item := range c.items {
		if strings.Contains(strings.ToLower(item.Name), search) {
			count++
		}
	}
	return count
}

// Cursor returns the current cursor position.
func (c *CollectionTree) Cursor() int {
	return c.cursor
}

// SetCursor sets the cursor position.
func (c *CollectionTree) SetCursor(pos int) {
	c.cursor = pos
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}
}

// Selected returns the currently selected item.
func (c *CollectionTree) Selected() *TreeItem {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return nil
	}
	return &c.items[c.cursor]
}

// IsExpanded returns true if the item at index is expanded.
func (c *CollectionTree) IsExpanded(index int) bool {
	if index < 0 || index >= len(c.items) {
		return false
	}
	return c.items[index].Expanded
}

// Expand expands the item at index.
func (c *CollectionTree) Expand(index int) {
	if index < 0 || index >= len(c.items) {
		return
	}
	item := c.items[index]
	if !item.Expandable {
		return
	}
	c.expanded[item.ID] = true
	c.rebuildItems()
}

// Collapse collapses the item at index.
func (c *CollectionTree) Collapse(index int) {
	if index < 0 || index >= len(c.items) {
		return
	}
	item := c.items[index]
	c.expanded[item.ID] = false
	c.rebuildItems()
}

// SetSearch sets the search filter.
func (c *CollectionTree) SetSearch(query string) {
	c.search = query
}

// ClearSearch clears the search filter.
func (c *CollectionTree) ClearSearch() {
	c.search = ""
}

func (c *CollectionTree) rebuildItems() {
	c.items = nil

	for _, coll := range c.collections {
		c.addCollectionItems(coll, 0)
	}
}

func (c *CollectionTree) addCollectionItems(coll *core.Collection, level int) {
	id := coll.ID()
	expanded := c.expanded[id]
	hasChildren := len(coll.Folders()) > 0 || len(coll.Requests()) > 0 || len(coll.WebSockets()) > 0

	c.items = append(c.items, TreeItem{
		ID:         id,
		Name:       coll.Name(),
		Type:       ItemCollection,
		Level:      level,
		Expandable: hasChildren,
		Expanded:   expanded,
		Collection: coll,
	})

	if expanded {
		// Add folders
		for _, folder := range coll.Folders() {
			c.addFolderItems(folder, level+1)
		}

		// Add requests at root level
		for _, req := range coll.Requests() {
			c.items = append(c.items, TreeItem{
				ID:      req.ID(),
				Name:    req.Name(),
				Type:    ItemRequest,
				Level:   level + 1,
				Method:  req.Method(),
				Request: req,
			})
		}

		// Add WebSocket definitions
		for _, ws := range coll.WebSockets() {
			c.items = append(c.items, TreeItem{
				ID:        ws.ID,
				Name:      ws.Name,
				Type:      ItemWebSocket,
				Level:     level + 1,
				Method:    "WS",
				WebSocket: ws,
			})
		}
	}
}

func (c *CollectionTree) addFolderItems(folder *core.Folder, level int) {
	id := folder.ID()
	expanded := c.expanded[id]
	hasChildren := len(folder.Folders()) > 0 || len(folder.Requests()) > 0

	c.items = append(c.items, TreeItem{
		ID:         id,
		Name:       folder.Name(),
		Type:       ItemFolder,
		Level:      level,
		Expandable: hasChildren,
		Expanded:   expanded,
		Folder:     folder,
	})

	if expanded {
		// Add nested folders
		for _, subFolder := range folder.Folders() {
			c.addFolderItems(subFolder, level+1)
		}

		// Add requests
		for _, req := range folder.Requests() {
			c.items = append(c.items, TreeItem{
				ID:      req.ID(),
				Name:    req.Name(),
				Type:    ItemRequest,
				Level:   level + 1,
				Method:  req.Method(),
				Request: req,
			})
		}
	}
}

// --- State accessors for E2E testing ---

// ViewModeName returns the view mode as a string.
func (c *CollectionTree) ViewModeName() string {
	if c.viewMode == ViewHistory {
		return "history"
	}
	return "collections"
}

// SearchQuery returns the current search query.
func (c *CollectionTree) SearchQuery() string {
	return c.search
}

// HistoryEntries returns the current history entries.
func (c *CollectionTree) HistoryEntries() []history.Entry {
	return c.historyEntries
}

// GPressed returns true if waiting for second 'g' in gg sequence.
func (c *CollectionTree) GPressed() bool {
	return c.gPressed
}

// HistoryCursor returns the current history cursor position.
func (c *CollectionTree) HistoryCursor() int {
	return c.historyCursor
}
