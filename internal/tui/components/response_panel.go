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
	ResponseTabConsole
)

var responseTabNames = []string{"Body", "Headers", "Cookies", "Timing", "Console"}

// CopyMsg is sent when content should be copied.
type CopyMsg struct {
	Content string
}

// ConsoleOutputMsg is sent when console output is available from scripts.
type ConsoleOutputMsg struct {
	Messages []ConsoleMessage
}

// ConsoleMessage represents a single console message.
type ConsoleMessage struct {
	Level   string // "log", "error", "warn", "info"
	Message string
}

// ResponsePanel displays response details.
type ResponsePanel struct {
	title           string
	focused         bool
	width           int
	height          int
	response        *core.Response
	activeTab       ResponseTab
	scrollOffset    int
	loading         bool
	err             error
	consoleMessages []ConsoleMessage
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
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "[":
			// Switch to previous tab
			p.prevTab()
			return p, nil
		case "]":
			// Switch to next tab
			p.nextTab()
			return p, nil
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

	// Loading state
	if p.loading {
		emptyHeight := innerHeight - 1
		if emptyHeight < 1 {
			emptyHeight = 1
		}
		loadingStyle := lipgloss.NewStyle().
			Width(innerWidth).
			Height(emptyHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("214"))

		content := loadingStyle.Render("Loading...")
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Error state
	if p.err != nil {
		emptyHeight := innerHeight - 1
		if emptyHeight < 1 {
			emptyHeight = 1
		}
		errorStyle := lipgloss.NewStyle().
			Width(innerWidth).
			Height(emptyHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("196"))

		content := errorStyle.Render("Error: " + p.err.Error())
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Empty state - still show tabs but with "No response yet" content
	if p.response == nil {
		// Tab bar (now 2 lines: tabs + indicator)
		tabBar := p.renderTabBar()

		emptyHeight := innerHeight - 3 // -1 title, -2 tabBar
		if emptyHeight < 1 {
			emptyHeight = 1
		}
		emptyStyle := lipgloss.NewStyle().
			Width(innerWidth).
			Height(emptyHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240"))

		emptyContent := emptyStyle.Render("No response yet")
		content := tabBar + "\n" + emptyContent
		return p.wrapWithBorder(title + "\n" + content)
	}

	// Status line
	statusLine := p.renderStatusLine()

	// Separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", innerWidth))

	// Tab bar (now 2 lines: tabs + indicator)
	tabBar := p.renderTabBar()

	// Tab content: innerHeight - title(1) - status(1) - separator(1) - tabBar(2)
	contentHeight := innerHeight - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	tabContent := p.renderTabContent(contentHeight)

	content := statusLine + "\n" + separator + "\n" + tabBar + "\n" + tabContent
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
	innerWidth := p.width - 2

	// Build two-line tab bar with underline indicator
	var topLine, bottomLine []string
	for i, name := range responseTabNames {
		if ResponseTab(i) == p.activeTab {
			activeColor := "214" // Orange
			if !p.focused {
				activeColor = "252" // White when not focused
			}
			activeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(activeColor)).
				Bold(true).
				Padding(0, 1)
			topLine = append(topLine, activeStyle.Render(name))
			bottomLine = append(bottomLine, lipgloss.NewStyle().
				Foreground(lipgloss.Color(activeColor)).
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
	case ResponseTabConsole:
		lines = p.renderConsoleTab()
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

	content := body.String()

	// Apply JSON syntax highlighting if content is JSON
	if IsJSON(content) {
		highlighter := NewJSONHighlighter()
		return highlighter.HighlightLines(content)
	}

	return strings.Split(content, "\n")
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
	if p.response == nil {
		return []string{"No cookies"}
	}

	headers := p.response.Headers()
	if headers == nil {
		return []string{"No cookies"}
	}

	setCookies := headers.GetAll("Set-Cookie")
	if len(setCookies) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		return []string{
			"No cookies in response",
			"",
			hintStyle.Render("Set-Cookie headers will be parsed and shown here."),
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Cookies: %d", len(setCookies)))
	lines = append(lines, "")

	for i, cookie := range setCookies {
		// Parse the cookie
		parsed := parseCookie(cookie)
		lines = append(lines, fmt.Sprintf("Cookie %d:", i+1))
		lines = append(lines, fmt.Sprintf("  Name:  %s", parsed.Name))
		lines = append(lines, fmt.Sprintf("  Value: %s", truncateValue(parsed.Value, 40)))
		if parsed.Domain != "" {
			lines = append(lines, fmt.Sprintf("  Domain: %s", parsed.Domain))
		}
		if parsed.Path != "" {
			lines = append(lines, fmt.Sprintf("  Path:   %s", parsed.Path))
		}
		if parsed.Expires != "" {
			lines = append(lines, fmt.Sprintf("  Expires: %s", parsed.Expires))
		}
		var flags []string
		if parsed.HttpOnly {
			flags = append(flags, "HttpOnly")
		}
		if parsed.Secure {
			flags = append(flags, "Secure")
		}
		if parsed.SameSite != "" {
			flags = append(flags, "SameSite="+parsed.SameSite)
		}
		if len(flags) > 0 {
			lines = append(lines, fmt.Sprintf("  Flags:  %s", strings.Join(flags, ", ")))
		}
		lines = append(lines, "")
	}

	return lines
}

// parsedCookie holds parsed cookie attributes.
type parsedCookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  string
	HttpOnly bool
	Secure   bool
	SameSite string
}

// parseCookie parses a Set-Cookie header value.
func parseCookie(raw string) parsedCookie {
	var c parsedCookie

	parts := strings.Split(raw, ";")
	if len(parts) == 0 {
		return c
	}

	// First part is name=value
	nameValue := strings.TrimSpace(parts[0])
	if idx := strings.Index(nameValue, "="); idx > 0 {
		c.Name = nameValue[:idx]
		c.Value = nameValue[idx+1:]
	} else {
		c.Name = nameValue
	}

	// Parse attributes
	for i := 1; i < len(parts); i++ {
		attr := strings.TrimSpace(parts[i])
		attrLower := strings.ToLower(attr)

		if attrLower == "httponly" {
			c.HttpOnly = true
		} else if attrLower == "secure" {
			c.Secure = true
		} else if strings.HasPrefix(attrLower, "domain=") {
			c.Domain = attr[7:]
		} else if strings.HasPrefix(attrLower, "path=") {
			c.Path = attr[5:]
		} else if strings.HasPrefix(attrLower, "expires=") {
			c.Expires = attr[8:]
		} else if strings.HasPrefix(attrLower, "samesite=") {
			c.SameSite = attr[9:]
		}
	}

	return c
}

// truncateValue truncates a string to maxLen with ellipsis.
func truncateValue(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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

func (p *ResponsePanel) renderConsoleTab() []string {
	if len(p.consoleMessages) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		return []string{
			"No console output",
			"",
			hintStyle.Render("Console output from pre-request scripts"),
			hintStyle.Render("and test scripts will appear here."),
		}
	}

	var lines []string
	for _, msg := range p.consoleMessages {
		// Color based on level
		var style lipgloss.Style
		switch msg.Level {
		case "error":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("160")) // Red
		case "warn":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Orange
		case "info":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // Blue
		default: // "log"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // White
		}

		prefix := fmt.Sprintf("[%s] ", msg.Level)
		lines = append(lines, style.Render(prefix+msg.Message))
	}

	return lines
}

func (p *ResponsePanel) wrapWithBorder(content string) string {
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
	p.err = nil
	p.consoleMessages = nil // Clear console for new response
}

// SetConsoleMessages sets the console output messages.
func (p *ResponsePanel) SetConsoleMessages(messages []ConsoleMessage) {
	p.consoleMessages = messages
}

// AddConsoleMessage adds a single console message.
func (p *ResponsePanel) AddConsoleMessage(level, message string) {
	p.consoleMessages = append(p.consoleMessages, ConsoleMessage{
		Level:   level,
		Message: message,
	})
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

// SetError sets an error to display.
func (p *ResponsePanel) SetError(err error) {
	p.err = err
	p.response = nil
	p.loading = false // Clear loading state on error
}

// Error returns the current error.
func (p *ResponsePanel) Error() error {
	return p.err
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

// --- State accessors for E2E testing ---

// HasResponse returns true if a response is loaded.
func (p *ResponsePanel) HasResponse() bool {
	return p.response != nil
}

// StatusCode returns the response status code.
func (p *ResponsePanel) StatusCode() int {
	if p.response == nil {
		return 0
	}
	return p.response.Status().Code()
}

// StatusText returns the response status text.
func (p *ResponsePanel) StatusText() string {
	if p.response == nil {
		return ""
	}
	return p.response.Status().Text()
}

// ResponseTime returns the response time in milliseconds.
func (p *ResponsePanel) ResponseTime() int64 {
	if p.response == nil {
		return 0
	}
	return p.response.Timing().Total.Milliseconds()
}

// BodySize returns the response body size in bytes.
func (p *ResponsePanel) BodySize() int64 {
	if p.response == nil {
		return 0
	}
	return p.response.Body().Size()
}

// BodyPreview returns the first n characters of the body.
func (p *ResponsePanel) BodyPreview(n int) string {
	if p.response == nil {
		return ""
	}
	body := p.response.Body().String()
	if len(body) > n {
		return body[:n]
	}
	return body
}

// ActiveTabName returns the active tab name as a string.
func (p *ResponsePanel) ActiveTabName() string {
	return responseTabNames[p.activeTab]
}

// ErrorString returns the error as a string, or empty if no error.
func (p *ResponsePanel) ErrorString() string {
	if p.err == nil {
		return ""
	}
	return p.err.Error()
}
