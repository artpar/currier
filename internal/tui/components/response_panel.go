package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/tui"
)

// ResponseTab represents the active tab in the response panel.
type ResponseTab int

const (
	ResponseTabBody ResponseTab = iota
	ResponseTabHeaders
	ResponseTabCookies
	ResponseTabTiming
)

var responseTabNames = []string{"Body", "Headers", "Cookies", "Timing"}

// CopyMsg is sent when content should be copied.
type CopyMsg struct {
	Content string
}

// ResponsePanel displays response details.
type ResponsePanel struct {
	title        string
	focused      bool
	width        int
	height       int
	response     *core.Response
	activeTab    ResponseTab
	scrollOffset int
	loading      bool
}

// NewResponsePanel creates a new response panel.
func NewResponsePanel() *ResponsePanel {
	return &ResponsePanel{
		title:     "Response",
		activeTab: ResponseTabBody,
	}
}

// Init initializes the component.
func (p *ResponsePanel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (p *ResponsePanel) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
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

func (p *ResponsePanel) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		p.nextTab()
	case tea.KeyShiftTab:
		p.prevTab()
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			p.scrollOffset++
		case "k":
			if p.scrollOffset > 0 {
				p.scrollOffset--
			}
		case "y":
			if p.response != nil && p.activeTab == ResponseTabBody {
				return p, func() tea.Msg {
					return CopyMsg{Content: p.response.Body().String()}
				}
			}
		case "G":
			// Go to bottom
			p.scrollOffset = p.maxScrollOffset()
		case "g":
			// Go to top (gg sequence would need state tracking)
			p.scrollOffset = 0
		}
	}

	return p, nil
}

func (p *ResponsePanel) nextTab() {
	p.activeTab = ResponseTab((int(p.activeTab) + 1) % len(responseTabNames))
	p.scrollOffset = 0
}

func (p *ResponsePanel) prevTab() {
	p.activeTab = ResponseTab((int(p.activeTab) - 1 + len(responseTabNames)) % len(responseTabNames))
	p.scrollOffset = 0
}

func (p *ResponsePanel) maxScrollOffset() int {
	if p.response == nil {
		return 0
	}
	body := p.response.Body().String()
	lines := strings.Count(body, "\n") + 1
	visibleLines := p.height - 8
	if lines > visibleLines {
		return lines - visibleLines
	}
	return 0
}

// View renders the component.
func (p *ResponsePanel) View() string {
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

	// Loading state
	if p.loading {
		loadingStyle := lipgloss.NewStyle().
			Width(p.width - 4).
			Height(p.height - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("214"))

		content := loadingStyle.Render("Loading...")
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Empty state
	if p.response == nil {
		emptyStyle := lipgloss.NewStyle().
			Width(p.width - 4).
			Height(p.height - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240"))

		content := emptyStyle.Render("No response yet")
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Status line
	statusLine := p.renderStatusLine()

	// Tab bar
	tabBar := p.renderTabBar()

	// Tab content
	contentHeight := p.height - 7 // Title + status + tabs + borders
	if contentHeight < 1 {
		contentHeight = 1
	}
	tabContent := p.renderTabContent(contentHeight)

	content := statusLine + "\n" + tabBar + "\n" + tabContent
	return p.wrapWithBorder(title + "\n" + content)
}

func (p *ResponsePanel) renderStatusLine() string {
	status := p.response.Status()
	timing := p.response.Timing()

	// Status with color
	statusStyle := p.statusStyle(status.Code())
	statusStr := statusStyle.Render(fmt.Sprintf("%d %s", status.Code(), status.Text()))

	// Timing
	duration := timing.EndTime.Sub(timing.StartTime)
	timeStr := fmt.Sprintf("%.0fms", float64(duration.Milliseconds()))

	// Size
	sizeStr := p.formatSize(p.response.Body().Size())

	return fmt.Sprintf("%s  %s  %s", statusStr, timeStr, sizeStr)
}

func (p *ResponsePanel) statusStyle(code int) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	switch {
	case code >= 200 && code < 300:
		return style.Background(lipgloss.Color("34")).Foreground(lipgloss.Color("255"))
	case code >= 300 && code < 400:
		return style.Background(lipgloss.Color("214")).Foreground(lipgloss.Color("0"))
	case code >= 400 && code < 500:
		return style.Background(lipgloss.Color("208")).Foreground(lipgloss.Color("255"))
	case code >= 500:
		return style.Background(lipgloss.Color("160")).Foreground(lipgloss.Color("255"))
	default:
		return style.Background(lipgloss.Color("240"))
	}
}

func (p *ResponsePanel) formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func (p *ResponsePanel) renderTabBar() string {
	var tabs []string
	for i, name := range responseTabNames {
		style := lipgloss.NewStyle().Padding(0, 1)
		if ResponseTab(i) == p.activeTab {
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

func (p *ResponsePanel) renderTabContent(height int) string {
	var lines []string

	switch p.activeTab {
	case ResponseTabBody:
		lines = p.renderBodyTab()
	case ResponseTabHeaders:
		lines = p.renderHeadersTab()
	case ResponseTabCookies:
		lines = p.renderCookiesTab()
	case ResponseTabTiming:
		lines = p.renderTimingTab()
	}

	// Apply scroll offset
	if p.scrollOffset > 0 && p.scrollOffset < len(lines) {
		lines = lines[p.scrollOffset:]
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

func (p *ResponsePanel) renderBodyTab() []string {
	if p.response == nil {
		return []string{"No body"}
	}

	body := p.response.Body()
	if body.IsEmpty() {
		return []string{"(empty body)"}
	}

	// TODO: Add JSON formatting/syntax highlighting
	return strings.Split(body.String(), "\n")
}

func (p *ResponsePanel) renderHeadersTab() []string {
	if p.response == nil {
		return []string{"No headers"}
	}

	headers := p.response.Headers()
	keys := headers.Keys()
	if len(keys) == 0 {
		return []string{"No response headers"}
	}

	var lines []string
	for _, key := range keys {
		value := headers.Get(key)
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}
	return lines
}

func (p *ResponsePanel) renderCookiesTab() []string {
	// TODO: Parse Set-Cookie headers
	return []string{"Cookies not yet implemented"}
}

func (p *ResponsePanel) renderTimingTab() []string {
	if p.response == nil {
		return []string{"No timing info"}
	}

	timing := p.response.Timing()
	duration := timing.EndTime.Sub(timing.StartTime)

	return []string{
		fmt.Sprintf("Total Time: %.2fms", float64(duration.Milliseconds())),
		"",
		"Breakdown:",
		fmt.Sprintf("  DNS Lookup:     %.2fms", float64(timing.DNSLookup.Milliseconds())),
		fmt.Sprintf("  TCP Connection: %.2fms", float64(timing.TCPConnection.Milliseconds())),
		fmt.Sprintf("  TLS Handshake:  %.2fms", float64(timing.TLSHandshake.Milliseconds())),
	}
}

func (p *ResponsePanel) wrapWithBorder(content string) string {
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
func (p *ResponsePanel) Title() string {
	if p.response != nil {
		status := p.response.Status()
		return fmt.Sprintf("Response: %d %s", status.Code(), status.Text())
	}
	return p.title
}

// Focused returns true if focused.
func (p *ResponsePanel) Focused() bool {
	return p.focused
}

// Focus sets the component as focused.
func (p *ResponsePanel) Focus() {
	p.focused = true
}

// Blur removes focus.
func (p *ResponsePanel) Blur() {
	p.focused = false
}

// SetSize sets dimensions.
func (p *ResponsePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Width returns the width.
func (p *ResponsePanel) Width() int {
	return p.width
}

// Height returns the height.
func (p *ResponsePanel) Height() int {
	return p.height
}

// Response returns the current response.
func (p *ResponsePanel) Response() *core.Response {
	return p.response
}

// SetResponse sets the response to display.
func (p *ResponsePanel) SetResponse(resp *core.Response) {
	p.response = resp
	p.scrollOffset = 0
	p.loading = false
}

// ActiveTab returns the currently active tab.
func (p *ResponsePanel) ActiveTab() ResponseTab {
	return p.activeTab
}

// SetActiveTab sets the active tab.
func (p *ResponsePanel) SetActiveTab(tab ResponseTab) {
	p.activeTab = tab
	p.scrollOffset = 0
}

// ScrollOffset returns the scroll offset.
func (p *ResponsePanel) ScrollOffset() int {
	return p.scrollOffset
}

// SetScrollOffset sets the scroll offset.
func (p *ResponsePanel) SetScrollOffset(offset int) {
	p.scrollOffset = offset
}

// IsLoading returns true if loading.
func (p *ResponsePanel) IsLoading() bool {
	return p.loading
}

// SetLoading sets the loading state.
func (p *ResponsePanel) SetLoading(loading bool) {
	p.loading = loading
}

// IsSuccess returns true if status is 2xx.
func (p *ResponsePanel) IsSuccess() bool {
	if p.response == nil {
		return false
	}
	code := p.response.Status().Code()
	return code >= 200 && code < 300
}

// IsClientError returns true if status is 4xx.
func (p *ResponsePanel) IsClientError() bool {
	if p.response == nil {
		return false
	}
	code := p.response.Status().Code()
	return code >= 400 && code < 500
}

// IsServerError returns true if status is 5xx.
func (p *ResponsePanel) IsServerError() bool {
	if p.response == nil {
		return false
	}
	code := p.response.Status().Code()
	return code >= 500
}
