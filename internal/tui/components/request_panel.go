package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/script"
	"github.com/artpar/currier/internal/tui"
)

// RequestTab represents the active tab in the request panel.
type RequestTab int

const (
	TabURL RequestTab = iota
	TabHeaders
	TabQuery
	TabBody
	TabAuth
	TabPreRequest
	TabTests
)

var tabNames = []string{"URL", "Headers", "Query", "Body", "Auth", "Pre-req", "Tests"}

// SendRequestMsg is sent when user wants to send the request.
type SendRequestMsg struct {
	Request *core.RequestDefinition
}

// ResponseReceivedMsg is sent when a response is received.
type ResponseReceivedMsg struct {
	Response    *core.Response
	TestResults []script.TestResult
	Console     []ConsoleMessage // Console output from scripts
}

// RequestErrorMsg is sent when a request fails.
type RequestErrorMsg struct {
	Error error
}

// HTTP methods for cycling
var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// RequestPanel displays and edits request details.
type RequestPanel struct {
	title         string
	focused       bool
	width         int
	height        int
	request       *core.RequestDefinition
	activeTab     RequestTab
	cursor        int
	offset        int
	editingURL    bool   // True when editing URL inline
	urlInput      string // Current URL input while editing
	urlCursor     int    // Cursor position in URL input
	editingMethod bool   // True when editing HTTP method
	methodIndex   int    // Current index in httpMethods

	// Header editing state
	editingHeader     bool   // True when editing a header
	headerEditMode    string // "key" or "value"
	headerKeyInput    string // Current key input
	headerValueInput  string // Current value input
	headerKeyCursor   int    // Cursor position in key
	headerValueCursor int    // Cursor position in value
	headerIsNew       bool   // True if adding new header
	headerOrigKey     string // Original key when editing (for replacement)
	headerKeys        []string // Ordered list of header keys for stable navigation

	// Body editing state
	editingBody    bool     // True when editing body
	bodyLines      []string // Body split into lines
	bodyCursorLine int      // Current line
	bodyCursorCol  int      // Current column

	// Query param editing state
	editingQuery     bool     // True when editing a query param
	queryEditMode    string   // "key" or "value"
	queryKeyInput    string   // Current key input
	queryValueInput  string   // Current value input
	queryKeyCursor   int      // Cursor position in key
	queryValueCursor int      // Cursor position in value
	queryIsNew       bool     // True if adding new param
	queryOrigKey     string   // Original key when editing (for replacement)
	queryKeys        []string // Ordered list of param keys for stable navigation

	// Auth editing state
	editingAuth      bool   // True when editing authentication
	authTypeIndex    int    // Index in auth types list
	authFieldIndex   int    // Which field is selected (0=type, 1=first field, etc.)
	authEditingField bool   // True when actively editing a field value
	authFieldInput   string // Current input for the active field
	authFieldCursor  int    // Cursor position in field input

	// Pre-request script editing state
	editingPreScript    bool     // True when editing pre-request script
	preScriptLines      []string // Pre-request script split into lines
	preScriptCursorLine int      // Current line
	preScriptCursorCol  int      // Current column

	// Test script editing state
	editingTestScript    bool     // True when editing test script
	testScriptLines      []string // Test script split into lines
	testScriptCursorLine int      // Current line
	testScriptCursorCol  int      // Current column
}

// NewRequestPanel creates a new request panel.
func NewRequestPanel() *RequestPanel {
	return &RequestPanel{
		title:     "Request",
		activeTab: TabURL,
	}
}

// Init initializes the component.
func (p *RequestPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (p *RequestPanel) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
	if !p.focused {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			p.width = msg.Width
			p.height = msg.Height
		case tui.FocusMsg:
			p.focused = true
		}
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case tui.FocusMsg:
		p.focused = true

	case tui.BlurMsg:
		p.focused = false

	case tea.KeyMsg:
		return p.handleKeyMsg(msg)
	}

	return p, nil
}

func (p *RequestPanel) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Handle URL editing mode
	if p.editingURL {
		return p.handleURLEditInput(msg)
	}

	// Handle method editing mode
	if p.editingMethod {
		return p.handleMethodEditInput(msg)
	}

	// Handle header editing mode
	if p.editingHeader {
		return p.handleHeaderEditInput(msg)
	}

	// Handle query editing mode
	if p.editingQuery {
		return p.handleQueryEditInput(msg)
	}

	// Handle body editing mode
	if p.editingBody {
		return p.handleBodyEditInput(msg)
	}

	// Handle auth editing mode
	if p.editingAuth {
		return p.handleAuthEditInput(msg)
	}

	// Handle pre-request script editing mode
	if p.editingPreScript {
		return p.handlePreScriptEditInput(msg)
	}

	// Handle test script editing mode
	if p.editingTestScript {
		return p.handleTestScriptEditInput(msg)
	}

	switch msg.Type {
	case tea.KeyEnter:
		// Send request from any tab when not in edit mode
		if p.request != nil {
			return p, func() tea.Msg {
				return SendRequestMsg{Request: p.request}
			}
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "e":
			// Enter URL edit mode
			if p.activeTab == TabURL && p.request != nil {
				p.editingURL = true
				p.urlInput = p.request.FullURL()
				p.urlCursor = len(p.urlInput)
				return p, nil
			}
			// Enter header edit mode (edit existing)
			if p.activeTab == TabHeaders && p.request != nil {
				p.syncHeaderKeys()
				if p.cursor < len(p.headerKeys) {
					key := p.headerKeys[p.cursor]
					value := p.request.GetHeader(key)
					p.editingHeader = true
					p.headerIsNew = false
					p.headerEditMode = "key"
					p.headerKeyInput = key
					p.headerValueInput = value
					p.headerKeyCursor = len(key)
					p.headerValueCursor = len(value)
					p.headerOrigKey = key
					return p, nil
				}
			}
			// Enter query param edit mode (edit existing)
			if p.activeTab == TabQuery && p.request != nil {
				p.syncQueryKeys()
				if p.cursor < len(p.queryKeys) {
					key := p.queryKeys[p.cursor]
					value := p.request.GetQueryParam(key)
					p.editingQuery = true
					p.queryIsNew = false
					p.queryEditMode = "key"
					p.queryKeyInput = key
					p.queryValueInput = value
					p.queryKeyCursor = len(key)
					p.queryValueCursor = len(value)
					p.queryOrigKey = key
					return p, nil
				}
			}
			// Enter body edit mode
			if p.activeTab == TabBody && p.request != nil {
				body := p.request.Body()
				p.bodyLines = strings.Split(body, "\n")
				if len(p.bodyLines) == 0 {
					p.bodyLines = []string{""}
				}
				p.bodyCursorLine = 0
				p.bodyCursorCol = 0
				p.editingBody = true
				return p, nil
			}
			// Enter auth edit mode
			if p.activeTab == TabAuth && p.request != nil {
				p.editingAuth = true
				p.authFieldIndex = 0 // Start at auth type
				p.authEditingField = false
				// Initialize auth type index based on current auth
				auth := p.request.Auth()
				if auth == nil || auth.Type == "" {
					p.authTypeIndex = 0 // No Auth
				} else {
					authTypes := core.CommonAuthTypes()
					for i, at := range authTypes {
						if string(at) == auth.Type {
							p.authTypeIndex = i
							break
						}
					}
				}
				return p, nil
			}
			// Enter pre-request script edit mode
			if p.activeTab == TabPreRequest && p.request != nil {
				script := p.request.PreScript()
				p.preScriptLines = strings.Split(script, "\n")
				if len(p.preScriptLines) == 0 {
					p.preScriptLines = []string{""}
				}
				p.preScriptCursorLine = 0
				p.preScriptCursorCol = 0
				p.editingPreScript = true
				return p, nil
			}
			// Enter test script edit mode
			if p.activeTab == TabTests && p.request != nil {
				script := p.request.PostScript()
				p.testScriptLines = strings.Split(script, "\n")
				if len(p.testScriptLines) == 0 {
					p.testScriptLines = []string{""}
				}
				p.testScriptCursorLine = 0
				p.testScriptCursorCol = 0
				p.editingTestScript = true
				return p, nil
			}
		case "a":
			// Add new header
			if p.activeTab == TabHeaders && p.request != nil {
				p.editingHeader = true
				p.headerIsNew = true
				p.headerEditMode = "key"
				p.headerKeyInput = ""
				p.headerValueInput = ""
				p.headerKeyCursor = 0
				p.headerValueCursor = 0
				p.headerOrigKey = ""
				return p, nil
			}
			// Add new query param
			if p.activeTab == TabQuery && p.request != nil {
				p.editingQuery = true
				p.queryIsNew = true
				p.queryEditMode = "key"
				p.queryKeyInput = ""
				p.queryValueInput = ""
				p.queryKeyCursor = 0
				p.queryValueCursor = 0
				p.queryOrigKey = ""
				return p, nil
			}
		case "d":
			// Delete header at cursor
			if p.activeTab == TabHeaders && p.request != nil {
				p.syncHeaderKeys()
				if p.cursor < len(p.headerKeys) {
					key := p.headerKeys[p.cursor]
					p.request.RemoveHeader(key)
					p.syncHeaderKeys()
					if p.cursor >= len(p.headerKeys) && p.cursor > 0 {
						p.cursor--
					}
					return p, nil
				}
			}
			// Delete query param at cursor
			if p.activeTab == TabQuery && p.request != nil {
				p.syncQueryKeys()
				if p.cursor < len(p.queryKeys) {
					key := p.queryKeys[p.cursor]
					p.request.RemoveQueryParam(key)
					p.syncQueryKeys()
					if p.cursor >= len(p.queryKeys) && p.cursor > 0 {
						p.cursor--
					}
					return p, nil
				}
			}
		case "m":
			// Enter method edit mode
			if p.request == nil {
				return p, nil
			}
			if p.activeTab != TabURL {
				return p, func() tea.Msg {
					return FeedbackMsg{Message: "Switch to URL tab to change method (press [)", IsError: false}
				}
			}
			p.editingMethod = true
			// Find current method index
			currentMethod := strings.ToUpper(p.request.Method())
			p.methodIndex = 0
			for i, m := range httpMethods {
				if m == currentMethod {
					p.methodIndex = i
					break
				}
			}
			return p, nil
		case "[":
			// Switch to previous tab
			p.prevTab()
			return p, nil
		case "]":
			// Switch to next tab
			p.nextTab()
			return p, nil
		case "j":
			p.moveCursor(1)
		case "k":
			p.moveCursor(-1)
		}
	}

	return p, nil
}

// syncHeaderKeys updates the ordered list of header keys for stable navigation.
func (p *RequestPanel) syncHeaderKeys() {
	if p.request == nil {
		p.headerKeys = nil
		return
	}
	headers := p.request.Headers()
	newKeys := make([]string, 0, len(headers))
	seen := make(map[string]bool)

	// Keep existing keys that still exist
	for _, key := range p.headerKeys {
		if _, exists := headers[key]; exists {
			newKeys = append(newKeys, key)
			seen[key] = true
		}
	}
	// Add any new keys
	for key := range headers {
		if !seen[key] {
			newKeys = append(newKeys, key)
		}
	}
	p.headerKeys = newKeys
}

// syncQueryKeys updates the ordered list of query param keys for stable navigation.
func (p *RequestPanel) syncQueryKeys() {
	if p.request == nil {
		p.queryKeys = nil
		return
	}
	params := p.request.QueryParams()
	newKeys := make([]string, 0, len(params))
	seen := make(map[string]bool)

	// Keep existing keys that still exist
	for _, key := range p.queryKeys {
		if _, exists := params[key]; exists {
			newKeys = append(newKeys, key)
			seen[key] = true
		}
	}
	// Add any new keys
	for key := range params {
		if !seen[key] {
			newKeys = append(newKeys, key)
		}
	}
	p.queryKeys = newKeys
}

// StartURLEdit enters URL edit mode externally.
func (p *RequestPanel) StartURLEdit() {
	if p.request == nil {
		return
	}
	p.editingURL = true
	p.urlInput = p.request.FullURL()
	p.urlCursor = len(p.urlInput)
	p.activeTab = TabURL
}

// handleHeaderEditInput handles keyboard input while editing a header.
func (p *RequestPanel) handleHeaderEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Save and exit (vim-like behavior)
		if p.headerKeyInput != "" {
			if !p.headerIsNew && p.headerOrigKey != p.headerKeyInput {
				// Key was renamed - remove old key
				p.request.RemoveHeader(p.headerOrigKey)
			}
			p.request.SetHeader(p.headerKeyInput, p.headerValueInput)
			p.syncHeaderKeys()
		}
		p.editingHeader = false
		return p, nil

	case tea.KeyEnter:
		// Save header
		if p.headerKeyInput != "" {
			if !p.headerIsNew && p.headerOrigKey != p.headerKeyInput {
				// Key was renamed - remove old key
				p.request.RemoveHeader(p.headerOrigKey)
			}
			p.request.SetHeader(p.headerKeyInput, p.headerValueInput)
			p.syncHeaderKeys()
		}
		p.editingHeader = false
		return p, nil

	case tea.KeyTab:
		// Switch between key and value
		if p.headerEditMode == "key" {
			p.headerEditMode = "value"
		} else {
			p.headerEditMode = "key"
		}
		return p, nil

	case tea.KeyBackspace:
		if p.headerEditMode == "key" {
			if p.headerKeyCursor > 0 {
				p.headerKeyInput = p.headerKeyInput[:p.headerKeyCursor-1] + p.headerKeyInput[p.headerKeyCursor:]
				p.headerKeyCursor--
			}
		} else {
			if p.headerValueCursor > 0 {
				p.headerValueInput = p.headerValueInput[:p.headerValueCursor-1] + p.headerValueInput[p.headerValueCursor:]
				p.headerValueCursor--
			}
		}
		return p, nil

	case tea.KeyLeft:
		if p.headerEditMode == "key" {
			if p.headerKeyCursor > 0 {
				p.headerKeyCursor--
			}
		} else {
			if p.headerValueCursor > 0 {
				p.headerValueCursor--
			}
		}
		return p, nil

	case tea.KeyRight:
		if p.headerEditMode == "key" {
			if p.headerKeyCursor < len(p.headerKeyInput) {
				p.headerKeyCursor++
			}
		} else {
			if p.headerValueCursor < len(p.headerValueInput) {
				p.headerValueCursor++
			}
		}
		return p, nil

	case tea.KeyDelete:
		if p.headerEditMode == "key" {
			if p.headerKeyCursor < len(p.headerKeyInput) {
				p.headerKeyInput = p.headerKeyInput[:p.headerKeyCursor] + p.headerKeyInput[p.headerKeyCursor+1:]
			}
		} else {
			if p.headerValueCursor < len(p.headerValueInput) {
				p.headerValueInput = p.headerValueInput[:p.headerValueCursor] + p.headerValueInput[p.headerValueCursor+1:]
			}
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		if p.headerEditMode == "key" {
			p.headerKeyCursor = 0
		} else {
			p.headerValueCursor = 0
		}
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		if p.headerEditMode == "key" {
			p.headerKeyCursor = len(p.headerKeyInput)
		} else {
			p.headerValueCursor = len(p.headerValueInput)
		}
		return p, nil

	case tea.KeyCtrlU:
		if p.headerEditMode == "key" {
			p.headerKeyInput = ""
			p.headerKeyCursor = 0
		} else {
			p.headerValueInput = ""
			p.headerValueCursor = 0
		}
		return p, nil

	case tea.KeySpace:
		if p.headerEditMode == "key" {
			p.headerKeyInput = p.headerKeyInput[:p.headerKeyCursor] + " " + p.headerKeyInput[p.headerKeyCursor:]
			p.headerKeyCursor++
		} else {
			p.headerValueInput = p.headerValueInput[:p.headerValueCursor] + " " + p.headerValueInput[p.headerValueCursor:]
			p.headerValueCursor++
		}
		return p, nil

	case tea.KeyRunes:
		char := string(msg.Runes)
		if p.headerEditMode == "key" {
			p.headerKeyInput = p.headerKeyInput[:p.headerKeyCursor] + char + p.headerKeyInput[p.headerKeyCursor:]
			p.headerKeyCursor += len(char)
		} else {
			p.headerValueInput = p.headerValueInput[:p.headerValueCursor] + char + p.headerValueInput[p.headerValueCursor:]
			p.headerValueCursor += len(char)
		}
		return p, nil
	}

	return p, nil
}

// handleQueryEditInput handles keyboard input while editing a query param.
func (p *RequestPanel) handleQueryEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Save and exit (vim-like behavior)
		if p.queryKeyInput != "" {
			if !p.queryIsNew && p.queryOrigKey != p.queryKeyInput {
				// Key was renamed - remove old key
				p.request.RemoveQueryParam(p.queryOrigKey)
			}
			p.request.SetQueryParam(p.queryKeyInput, p.queryValueInput)
			p.syncQueryKeys()
		}
		p.editingQuery = false
		return p, nil

	case tea.KeyEnter:
		// Save query param
		if p.queryKeyInput != "" {
			if !p.queryIsNew && p.queryOrigKey != p.queryKeyInput {
				// Key was renamed - remove old key
				p.request.RemoveQueryParam(p.queryOrigKey)
			}
			p.request.SetQueryParam(p.queryKeyInput, p.queryValueInput)
			p.syncQueryKeys()
		}
		p.editingQuery = false
		return p, nil

	case tea.KeyTab:
		// Switch between key and value
		if p.queryEditMode == "key" {
			p.queryEditMode = "value"
		} else {
			p.queryEditMode = "key"
		}
		return p, nil

	case tea.KeyBackspace:
		if p.queryEditMode == "key" {
			if p.queryKeyCursor > 0 {
				p.queryKeyInput = p.queryKeyInput[:p.queryKeyCursor-1] + p.queryKeyInput[p.queryKeyCursor:]
				p.queryKeyCursor--
			}
		} else {
			if p.queryValueCursor > 0 {
				p.queryValueInput = p.queryValueInput[:p.queryValueCursor-1] + p.queryValueInput[p.queryValueCursor:]
				p.queryValueCursor--
			}
		}
		return p, nil

	case tea.KeyLeft:
		if p.queryEditMode == "key" {
			if p.queryKeyCursor > 0 {
				p.queryKeyCursor--
			}
		} else {
			if p.queryValueCursor > 0 {
				p.queryValueCursor--
			}
		}
		return p, nil

	case tea.KeyRight:
		if p.queryEditMode == "key" {
			if p.queryKeyCursor < len(p.queryKeyInput) {
				p.queryKeyCursor++
			}
		} else {
			if p.queryValueCursor < len(p.queryValueInput) {
				p.queryValueCursor++
			}
		}
		return p, nil

	case tea.KeyDelete:
		if p.queryEditMode == "key" {
			if p.queryKeyCursor < len(p.queryKeyInput) {
				p.queryKeyInput = p.queryKeyInput[:p.queryKeyCursor] + p.queryKeyInput[p.queryKeyCursor+1:]
			}
		} else {
			if p.queryValueCursor < len(p.queryValueInput) {
				p.queryValueInput = p.queryValueInput[:p.queryValueCursor] + p.queryValueInput[p.queryValueCursor+1:]
			}
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		if p.queryEditMode == "key" {
			p.queryKeyCursor = 0
		} else {
			p.queryValueCursor = 0
		}
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		if p.queryEditMode == "key" {
			p.queryKeyCursor = len(p.queryKeyInput)
		} else {
			p.queryValueCursor = len(p.queryValueInput)
		}
		return p, nil

	case tea.KeyCtrlU:
		if p.queryEditMode == "key" {
			p.queryKeyInput = ""
			p.queryKeyCursor = 0
		} else {
			p.queryValueInput = ""
			p.queryValueCursor = 0
		}
		return p, nil

	case tea.KeySpace:
		if p.queryEditMode == "key" {
			p.queryKeyInput = p.queryKeyInput[:p.queryKeyCursor] + " " + p.queryKeyInput[p.queryKeyCursor:]
			p.queryKeyCursor++
		} else {
			p.queryValueInput = p.queryValueInput[:p.queryValueCursor] + " " + p.queryValueInput[p.queryValueCursor:]
			p.queryValueCursor++
		}
		return p, nil

	case tea.KeyRunes:
		char := string(msg.Runes)
		if p.queryEditMode == "key" {
			p.queryKeyInput = p.queryKeyInput[:p.queryKeyCursor] + char + p.queryKeyInput[p.queryKeyCursor:]
			p.queryKeyCursor += len(char)
		} else {
			p.queryValueInput = p.queryValueInput[:p.queryValueCursor] + char + p.queryValueInput[p.queryValueCursor:]
			p.queryValueCursor += len(char)
		}
		return p, nil
	}

	return p, nil
}

// handleBodyEditInput handles keyboard input while editing the body.
func (p *RequestPanel) handleBodyEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Insert tab character (useful for JSON/code formatting)
		line := p.bodyLines[p.bodyCursorLine]
		p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol] + "\t" + line[p.bodyCursorCol:]
		p.bodyCursorCol++
		return p, nil

	case tea.KeyEsc:
		// Save and exit (vim-like: Esc returns to normal mode with changes saved)
		body := strings.Join(p.bodyLines, "\n")
		p.request.SetBody(body)
		p.editingBody = false
		return p, nil

	case tea.KeyEnter:
		// Insert new line
		line := p.bodyLines[p.bodyCursorLine]
		before := line[:p.bodyCursorCol]
		after := line[p.bodyCursorCol:]
		p.bodyLines[p.bodyCursorLine] = before
		// Insert new line after current
		newLines := make([]string, 0, len(p.bodyLines)+1)
		newLines = append(newLines, p.bodyLines[:p.bodyCursorLine+1]...)
		newLines = append(newLines, after)
		newLines = append(newLines, p.bodyLines[p.bodyCursorLine+1:]...)
		p.bodyLines = newLines
		p.bodyCursorLine++
		p.bodyCursorCol = 0
		return p, nil

	case tea.KeyBackspace:
		if p.bodyCursorCol > 0 {
			line := p.bodyLines[p.bodyCursorLine]
			p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol-1] + line[p.bodyCursorCol:]
			p.bodyCursorCol--
		} else if p.bodyCursorLine > 0 {
			// Join with previous line
			prevLine := p.bodyLines[p.bodyCursorLine-1]
			currLine := p.bodyLines[p.bodyCursorLine]
			p.bodyCursorCol = len(prevLine)
			p.bodyLines[p.bodyCursorLine-1] = prevLine + currLine
			// Remove current line
			p.bodyLines = append(p.bodyLines[:p.bodyCursorLine], p.bodyLines[p.bodyCursorLine+1:]...)
			p.bodyCursorLine--
		}
		return p, nil

	case tea.KeyDelete:
		line := p.bodyLines[p.bodyCursorLine]
		if p.bodyCursorCol < len(line) {
			p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol] + line[p.bodyCursorCol+1:]
		} else if p.bodyCursorLine < len(p.bodyLines)-1 {
			// Join with next line
			nextLine := p.bodyLines[p.bodyCursorLine+1]
			p.bodyLines[p.bodyCursorLine] = line + nextLine
			p.bodyLines = append(p.bodyLines[:p.bodyCursorLine+1], p.bodyLines[p.bodyCursorLine+2:]...)
		}
		return p, nil

	case tea.KeyLeft:
		if p.bodyCursorCol > 0 {
			p.bodyCursorCol--
		} else if p.bodyCursorLine > 0 {
			p.bodyCursorLine--
			p.bodyCursorCol = len(p.bodyLines[p.bodyCursorLine])
		}
		return p, nil

	case tea.KeyRight:
		line := p.bodyLines[p.bodyCursorLine]
		if p.bodyCursorCol < len(line) {
			p.bodyCursorCol++
		} else if p.bodyCursorLine < len(p.bodyLines)-1 {
			p.bodyCursorLine++
			p.bodyCursorCol = 0
		}
		return p, nil

	case tea.KeyUp:
		if p.bodyCursorLine > 0 {
			p.bodyCursorLine--
			if p.bodyCursorCol > len(p.bodyLines[p.bodyCursorLine]) {
				p.bodyCursorCol = len(p.bodyLines[p.bodyCursorLine])
			}
		}
		return p, nil

	case tea.KeyDown:
		if p.bodyCursorLine < len(p.bodyLines)-1 {
			p.bodyCursorLine++
			if p.bodyCursorCol > len(p.bodyLines[p.bodyCursorLine]) {
				p.bodyCursorCol = len(p.bodyLines[p.bodyCursorLine])
			}
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		p.bodyCursorCol = 0
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		p.bodyCursorCol = len(p.bodyLines[p.bodyCursorLine])
		return p, nil

	case tea.KeyCtrlU:
		// Clear current line
		p.bodyLines[p.bodyCursorLine] = ""
		p.bodyCursorCol = 0
		return p, nil

	case tea.KeySpace:
		line := p.bodyLines[p.bodyCursorLine]
		p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol] + " " + line[p.bodyCursorCol:]
		p.bodyCursorCol++
		return p, nil

	case tea.KeyRunes:
		char := string(msg.Runes)
		line := p.bodyLines[p.bodyCursorLine]
		p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol] + char + line[p.bodyCursorCol:]
		p.bodyCursorCol += len(char)
		return p, nil
	}

	return p, nil
}

// handlePreScriptEditInput handles keyboard input while editing the pre-request script.
func (p *RequestPanel) handlePreScriptEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Insert tab character
		line := p.preScriptLines[p.preScriptCursorLine]
		p.preScriptLines[p.preScriptCursorLine] = line[:p.preScriptCursorCol] + "\t" + line[p.preScriptCursorCol:]
		p.preScriptCursorCol++
		return p, nil

	case tea.KeyEsc:
		// Save and exit
		script := strings.Join(p.preScriptLines, "\n")
		p.request.SetPreScript(script)
		p.editingPreScript = false
		return p, nil

	case tea.KeyEnter:
		// Insert new line
		line := p.preScriptLines[p.preScriptCursorLine]
		before := line[:p.preScriptCursorCol]
		after := line[p.preScriptCursorCol:]
		p.preScriptLines[p.preScriptCursorLine] = before
		newLines := make([]string, 0, len(p.preScriptLines)+1)
		newLines = append(newLines, p.preScriptLines[:p.preScriptCursorLine+1]...)
		newLines = append(newLines, after)
		newLines = append(newLines, p.preScriptLines[p.preScriptCursorLine+1:]...)
		p.preScriptLines = newLines
		p.preScriptCursorLine++
		p.preScriptCursorCol = 0
		return p, nil

	case tea.KeyBackspace:
		if p.preScriptCursorCol > 0 {
			line := p.preScriptLines[p.preScriptCursorLine]
			p.preScriptLines[p.preScriptCursorLine] = line[:p.preScriptCursorCol-1] + line[p.preScriptCursorCol:]
			p.preScriptCursorCol--
		} else if p.preScriptCursorLine > 0 {
			prevLine := p.preScriptLines[p.preScriptCursorLine-1]
			currLine := p.preScriptLines[p.preScriptCursorLine]
			p.preScriptCursorCol = len(prevLine)
			p.preScriptLines[p.preScriptCursorLine-1] = prevLine + currLine
			p.preScriptLines = append(p.preScriptLines[:p.preScriptCursorLine], p.preScriptLines[p.preScriptCursorLine+1:]...)
			p.preScriptCursorLine--
		}
		return p, nil

	case tea.KeyDelete:
		line := p.preScriptLines[p.preScriptCursorLine]
		if p.preScriptCursorCol < len(line) {
			p.preScriptLines[p.preScriptCursorLine] = line[:p.preScriptCursorCol] + line[p.preScriptCursorCol+1:]
		} else if p.preScriptCursorLine < len(p.preScriptLines)-1 {
			nextLine := p.preScriptLines[p.preScriptCursorLine+1]
			p.preScriptLines[p.preScriptCursorLine] = line + nextLine
			p.preScriptLines = append(p.preScriptLines[:p.preScriptCursorLine+1], p.preScriptLines[p.preScriptCursorLine+2:]...)
		}
		return p, nil

	case tea.KeyLeft:
		if p.preScriptCursorCol > 0 {
			p.preScriptCursorCol--
		} else if p.preScriptCursorLine > 0 {
			p.preScriptCursorLine--
			p.preScriptCursorCol = len(p.preScriptLines[p.preScriptCursorLine])
		}
		return p, nil

	case tea.KeyRight:
		line := p.preScriptLines[p.preScriptCursorLine]
		if p.preScriptCursorCol < len(line) {
			p.preScriptCursorCol++
		} else if p.preScriptCursorLine < len(p.preScriptLines)-1 {
			p.preScriptCursorLine++
			p.preScriptCursorCol = 0
		}
		return p, nil

	case tea.KeyUp:
		if p.preScriptCursorLine > 0 {
			p.preScriptCursorLine--
			if p.preScriptCursorCol > len(p.preScriptLines[p.preScriptCursorLine]) {
				p.preScriptCursorCol = len(p.preScriptLines[p.preScriptCursorLine])
			}
		}
		return p, nil

	case tea.KeyDown:
		if p.preScriptCursorLine < len(p.preScriptLines)-1 {
			p.preScriptCursorLine++
			if p.preScriptCursorCol > len(p.preScriptLines[p.preScriptCursorLine]) {
				p.preScriptCursorCol = len(p.preScriptLines[p.preScriptCursorLine])
			}
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		p.preScriptCursorCol = 0
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		p.preScriptCursorCol = len(p.preScriptLines[p.preScriptCursorLine])
		return p, nil

	case tea.KeyCtrlU:
		p.preScriptLines[p.preScriptCursorLine] = ""
		p.preScriptCursorCol = 0
		return p, nil

	case tea.KeySpace:
		line := p.preScriptLines[p.preScriptCursorLine]
		p.preScriptLines[p.preScriptCursorLine] = line[:p.preScriptCursorCol] + " " + line[p.preScriptCursorCol:]
		p.preScriptCursorCol++
		return p, nil

	case tea.KeyRunes:
		char := string(msg.Runes)
		line := p.preScriptLines[p.preScriptCursorLine]
		p.preScriptLines[p.preScriptCursorLine] = line[:p.preScriptCursorCol] + char + line[p.preScriptCursorCol:]
		p.preScriptCursorCol += len(char)
		return p, nil
	}

	return p, nil
}

// handleTestScriptEditInput handles keyboard input while editing the test script.
func (p *RequestPanel) handleTestScriptEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Insert tab character
		line := p.testScriptLines[p.testScriptCursorLine]
		p.testScriptLines[p.testScriptCursorLine] = line[:p.testScriptCursorCol] + "\t" + line[p.testScriptCursorCol:]
		p.testScriptCursorCol++
		return p, nil

	case tea.KeyEsc:
		// Save and exit
		script := strings.Join(p.testScriptLines, "\n")
		p.request.SetPostScript(script)
		p.editingTestScript = false
		return p, nil

	case tea.KeyEnter:
		// Insert new line
		line := p.testScriptLines[p.testScriptCursorLine]
		before := line[:p.testScriptCursorCol]
		after := line[p.testScriptCursorCol:]
		p.testScriptLines[p.testScriptCursorLine] = before
		newLines := make([]string, 0, len(p.testScriptLines)+1)
		newLines = append(newLines, p.testScriptLines[:p.testScriptCursorLine+1]...)
		newLines = append(newLines, after)
		newLines = append(newLines, p.testScriptLines[p.testScriptCursorLine+1:]...)
		p.testScriptLines = newLines
		p.testScriptCursorLine++
		p.testScriptCursorCol = 0
		return p, nil

	case tea.KeyBackspace:
		if p.testScriptCursorCol > 0 {
			line := p.testScriptLines[p.testScriptCursorLine]
			p.testScriptLines[p.testScriptCursorLine] = line[:p.testScriptCursorCol-1] + line[p.testScriptCursorCol:]
			p.testScriptCursorCol--
		} else if p.testScriptCursorLine > 0 {
			prevLine := p.testScriptLines[p.testScriptCursorLine-1]
			currLine := p.testScriptLines[p.testScriptCursorLine]
			p.testScriptCursorCol = len(prevLine)
			p.testScriptLines[p.testScriptCursorLine-1] = prevLine + currLine
			p.testScriptLines = append(p.testScriptLines[:p.testScriptCursorLine], p.testScriptLines[p.testScriptCursorLine+1:]...)
			p.testScriptCursorLine--
		}
		return p, nil

	case tea.KeyDelete:
		line := p.testScriptLines[p.testScriptCursorLine]
		if p.testScriptCursorCol < len(line) {
			p.testScriptLines[p.testScriptCursorLine] = line[:p.testScriptCursorCol] + line[p.testScriptCursorCol+1:]
		} else if p.testScriptCursorLine < len(p.testScriptLines)-1 {
			nextLine := p.testScriptLines[p.testScriptCursorLine+1]
			p.testScriptLines[p.testScriptCursorLine] = line + nextLine
			p.testScriptLines = append(p.testScriptLines[:p.testScriptCursorLine+1], p.testScriptLines[p.testScriptCursorLine+2:]...)
		}
		return p, nil

	case tea.KeyLeft:
		if p.testScriptCursorCol > 0 {
			p.testScriptCursorCol--
		} else if p.testScriptCursorLine > 0 {
			p.testScriptCursorLine--
			p.testScriptCursorCol = len(p.testScriptLines[p.testScriptCursorLine])
		}
		return p, nil

	case tea.KeyRight:
		line := p.testScriptLines[p.testScriptCursorLine]
		if p.testScriptCursorCol < len(line) {
			p.testScriptCursorCol++
		} else if p.testScriptCursorLine < len(p.testScriptLines)-1 {
			p.testScriptCursorLine++
			p.testScriptCursorCol = 0
		}
		return p, nil

	case tea.KeyUp:
		if p.testScriptCursorLine > 0 {
			p.testScriptCursorLine--
			if p.testScriptCursorCol > len(p.testScriptLines[p.testScriptCursorLine]) {
				p.testScriptCursorCol = len(p.testScriptLines[p.testScriptCursorLine])
			}
		}
		return p, nil

	case tea.KeyDown:
		if p.testScriptCursorLine < len(p.testScriptLines)-1 {
			p.testScriptCursorLine++
			if p.testScriptCursorCol > len(p.testScriptLines[p.testScriptCursorLine]) {
				p.testScriptCursorCol = len(p.testScriptLines[p.testScriptCursorLine])
			}
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		p.testScriptCursorCol = 0
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		p.testScriptCursorCol = len(p.testScriptLines[p.testScriptCursorLine])
		return p, nil

	case tea.KeyCtrlU:
		p.testScriptLines[p.testScriptCursorLine] = ""
		p.testScriptCursorCol = 0
		return p, nil

	case tea.KeySpace:
		line := p.testScriptLines[p.testScriptCursorLine]
		p.testScriptLines[p.testScriptCursorLine] = line[:p.testScriptCursorCol] + " " + line[p.testScriptCursorCol:]
		p.testScriptCursorCol++
		return p, nil

	case tea.KeyRunes:
		char := string(msg.Runes)
		line := p.testScriptLines[p.testScriptCursorLine]
		p.testScriptLines[p.testScriptCursorLine] = line[:p.testScriptCursorCol] + char + line[p.testScriptCursorCol:]
		p.testScriptCursorCol += len(char)
		return p, nil
	}

	return p, nil
}

// getAuthFieldsForType returns the field labels for the given auth type.
func getAuthFieldsForType(authType core.AuthType) []string {
	switch authType {
	case core.AuthTypeBasic:
		return []string{"Username", "Password"}
	case core.AuthTypeBearer:
		return []string{"Token"}
	case core.AuthTypeAPIKey:
		return []string{"Key Name", "Key Value", "Add to"}
	case core.AuthTypeOAuth2:
		return []string{"Access Token", "Token Type"}
	default:
		return []string{}
	}
}

// getAuthFieldValue gets the current value for a field index.
func (p *RequestPanel) getAuthFieldValue(fieldIdx int) string {
	auth := p.request.Auth()
	if auth == nil {
		return ""
	}
	authType := auth.GetAuthType()
	switch authType {
	case core.AuthTypeBasic:
		switch fieldIdx {
		case 1:
			return auth.Username
		case 2:
			return auth.Password
		}
	case core.AuthTypeBearer:
		if fieldIdx == 1 {
			return auth.Token
		}
	case core.AuthTypeAPIKey:
		switch fieldIdx {
		case 1:
			return auth.Key
		case 2:
			return auth.Value
		case 3:
			if auth.In == "" {
				return "header"
			}
			return auth.In
		}
	case core.AuthTypeOAuth2:
		if auth.OAuth2 != nil {
			switch fieldIdx {
			case 1:
				return auth.OAuth2.AccessToken
			case 2:
				prefix := auth.OAuth2.HeaderPrefix
				if prefix == "" {
					return "Bearer"
				}
				return prefix
			}
		}
	}
	return ""
}

// setAuthFieldValue sets the value for a field index.
func (p *RequestPanel) setAuthFieldValue(fieldIdx int, value string) {
	auth := p.request.Auth()
	if auth == nil {
		return
	}
	authType := auth.GetAuthType()
	switch authType {
	case core.AuthTypeBasic:
		switch fieldIdx {
		case 1:
			auth.Username = value
		case 2:
			auth.Password = value
		}
	case core.AuthTypeBearer:
		if fieldIdx == 1 {
			auth.Token = value
		}
	case core.AuthTypeAPIKey:
		switch fieldIdx {
		case 1:
			auth.Key = value
		case 2:
			auth.Value = value
		case 3:
			auth.In = value
		}
	case core.AuthTypeOAuth2:
		if auth.OAuth2 == nil {
			auth.OAuth2 = &core.OAuth2Config{}
		}
		switch fieldIdx {
		case 1:
			auth.OAuth2.AccessToken = value
		case 2:
			auth.OAuth2.HeaderPrefix = value
		}
	}
}

// handleAuthEditInput handles keyboard input while editing authentication.
func (p *RequestPanel) handleAuthEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	authTypes := core.CommonAuthTypes()
	currentAuthType := authTypes[p.authTypeIndex]
	fields := getAuthFieldsForType(currentAuthType)
	maxField := len(fields) // 0 = type, 1..n = fields

	// If editing a field value
	if p.authEditingField {
		switch msg.Type {
		case tea.KeyEsc, tea.KeyEnter:
			// Save field and exit field edit
			p.setAuthFieldValue(p.authFieldIndex, p.authFieldInput)
			p.authEditingField = false
			return p, nil

		case tea.KeyTab:
			// Save and move to next field
			p.setAuthFieldValue(p.authFieldIndex, p.authFieldInput)
			p.authEditingField = false
			p.authFieldIndex++
			if p.authFieldIndex > maxField {
				p.authFieldIndex = 0
			}
			return p, nil

		case tea.KeyBackspace:
			if p.authFieldCursor > 0 {
				p.authFieldInput = p.authFieldInput[:p.authFieldCursor-1] + p.authFieldInput[p.authFieldCursor:]
				p.authFieldCursor--
			}
			return p, nil

		case tea.KeyLeft:
			if p.authFieldCursor > 0 {
				p.authFieldCursor--
			}
			return p, nil

		case tea.KeyRight:
			if p.authFieldCursor < len(p.authFieldInput) {
				p.authFieldCursor++
			}
			return p, nil

		case tea.KeyHome, tea.KeyCtrlA:
			p.authFieldCursor = 0
			return p, nil

		case tea.KeyEnd, tea.KeyCtrlE:
			p.authFieldCursor = len(p.authFieldInput)
			return p, nil

		case tea.KeyCtrlU:
			p.authFieldInput = ""
			p.authFieldCursor = 0
			return p, nil

		case tea.KeySpace:
			p.authFieldInput = p.authFieldInput[:p.authFieldCursor] + " " + p.authFieldInput[p.authFieldCursor:]
			p.authFieldCursor++
			return p, nil

		case tea.KeyRunes:
			char := string(msg.Runes)
			p.authFieldInput = p.authFieldInput[:p.authFieldCursor] + char + p.authFieldInput[p.authFieldCursor:]
			p.authFieldCursor += len(char)
			return p, nil
		}
		return p, nil
	}

	// Navigation mode (not editing a field value)
	switch msg.Type {
	case tea.KeyTab:
		// Capture Tab to prevent pane switching
		return p, nil

	case tea.KeyEsc:
		// Exit auth edit mode
		p.editingAuth = false
		return p, nil

	case tea.KeyEnter:
		if p.authFieldIndex == 0 {
			// On type field - cycle through types
			p.authTypeIndex = (p.authTypeIndex + 1) % len(authTypes)
			newType := authTypes[p.authTypeIndex]
			p.applyAuthType(newType)
		} else {
			// On a field - enter edit mode
			p.authEditingField = true
			p.authFieldInput = p.getAuthFieldValue(p.authFieldIndex)
			p.authFieldCursor = len(p.authFieldInput)
		}
		return p, nil

	case tea.KeyUp:
		p.authFieldIndex--
		if p.authFieldIndex < 0 {
			p.authFieldIndex = maxField
		}
		return p, nil

	case tea.KeyDown:
		p.authFieldIndex++
		if p.authFieldIndex > maxField {
			p.authFieldIndex = 0
		}
		return p, nil

	case tea.KeyLeft:
		if p.authFieldIndex == 0 {
			// Cycle auth type backward
			p.authTypeIndex--
			if p.authTypeIndex < 0 {
				p.authTypeIndex = len(authTypes) - 1
			}
			p.applyAuthType(authTypes[p.authTypeIndex])
		}
		return p, nil

	case tea.KeyRight:
		if p.authFieldIndex == 0 {
			// Cycle auth type forward
			p.authTypeIndex = (p.authTypeIndex + 1) % len(authTypes)
			p.applyAuthType(authTypes[p.authTypeIndex])
		}
		return p, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			p.authFieldIndex++
			if p.authFieldIndex > maxField {
				p.authFieldIndex = 0
			}
		case "k":
			p.authFieldIndex--
			if p.authFieldIndex < 0 {
				p.authFieldIndex = maxField
			}
		case "h":
			if p.authFieldIndex == 0 {
				p.authTypeIndex--
				if p.authTypeIndex < 0 {
					p.authTypeIndex = len(authTypes) - 1
				}
				p.applyAuthType(authTypes[p.authTypeIndex])
			}
		case "l":
			if p.authFieldIndex == 0 {
				p.authTypeIndex = (p.authTypeIndex + 1) % len(authTypes)
				p.applyAuthType(authTypes[p.authTypeIndex])
			}
		case "e":
			// Enter field edit mode (if on a field, not type)
			if p.authFieldIndex > 0 {
				p.authEditingField = true
				p.authFieldInput = p.getAuthFieldValue(p.authFieldIndex)
				p.authFieldCursor = len(p.authFieldInput)
			}
		}
		return p, nil
	}

	return p, nil
}

// applyAuthType sets the auth type on the request.
func (p *RequestPanel) applyAuthType(authType core.AuthType) {
	auth := p.request.Auth()
	if auth == nil {
		newAuth := core.AuthConfig{}
		newAuth.SetAuthType(authType)
		// Initialize OAuth2 config if needed
		if authType == core.AuthTypeOAuth2 {
			newAuth.OAuth2 = &core.OAuth2Config{
				HeaderPrefix: "Bearer",
			}
		}
		p.request.SetAuth(newAuth)
		return
	}
	auth.SetAuthType(authType)

	// Initialize OAuth2 config if needed
	if authType == core.AuthTypeOAuth2 && auth.OAuth2 == nil {
		auth.OAuth2 = &core.OAuth2Config{
			HeaderPrefix: "Bearer",
		}
	}
}

func (p *RequestPanel) handleURLEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Capture Tab to prevent pane switching during URL edit
		// Tab does nothing in URL edit mode
		return p, nil

	case tea.KeyEsc:
		// Save URL and exit edit mode (vim-like behavior)
		if p.request != nil && p.urlInput != "" {
			p.request.SetURL(p.urlInput)
			p.syncQueryKeys()
		}
		p.editingURL = false
		p.urlInput = ""
		return p, nil

	case tea.KeyEnter:
		// Save URL and exit edit mode
		if p.request != nil && p.urlInput != "" {
			p.request.SetURL(p.urlInput)
			// Sync query keys after URL is set (may contain query params)
			p.syncQueryKeys()
		}
		p.editingURL = false
		p.urlInput = ""
		return p, nil

	case tea.KeyBackspace:
		if p.urlCursor > 0 {
			p.urlInput = p.urlInput[:p.urlCursor-1] + p.urlInput[p.urlCursor:]
			p.urlCursor--
		}
		return p, nil

	case tea.KeyDelete:
		if p.urlCursor < len(p.urlInput) {
			p.urlInput = p.urlInput[:p.urlCursor] + p.urlInput[p.urlCursor+1:]
		}
		return p, nil

	case tea.KeyLeft:
		if p.urlCursor > 0 {
			p.urlCursor--
		}
		return p, nil

	case tea.KeyRight:
		if p.urlCursor < len(p.urlInput) {
			p.urlCursor++
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		p.urlCursor = 0
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		p.urlCursor = len(p.urlInput)
		return p, nil

	case tea.KeyCtrlU:
		// Clear input
		p.urlInput = ""
		p.urlCursor = 0
		return p, nil

	case tea.KeySpace:
		// Insert space at cursor position
		p.urlInput = p.urlInput[:p.urlCursor] + " " + p.urlInput[p.urlCursor:]
		p.urlCursor++
		return p, nil

	case tea.KeyRunes:
		// Insert characters at cursor position
		char := string(msg.Runes)
		p.urlInput = p.urlInput[:p.urlCursor] + char + p.urlInput[p.urlCursor:]
		p.urlCursor += len(char)
		return p, nil
	}

	return p, nil
}

func (p *RequestPanel) handleMethodEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Capture Tab to prevent pane switching during method selection
		return p, nil

	case tea.KeyEsc:
		// Save method and exit edit mode (vim-like behavior)
		if p.request != nil {
			p.request.SetMethod(httpMethods[p.methodIndex])
		}
		p.editingMethod = false
		return p, nil

	case tea.KeyEnter:
		// Save method and exit edit mode
		if p.request != nil {
			p.request.SetMethod(httpMethods[p.methodIndex])
		}
		p.editingMethod = false
		return p, nil

	case tea.KeyUp, tea.KeyLeft:
		// Previous method
		p.methodIndex--
		if p.methodIndex < 0 {
			p.methodIndex = len(httpMethods) - 1
		}
		return p, nil

	case tea.KeyDown, tea.KeyRight:
		// Next method
		p.methodIndex++
		if p.methodIndex >= len(httpMethods) {
			p.methodIndex = 0
		}
		return p, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j", "l":
			// Next method
			p.methodIndex++
			if p.methodIndex >= len(httpMethods) {
				p.methodIndex = 0
			}
		case "k", "h":
			// Previous method
			p.methodIndex--
			if p.methodIndex < 0 {
				p.methodIndex = len(httpMethods) - 1
			}
		}
		return p, nil
	}

	return p, nil
}

func (p *RequestPanel) nextTab() {
	p.activeTab = RequestTab((int(p.activeTab) + 1) % len(tabNames))
	p.cursor = 0
}

func (p *RequestPanel) prevTab() {
	p.activeTab = RequestTab((int(p.activeTab) - 1 + len(tabNames)) % len(tabNames))
	p.cursor = 0
}

func (p *RequestPanel) moveCursor(delta int) {
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}

	// Limit based on content
	maxCursor := p.maxCursorForTab()
	if p.cursor > maxCursor {
		p.cursor = maxCursor
	}
}

func (p *RequestPanel) maxCursorForTab() int {
	if p.request == nil {
		return 0
	}

	switch p.activeTab {
	case TabHeaders:
		return len(p.request.Headers()) - 1
	case TabQuery:
		return len(p.request.QueryParams()) - 1
	default:
		return 0
	}
}

// View renders the component.
func (p *RequestPanel) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	// Account for borders
	innerWidth := p.width - 2
	innerHeight := p.height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Bold(true)

	if p.focused {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62"))
	} else {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238"))
	}

	title := titleStyle.Render(p.title)

	// Empty state
	if p.request == nil {
		emptyHeight := innerHeight - 1 // minus title
		if emptyHeight < 1 {
			emptyHeight = 1
		}
		emptyStyle := lipgloss.NewStyle().
			Width(innerWidth).
			Height(emptyHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240"))

		content := emptyStyle.Render("Press n to create a new request\nor select from Collections (press 1)")
		return p.wrapWithBorder(title + "\n" + content)
	}

	// URL bar with method badge and send hint
	urlBar := p.renderURLBar()

	// Separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", innerWidth))

	// Tab bar (now 2 lines: tabs + indicator)
	tabBar := p.renderTabBar()

	// Tab content: innerHeight - title(1) - urlBar(1) - separator(1) - tabBar(2)
	contentHeight := innerHeight - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	tabContent := p.renderTabContent(contentHeight)

	content := urlBar + "\n" + separator + "\n" + tabBar + "\n" + tabContent
	return p.wrapWithBorder(title + "\n" + content)
}

func (p *RequestPanel) renderURLBar() string {
	innerWidth := p.width - 2

	// Empty state - no request selected
	if p.request == nil {
		emptyStyle := lipgloss.NewStyle().
			Width(innerWidth).
			Foreground(lipgloss.Color("243")).
			Align(lipgloss.Center).
			Italic(true)
		return emptyStyle.Render("Press n to create a new request, or select from Collections (1)")
	}

	// Check if in method edit mode
	if p.editingMethod {
		return p.renderMethodSelector()
	}

	// Method button style (Postman-like dropdown look)
	method := p.request.Method()
	methodBtnStyle := p.methodStyle(method).
		Padding(0, 1)
	methodBtn := methodBtnStyle.Render(method + " ▾")

	// Send button - brighter when focused on URL tab
	sendStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 1)
	if p.focused && p.activeTab == TabURL {
		sendStyle = sendStyle.Background(lipgloss.Color("208"))
	}
	sendBtn := sendStyle.Render("↵ Send")

	// Calculate URL field width (method + spacing + url + spacing + send)
	sendBtnWidth := lipgloss.Width(sendBtn)
	methodWidth := lipgloss.Width(methodBtn)
	urlFieldWidth := innerWidth - methodWidth - sendBtnWidth - 4
	if urlFieldWidth < 10 {
		urlFieldWidth = 10
	}

	// URL content with placeholder support
	var urlContent string
	var isPlaceholder bool
	if p.editingURL {
		// Show editable URL with cursor
		url := p.urlInput
		cursor := p.urlCursor

		if cursor >= len(url) {
			urlContent = url + "▌"
		} else {
			urlContent = url[:cursor] + "▌" + url[cursor:]
		}

		// Truncate keeping cursor visible
		if len(urlContent) > urlFieldWidth {
			start := 0
			if cursor > urlFieldWidth-5 {
				start = cursor - urlFieldWidth + 5
			}
			end := start + urlFieldWidth
			if end > len(urlContent) {
				end = len(urlContent)
			}
			if start > 0 {
				urlContent = "…" + urlContent[start:end]
			} else {
				urlContent = urlContent[:end]
			}
		}
	} else {
		urlContent = p.request.FullURL()
		if urlContent == "" {
			urlContent = "Enter request URL..."
			isPlaceholder = true
		} else if len(urlContent) > urlFieldWidth {
			urlContent = urlContent[:urlFieldWidth-3] + "..."
		}
	}

	// Pad URL to fill field
	for len(urlContent) < urlFieldWidth {
		urlContent += " "
	}

	// URL field style with visual states
	urlFieldStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	if p.editingURL {
		// Editing: bright orange cursor, darker background
		urlFieldStyle = urlFieldStyle.
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("229"))
	} else if isPlaceholder {
		// Placeholder: italic, muted
		urlFieldStyle = urlFieldStyle.
			Foreground(lipgloss.Color("243")).
			Italic(true)
	} else if p.focused && p.activeTab == TabURL {
		// Focused: subtle highlight
		urlFieldStyle = urlFieldStyle.
			Background(lipgloss.Color("238"))
	}

	urlField := urlFieldStyle.Render(urlContent)

	// Compose URL bar
	urlBar := methodBtn + " " + urlField + " " + sendBtn

	return urlBar
}

func (p *RequestPanel) renderMethodSelector() string {
	innerWidth := p.width - 2

	// Build dropdown-style method selector
	var methods []string

	for i, m := range httpMethods {
		if i == p.methodIndex {
			// Selected method - colored badge with arrow
			style := p.methodStyle(m)
			methods = append(methods, style.Render("▸ "+m))
		} else {
			// Other methods - muted
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)
			methods = append(methods, style.Render("  "+m))
		}
	}

	// Dropdown box styling
	dropdownStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Padding(0, 1)

	methodList := strings.Join(methods, "\n")
	dropdown := dropdownStyle.Render(methodList)

	// Hints below dropdown
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	hints := hintStyle.Render("↑/↓ j/k Select   Enter/Esc: save & exit")

	// Pad to fill width
	lines := strings.Split(dropdown, "\n")
	for i, line := range lines {
		if len(line) < innerWidth {
			lines[i] = line + strings.Repeat(" ", innerWidth-lipgloss.Width(line))
		}
	}

	return strings.Join(lines, "\n") + "\n" + hints
}

func (p *RequestPanel) renderTabBar() string {
	innerWidth := p.width - 2
	var tabs []string

	for i, name := range tabNames {
		if RequestTab(i) == p.activeTab {
			// Active tab - colored indicator
			activeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true).
				Padding(0, 1)
			indicator := lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Render("━")
			tabs = append(tabs, activeStyle.Render(name)+"\n"+indicator+strings.Repeat("━", len(name)))
		} else {
			// Inactive tab
			inactiveStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)
			tabs = append(tabs, inactiveStyle.Render(name)+"\n"+strings.Repeat(" ", len(name)+2))
		}
	}

	// Build two-line tab bar
	var topLine, bottomLine []string
	for i, name := range tabNames {
		if RequestTab(i) == p.activeTab {
			activeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true).
				Padding(0, 1)
			topLine = append(topLine, activeStyle.Render(name))
			bottomLine = append(bottomLine, lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Render(strings.Repeat("━", len(name)+2)))
		} else {
			inactiveStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)
			topLine = append(topLine, inactiveStyle.Render(name))
			bottomLine = append(bottomLine, strings.Repeat(" ", len(name)+2))
		}
	}

	// Fill remaining width with separator line
	topRow := strings.Join(topLine, " ")
	bottomRow := strings.Join(bottomLine, " ")
	remainingWidth := innerWidth - lipgloss.Width(bottomRow)
	if remainingWidth > 0 {
		bottomRow += lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Render(strings.Repeat("─", remainingWidth))
	}

	return topRow + "\n" + bottomRow
}

func (p *RequestPanel) renderTabContent(height int) string {
	var lines []string

	switch p.activeTab {
	case TabURL:
		lines = p.renderURLTab()
	case TabHeaders:
		lines = p.renderHeadersTab()
	case TabQuery:
		lines = p.renderQueryTab()
	case TabBody:
		lines = p.renderBodyTab()
	case TabAuth:
		lines = p.renderAuthTab()
	case TabPreRequest:
		lines = p.renderPreRequestTab()
	case TabTests:
		lines = p.renderTestsTab()
	}

	// Pad or truncate to height
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

func (p *RequestPanel) renderURLTab() []string {
	if p.request == nil {
		return []string{"No request"}
	}

	return []string{
		fmt.Sprintf("URL: %s", p.request.FullURL()),
		fmt.Sprintf("Method: %s", p.request.Method()),
		"",
		"Press m to change method",
		"Press e to edit URL",
		"Press Enter to send request",
	}
}

func (p *RequestPanel) renderHeadersTab() []string {
	if p.request == nil {
		return []string{"No headers"}
	}

	p.syncHeaderKeys()
	innerWidth := p.width - 4
	keyWidth := innerWidth * 35 / 100
	if keyWidth < 10 {
		keyWidth = 10
	}
	valueWidth := innerWidth - keyWidth - 5
	if valueWidth < 10 {
		valueWidth = 10
	}

	var lines []string

	// Header row
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	lines = append(lines, headerStyle.Render(fmt.Sprintf("  %-*s │ %s", keyWidth, "Key", "Value")))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", innerWidth)))

	// If adding new header, show it at the top
	if p.editingHeader && p.headerIsNew {
		lines = append(lines, p.renderHeaderEditRow(keyWidth, valueWidth))
	}

	// Existing headers
	for i, key := range p.headerKeys {
		value := p.request.GetHeader(key)
		prefix := "  "

		if i == p.cursor && p.focused {
			if p.editingHeader && !p.headerIsNew {
				// Show edit mode for this row
				lines = append(lines, p.renderHeaderEditRow(keyWidth, valueWidth))
			} else {
				prefix = "> "
				keyStr := key
				if len(keyStr) > keyWidth {
					keyStr = keyStr[:keyWidth-1] + "…"
				}
				valueStr := value
				if len(valueStr) > valueWidth {
					valueStr = valueStr[:valueWidth-1] + "…"
				}
				selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
				lines = append(lines, selectedStyle.Render(fmt.Sprintf("%s%-*s │ %s", prefix, keyWidth, keyStr, valueStr)))
			}
		} else {
			keyStr := key
			if len(keyStr) > keyWidth {
				keyStr = keyStr[:keyWidth-1] + "…"
			}
			valueStr := value
			if len(valueStr) > valueWidth {
				valueStr = valueStr[:valueWidth-1] + "…"
			}
			lines = append(lines, fmt.Sprintf("%s%-*s │ %s", prefix, keyWidth, keyStr, valueStr))
		}
	}

	// Empty state or hint
	if len(p.headerKeys) == 0 && !p.editingHeader {
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		lines = append(lines, hintStyle.Render("  Press 'a' to add a header"))
	}

	// Editing hints with active field indicator
	if p.editingHeader {
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		activeField := "Key"
		if p.headerEditMode == "value" {
			activeField = "Value"
		}
		lines = append(lines, hintStyle.Render(fmt.Sprintf("  Editing: %s │ Tab: switch │ Enter/Esc: save", activeField)))
	} else if p.focused {
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		lines = append(lines, hintStyle.Render("  a: add │ e: edit │ d: delete"))
	}

	return lines
}

// renderHeaderEditRow renders a single header row in edit mode.
func (p *RequestPanel) renderHeaderEditRow(keyWidth, valueWidth int) string {
	// Prepare key content with cursor
	keyContent := p.headerKeyInput
	if p.headerEditMode == "key" {
		if p.headerKeyCursor >= len(keyContent) {
			keyContent += "▌"
		} else {
			keyContent = keyContent[:p.headerKeyCursor] + "▌" + keyContent[p.headerKeyCursor:]
		}
	}
	if len(keyContent) > keyWidth {
		keyContent = keyContent[:keyWidth-1] + "…"
	}

	// Prepare value content with cursor
	valueContent := p.headerValueInput
	if p.headerEditMode == "value" {
		if p.headerValueCursor >= len(valueContent) {
			valueContent += "▌"
		} else {
			valueContent = valueContent[:p.headerValueCursor] + "▌" + valueContent[p.headerValueCursor:]
		}
	}
	if len(valueContent) > valueWidth {
		valueContent = valueContent[:valueWidth-1] + "…"
	}

	// Styling for active field
	keyStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
	valueStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))

	if p.headerEditMode == "key" {
		keyStyle = keyStyle.Background(lipgloss.Color("238")).Foreground(lipgloss.Color("214"))
	} else {
		valueStyle = valueStyle.Background(lipgloss.Color("238")).Foreground(lipgloss.Color("214"))
	}

	// Pad to width
	for len(keyContent) < keyWidth {
		keyContent += " "
	}
	for len(valueContent) < valueWidth {
		valueContent += " "
	}

	return fmt.Sprintf("> %s │ %s", keyStyle.Render(keyContent), valueStyle.Render(valueContent))
}

func (p *RequestPanel) renderQueryTab() []string {
	if p.request == nil {
		return []string{"No query params"}
	}

	var lines []string

	// Sync query keys for stable ordering
	p.syncQueryKeys()

	// Show editing row if adding new or editing
	if p.editingQuery {
		if p.queryIsNew {
			lines = append(lines, p.renderQueryEditRow())
		}
		for i, key := range p.queryKeys {
			if !p.queryIsNew && i == p.cursor {
				lines = append(lines, p.renderQueryEditRow())
			} else {
				prefix := "  "
				if i == p.cursor && p.focused {
					prefix = "> "
				}
				value := p.request.GetQueryParam(key)
				lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, value))
			}
		}
		// Add hints with active field indicator
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		activeField := "Key"
		if p.queryEditMode == "value" {
			activeField = "Value"
		}
		lines = append(lines, hintStyle.Render(fmt.Sprintf("  Editing: %s │ Tab: switch │ Enter/Esc: save", activeField)))
	} else {
		if len(p.queryKeys) == 0 {
			lines = []string{"  No query parameters defined"}
			if p.focused {
				lines = append(lines, "")
				hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
				lines = append(lines, hintStyle.Render("  Press 'a' to add a parameter"))
			}
		} else {
			for i, key := range p.queryKeys {
				prefix := "  "
				if i == p.cursor && p.focused {
					prefix = "> "
				}
				value := p.request.GetQueryParam(key)
				lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, value))
			}
			// Add hints when focused
			if p.focused {
				lines = append(lines, "")
				hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
				lines = append(lines, hintStyle.Render("  e: edit │ a: add │ d: delete │ j/k: navigate"))
			}
		}
	}

	return lines
}

func (p *RequestPanel) renderQueryEditRow() string {
	// Similar to header edit row
	keyStyle := lipgloss.NewStyle()
	valueStyle := lipgloss.NewStyle()

	keyContent := p.queryKeyInput
	valueContent := p.queryValueInput

	// Add cursor indicator
	if p.queryEditMode == "key" {
		keyStyle = keyStyle.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("229"))
		if p.queryKeyCursor >= len(keyContent) {
			keyContent += "▌"
		} else {
			keyContent = keyContent[:p.queryKeyCursor] + "▌" + keyContent[p.queryKeyCursor:]
		}
	} else {
		valueStyle = valueStyle.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("229"))
		if p.queryValueCursor >= len(valueContent) {
			valueContent += "▌"
		} else {
			valueContent = valueContent[:p.queryValueCursor] + "▌" + valueContent[p.queryValueCursor:]
		}
	}

	return fmt.Sprintf("> %s │ %s", keyStyle.Render(keyContent), valueStyle.Render(valueContent))
}

func (p *RequestPanel) renderBodyTab() []string {
	if p.request == nil {
		return []string{"No body"}
	}

	var lines []string
	innerWidth := p.width - 4

	if p.editingBody {
		// Show editable body with cursor
		for i, line := range p.bodyLines {
			displayLine := line
			if i == p.bodyCursorLine {
				// Insert cursor at position
				if p.bodyCursorCol >= len(displayLine) {
					displayLine += "▌"
				} else {
					displayLine = displayLine[:p.bodyCursorCol] + "▌" + displayLine[p.bodyCursorCol:]
				}
			}
			// Truncate if too long
			if len(displayLine) > innerWidth {
				displayLine = displayLine[:innerWidth-1] + "…"
			}
			// Highlight current line
			if i == p.bodyCursorLine {
				lineStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
				displayLine = lineStyle.Render(displayLine)
			}
			lines = append(lines, displayLine)
		}

		// Add hints
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		lines = append(lines, hintStyle.Render("  Esc: save and exit │ ↑↓←→: navigate │ Enter: new line"))
	} else {
		body := p.request.Body()
		if body == "" {
			lines = []string{""}
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
			lines = append(lines, hintStyle.Render("  No body defined. Press 'e' to edit."))
		} else {
			bodyLines := strings.Split(body, "\n")
			for _, line := range bodyLines {
				if len(line) > innerWidth {
					line = line[:innerWidth-1] + "…"
				}
				lines = append(lines, line)
			}
		}

		// Add hint when focused (shows exit behavior before entering edit mode)
		if p.focused {
			lines = append(lines, "")
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
			lines = append(lines, hintStyle.Render("  Press 'e' to edit (Esc saves and exits)"))
		}
	}

	return lines
}

func (p *RequestPanel) renderAuthTab() []string {
	if p.request == nil {
		return []string{"No auth"}
	}

	innerWidth := p.width - 4
	labelWidth := 15
	valueWidth := innerWidth - labelWidth - 6
	if valueWidth < 10 {
		valueWidth = 10
	}

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	editingStyle := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("229"))

	var lines []string

	// Get auth types and current selection
	authTypes := core.CommonAuthTypes()
	currentAuthType := authTypes[p.authTypeIndex]
	fields := getAuthFieldsForType(currentAuthType)

	// Auth Type selector (field 0)
	typePrefix := "  "
	if p.editingAuth && p.authFieldIndex == 0 {
		typePrefix = "> "
	}
	typeName := core.AuthTypeNames[currentAuthType]
	if p.editingAuth && p.authFieldIndex == 0 {
		// Show as dropdown-style selector
		lines = append(lines, selectedStyle.Render(fmt.Sprintf("%s%-*s: ◀ %s ▶", typePrefix, labelWidth, "Auth Type", typeName)))
	} else {
		lines = append(lines, fmt.Sprintf("%s%-*s: %s", typePrefix, labelWidth, labelStyle.Render("Auth Type"), valueStyle.Render(typeName)))
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", innerWidth)))

	// Render fields based on auth type
	if currentAuthType == core.AuthTypeNone {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("  No authentication will be applied to this request."))
	} else {
		for i, fieldLabel := range fields {
			fieldIdx := i + 1 // Field indices start at 1 (0 is type)
			prefix := "  "
			if p.editingAuth && p.authFieldIndex == fieldIdx {
				prefix = "> "
			}

			// Get current value for this field
			value := p.getAuthFieldValue(fieldIdx)

			// Special handling for password fields - mask them
			displayValue := value
			if fieldLabel == "Password" && !p.authEditingField {
				if len(value) > 0 {
					displayValue = strings.Repeat("•", len(value))
				}
			}

			// Special handling for "Add to" field (API Key location)
			if fieldLabel == "Add to" {
				if value == "" {
					value = "header"
					displayValue = "header"
				}
				if p.editingAuth && p.authFieldIndex == fieldIdx {
					// Show as selector
					lines = append(lines, selectedStyle.Render(fmt.Sprintf("%s%-*s: ◀ %s ▶", prefix, labelWidth, fieldLabel, displayValue)))
					continue
				}
			}

			// Check if actively editing this field
			if p.editingAuth && p.authEditingField && p.authFieldIndex == fieldIdx {
				// Show with cursor
				editContent := p.authFieldInput
				if p.authFieldCursor >= len(editContent) {
					editContent += "▌"
				} else {
					editContent = editContent[:p.authFieldCursor] + "▌" + editContent[p.authFieldCursor:]
				}
				if len(editContent) > valueWidth {
					editContent = editContent[:valueWidth-1] + "…"
				}
				// Pad to valueWidth
				for len(editContent) < valueWidth {
					editContent += " "
				}
				lines = append(lines, fmt.Sprintf("%s%-*s: %s", prefix, labelWidth, selectedStyle.Render(fieldLabel), editingStyle.Render(editContent)))
			} else if p.editingAuth && p.authFieldIndex == fieldIdx {
				// Selected but not editing - highlight
				if displayValue == "" {
					displayValue = "(empty)"
				}
				lines = append(lines, selectedStyle.Render(fmt.Sprintf("%s%-*s: %s", prefix, labelWidth, fieldLabel, displayValue)))
			} else {
				// Normal display
				if displayValue == "" {
					displayValue = hintStyle.Render("(not set)")
				}
				lines = append(lines, fmt.Sprintf("%s%-*s: %s", prefix, labelWidth, labelStyle.Render(fieldLabel), displayValue))
			}
		}
	}

	// Add hints
	lines = append(lines, "")
	if p.editingAuth {
		if p.authEditingField {
			lines = append(lines, hintStyle.Render("  Type to edit │ Tab: next field │ Enter/Esc: save"))
		} else if p.authFieldIndex == 0 {
			lines = append(lines, hintStyle.Render("  ←/→ h/l: change type │ j/k: next field │ Esc: done"))
		} else {
			lines = append(lines, hintStyle.Render("  Enter/e: edit │ j/k: navigate │ Esc: done"))
		}
	} else if p.focused {
		lines = append(lines, hintStyle.Render("  Press 'e' to configure authentication"))
	}

	return lines
}

func (p *RequestPanel) renderPreRequestTab() []string {
	if p.request == nil {
		return []string{"No request"}
	}

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	innerWidth := p.width - 4
	var lines []string

	if p.editingPreScript {
		// Show editable script with cursor
		for i, line := range p.preScriptLines {
			displayLine := line
			if i == p.preScriptCursorLine {
				// Insert cursor at position
				if p.preScriptCursorCol >= len(displayLine) {
					displayLine += "▌"
				} else {
					displayLine = displayLine[:p.preScriptCursorCol] + "▌" + displayLine[p.preScriptCursorCol:]
				}
			}
			// Truncate if too long
			if len(displayLine) > innerWidth {
				displayLine = displayLine[:innerWidth-1] + "…"
			}
			// Highlight current line
			if i == p.preScriptCursorLine {
				lineStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
				displayLine = lineStyle.Render(displayLine)
			}
			lines = append(lines, displayLine)
		}

		// Add hints
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("  Esc: save and exit │ ↑↓←→: navigate │ Enter: new line"))
	} else {
		script := p.request.PreScript()
		if script == "" {
			lines = []string{""}
			lines = append(lines, hintStyle.Render("  No pre-request script defined."))
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Pre-request scripts run before the request is sent."))
			lines = append(lines, hintStyle.Render("  Use them to set variables, modify headers, etc."))
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Example:"))
			lines = append(lines, hintStyle.Render("    currier.request.headers['X-Timestamp'] = Date.now();"))
		} else {
			scriptLines := strings.Split(script, "\n")
			for _, line := range scriptLines {
				if len(line) > innerWidth {
					line = line[:innerWidth-1] + "…"
				}
				lines = append(lines, line)
			}
		}

		// Add hint when focused
		if p.focused {
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Press 'e' to edit (Esc saves and exits)"))
		}
	}

	return lines
}

func (p *RequestPanel) renderTestsTab() []string {
	if p.request == nil {
		return []string{"No request"}
	}

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	innerWidth := p.width - 4
	var lines []string

	if p.editingTestScript {
		// Show editable script with cursor
		for i, line := range p.testScriptLines {
			displayLine := line
			if i == p.testScriptCursorLine {
				// Insert cursor at position
				if p.testScriptCursorCol >= len(displayLine) {
					displayLine += "▌"
				} else {
					displayLine = displayLine[:p.testScriptCursorCol] + "▌" + displayLine[p.testScriptCursorCol:]
				}
			}
			// Truncate if too long
			if len(displayLine) > innerWidth {
				displayLine = displayLine[:innerWidth-1] + "…"
			}
			// Highlight current line
			if i == p.testScriptCursorLine {
				lineStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
				displayLine = lineStyle.Render(displayLine)
			}
			lines = append(lines, displayLine)
		}

		// Add hints
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("  Esc: save and exit │ ↑↓←→: navigate │ Enter: new line"))
	} else {
		script := p.request.PostScript()
		if script == "" {
			lines = []string{""}
			lines = append(lines, hintStyle.Render("  No test script defined."))
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Test scripts run after the response is received."))
			lines = append(lines, hintStyle.Render("  Use currier.test() and currier.expect() to validate."))
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Example:"))
			lines = append(lines, hintStyle.Render("    currier.test('Status is 200', function() {"))
			lines = append(lines, hintStyle.Render("      currier.expect(currier.response.status).toBe(200);"))
			lines = append(lines, hintStyle.Render("    });"))
		} else {
			scriptLines := strings.Split(script, "\n")
			for _, line := range scriptLines {
				if len(line) > innerWidth {
					line = line[:innerWidth-1] + "…"
				}
				lines = append(lines, line)
			}
		}

		// Add hint when focused
		if p.focused {
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("  Press 'e' to edit (Esc saves and exits)"))
		}
	}

	return lines
}

func (p *RequestPanel) methodStyle(method string) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	switch strings.ToUpper(method) {
	case "GET":
		return style.Background(lipgloss.Color("34")).Foreground(lipgloss.Color("255"))
	case "POST":
		return style.Background(lipgloss.Color("214")).Foreground(lipgloss.Color("0"))
	case "PUT":
		return style.Background(lipgloss.Color("33")).Foreground(lipgloss.Color("255"))
	case "PATCH":
		return style.Background(lipgloss.Color("141")).Foreground(lipgloss.Color("255"))
	case "DELETE":
		return style.Background(lipgloss.Color("160")).Foreground(lipgloss.Color("255"))
	default:
		return style.Background(lipgloss.Color("240"))
	}
}

func (p *RequestPanel) wrapWithBorder(content string) string {
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	if p.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		// Use brighter color (244 instead of 240) for better visibility
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("244"))
	}

	return borderStyle.Render(content)
}

// Title returns the component title.
func (p *RequestPanel) Title() string {
	if p.request != nil {
		return fmt.Sprintf("Request: %s", p.request.Name())
	}
	return p.title
}

// Focused returns true if focused.
func (p *RequestPanel) Focused() bool {
	return p.focused
}

// Focus sets the component as focused.
func (p *RequestPanel) Focus() {
	p.focused = true
}

// Blur removes focus.
func (p *RequestPanel) Blur() {
	p.focused = false
}

// IsEditing returns true if the panel is in any editing mode.
func (p *RequestPanel) IsEditing() bool {
	return p.editingURL || p.editingMethod || p.editingHeader || p.editingQuery || p.editingBody || p.editingAuth || p.editingPreScript || p.editingTestScript
}

// SetSize sets dimensions.
func (p *RequestPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Width returns the width.
func (p *RequestPanel) Width() int {
	return p.width
}

// Height returns the height.
func (p *RequestPanel) Height() int {
	return p.height
}

// Request returns the current request.
func (p *RequestPanel) Request() *core.RequestDefinition {
	return p.request
}

// SetRequest sets the request to display.
func (p *RequestPanel) SetRequest(req *core.RequestDefinition) {
	p.request = req
	p.cursor = 0
}

// ActiveTab returns the currently active tab.
func (p *RequestPanel) ActiveTab() RequestTab {
	return p.activeTab
}

// SetActiveTab sets the active tab.
func (p *RequestPanel) SetActiveTab(tab RequestTab) {
	p.activeTab = tab
	p.cursor = 0
}

// Cursor returns the cursor position.
func (p *RequestPanel) Cursor() int {
	return p.cursor
}

// SetCursor sets the cursor position.
func (p *RequestPanel) SetCursor(pos int) {
	p.cursor = pos
}

// Headers returns the request headers.
func (p *RequestPanel) Headers() map[string]string {
	if p.request == nil {
		return nil
	}
	return p.request.Headers()
}

// AddHeader adds a header to the request.
func (p *RequestPanel) AddHeader(key, value string) {
	if p.request != nil {
		p.request.SetHeader(key, value)
	}
}

// RemoveHeader removes a header from the request.
func (p *RequestPanel) RemoveHeader(key string) {
	if p.request != nil {
		p.request.RemoveHeader(key)
	}
}

// Body returns the request body.
func (p *RequestPanel) Body() string {
	if p.request == nil {
		return ""
	}
	return p.request.Body()
}

// SetBody sets the request body.
func (p *RequestPanel) SetBody(body string) {
	if p.request != nil {
		p.request.SetBody(body)
	}
}

// --- State accessors for E2E testing ---

// HasRequest returns true if a request is loaded.
func (p *RequestPanel) HasRequest() bool {
	return p.request != nil
}

// URL returns the current URL (from input buffer if editing, otherwise from request).
func (p *RequestPanel) URL() string {
	// When editing, return the input buffer
	if p.editingURL {
		return p.urlInput
	}
	if p.request == nil {
		return ""
	}
	return p.request.FullURL()
}

// Method returns the current HTTP method.
func (p *RequestPanel) Method() string {
	if p.request == nil {
		return ""
	}
	return p.request.Method()
}

// HeadersMap returns headers as a map.
func (p *RequestPanel) HeadersMap() map[string]string {
	if p.request == nil {
		return nil
	}
	return p.request.Headers()
}

// QueryParamsMap returns query parameters as a map.
func (p *RequestPanel) QueryParamsMap() map[string]string {
	if p.request == nil {
		return nil
	}
	return p.request.QueryParams()
}

// ActiveTabName returns the active tab name as a string.
func (p *RequestPanel) ActiveTabName() string {
	return tabNames[p.activeTab]
}

// IsEditingMethod returns true if in method edit mode.
func (p *RequestPanel) IsEditingMethod() bool {
	return p.editingMethod
}

// EditingField returns which field is being edited.
func (p *RequestPanel) EditingField() string {
	if p.editingURL {
		return "url"
	}
	if p.editingHeader {
		if p.headerEditMode == "key" {
			return "header_key"
		}
		return "header_value"
	}
	if p.editingQuery {
		if p.queryEditMode == "key" {
			return "query_key"
		}
		return "query_value"
	}
	if p.editingBody {
		return "body"
	}
	if p.editingMethod {
		return "method"
	}
	if p.editingPreScript {
		return "pre_script"
	}
	if p.editingTestScript {
		return "test_script"
	}
	return ""
}

// CursorPosition returns the current cursor position in the active field.
func (p *RequestPanel) CursorPosition() int {
	if p.editingURL {
		return p.urlCursor
	}
	if p.editingBody {
		return p.bodyCursorCol // Return column position
	}
	if p.editingHeader {
		if p.headerEditMode == "key" {
			return p.headerKeyCursor
		}
		return p.headerValueCursor
	}
	if p.editingQuery {
		if p.queryEditMode == "key" {
			return p.queryKeyCursor
		}
		return p.queryValueCursor
	}
	if p.editingPreScript {
		return p.preScriptCursorCol
	}
	if p.editingTestScript {
		return p.testScriptCursorCol
	}
	return 0
}

// PreRequestScript returns the pre-request script.
func (p *RequestPanel) PreRequestScript() string {
	if p.request == nil {
		return ""
	}
	return p.request.PreScript()
}

// TestScript returns the test script.
func (p *RequestPanel) TestScript() string {
	if p.request == nil {
		return ""
	}
	return p.request.PostScript()
}

// SetPreRequestScript sets the pre-request script.
func (p *RequestPanel) SetPreRequestScript(script string) {
	if p.request != nil {
		p.request.SetPreScript(script)
	}
}

// SetTestScript sets the test script.
func (p *RequestPanel) SetTestScript(script string) {
	if p.request != nil {
		p.request.SetPostScript(script)
	}
}
