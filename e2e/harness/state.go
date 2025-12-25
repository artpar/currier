package harness

import (
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/tui/views"
)

// State represents a snapshot of the entire TUI state for verification.
type State struct {
	MainView *MainViewState
	Request  *RequestPanelState
	Response *ResponsePanelState
	Tree     *CollectionTreeState
}

// MainViewState captures the main view state.
type MainViewState struct {
	FocusedPane  string // "collections", "request", "response"
	Mode         string // "NORMAL", "INSERT", "METHOD", "SEARCH"
	ShowingHelp  bool
	Notification string
}

// RequestPanelState captures the request panel state.
type RequestPanelState struct {
	HasRequest   bool
	URL          string
	Method       string
	Headers      map[string]string
	QueryParams  map[string]string
	Body         string
	ActiveTab    string
	IsEditing    bool
	EditingField string // "url", "header_key", "header_value", "query_key", "query_value", "body", ""
	CursorPos    int
}

// ResponsePanelState captures the response panel state.
type ResponsePanelState struct {
	HasResponse  bool
	StatusCode   int
	StatusText   string
	ResponseTime int64 // milliseconds
	BodySize     int64
	BodyPreview  string // First N chars of body
	ActiveTab    string
	IsLoading    bool
	Error        string
}

// CollectionTreeState captures the collection tree state.
type CollectionTreeState struct {
	ViewMode       string // "collections" or "history"
	ItemCount      int
	SelectedIndex  int
	SelectedType   string // "collection", "folder", "request"
	SelectedName   string
	IsSearching    bool
	SearchQuery    string
	HistoryCount   int
	HistoryEntries []HistoryEntryInfo
}

// HistoryEntryInfo is a summary of a history entry.
type HistoryEntryInfo struct {
	Method     string
	URL        string
	StatusCode int
}

// CaptureState captures the current state of the TUI session.
func (s *TUISession) CaptureState() *State {
	state := &State{
		MainView: s.captureMainViewState(),
		Request:  s.captureRequestPanelState(),
		Response: s.captureResponsePanelState(),
		Tree:     s.captureCollectionTreeState(),
	}
	return state
}

func (s *TUISession) captureMainViewState() *MainViewState {
	mv := s.model

	// Determine mode from component states
	mode := "NORMAL"
	if mv.RequestPanel().IsEditing() {
		mode = "INSERT"
	} else if mv.RequestPanel().IsEditingMethod() {
		mode = "METHOD"
	} else if mv.CollectionTree().IsSearching() {
		mode = "SEARCH"
	}

	// Determine focused pane
	focusedPane := "collections"
	switch mv.FocusedPane() {
	case views.PaneCollections:
		focusedPane = "collections"
	case views.PaneRequest:
		focusedPane = "request"
	case views.PaneResponse:
		focusedPane = "response"
	}

	return &MainViewState{
		FocusedPane:  focusedPane,
		Mode:         mode,
		ShowingHelp:  mv.ShowingHelp(),
		Notification: mv.Notification(),
	}
}

func (s *TUISession) captureRequestPanelState() *RequestPanelState {
	rp := s.model.RequestPanel()

	state := &RequestPanelState{
		HasRequest:   rp.HasRequest(),
		ActiveTab:    rp.ActiveTabName(),
		IsEditing:    rp.IsEditing(),
		EditingField: rp.EditingField(),
		CursorPos:    rp.CursorPosition(),
	}

	if rp.HasRequest() {
		state.URL = rp.URL()
		state.Method = rp.Method()
		state.Headers = rp.HeadersMap()
		state.QueryParams = rp.QueryParamsMap()
		state.Body = rp.Body()
	}

	return state
}

func (s *TUISession) captureResponsePanelState() *ResponsePanelState {
	resp := s.model.ResponsePanel()

	state := &ResponsePanelState{
		HasResponse: resp.HasResponse(),
		IsLoading:   resp.IsLoading(),
		ActiveTab:   resp.ActiveTabName(),
		Error:       resp.ErrorString(),
	}

	if resp.HasResponse() {
		state.StatusCode = resp.StatusCode()
		state.StatusText = resp.StatusText()
		state.ResponseTime = resp.ResponseTime()
		state.BodySize = resp.BodySize()
		state.BodyPreview = resp.BodyPreview(500)
	}

	return state
}

func (s *TUISession) captureCollectionTreeState() *CollectionTreeState {
	tree := s.model.CollectionTree()

	state := &CollectionTreeState{
		ViewMode:      tree.ViewModeName(),
		ItemCount:     tree.ItemCount(),
		SelectedIndex: tree.Cursor(),
		IsSearching:   tree.IsSearching(),
		SearchQuery:   tree.SearchQuery(),
	}

	// Capture selected item info
	if selected := tree.Selected(); selected != nil {
		state.SelectedName = selected.Name
		switch selected.Type {
		case 0:
			state.SelectedType = "collection"
		case 1:
			state.SelectedType = "folder"
		case 2:
			state.SelectedType = "request"
		}
	}

	// Capture history entries if in history mode
	if tree.ViewModeName() == "history" {
		entries := tree.HistoryEntries()
		state.HistoryCount = len(entries)
		for _, e := range entries {
			state.HistoryEntries = append(state.HistoryEntries, HistoryEntryInfo{
				Method:     e.RequestMethod,
				URL:        e.RequestURL,
				StatusCode: e.ResponseStatus,
			})
		}
	}

	return state
}

// State returns the current state (alias for CaptureState).
func (s *TUISession) State() *State {
	return s.CaptureState()
}

// DBVerifier provides database verification capabilities.
type DBVerifier struct {
	store history.Store
}

// NewDBVerifier creates a new database verifier.
func NewDBVerifier(store history.Store) *DBVerifier {
	return &DBVerifier{store: store}
}

// EntryCount returns the number of history entries.
func (d *DBVerifier) EntryCount() int {
	if d.store == nil {
		return 0
	}
	// This would need a Count method on the store
	return 0
}

// HasEntryWithURL checks if an entry with the given URL exists.
func (d *DBVerifier) HasEntryWithURL(url string) bool {
	if d.store == nil {
		return false
	}
	// This would need a query method on the store
	return false
}
