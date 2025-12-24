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

// RequestPanel displays and edits request details.
type RequestPanel struct {
	title     string
	focused   bool
	width     int
	height    int
	request   *core.RequestDefinition
	activeTab RequestTab
	cursor    int
	offset    int
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
		case "j":
			p.moveCursor(1)
		case "k":
			p.moveCursor(-1)
		}
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

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Width(p.width - 2).
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
		emptyStyle := lipgloss.NewStyle().
			Width(p.width - 4).
			Height(p.height - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240"))

		content := emptyStyle.Render("No request selected")
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Method and URL line
	methodStyle := p.methodStyle(p.request.Method())
	urlLine := methodStyle.Render(p.request.Method()) + " " + p.request.URL()

	// Tab bar
	tabBar := p.renderTabBar()

	// Tab content
	contentHeight := p.height - 7 // Title + URL + tabs + borders
	if contentHeight < 1 {
		contentHeight = 1
	}
	tabContent := p.renderTabContent(contentHeight)

	content := urlLine + "\n" + tabBar + "\n" + tabContent
	return p.wrapWithBorder(title + "\n" + content)
}

func (p *RequestPanel) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		style := lipgloss.NewStyle().Padding(0, 1)
		if RequestTab(i) == p.activeTab {
			if p.focused {
				style = style.
					Background(lipgloss.Color("62")).
					Foreground(lipgloss.Color("229")).
					Bold(true)
			} else {
				style = style.
					Background(lipgloss.Color("240")).
					Bold(true)
			}
		}
		tabs = append(tabs, style.Render(name))
	}
	return strings.Join(tabs, " ")
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
		Width(p.width).
		Height(p.height).
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
