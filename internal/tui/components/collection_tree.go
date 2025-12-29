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

// DeleteRequestMsg is sent when a request is deleted from a collection.
type DeleteRequestMsg struct {
	Collection *core.Collection
	RequestID  string
}

// SaveToCollectionMsg is sent when a request is saved to a collection.
type SaveToCollectionMsg struct {
	Collection *core.Collection
	Request    *core.RequestDefinition
}

// CreateCollectionMsg is sent when a new collection is created.
type CreateCollectionMsg struct {
	Collection *core.Collection
}

// DeleteCollectionMsg is sent when a collection is deleted.
type DeleteCollectionMsg struct {
	CollectionID string
}

// RenameCollectionMsg is sent when a collection is renamed.
type RenameCollectionMsg struct {
	Collection *core.Collection
}

// MoveRequestMsg is sent when a request is moved between collections or into folders.
type MoveRequestMsg struct {
	SourceCollection *core.Collection
	TargetCollection *core.Collection
	TargetFolder     *core.Folder // nil if moved to collection root
	Request          *core.RequestDefinition
}

// DuplicateRequestMsg is sent when a request is duplicated.
type DuplicateRequestMsg struct {
	Collection *core.Collection
	Request    *core.RequestDefinition
}

// RenameRequestMsg is sent when a request is renamed.
type RenameRequestMsg struct {
	Collection *core.Collection
	Request    *core.RequestDefinition
}

// CreateFolderMsg is sent when a folder is created in a collection.
type CreateFolderMsg struct {
	Collection *core.Collection
	Folder     *core.Folder
}

// RenameFolderMsg is sent when a folder is renamed.
type RenameFolderMsg struct {
	Collection *core.Collection
	Folder     *core.Folder
}

// DeleteFolderMsg is sent when a folder is deleted.
type DeleteFolderMsg struct {
	Collection *core.Collection
	FolderID   string
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

	// Rename mode
	renaming        bool   // True when renaming a collection, folder, or request
	renameBuffer    string // Buffer for the new name
	renamingCollID  string // ID of collection being renamed or containing the item
	renamingReqID   string // ID of request being renamed
	renamingFolderID string // ID of folder being renamed

	// Move mode
	moving          bool                    // True when moving a request
	movingRequest   *core.RequestDefinition // Request being moved
	sourceCollID    string                  // Source collection ID
	moveTargets     []TreeItem              // Collections available as move targets
	moveCursor      int                     // Cursor for selecting target collection

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
		viewMode: ViewHistory,
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

	// Handle rename mode input
	if c.renaming {
		return c.handleRenameInput(msg)
	}

	// Handle move mode input
	if c.moving {
		return c.handleMoveInput(msg)
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
		c.gPressed = false
		return c, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			c.searching = true
			c.search = ""
			c.gPressed = false
			return c, nil
		case "j":
			c.moveCursorPos(1)
		case "k":
			c.moveCursorPos(-1)
		case "l":
			c.expandCurrent()
		case "h":
			c.collapseCurrent()
		case "H":
			// Switch to History section
			c.viewMode = ViewHistory
			c.loadHistory()
			c.gPressed = false
			return c, nil
		case "d":
			// Delete selected request
			c.gPressed = false
			return c.handleDeleteRequest()
		case "N":
			// Create new collection
			c.gPressed = false
			return c.handleCreateCollection()
		case "F":
			// Create new folder in current collection
			c.gPressed = false
			return c.handleCreateFolder()
		case "D":
			// Delete selected collection or folder
			c.gPressed = false
			displayItems := c.getDisplayItems()
			if c.cursor >= 0 && c.cursor < len(displayItems) {
				item := displayItems[c.cursor]
				if item.Type == ItemFolder {
					return c.handleDeleteFolder()
				}
			}
			return c.handleDeleteCollection()
		case "r":
			// Rename selected collection
			c.gPressed = false
			return c.startRename()
		case "R":
			// Rename selected request
			c.gPressed = false
			return c.startRenameRequest()
		case "m":
			// Move request to another collection
			c.gPressed = false
			return c.startMove()
		case "y":
			// Duplicate/copy request
			c.gPressed = false
			return c.handleDuplicateRequest()
		case "G":
			displayItems := c.getDisplayItems()
			if len(displayItems) > 0 {
				c.cursor = len(displayItems) - 1
				// Adjust offset to ensure cursor is visible
				visibleHeight := c.contentHeight()
				if c.cursor >= c.offset+visibleHeight {
					c.offset = c.cursor - visibleHeight + 1
				}
			}
			c.gPressed = false
		case "g":
			if c.gPressed {
				c.cursor = 0
				c.offset = 0
				c.gPressed = false
			} else {
				c.gPressed = true
			}
			return c, nil
		default:
			c.gPressed = false
		}

	case tea.KeyEnter:
		c.gPressed = false
		return c.handleEnter()
	}

	c.gPressed = false
	return c, nil
}

func (c *CollectionTree) handleHistoryKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		c.gPressed = false
		// If searching, clear search filter first
		if c.historySearch != "" {
			c.historySearch = ""
			c.loadHistory()
			return c, nil
		}
		// Otherwise, switch to Collections section
		c.viewMode = ViewCollections
		return c, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			c.searching = true
			c.historySearch = ""
			c.gPressed = false
			return c, nil
		case "j":
			c.moveHistoryCursor(1)
		case "k":
			c.moveHistoryCursor(-1)
		case "C":
			// Switch to Collections section
			c.viewMode = ViewCollections
			c.gPressed = false
			return c, nil
		case "H":
			// Toggle - switch to Collections section
			c.viewMode = ViewCollections
			c.gPressed = false
			return c, nil
		case "h", "l":
			// No-op in history view (no expand/collapse) but handle gracefully
			c.gPressed = false
			return c, nil
		case "G":
			if len(c.historyEntries) > 0 {
				c.historyCursor = len(c.historyEntries) - 1
				// Adjust offset to ensure cursor is visible
				visibleHeight := c.historyContentHeight()
				if c.historyCursor >= c.historyOffset+visibleHeight {
					c.historyOffset = c.historyCursor - visibleHeight + 1
				}
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
			c.gPressed = false
			return c, nil
		default:
			c.gPressed = false
		}

	case tea.KeyEnter:
		c.gPressed = false
		return c.handleHistoryEnter()
	}

	c.gPressed = false
	return c, nil
}

func (c *CollectionTree) moveHistoryCursor(delta int) {
	// Use pure functions - explicit state changes
	c.historyCursor = MoveCursor(c.historyCursor, delta, len(c.historyEntries))
	c.historyOffset = AdjustOffset(c.historyCursor, c.historyOffset, c.historyContentHeight())
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
		// Toggle expand/collapse using pure function
		c.expanded = ToggleExpand(c.expanded, item.ID, !item.Expanded)
		c.rebuildItems()
		// Only refilter if search is active
		if c.search != "" {
			c.filteredItems = FilterItemsBySearch(c.items, c.search)
		}
		// cursor/offset unchanged - explicit
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

func (c *CollectionTree) handleDeleteRequest() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only delete requests
	if item.Type != ItemRequest {
		return c, nil
	}

	requestID := item.Request.ID()

	// Find and remove the request from its collection
	modifiedCollection := c.DeleteRequest(requestID)
	if modifiedCollection == nil {
		return c, nil
	}

	// Emit message for persistence
	return c, func() tea.Msg {
		return DeleteRequestMsg{
			Collection: modifiedCollection,
			RequestID:  requestID,
		}
	}
}

func (c *CollectionTree) handleDuplicateRequest() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only duplicate requests
	if item.Type != ItemRequest {
		return c, nil
	}

	// Find the collection containing this request
	var targetColl *core.Collection
	for _, coll := range c.collections {
		if _, found := coll.FindRequest(item.Request.ID()); found {
			targetColl = coll
			break
		}
	}

	if targetColl == nil {
		return c, nil
	}

	// Clone the request with a new name
	cloned := item.Request.Clone()
	cloned.SetName(item.Request.Name() + " (Copy)")

	// Add to the same collection
	targetColl.AddRequest(cloned)
	c.rebuildItems()

	// Move cursor to the new request
	for i, treeItem := range c.items {
		if treeItem.Type == ItemRequest && treeItem.Request != nil && treeItem.Request.ID() == cloned.ID() {
			c.cursor = i
			c.offset = AdjustOffset(c.cursor, c.offset, c.contentHeight())
			break
		}
	}

	// Emit message for persistence
	return c, func() tea.Msg {
		return DuplicateRequestMsg{
			Collection: targetColl,
			Request:    cloned,
		}
	}
}

func (c *CollectionTree) handleCreateCollection() (tui.Component, tea.Cmd) {
	// Create a new collection with a default name
	newCollection := core.NewCollection("New Collection")

	// Add to collections list
	c.collections = append(c.collections, newCollection)
	c.rebuildItems()

	// Move cursor to the new collection (last item at level 0)
	for i, item := range c.items {
		if item.ID == newCollection.ID() {
			c.cursor = i
			break
		}
	}

	// Emit message for persistence
	return c, func() tea.Msg {
		return CreateCollectionMsg{
			Collection: newCollection,
		}
	}
}

func (c *CollectionTree) handleCreateFolder() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Find the collection to add folder to
	var targetColl *core.Collection

	switch item.Type {
	case ItemCollection:
		// If on a collection, add folder to it
		targetColl = item.Collection
	case ItemFolder:
		// If on a folder, find its parent collection
		for _, coll := range c.collections {
			if coll.FindFolder(item.ID) != nil {
				targetColl = coll
				break
			}
		}
	case ItemRequest:
		// If on a request, find its parent collection
		for _, coll := range c.collections {
			if _, found := coll.FindRequest(item.Request.ID()); found {
				targetColl = coll
				break
			}
		}
	}

	if targetColl == nil {
		return c, nil
	}

	// Create new folder
	newFolder := targetColl.AddFolder("New Folder")

	// Expand the collection so the new folder is visible
	c.expanded[targetColl.ID()] = true
	c.rebuildItems()

	// Move cursor to the new folder
	for i, treeItem := range c.items {
		if treeItem.Type == ItemFolder && treeItem.ID == newFolder.ID() {
			c.cursor = i
			c.offset = AdjustOffset(c.cursor, c.offset, c.contentHeight())
			break
		}
	}

	// Emit message for persistence
	return c, func() tea.Msg {
		return CreateFolderMsg{
			Collection: targetColl,
			Folder:     newFolder,
		}
	}
}

func (c *CollectionTree) handleDeleteCollection() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only delete collections (not folders or requests)
	if item.Type != ItemCollection {
		return c, nil
	}

	collectionID := item.ID

	// Remove from collections list
	for i, coll := range c.collections {
		if coll.ID() == collectionID {
			c.collections = append(c.collections[:i], c.collections[i+1:]...)
			break
		}
	}

	c.rebuildItems()

	// Adjust cursor if needed
	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}

	// Emit message for persistence (delete from disk)
	return c, func() tea.Msg {
		return DeleteCollectionMsg{
			CollectionID: collectionID,
		}
	}
}

func (c *CollectionTree) handleDeleteFolder() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only delete folders
	if item.Type != ItemFolder {
		return c, nil
	}

	folderID := item.ID

	// Find which collection contains this folder
	var modifiedColl *core.Collection
	for _, coll := range c.collections {
		if coll.RemoveFolderRecursive(folderID) {
			modifiedColl = coll
			break
		}
	}

	if modifiedColl == nil {
		return c, nil
	}

	c.rebuildItems()

	// Adjust cursor if needed
	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}

	// Emit message for persistence
	return c, func() tea.Msg {
		return DeleteFolderMsg{
			Collection: modifiedColl,
			FolderID:   folderID,
		}
	}
}

func (c *CollectionTree) startRename() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only rename collections
	if item.Type != ItemCollection {
		return c, nil
	}

	// Start rename mode with current name
	c.renaming = true
	c.renamingCollID = item.ID
	c.renamingReqID = ""
	c.renameBuffer = item.Name

	return c, nil
}

func (c *CollectionTree) startRenameRequest() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Handle requests
	if item.Type == ItemRequest {
		// Find the collection containing this request
		var collID string
		for _, coll := range c.collections {
			if _, found := coll.FindRequest(item.Request.ID()); found {
				collID = coll.ID()
				break
			}
		}

		if collID == "" {
			return c, nil
		}

		// Start rename mode with current name
		c.renaming = true
		c.renamingCollID = collID
		c.renamingReqID = item.Request.ID()
		c.renamingFolderID = ""
		c.renameBuffer = item.Request.Name()

		return c, nil
	}

	// Handle folders
	if item.Type == ItemFolder {
		// Find the collection containing this folder
		var collID string
		for _, coll := range c.collections {
			if coll.FindFolder(item.ID) != nil {
				collID = coll.ID()
				break
			}
		}

		if collID == "" {
			return c, nil
		}

		// Start rename mode with current name
		c.renaming = true
		c.renamingCollID = collID
		c.renamingReqID = ""
		c.renamingFolderID = item.ID
		c.renameBuffer = item.Name

		return c, nil
	}

	return c, nil
}

func (c *CollectionTree) handleRenameInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel rename
		c.renaming = false
		c.renameBuffer = ""
		c.renamingCollID = ""
		c.renamingReqID = ""
		c.renamingFolderID = ""
		return c, nil

	case tea.KeyEnter:
		// Confirm rename
		if c.renameBuffer == "" {
			// Don't allow empty names
			return c, nil
		}

		// Check if we're renaming a request
		if c.renamingReqID != "" {
			// Renaming a request
			var renamedColl *core.Collection
			var renamedReq *core.RequestDefinition
			for _, coll := range c.collections {
				if coll.ID() == c.renamingCollID {
					if req, found := coll.FindRequest(c.renamingReqID); found {
						req.SetName(c.renameBuffer)
						renamedColl = coll
						renamedReq = req
					}
					break
				}
			}

			c.renaming = false
			c.renameBuffer = ""
			c.renamingCollID = ""
			c.renamingReqID = ""
			c.renamingFolderID = ""
			c.rebuildItems()

			if renamedColl != nil && renamedReq != nil {
				return c, func() tea.Msg {
					return RenameRequestMsg{Collection: renamedColl, Request: renamedReq}
				}
			}
			return c, nil
		}

		// Check if we're renaming a folder
		if c.renamingFolderID != "" {
			var renamedColl *core.Collection
			var renamedFolder *core.Folder
			for _, coll := range c.collections {
				if coll.ID() == c.renamingCollID {
					if folder := coll.FindFolder(c.renamingFolderID); folder != nil {
						folder.SetName(c.renameBuffer)
						renamedColl = coll
						renamedFolder = folder
					}
					break
				}
			}

			c.renaming = false
			c.renameBuffer = ""
			c.renamingCollID = ""
			c.renamingReqID = ""
			c.renamingFolderID = ""
			c.rebuildItems()

			if renamedColl != nil && renamedFolder != nil {
				return c, func() tea.Msg {
					return RenameFolderMsg{Collection: renamedColl, Folder: renamedFolder}
				}
			}
			return c, nil
		}

		// Renaming a collection
		var renamedColl *core.Collection
		for _, coll := range c.collections {
			if coll.ID() == c.renamingCollID {
				coll.SetName(c.renameBuffer)
				renamedColl = coll
				break
			}
		}

		c.renaming = false
		c.renameBuffer = ""
		c.renamingCollID = ""
		c.renamingReqID = ""
		c.renamingFolderID = ""
		c.rebuildItems()

		if renamedColl != nil {
			return c, func() tea.Msg {
				return RenameCollectionMsg{Collection: renamedColl}
			}
		}
		return c, nil

	case tea.KeyBackspace:
		if len(c.renameBuffer) > 0 {
			c.renameBuffer = c.renameBuffer[:len(c.renameBuffer)-1]
		}
		return c, nil

	case tea.KeyRunes:
		c.renameBuffer += string(msg.Runes)
		return c, nil

	case tea.KeySpace:
		c.renameBuffer += " "
		return c, nil
	}

	return c, nil
}

// IsRenaming returns true if currently in rename mode.
func (c *CollectionTree) IsRenaming() bool {
	return c.renaming
}

// IsMoving returns true if currently in move mode.
func (c *CollectionTree) IsMoving() bool {
	return c.moving
}

func (c *CollectionTree) startMove() (tui.Component, tea.Cmd) {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return c, nil
	}

	item := displayItems[c.cursor]

	// Only move requests
	if item.Type != ItemRequest {
		return c, nil
	}

	// Find source collection
	var sourceCollID string
	for _, coll := range c.collections {
		if req, found := coll.FindRequest(item.Request.ID()); found && req != nil {
			sourceCollID = coll.ID()
			break
		}
	}

	if sourceCollID == "" {
		return c, nil
	}

	// Get move targets (all collections except source)
	targets := c.getMoveTargets(sourceCollID)
	if len(targets) == 0 {
		// No other collections to move to
		return c, nil
	}

	c.moving = true
	c.movingRequest = item.Request
	c.sourceCollID = sourceCollID
	c.moveTargets = targets
	c.moveCursor = 0

	return c, nil
}

func (c *CollectionTree) getMoveTargets(excludeID string) []TreeItem {
	var targets []TreeItem
	for _, coll := range c.collections {
		// Add the collection itself (can move to any collection, including source)
		targets = append(targets, TreeItem{
			ID:         coll.ID(),
			Name:       coll.Name(),
			Type:       ItemCollection,
			Level:      0,
			Collection: coll,
		})
		// Add folders from this collection
		targets = append(targets, c.getFolderTargets(coll, coll.Folders(), 1)...)
	}
	return targets
}

func (c *CollectionTree) getFolderTargets(coll *core.Collection, folders []*core.Folder, level int) []TreeItem {
	var targets []TreeItem
	for _, folder := range folders {
		targets = append(targets, TreeItem{
			ID:         folder.ID(),
			Name:       folder.Name(),
			Type:       ItemFolder,
			Level:      level,
			Collection: coll,
			Folder:     folder,
		})
		// Add nested folders recursively
		targets = append(targets, c.getFolderTargets(coll, folder.Folders(), level+1)...)
	}
	return targets
}

func (c *CollectionTree) handleMoveInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel move
		c.moving = false
		c.movingRequest = nil
		c.sourceCollID = ""
		c.moveTargets = nil
		c.moveCursor = 0
		return c, nil

	case tea.KeyEnter:
		// Confirm move
		if c.moveCursor < 0 || c.moveCursor >= len(c.moveTargets) {
			return c, nil
		}

		targetItem := c.moveTargets[c.moveCursor]
		targetColl := targetItem.Collection
		targetFolder := targetItem.Folder // nil if moving to collection root

		// Find source collection and remove request
		var sourceColl *core.Collection
		for _, coll := range c.collections {
			if coll.ID() == c.sourceCollID {
				sourceColl = coll
				coll.RemoveRequestRecursive(c.movingRequest.ID())
				break
			}
		}

		// Add to target folder or collection root
		if targetFolder != nil {
			targetFolder.AddRequest(c.movingRequest)
		} else {
			targetColl.AddRequest(c.movingRequest)
		}

		movedRequest := c.movingRequest

		// Exit move mode
		c.moving = false
		c.movingRequest = nil
		c.sourceCollID = ""
		c.moveTargets = nil
		c.moveCursor = 0
		c.rebuildItems()

		if sourceColl != nil && targetColl != nil {
			return c, func() tea.Msg {
				return MoveRequestMsg{
					SourceCollection: sourceColl,
					TargetCollection: targetColl,
					TargetFolder:     targetFolder,
					Request:          movedRequest,
				}
			}
		}
		return c, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if c.moveCursor < len(c.moveTargets)-1 {
				c.moveCursor++
			}
		case "k":
			if c.moveCursor > 0 {
				c.moveCursor--
			}
		}
		return c, nil
	}

	// Handle j/k via KeyDown/KeyUp
	switch msg.Type {
	case tea.KeyDown:
		if c.moveCursor < len(c.moveTargets)-1 {
			c.moveCursor++
		}
	case tea.KeyUp:
		if c.moveCursor > 0 {
			c.moveCursor--
		}
	}

	return c, nil
}

func (c *CollectionTree) moveCursorPos(delta int) {
	displayItems := c.getDisplayItems()
	// Use pure functions - explicit state changes
	c.cursor = MoveCursor(c.cursor, delta, len(displayItems))
	c.offset = AdjustOffset(c.cursor, c.offset, c.contentHeight())
}

// getDisplayItems returns the items to display (filtered or all).
func (c *CollectionTree) getDisplayItems() []TreeItem {
	if c.search == "" {
		return c.items
	}
	return c.filteredItems
}

// applyFilter filters items based on search query.
// Note: Only resets cursor/offset when search is active (filter changes visible items).
// When search is empty, cursor/offset are preserved (items unchanged).
func (c *CollectionTree) applyFilter() {
	if c.search == "" {
		c.filteredItems = nil
		// Don't reset cursor/offset - items are the same, just no filter
		return
	}
	// Use pure function - no duplicate logic
	c.filteredItems = FilterItemsBySearch(c.items, c.search)
	// Reset cursor when search changes (filter changes visible items)
	c.cursor = 0
	c.offset = 0
}

// contentHeight returns the height available for content.
func (c *CollectionTree) contentHeight() int {
	// In stacked layout: borders (2) + search (1) + history header (1) + collections header (1)
	// History gets ~30%, Collections gets ~70% of remaining
	innerHeight := c.height - 2
	availableHeight := innerHeight - 3 // headers (2) + search bar (1)
	if availableHeight < 2 {
		availableHeight = 2
	}
	historyHeight := availableHeight * 3 / 10
	if historyHeight < 1 {
		historyHeight = 1
	}
	collectionHeight := availableHeight - historyHeight
	if collectionHeight < 1 {
		collectionHeight = 1
	}
	return collectionHeight
}

// historyContentHeight returns the height available for history content.
func (c *CollectionTree) historyContentHeight() int {
	innerHeight := c.height - 2
	availableHeight := innerHeight - 3 // headers (2) + search bar (1)
	if availableHeight < 2 {
		availableHeight = 2
	}
	historyHeight := availableHeight * 3 / 10
	if historyHeight < 1 {
		historyHeight = 1
	}
	return historyHeight
}

func (c *CollectionTree) expandCurrent() {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return
	}
	item := displayItems[c.cursor]
	if !item.Expandable || item.Expanded {
		return
	}
	// Use pure function for immutable expand toggle
	c.expanded = ToggleExpand(c.expanded, item.ID, true)
	c.rebuildItems()
	// Only refilter if search is active
	if c.search != "" {
		c.filteredItems = FilterItemsBySearch(c.items, c.search)
	}
	// cursor/offset unchanged - explicit
}

func (c *CollectionTree) collapseCurrent() {
	displayItems := c.getDisplayItems()
	if c.cursor < 0 || c.cursor >= len(displayItems) {
		return
	}
	item := displayItems[c.cursor]
	if !item.Expandable || !item.Expanded {
		return
	}
	// Use pure function for immutable expand toggle
	c.expanded = ToggleExpand(c.expanded, item.ID, false)
	c.rebuildItems()
	// Only refilter if search is active
	if c.search != "" {
		c.filteredItems = FilterItemsBySearch(c.items, c.search)
	}
	// cursor/offset unchanged - explicit
}

// View renders the component with stacked History and Collections.
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

	// Render move mode overlay
	if c.moving {
		return c.renderMoveMode(innerWidth, innerHeight)
	}

	// Section header styles
	activeHeaderStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Bold(true)
	inactiveHeaderStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(lipgloss.Color("243"))

	if c.focused {
		activeHeaderStyle = activeHeaderStyle.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62"))
	} else {
		activeHeaderStyle = activeHeaderStyle.
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238"))
	}

	// Always reserve 1 line for search bar (prevents layout jump)
	searchBar := c.renderSearchBar()

	// Calculate heights: 2 headers + search bar (always 1) + content
	// History gets ~30%, Collections gets ~70%
	availableHeight := innerHeight - 3 // subtract headers (2) and search bar (1)
	if availableHeight < 2 {
		availableHeight = 2
	}
	historyHeight := availableHeight * 3 / 10
	if historyHeight < 1 {
		historyHeight = 1
	}
	collectionHeight := availableHeight - historyHeight
	if collectionHeight < 1 {
		collectionHeight = 1
	}

	var parts []string

	// Search bar always at top (space always reserved)
	parts = append(parts, searchBar)

	// History section
	if c.viewMode == ViewHistory {
		parts = append(parts, activeHeaderStyle.Render("History"))
	} else {
		parts = append(parts, inactiveHeaderStyle.Render("History (H)"))
	}
	historyContent := c.renderHistoryContent(innerWidth, historyHeight)
	parts = append(parts, historyContent)

	// Collections section
	if c.viewMode == ViewCollections {
		parts = append(parts, activeHeaderStyle.Render("Collections"))
	} else {
		parts = append(parts, inactiveHeaderStyle.Render("Collections (C)"))
	}
	collectionsContent := c.renderCollectionContent(innerWidth, collectionHeight)
	parts = append(parts, collectionsContent)

	// Border style
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	if c.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("244"))
	}

	return borderStyle.Render(strings.Join(parts, "\n"))
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
	// Selection indicator prefix
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	// Format: [PREFIX][METHOD] URL - status - time ago
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
	// prefix ~2 chars, methodBadge ~6 chars, status ~3 chars, time ~10 chars, spaces ~5 chars
	availableWidth := width - 27
	if availableWidth < 10 {
		availableWidth = 10
	}
	if len(url) > availableWidth {
		url = url[:availableWidth-3] + "..."
	}

	line := fmt.Sprintf("%s%s %s %s %s", prefix, methodBadge, url, statusStr, timeStr)

	// Pad to full width
	if len(line) < width {
		line += strings.Repeat(" ", width-lipgloss.Width(line))
	}

	// Apply selection styling
	style := lipgloss.NewStyle()
	if selected {
		// Only show bright highlight if focused AND this section (history) is active
		if c.focused && c.viewMode == ViewHistory {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		} else {
			// Dimmer highlight when unfocused or section is inactive
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

func (c *CollectionTree) renderMoveMode(innerWidth, innerHeight int) string {
	headerStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62"))

	// Header showing what's being moved
	reqName := "Request"
	if c.movingRequest != nil {
		reqName = c.movingRequest.Name()
		if len(reqName) > innerWidth-15 {
			reqName = reqName[:innerWidth-18] + "..."
		}
	}
	header := headerStyle.Render("Move: " + reqName)

	// Render target list
	var lines []string
	lines = append(lines, header)
	lines = append(lines, "") // Empty line

	for i, target := range c.moveTargets {
		prefix := "  "
		if i == c.moveCursor {
			prefix = "→ "
		}

		// Indentation based on level
		indent := strings.Repeat("  ", target.Level)

		// Icon based on type
		icon := "📁 "
		if target.Type == ItemFolder {
			icon = "📂 "
		}

		line := prefix + indent + icon + target.Name

		// Pad/truncate to width
		if lipgloss.Width(line) > innerWidth {
			// Truncate name while keeping structure
			maxNameLen := innerWidth - lipgloss.Width(prefix+indent+icon) - 3
			if maxNameLen > 0 && len(target.Name) > maxNameLen {
				line = prefix + indent + icon + target.Name[:maxNameLen] + "..."
			}
		}
		if lipgloss.Width(line) < innerWidth {
			line += strings.Repeat(" ", innerWidth-lipgloss.Width(line))
		}

		// Highlight selected
		style := lipgloss.NewStyle()
		if i == c.moveCursor {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		}
		lines = append(lines, style.Render(line))
	}

	// Fill remaining height
	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	// Border
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())
	if c.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("243"))
	}

	content := strings.Join(lines[:innerHeight], "\n")
	return borderStyle.Render(content)
}

func (c *CollectionTree) renderSearchBar() string {
	width := c.width - 2 // Account for borders only

	// Search icon and input
	searchIcon := "/ "
	query := c.search
	if c.viewMode == ViewHistory {
		query = c.historySearch
	}

	// When not searching and no query, show placeholder hint
	placeholder := ""
	if !c.searching && query == "" {
		placeholder = "search..."
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
	var content string
	if placeholder != "" {
		// Show dimmed placeholder when not searching
		placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		content = searchIcon + placeholderStyle.Render(placeholder)
	} else {
		content = searchIcon + query + cursor
	}

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

	// Selection indicator prefix
	selPrefix := " "
	if selected {
		selPrefix = "→"
	}

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

	// Name - show rename buffer if renaming this collection, folder, or request
	name := item.Name
	if c.renaming && item.Type == ItemCollection && item.ID == c.renamingCollID && c.renamingReqID == "" && c.renamingFolderID == "" {
		name = c.renameBuffer + "▏" // Show cursor
	}
	if c.renaming && item.Type == ItemFolder && item.ID == c.renamingFolderID {
		name = c.renameBuffer + "▏" // Show cursor
	}
	if c.renaming && item.Type == ItemRequest && item.Request != nil && item.Request.ID() == c.renamingReqID {
		name = c.renameBuffer + "▏" // Show cursor
	}
	availableWidth := width - len(selPrefix) - len(indent) - len(indicator) - len(icon) - 2
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

	line := selPrefix + indent + indicator + icon + name

	// Pad to full width
	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	}

	// Apply selection styling
	style := lipgloss.NewStyle()
	if selected {
		// Only show bright highlight if focused AND this section (collections) is active
		if c.focused && c.viewMode == ViewCollections {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		} else {
			// Dimmer highlight when unfocused or section is inactive
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
	c.offset = 0
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
			// Ensure cursor is visible by adjusting offset
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
			break
		}
	}

	return true
}

// DeleteRequest removes a request from any collection by ID.
// Returns the collection that was modified (for persistence), or nil if not found.
func (c *CollectionTree) DeleteRequest(id string) *core.Collection {
	// Find which collection contains this request
	for _, coll := range c.collections {
		if coll.RemoveRequestRecursive(id) {
			// Rebuild tree
			c.rebuildItems()
			// Adjust cursor if needed
			if c.cursor >= len(c.items) {
				c.cursor = len(c.items) - 1
			}
			if c.cursor < 0 {
				c.cursor = 0
			}
			return coll
		}
	}
	return nil
}

// GetSelectedItem returns the currently selected tree item, or nil if none.
func (c *CollectionTree) GetSelectedItem() *TreeItem {
	items := c.getDisplayItems()
	if c.cursor >= 0 && c.cursor < len(items) {
		return &items[c.cursor]
	}
	return nil
}

// SetHistoryStore sets the history store for browsing request history.
func (c *CollectionTree) SetHistoryStore(store history.Store) {
	c.historyStore = store
	c.loadHistory() // Load history immediately when store is set
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

// RebuildItems rebuilds the tree items from the collections.
// Call this after modifying collections externally.
func (c *CollectionTree) RebuildItems() {
	c.rebuildItems()
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
