package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/tui"
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
	Method      string
}

// TreeItemType identifies the type of tree item.
type TreeItemType int

const (
	ItemCollection TreeItemType = iota
	ItemFolder
	ItemRequest
)

// SelectionMsg is sent when a request is selected.
type SelectionMsg struct {
	Request *core.RequestDefinition
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

func (c *CollectionTree) handleSearchInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Exit search mode but keep filter
		c.searching = false
		return c, nil

	case tea.KeyEnter:
		// Exit search mode and keep filter
		c.searching = false
		return c, nil

	case tea.KeyBackspace:
		if len(c.search) > 0 {
			c.search = c.search[:len(c.search)-1]
			c.applyFilter()
		}
		return c, nil

	case tea.KeyCtrlU:
		// Clear search
		c.search = ""
		c.applyFilter()
		return c, nil

	case tea.KeyRunes:
		c.search += string(msg.Runes)
		c.applyFilter()
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
	height := c.height - 4 // Title + borders
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

	// Title
	titleStyle := lipgloss.NewStyle().
		Width(c.width - 2).
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

	title := titleStyle.Render(c.title)

	// Search bar (if searching or has active filter)
	var searchBar string
	if c.searching || c.search != "" {
		searchBar = c.renderSearchBar()
	}

	// Content
	contentHeight := c.contentHeight()

	displayItems := c.getDisplayItems()
	var lines []string
	for i := c.offset; i < len(displayItems) && len(lines) < contentHeight; i++ {
		item := displayItems[i]
		line := c.renderItem(item, i == c.cursor)
		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	for len(lines) < contentHeight {
		lines = append(lines, strings.Repeat(" ", c.width-2))
	}

	content := strings.Join(lines, "\n")

	// Combine all parts
	var parts []string
	parts = append(parts, title)
	if searchBar != "" {
		parts = append(parts, searchBar)
	}
	parts = append(parts, content)

	// Border
	borderStyle := lipgloss.NewStyle().
		Width(c.width).
		Height(c.height).
		BorderStyle(lipgloss.RoundedBorder())

	if c.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240"))
	}

	return borderStyle.Render(strings.Join(parts, "\n"))
}

func (c *CollectionTree) renderSearchBar() string {
	width := c.width - 4

	// Search icon and input
	searchIcon := "🔍 "
	query := c.search

	// Cursor indicator when in search mode
	cursor := ""
	if c.searching {
		cursor = "▌"
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

	// Truncate if too long
	if len(content) > width {
		content = content[:width-3] + "..."
	}

	// Pad to full width
	if len(content) < width {
		content += strings.Repeat(" ", width-len(content))
	}

	return style.Render(content)
}

func (c *CollectionTree) renderItem(item TreeItem, selected bool) string {
	width := c.width - 4 // Account for borders and padding

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
	if len(name) > availableWidth {
		name = name[:availableWidth-3] + "..."
	}

	line := indent + indicator + icon + name

	// Pad to full width
	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	}

	// Apply selection styling
	style := lipgloss.NewStyle()
	if selected && c.focused {
		style = style.
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("229"))
	}

	return style.Render(line)
}

func (c *CollectionTree) methodBadge(method string) string {
	style := lipgloss.NewStyle().Bold(true)

	switch strings.ToUpper(method) {
	case "GET":
		return style.Foreground(lipgloss.Color("34")).Render("GET ")
	case "POST":
		return style.Foreground(lipgloss.Color("214")).Render("POST")
	case "PUT":
		return style.Foreground(lipgloss.Color("33")).Render("PUT ")
	case "PATCH":
		return style.Foreground(lipgloss.Color("141")).Render("PTCH")
	case "DELETE":
		return style.Foreground(lipgloss.Color("160")).Render("DEL ")
	case "HEAD":
		return style.Foreground(lipgloss.Color("245")).Render("HEAD")
	case "OPTIONS":
		return style.Foreground(lipgloss.Color("245")).Render("OPT ")
	default:
		return style.Foreground(lipgloss.Color("245")).Render(fmt.Sprintf("%-4s", method))
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
	hasChildren := len(coll.Folders()) > 0 || len(coll.Requests()) > 0

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
