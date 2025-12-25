package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
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
	TabTests
)

var tabNames = []string{"URL", "Headers", "Query", "Body", "Auth", "Tests"}

// SendRequestMsg is sent when user wants to send the request.
type SendRequestMsg struct {
	Request *core.RequestDefinition
}

// ResponseReceivedMsg is sent when a response is received.
type ResponseReceivedMsg struct {
	Response *core.Response
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

	// Handle body editing mode
	if p.editingBody {
		return p.handleBodyEditInput(msg)
	}

	switch msg.Type {
	case tea.KeyTab:
		p.nextTab()
	case tea.KeyShiftTab:
		p.prevTab()
	case tea.KeyEnter:
		if p.activeTab == TabURL && p.request != nil {
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
				p.urlInput = p.request.URL()
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
		case "m":
			// Enter method edit mode
			if p.activeTab == TabURL && p.request != nil {
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
			}
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

// handleHeaderEditInput handles keyboard input while editing a header.
func (p *RequestPanel) handleHeaderEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel editing
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

// handleBodyEditInput handles keyboard input while editing the body.
func (p *RequestPanel) handleBodyEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Save and exit
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

	case tea.KeyRunes:
		char := string(msg.Runes)
		line := p.bodyLines[p.bodyCursorLine]
		p.bodyLines[p.bodyCursorLine] = line[:p.bodyCursorCol] + char + line[p.bodyCursorCol:]
		p.bodyCursorCol += len(char)
		return p, nil
	}

	return p, nil
}

func (p *RequestPanel) handleURLEditInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel editing, discard changes
		p.editingURL = false
		p.urlInput = ""
		return p, nil

	case tea.KeyEnter:
		// Save URL and exit edit mode
		if p.request != nil && p.urlInput != "" {
			p.request.SetURL(p.urlInput)
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
	case tea.KeyEsc:
		// Cancel editing
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

		content := emptyStyle.Render("No request selected")
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
		return emptyStyle.Render("Select a request from Collections (press 1)")
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
		urlContent = p.request.URL()
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

	// Add hint line below when focused
	if p.focused && p.activeTab == TabURL && !p.editingURL && !p.editingMethod {
		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		hint := hintStyle.Render("  Press e to edit URL, m to change method")
		return urlBar + "\n" + hint
	}

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
	hints := hintStyle.Render("↑/↓ j/k Select   Enter Save   Esc Cancel")

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
		fmt.Sprintf("URL: %s", p.request.URL()),
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

	// Editing hints
	if p.editingHeader {
		lines = append(lines, "")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		lines = append(lines, hintStyle.Render("  Tab: switch field │ Enter: save │ Esc: cancel"))
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

	params := p.request.QueryParams()
	if len(params) == 0 {
		return []string{"No query parameters defined"}
	}

	var lines []string
	i := 0
	for key, value := range params {
		prefix := "  "
		if i == p.cursor && p.focused {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, value))
		i++
	}
	return lines
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

		// Add hint when focused
		if p.focused {
			lines = append(lines, "")
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
			lines = append(lines, hintStyle.Render("  Press 'e' to edit body"))
		}
	}

	return lines
}

func (p *RequestPanel) renderAuthTab() []string {
	if p.request == nil {
		return []string{"No auth"}
	}

	auth := p.request.Auth()
	if auth == nil {
		return []string{"No authentication configured"}
	}

	return []string{
		fmt.Sprintf("Type: %s", auth.Type),
	}
}

func (p *RequestPanel) renderTestsTab() []string {
	return []string{"Tests not yet implemented"}
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
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240"))
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
	return p.editingURL || p.editingMethod || p.editingHeader || p.editingBody
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
