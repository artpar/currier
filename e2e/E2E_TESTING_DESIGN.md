# Systematic TUI E2E Testing Framework

## Problem Statement

Current tests are "unit tests cosplaying as E2E tests":
- Test components in isolation, not user workflows
- Pre-populate data instead of creating through UI
- Only verify rendered output, not internal state or side effects
- Test happy paths only

## Design Principles

1. **User Journey First** - Every test follows an actual user workflow
2. **State Verification** - Tests verify internal state, not just visual output
3. **Side Effect Verification** - Tests verify database, file system, and other side effects
4. **Failure Mode Testing** - Tests verify error handling and edge cases
5. **Coverage Matrix** - Systematic enumeration of what to test

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Test File (e.g., workflow_test.go)       │
│  - Defines user journeys                                     │
│  - Uses fluent API for readability                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Journey Builder                           │
│  - Fluent API for defining test steps                       │
│  - Automatic state snapshots between steps                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    TUI Session (Enhanced)                    │
│  - SendKey/Type (existing)                                  │
│  - State() - access internal component state                │
│  - DB() - access database for verification                  │
│  - Files() - access file system for verification            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Assertions Layer                          │
│  - StateEquals, StateContains                               │
│  - DBHasEntry, DBEntryCount                                 │
│  - FileExists, FileContains                                 │
│  - ModeIs, FocusedPaneIs                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## User Journeys to Test

### Core Workflows

| ID | Journey | Steps | Verifications |
|----|---------|-------|---------------|
| J1 | Create and send request | n → type URL → Enter | Request in collection, response shown, history saved |
| J2 | Edit existing request | Select → e → modify → Esc → Enter | Changes persisted, response updated |
| J3 | Browse history | H → navigate → Enter | Request loaded, can re-send |
| J4 | Search collections | / → type query → Enter | Filtered results, can select |
| J5 | Add headers | Select → ] → a → type → Tab → type → Enter | Headers shown in request |
| J6 | Add query params | Select → ] ] → a → type → Tab → type → Enter | URL updated with params |
| J7 | Edit body | Select → ] ] ] → e → type JSON → Esc | Body persisted |
| J8 | Change method | Select → m → j/k → Enter | Method updated, badge changes |
| J9 | Navigate panes | Tab / 1/2/3 | Focus indicator moves |
| J10 | Error handling | Send invalid URL | Error shown, can recover |

### Input Testing

| ID | Test | Steps | Verifications |
|----|------|-------|---------------|
| I1 | All printable chars | For each char: n → type char | Char appears in URL field |
| I2 | Space character | n → type "hello world" | Space preserved |
| I3 | Special URL chars | n → type "?foo=bar&x=1" | All chars preserved |
| I4 | Cursor navigation | n → type → Left → type | Insertion at cursor |
| I5 | Delete operations | n → type → Backspace | Char deleted |
| I6 | Clear field | n → type → Ctrl+U | Field empty |

### Mode Transitions

| ID | Test | Start Mode | Action | End Mode |
|----|------|------------|--------|----------|
| M1 | Enter edit | NORMAL | e | INSERT |
| M2 | Exit edit | INSERT | Esc | NORMAL |
| M3 | New request | NORMAL | n | INSERT |
| M4 | Method select | NORMAL | m | METHOD |
| M5 | Search mode | NORMAL | / | SEARCH |

---

## State Verification API

```go
// State accessor methods to add to TUI components

// RequestPanel state
type RequestPanelState struct {
    URL         string
    Method      string
    Headers     map[string]string
    QueryParams map[string]string
    Body        string
    ActiveTab   string
    IsEditing   bool
    EditingField string // "url", "header_key", "header_value", "body", ""
    CursorPos   int
}

// CollectionTree state
type CollectionTreeState struct {
    Collections     []CollectionInfo
    SelectedIndex   int
    SelectedItem    *TreeItemInfo
    IsSearching     bool
    SearchQuery     string
    ViewMode        string // "collections" or "history"
    ExpandedNodes   []string
}

// ResponsePanel state
type ResponsePanelState struct {
    HasResponse     bool
    StatusCode      int
    StatusText      string
    ResponseTime    int64
    BodyPreview     string
    ActiveTab       string
    IsLoading       bool
    Error           string
}

// MainView state
type MainViewState struct {
    FocusedPane     string // "collections", "request", "response"
    Mode            string // "NORMAL", "INSERT", "METHOD", "SEARCH"
    ShowingHelp     bool
    Notification    string
}
```

---

## Side Effect Verification API

```go
// Database verification
type DBVerifier struct {
    store history.Store
}

func (d *DBVerifier) EntryCount() int
func (d *DBVerifier) HasEntryWithURL(url string) bool
func (d *DBVerifier) LatestEntry() *history.Entry
func (d *DBVerifier) EntriesAfter(t time.Time) []history.Entry

// File system verification (for future collection persistence)
type FSVerifier struct {
    basePath string
}

func (f *FSVerifier) FileExists(path string) bool
func (f *FSVerifier) FileContains(path, content string) bool
func (f *FSVerifier) CollectionFile(name string) string
```

---

## Fluent Test API

```go
func TestJourney_CreateAndSendRequest(t *testing.T) {
    journey := e2e.NewJourney(t, "Create and send request").
        WithCleanDB().
        WithMockServer()

    journey.
        // Step 1: Create new request
        Step("Create new request").
            SendKey("n").
            ExpectMode("INSERT").
            ExpectFocus("request").
            ExpectState(func(s *State) {
                assert.Empty(t, s.Request.URL)
                assert.True(t, s.Request.IsEditing)
            }).

        // Step 2: Type URL
        Step("Enter URL").
            Type("https://httpbin.org/get").
            ExpectState(func(s *State) {
                assert.Equal(t, "https://httpbin.org/get", s.Request.URL)
            }).

        // Step 3: Send request
        Step("Send request").
            SendKey("Escape").  // Exit edit mode
            SendKey("Enter").   // Send
            WaitFor(func(s *State) bool {
                return s.Response.HasResponse || s.Response.Error != ""
            }, 5*time.Second).
            ExpectState(func(s *State) {
                assert.True(t, s.Response.HasResponse)
                assert.Equal(t, 200, s.Response.StatusCode)
            }).

        // Step 4: Verify history
        Step("Verify history saved").
            ExpectDB(func(db *DBVerifier) {
                assert.Equal(t, 1, db.EntryCount())
                assert.True(t, db.HasEntryWithURL("https://httpbin.org/get"))
            }).

        // Step 5: View history
        Step("Switch to history view").
            SendKey("H").
            ExpectState(func(s *State) {
                assert.Equal(t, "history", s.Tree.ViewMode)
                assert.GreaterOrEqual(t, len(s.Tree.HistoryEntries), 1)
            })

    journey.Run()
}
```

---

## Coverage Matrix

### Feature × Action Matrix

| Feature | Create | Read | Update | Delete | Navigate |
|---------|--------|------|--------|--------|----------|
| Request | J1 | J2 | J2 | - | J9 |
| Headers | J5 | J5 | J5 | J5 | J5 |
| Query Params | J6 | J6 | J6 | J6 | J6 |
| Body | J7 | J7 | J7 | J7 | - |
| Collection | - | J4 | - | - | J4 |
| History | J1 | J3 | - | - | J3 |

### Input × Field Matrix

| Input | URL | Header Key | Header Value | Query Key | Query Value | Body |
|-------|-----|------------|--------------|-----------|-------------|------|
| Printable | I1 | I1 | I1 | I1 | I1 | I1 |
| Space | I2 | I2 | I2 | I2 | I2 | I2 |
| Special | I3 | I3 | I3 | I3 | I3 | I3 |
| Cursor Nav | I4 | I4 | I4 | I4 | I4 | I4 |
| Delete | I5 | I5 | I5 | I5 | I5 | I5 |
| Clear | I6 | I6 | I6 | I6 | I6 | I6 |

---

## Implementation Plan

### Phase 1: State Access Layer
1. Add state getter methods to each component
2. Add State() method to TUISession
3. Add state snapshot capability

### Phase 2: Side Effect Verification
1. Add DBVerifier with history store access
2. Add FSVerifier for file system checks
3. Integrate into TUISession

### Phase 3: Journey Framework
1. Implement Journey builder
2. Implement Step builder with fluent API
3. Add WaitFor with polling

### Phase 4: Migrate Existing Tests
1. Convert existing tests to journey format
2. Add missing journey tests
3. Add input matrix tests

### Phase 5: CI Integration
1. Run journey tests in CI
2. Generate coverage report
3. Fail build if coverage drops

---

## File Structure

```
e2e/
├── E2E_TESTING_DESIGN.md     # This document
├── harness/
│   ├── tui_runner.go         # Enhanced with state access
│   ├── state.go              # State types and accessors
│   ├── db_verifier.go        # Database verification
│   ├── fs_verifier.go        # File system verification
│   └── journey.go            # Journey builder
├── journeys/
│   ├── request_test.go       # Request CRUD journeys
│   ├── history_test.go       # History journeys
│   ├── navigation_test.go    # Navigation journeys
│   └── input_test.go         # Input matrix tests
└── fixtures/
    ├── collections/          # Test collection files
    └── mock_responses/       # Mock server responses
```
