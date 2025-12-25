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
	if p.request == nil {
		return ""
	}

	innerWidth := p.width - 2

	// Check if in method edit mode
	if p.editingMethod {
		return p.renderMethodSelector()
	}

	// Method button style (Postman-like dropdown look)
	method := p.request.Method()
	methodStyle := p.methodStyle(method)
	methodBtn := methodStyle.Render(method + " ▾")

	// Send button
	sendStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 1)
	sendBtn := sendStyle.Render("Send")

	// Calculate URL field width (method + spacing + url + spacing + send)
	sendBtnWidth := lipgloss.Width(sendBtn)
	methodWidth := lipgloss.Width(methodBtn)
	urlFieldWidth := innerWidth - methodWidth - sendBtnWidth - 4
	if urlFieldWidth < 10 {
		urlFieldWidth = 10
	}

	// URL content
	var urlContent string
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
		if len(urlContent) > urlFieldWidth {
			urlContent = urlContent[:urlFieldWidth-3] + "..."
		}
	}

	// Pad URL to fill field
	for len(urlContent) < urlFieldWidth {
		urlContent += " "
	}

	// URL field style - simple background highlight instead of border
	urlFieldStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	if p.editingURL {
		urlFieldStyle = urlFieldStyle.
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("214"))
	}

	urlField := urlFieldStyle.Render(urlContent)

	// Compose URL bar with proper spacing
	return methodBtn + " " + urlField + " " + sendBtn
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

	headers := p.request.Headers()
	if len(headers) == 0 {
		return []string{"No headers defined"}
	}

	var lines []string
	i := 0
	for key, value := range headers {
		prefix := "  "
		if i == p.cursor && p.focused {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, key, value))
		i++
	}
	return lines
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

	body := p.request.Body()
	if body == "" {
		return []string{"No body defined"}
	}

	return strings.Split(body, "\n")
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
