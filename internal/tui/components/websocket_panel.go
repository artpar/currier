package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/artpar/currier/internal/tui"
)

// WebSocketTab represents the active tab in the WebSocket panel.
type WebSocketTab int

const (
	WebSocketTabMessages WebSocketTab = iota
	WebSocketTabConnection
	WebSocketTabScripts
	WebSocketTabAutoResponse
)

var wsTabNames = []string{"Messages", "Connection", "Scripts", "Auto-Response"}

// WebSocket message types for bubble tea
type (
	// WSConnectedMsg is sent when connection is established.
	WSConnectedMsg struct {
		ConnectionID string
	}

	// WSDisconnectedMsg is sent when connection is closed.
	WSDisconnectedMsg struct {
		ConnectionID string
		Error        error
	}

	// WSMessageReceivedMsg is sent when a message is received.
	WSMessageReceivedMsg struct {
		Message *core.WebSocketMessage
	}

	// WSMessageSentMsg is sent when a message is sent.
	WSMessageSentMsg struct {
		Message *core.WebSocketMessage
	}

	// WSStateChangedMsg is sent when connection state changes.
	WSStateChangedMsg struct {
		State interfaces.ConnectionState
	}

	// WSErrorMsg is sent when an error occurs.
	WSErrorMsg struct {
		Error error
	}

	// WSSendMessageCmd requests sending a message.
	WSSendMessageCmd struct {
		Content string
	}

	// WSConnectCmd requests connecting to the endpoint.
	WSConnectCmd struct {
		Definition *core.WebSocketDefinition
	}

	// WSDisconnectCmd requests disconnecting.
	WSDisconnectCmd struct{}

	// WSReconnectCmd requests reconnecting.
	WSReconnectCmd struct{}
)

// WebSocketPanel displays WebSocket connection and messages.
type WebSocketPanel struct {
	title      string
	focused    bool
	width      int
	height     int
	activeTab  WebSocketTab
	gPressed   bool
	autoScroll bool

	// Connection state
	definition      *core.WebSocketDefinition
	connectionState interfaces.ConnectionState
	connectionID    string

	// Messages
	messages     []*core.WebSocketMessage
	scrollOffset int

	// Input field
	inputText   string
	inputCursor int
	inputMode   bool // true when typing in input field

	// Tab scroll offsets
	tabScrollOffset [4]int
}

// NewWebSocketPanel creates a new WebSocket panel.
func NewWebSocketPanel() *WebSocketPanel {
	return &WebSocketPanel{
		title:           "WebSocket",
		activeTab:       WebSocketTabMessages,
		autoScroll:      true,
		messages:        make([]*core.WebSocketMessage, 0),
		connectionState: interfaces.ConnectionStateDisconnected,
	}
}

// Init initializes the component.
func (p *WebSocketPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (p *WebSocketPanel) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case tui.FocusMsg:
		p.focused = true

	case tui.BlurMsg:
		p.focused = false
		p.inputMode = false

	case WSConnectedMsg:
		p.connectionID = msg.ConnectionID
		p.connectionState = interfaces.ConnectionStateConnected

	case WSDisconnectedMsg:
		p.connectionID = ""
		p.connectionState = interfaces.ConnectionStateDisconnected

	case WSStateChangedMsg:
		p.connectionState = msg.State

	case WSMessageReceivedMsg:
		p.messages = append(p.messages, msg.Message)
		if p.autoScroll {
			p.scrollToBottom()
		}

	case WSMessageSentMsg:
		p.messages = append(p.messages, msg.Message)
		if p.autoScroll {
			p.scrollToBottom()
		}

	case tea.KeyMsg:
		if p.focused {
			return p.handleKeyMsg(msg)
		}
	}

	return p, nil
}

func (p *WebSocketPanel) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Input mode handling
	if p.inputMode {
		return p.handleInputKey(msg)
	}

	// Normal mode
	pageSize := p.height - 10
	if pageSize < 1 {
		pageSize = 5
	}

	switch msg.Type {
	case tea.KeyEnter:
		// Enter input mode or send message
		if p.activeTab == WebSocketTabMessages {
			if p.inputText != "" && p.connectionState == interfaces.ConnectionStateConnected {
				// Send message
				content := p.inputText
				p.inputText = ""
				p.inputCursor = 0
				return p, func() tea.Msg {
					return WSSendMessageCmd{Content: content}
				}
			}
			p.inputMode = true
		}
		return p, nil

	case tea.KeyEsc:
		p.inputMode = false
		p.gPressed = false
		return p, nil

	case tea.KeyCtrlC:
		// Disconnect
		if p.connectionState == interfaces.ConnectionStateConnected {
			return p, func() tea.Msg {
				return WSDisconnectCmd{}
			}
		}
		return p, nil

	case tea.KeyCtrlR:
		// Reconnect
		return p, func() tea.Msg {
			return WSReconnectCmd{}
		}

	case tea.KeyPgUp, tea.KeyCtrlU:
		p.scrollOffset -= pageSize
		if p.scrollOffset < 0 {
			p.scrollOffset = 0
		}
		p.autoScroll = false
		p.gPressed = false
		return p, nil

	case tea.KeyPgDown, tea.KeyCtrlD:
		p.scrollOffset += pageSize
		maxOffset := p.maxScrollOffset()
		if p.scrollOffset >= maxOffset {
			p.scrollOffset = maxOffset
			p.autoScroll = true
		}
		p.gPressed = false
		return p, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "[":
			p.prevTab()
		case "]":
			p.nextTab()
		case "j":
			p.scrollOffset++
			if p.scrollOffset >= p.maxScrollOffset() {
				p.autoScroll = true
			} else {
				p.autoScroll = false
			}
		case "k":
			if p.scrollOffset > 0 {
				p.scrollOffset--
				p.autoScroll = false
			}
		case "G":
			p.scrollToBottom()
			p.autoScroll = true
			p.gPressed = false
		case "g":
			if p.gPressed {
				p.scrollOffset = 0
				p.autoScroll = false
				p.gPressed = false
			} else {
				p.gPressed = true
			}
			return p, nil
		case "i":
			// Enter input mode
			if p.activeTab == WebSocketTabMessages {
				p.inputMode = true
			}
		case "y":
			// Copy last message
			if len(p.messages) > 0 {
				lastMsg := p.messages[len(p.messages)-1]
				return p, func() tea.Msg {
					return CopyMsg{Content: lastMsg.Content}
				}
			}
		case "c":
			// Connect
			if p.connectionState == interfaces.ConnectionStateDisconnected && p.definition != nil {
				return p, func() tea.Msg {
					return WSConnectCmd{Definition: p.definition}
				}
			}
		case "d":
			// Disconnect
			if p.connectionState == interfaces.ConnectionStateConnected {
				return p, func() tea.Msg {
					return WSDisconnectCmd{}
				}
			}
		default:
			p.gPressed = false
		}
	}

	return p, nil
}

func (p *WebSocketPanel) handleInputKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		p.inputMode = false
		return p, nil

	case tea.KeyEnter:
		// Send message
		if p.inputText != "" && p.connectionState == interfaces.ConnectionStateConnected {
			content := p.inputText
			p.inputText = ""
			p.inputCursor = 0
			return p, func() tea.Msg {
				return WSSendMessageCmd{Content: content}
			}
		}
		return p, nil

	case tea.KeyBackspace:
		if p.inputCursor > 0 && len(p.inputText) > 0 {
			p.inputText = p.inputText[:p.inputCursor-1] + p.inputText[p.inputCursor:]
			p.inputCursor--
		}
		return p, nil

	case tea.KeyDelete:
		if p.inputCursor < len(p.inputText) {
			p.inputText = p.inputText[:p.inputCursor] + p.inputText[p.inputCursor+1:]
		}
		return p, nil

	case tea.KeyLeft:
		if p.inputCursor > 0 {
			p.inputCursor--
		}
		return p, nil

	case tea.KeyRight:
		if p.inputCursor < len(p.inputText) {
			p.inputCursor++
		}
		return p, nil

	case tea.KeyHome, tea.KeyCtrlA:
		p.inputCursor = 0
		return p, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		p.inputCursor = len(p.inputText)
		return p, nil

	case tea.KeyCtrlU:
		// Clear input
		p.inputText = ""
		p.inputCursor = 0
		return p, nil

	case tea.KeyRunes:
		// Insert characters
		runes := string(msg.Runes)
		p.inputText = p.inputText[:p.inputCursor] + runes + p.inputText[p.inputCursor:]
		p.inputCursor += len(runes)
		return p, nil
	}

	return p, nil
}

func (p *WebSocketPanel) nextTab() {
	p.tabScrollOffset[p.activeTab] = p.scrollOffset
	p.activeTab = WebSocketTab((int(p.activeTab) + 1) % len(wsTabNames))
	p.scrollOffset = p.tabScrollOffset[p.activeTab]
}

func (p *WebSocketPanel) prevTab() {
	p.tabScrollOffset[p.activeTab] = p.scrollOffset
	p.activeTab = WebSocketTab((int(p.activeTab) - 1 + len(wsTabNames)) % len(wsTabNames))
	p.scrollOffset = p.tabScrollOffset[p.activeTab]
}

func (p *WebSocketPanel) scrollToBottom() {
	p.scrollOffset = p.maxScrollOffset()
}

func (p *WebSocketPanel) maxScrollOffset() int {
	// Get content lines for current tab
	width := p.width - 4 // Account for borders and padding
	if width < 1 {
		width = 1
	}

	var lines []string
	switch p.activeTab {
	case WebSocketTabMessages:
		lines = p.renderMessagesTab(width)
	case WebSocketTabConnection:
		lines = p.renderConnectionTab(width)
	case WebSocketTabScripts:
		lines = p.renderScriptsTab(width)
	case WebSocketTabAutoResponse:
		lines = p.renderAutoResponseTab(width)
	default:
		return 0
	}

	visibleLines := p.height - 12 // Account for header, tabs, input, borders
	if visibleLines < 1 {
		visibleLines = 1
	}
	if len(lines) > visibleLines {
		return len(lines) - visibleLines
	}
	return 0
}

// View renders the component.
func (p *WebSocketPanel) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	innerWidth := p.width - 2
	innerHeight := p.height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Title bar
	title := p.renderTitleBar(innerWidth)

	// Status line
	statusLine := p.renderStatusLine(innerWidth)

	// Tab bar
	tabBar := p.renderTabBar(innerWidth)

	// Tab content
	contentHeight := innerHeight - 6 // title, status, tabs(2), input
	if contentHeight < 1 {
		contentHeight = 1
	}
	tabContent := p.renderTabContent(innerWidth, contentHeight)

	// Input line (only for Messages tab)
	inputLine := p.renderInputLine(innerWidth)

	content := title + "\n" + statusLine + "\n" + tabBar + "\n" + tabContent + "\n" + inputLine
	return p.wrapWithBorder(content)
}

func (p *WebSocketPanel) renderTitleBar(width int) string {
	titleStyle := lipgloss.NewStyle().
		Width(width).
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

	titleText := p.title
	if p.definition != nil {
		titleText = fmt.Sprintf("WebSocket: %s", p.definition.Name)
	}

	return titleStyle.Render(titleText)
}

func (p *WebSocketPanel) renderStatusLine(width int) string {
	// Connection status with color
	var statusStyle lipgloss.Style
	var statusText string

	switch p.connectionState {
	case interfaces.ConnectionStateConnected:
		statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)
		statusText = "Connected"
	case interfaces.ConnectionStateConnecting:
		statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1)
		statusText = "Connecting..."
	case interfaces.ConnectionStateDisconnecting:
		statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("208")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)
		statusText = "Disconnecting..."
	case interfaces.ConnectionStateError:
		statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("160")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)
		statusText = "Error"
	default:
		statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)
		statusText = "Disconnected"
	}

	status := statusStyle.Render(statusText)

	// Endpoint
	endpoint := ""
	if p.definition != nil {
		endpoint = p.definition.Endpoint
		// Truncate if too long
		maxLen := width - lipgloss.Width(status) - 5
		if maxLen > 0 && len(endpoint) > maxLen {
			endpoint = endpoint[:maxLen-3] + "..."
		}
	}

	endpointStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	return fmt.Sprintf("%s  %s", status, endpointStyle.Render(endpoint))
}

func (p *WebSocketPanel) renderTabBar(width int) string {
	var topLine, bottomLine []string

	for i, name := range wsTabNames {
		if WebSocketTab(i) == p.activeTab {
			activeColor := "214"
			if !p.focused {
				activeColor = "252"
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

	topRow := strings.Join(topLine, " ")
	bottomRow := strings.Join(bottomLine, " ")
	remainingWidth := width - lipgloss.Width(bottomRow)
	if remainingWidth > 0 {
		bottomRow += lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Render(strings.Repeat("─", remainingWidth))
	}

	return topRow + "\n" + bottomRow
}

func (p *WebSocketPanel) renderTabContent(width, height int) string {
	var lines []string

	switch p.activeTab {
	case WebSocketTabMessages:
		lines = p.renderMessagesTab(width)
	case WebSocketTabConnection:
		lines = p.renderConnectionTab(width)
	case WebSocketTabScripts:
		lines = p.renderScriptsTab(width)
	case WebSocketTabAutoResponse:
		lines = p.renderAutoResponseTab(width)
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

func (p *WebSocketPanel) renderMessagesTab(width int) []string {
	if len(p.messages) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
		return []string{
			"",
			hintStyle.Render("No messages yet"),
			"",
			hintStyle.Render("Press 'c' to connect, 'i' or Enter to type a message"),
		}
	}

	var lines []string
	sentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))     // Blue for sent
	recvStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34"))     // Green for received
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))    // Gray for timestamp
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("160"))   // Red for errors
	autoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))    // Orange for auto-response

	for _, msg := range p.messages {
		// Direction indicator
		var prefix string
		var contentStyle lipgloss.Style

		if msg.IsSent() {
			prefix = "→ "
			contentStyle = sentStyle
			if msg.AutoResponse {
				prefix = "⟳ "
				contentStyle = autoStyle
			}
		} else {
			prefix = "← "
			contentStyle = recvStyle
		}

		if msg.Error != "" {
			prefix = "✗ "
			contentStyle = errorStyle
		}

		// Timestamp
		timestamp := timeStyle.Render(msg.Timestamp.Format("15:04:05"))

		// Content (truncate long messages)
		content := msg.Content
		maxContentLen := width - len(prefix) - 12 // Reserve space for timestamp
		if maxContentLen > 0 && len(content) > maxContentLen {
			content = content[:maxContentLen-3] + "..."
		}

		// Build line
		line := fmt.Sprintf("%s%s  %s", prefix, contentStyle.Render(content), timestamp)
		lines = append(lines, line)

		// Show error if present
		if msg.Error != "" {
			lines = append(lines, errorStyle.Render("  Error: "+msg.Error))
		}
	}

	return lines
}

func (p *WebSocketPanel) renderConnectionTab(width int) []string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)

	if p.definition == nil {
		return []string{
			"",
			hintStyle.Render("No WebSocket definition selected"),
		}
	}

	lines = append(lines, labelStyle.Render("Endpoint:"))
	lines = append(lines, valueStyle.Render("  "+p.definition.Endpoint))
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Connection ID:"))
	if p.connectionID != "" {
		lines = append(lines, valueStyle.Render("  "+p.connectionID))
	} else {
		lines = append(lines, valueStyle.Render("  (not connected)"))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Headers:"))
	if len(p.definition.Headers) == 0 {
		lines = append(lines, valueStyle.Render("  (none)"))
	} else {
		for k, v := range p.definition.Headers {
			lines = append(lines, valueStyle.Render(fmt.Sprintf("  %s: %s", k, v)))
		}
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Subprotocols:"))
	if len(p.definition.Subprotocols) == 0 {
		lines = append(lines, valueStyle.Render("  (none)"))
	} else {
		lines = append(lines, valueStyle.Render("  "+strings.Join(p.definition.Subprotocols, ", ")))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Settings:"))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Ping Interval: %ds", p.definition.PingInterval)))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Reconnect: %v (max %d attempts)", p.definition.ReconnectEnabled, p.definition.MaxReconnectAttempts)))

	return lines
}

func (p *WebSocketPanel) renderScriptsTab(width int) []string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	if p.definition == nil {
		return []string{
			"",
			hintStyle.Render("No WebSocket definition selected"),
		}
	}

	lines = append(lines, labelStyle.Render("Pre-Connect Script:"))
	if p.definition.PreConnectScript == "" {
		lines = append(lines, hintStyle.Render("  (none)"))
	} else {
		lines = append(lines, codeStyle.Render("  "+truncateScript(p.definition.PreConnectScript, width-4)))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Pre-Message Script:"))
	if p.definition.PreMessageScript == "" {
		lines = append(lines, hintStyle.Render("  (none)"))
	} else {
		lines = append(lines, codeStyle.Render("  "+truncateScript(p.definition.PreMessageScript, width-4)))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Post-Message Script:"))
	if p.definition.PostMessageScript == "" {
		lines = append(lines, hintStyle.Render("  (none)"))
	} else {
		lines = append(lines, codeStyle.Render("  "+truncateScript(p.definition.PostMessageScript, width-4)))
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Filter Script:"))
	if p.definition.FilterScript == "" {
		lines = append(lines, hintStyle.Render("  (none)"))
	} else {
		lines = append(lines, codeStyle.Render("  "+truncateScript(p.definition.FilterScript, width-4)))
	}

	return lines
}

func (p *WebSocketPanel) renderAutoResponseTab(width int) []string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	enabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("160"))

	if p.definition == nil {
		return []string{
			"",
			hintStyle.Render("No WebSocket definition selected"),
		}
	}

	if len(p.definition.AutoResponseRules) == 0 {
		return []string{
			"",
			hintStyle.Render("No auto-response rules configured"),
			"",
			hintStyle.Render("Auto-response rules automatically send messages"),
			hintStyle.Render("when incoming messages match a pattern."),
		}
	}

	lines = append(lines, labelStyle.Render(fmt.Sprintf("Auto-Response Rules (%d):", len(p.definition.AutoResponseRules))))
	lines = append(lines, "")

	for i, rule := range p.definition.AutoResponseRules {
		status := disabledStyle.Render("✗")
		if rule.Enabled {
			status = enabledStyle.Render("✓")
		}

		lines = append(lines, fmt.Sprintf("%s %d. %s", status, i+1, rule.Name))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("    Match: %s", truncateScript(rule.MatchScript, width-12))))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("    Reply: %s", truncateScript(rule.Response, width-12))))
		lines = append(lines, "")
	}

	return lines
}

func (p *WebSocketPanel) renderInputLine(width int) string {
	if p.activeTab != WebSocketTabMessages {
		return ""
	}

	// Input prompt
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	prompt := "> "
	if p.inputMode {
		prompt = ">> "
	}

	// Show cursor in input mode
	displayText := p.inputText
	if p.inputMode {
		// Insert cursor
		if p.inputCursor >= len(displayText) {
			displayText += "█"
		} else {
			displayText = displayText[:p.inputCursor] + "█" + displayText[p.inputCursor:]
		}
	}

	// Truncate if too long
	maxInputLen := width - len(prompt) - 20
	if maxInputLen > 0 && len(displayText) > maxInputLen {
		displayText = displayText[:maxInputLen]
	}

	inputLine := promptStyle.Render(prompt) + inputStyle.Render(displayText)

	// Hint
	hint := ""
	if p.connectionState != interfaces.ConnectionStateConnected {
		hint = hintStyle.Render("  [disconnected]")
	} else if !p.inputMode {
		hint = hintStyle.Render("  [i: type, Enter: send]")
	} else {
		hint = hintStyle.Render("  [Esc: cancel]")
	}

	return inputLine + hint
}

func (p *WebSocketPanel) wrapWithBorder(content string) string {
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	if p.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("244"))
	}

	return borderStyle.Render(content)
}

// Helper to truncate script text
func truncateScript(s string, maxLen int) string {
	// Remove newlines for display
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// Title returns the component title.
func (p *WebSocketPanel) Title() string {
	if p.definition != nil {
		return fmt.Sprintf("WS: %s", p.definition.Name)
	}
	return p.title
}

// Focused returns true if focused.
func (p *WebSocketPanel) Focused() bool {
	return p.focused
}

// Focus sets the component as focused.
func (p *WebSocketPanel) Focus() {
	p.focused = true
}

// Blur removes focus.
func (p *WebSocketPanel) Blur() {
	p.focused = false
	p.inputMode = false
}

// SetSize sets dimensions.
func (p *WebSocketPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Width returns the width.
func (p *WebSocketPanel) Width() int {
	return p.width
}

// Height returns the height.
func (p *WebSocketPanel) Height() int {
	return p.height
}

// SetDefinition sets the WebSocket definition.
func (p *WebSocketPanel) SetDefinition(def *core.WebSocketDefinition) {
	p.definition = def
	p.messages = nil
	p.scrollOffset = 0
	p.connectionID = ""
	p.connectionState = interfaces.ConnectionStateDisconnected
}

// Definition returns the current definition.
func (p *WebSocketPanel) Definition() *core.WebSocketDefinition {
	return p.definition
}

// ConnectionState returns the current connection state.
func (p *WebSocketPanel) ConnectionState() interfaces.ConnectionState {
	return p.connectionState
}

// SetConnectionState sets the connection state.
func (p *WebSocketPanel) SetConnectionState(state interfaces.ConnectionState) {
	p.connectionState = state
}

// ConnectionID returns the current connection ID.
func (p *WebSocketPanel) ConnectionID() string {
	return p.connectionID
}

// SetConnectionID sets the connection ID.
func (p *WebSocketPanel) SetConnectionID(id string) {
	p.connectionID = id
}

// Messages returns all messages.
func (p *WebSocketPanel) Messages() []*core.WebSocketMessage {
	return p.messages
}

// AddMessage adds a message to the display.
func (p *WebSocketPanel) AddMessage(msg *core.WebSocketMessage) {
	p.messages = append(p.messages, msg)
	if p.autoScroll {
		p.scrollToBottom()
	}
}

// ClearMessages clears all messages.
func (p *WebSocketPanel) ClearMessages() {
	p.messages = nil
	p.scrollOffset = 0
}

// MessageCount returns the number of messages.
func (p *WebSocketPanel) MessageCount() int {
	return len(p.messages)
}

// InputText returns the current input text.
func (p *WebSocketPanel) InputText() string {
	return p.inputText
}

// SetInputText sets the input text.
func (p *WebSocketPanel) SetInputText(text string) {
	p.inputText = text
	p.inputCursor = len(text)
}

// IsInputMode returns true if in input mode.
func (p *WebSocketPanel) IsInputMode() bool {
	return p.inputMode
}

// ActiveTab returns the active tab.
func (p *WebSocketPanel) ActiveTab() WebSocketTab {
	return p.activeTab
}

// SetActiveTab sets the active tab.
func (p *WebSocketPanel) SetActiveTab(tab WebSocketTab) {
	p.activeTab = tab
	p.scrollOffset = 0
}

// ActiveTabName returns the active tab name.
func (p *WebSocketPanel) ActiveTabName() string {
	return wsTabNames[p.activeTab]
}

// GPressed returns true if waiting for second 'g'.
func (p *WebSocketPanel) GPressed() bool {
	return p.gPressed
}

// AutoScroll returns true if auto-scrolling is enabled.
func (p *WebSocketPanel) AutoScroll() bool {
	return p.autoScroll
}

// SetAutoScroll sets auto-scroll mode.
func (p *WebSocketPanel) SetAutoScroll(enabled bool) {
	p.autoScroll = enabled
}

// Ensure time is imported for timestamp handling
var _ = time.Now
