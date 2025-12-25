package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/interpolate"
	httpclient "github.com/artpar/currier/internal/protocol/http"
	"github.com/artpar/currier/internal/tui"
	"github.com/artpar/currier/internal/tui/components"
)

// Pane represents which pane is focused.
type Pane int

const (
	PaneCollections Pane = iota
	PaneRequest
	PaneResponse
)

// MainView is the main three-pane view.
type MainView struct {
	width        int
	height       int
	focusedPane  Pane
	tree         *components.CollectionTree
	request      *components.RequestPanel
	response     *components.ResponsePanel
	showHelp     bool
	environment  *core.Environment
	interpolator *interpolate.Engine
	notification string    // Temporary notification message
	notifyUntil  time.Time // When to clear notification
	historyStore history.Store // Store for request history
	lastRequest  *core.RequestDefinition // Last sent request for history
}

// clearNotificationMsg is sent to clear the notification.
type clearNotificationMsg struct{}

// NewMainView creates a new main view.
func NewMainView() *MainView {
	view := &MainView{
		tree:         components.NewCollectionTree(),
		request:      components.NewRequestPanel(),
		response:     components.NewResponsePanel(),
		focusedPane:  PaneCollections,
		interpolator: interpolate.NewEngine(), // Default engine with builtins
	}
	view.tree.Focus()
	return view
}

// Init initializes the view.
func (v *MainView) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (v *MainView) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
	// Handle help overlay first
	if v.showHelp {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEsc || string(keyMsg.Runes) == "?" {
				v.showHelp = false
				return v, nil
			}
		}
		return v, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.updatePaneSizes()
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case components.SelectionMsg:
		v.request.SetRequest(msg.Request)
		v.focusPane(PaneRequest)
		return v, nil

	case components.SelectHistoryItemMsg:
		// Create a request from history entry
		name := msg.Entry.RequestName
		if name == "" {
			name = "History Request"
		}
		req := core.NewRequestDefinition(
			name,
			msg.Entry.RequestMethod,
			msg.Entry.RequestURL,
		)
		// Set body if present
		if msg.Entry.RequestBody != "" {
			req.SetBody(msg.Entry.RequestBody)
		}
		// Set headers if present
		for key, value := range msg.Entry.RequestHeaders {
			req.SetHeader(key, value)
		}
		v.request.SetRequest(req)
		v.focusPane(PaneRequest)
		return v, nil

	case components.SendRequestMsg:
		v.response.SetLoading(true)
		v.focusPane(PaneResponse)
		v.lastRequest = msg.Request // Save for history
		return v, sendRequest(msg.Request, v.interpolator)

	case components.ResponseReceivedMsg:
		v.response.SetLoading(false)
		v.response.SetResponse(msg.Response)
		// Save to history
		if v.historyStore != nil && v.lastRequest != nil {
			go v.saveToHistory(v.lastRequest, msg.Response, nil)
		}
		return v, nil

	case components.RequestErrorMsg:
		v.response.SetLoading(false)
		v.response.SetError(msg.Error)
		// Save failed request to history too
		if v.historyStore != nil && v.lastRequest != nil {
			go v.saveToHistory(v.lastRequest, nil, msg.Error)
		}
		return v, nil

	case components.CopyMsg:
		return v.handleCopy(msg.Content)

	case clearNotificationMsg:
		v.notification = ""
		return v, nil
	}

	// Forward messages to focused pane
	return v.forwardToFocusedPane(msg)
}

func (v *MainView) handleCopy(content string) (tui.Component, tea.Cmd) {
	err := clipboard.WriteAll(content)
	if err != nil {
		v.notification = "✗ Copy failed"
	} else {
		size := len(content)
		if size > 1024 {
			v.notification = fmt.Sprintf("✓ Copied %.1fKB", float64(size)/1024)
		} else {
			v.notification = fmt.Sprintf("✓ Copied %dB", size)
		}
	}
	v.notifyUntil = time.Now().Add(2 * time.Second)

	// Schedule clearing the notification
	return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func (v *MainView) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Check if we're in INSERT mode (editing text in any pane)
	// In INSERT mode, forward ALL keys to the focused pane except Ctrl+C
	isEditing := v.request.IsEditing() || v.tree.IsSearching()

	// Ctrl+C always quits
	if msg.Type == tea.KeyCtrlC {
		return v, tea.Quit
	}

	// In INSERT mode, forward everything to the focused pane
	// Only Escape exits insert mode (handled by the pane itself)
	if isEditing {
		return v.forwardToFocusedPane(msg)
	}

	// NORMAL mode - handle global shortcuts
	switch msg.Type {
	case tea.KeyTab:
		// Tab always cycles panes
		v.cycleFocusForward()
		return v, nil

	case tea.KeyShiftTab:
		// Shift+Tab always cycles panes backward
		v.cycleFocusBackward()
		return v, nil

	case tea.KeyEsc:
		// Already in normal mode, nothing to do
		return v, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			return v, tea.Quit
		case "?":
			v.showHelp = true
			return v, nil
		case "1":
			v.focusPane(PaneCollections)
			return v, nil
		case "2":
			v.focusPane(PaneRequest)
			return v, nil
		case "3":
			v.focusPane(PaneResponse)
			return v, nil
		case "n":
			// Create a new request and add it to a collection
			newReq := core.NewRequestDefinition("New Request", "GET", "")

			// Add to collection tree (creates default collection if none exist)
			v.tree.AddRequest(newReq, nil)

			// Set it in the request panel
			v.request.SetRequest(newReq)
			v.focusPane(PaneRequest)
			// Auto-enter URL edit mode
			v.request.StartURLEdit()
			return v, nil
		}
	}

	// Forward to focused pane for other keys
	return v.forwardToFocusedPane(msg)
}

func (v *MainView) forwardToFocusedPane(msg tea.Msg) (tui.Component, tea.Cmd) {
	var cmd tea.Cmd

	switch v.focusedPane {
	case PaneCollections:
		updated, c := v.tree.Update(msg)
		v.tree = updated.(*components.CollectionTree)
		cmd = c
	case PaneRequest:
		updated, c := v.request.Update(msg)
		v.request = updated.(*components.RequestPanel)
		cmd = c
	case PaneResponse:
		updated, c := v.response.Update(msg)
		v.response = updated.(*components.ResponsePanel)
		cmd = c
	}

	return v, cmd
}

func (v *MainView) cycleFocusForward() {
	v.focusPane(Pane((int(v.focusedPane) + 1) % 3))
}

func (v *MainView) cycleFocusBackward() {
	v.focusPane(Pane((int(v.focusedPane) + 2) % 3))
}

func (v *MainView) focusPane(pane Pane) {
	// Blur all
	v.tree.Blur()
	v.request.Blur()
	v.response.Blur()

	// Focus the target
	v.focusedPane = pane
	switch pane {
	case PaneCollections:
		v.tree.Focus()
	case PaneRequest:
		v.request.Focus()
	case PaneResponse:
		v.response.Focus()
	}
}

func (v *MainView) updatePaneSizes() {
	if v.width == 0 || v.height == 0 {
		return
	}

	// Postman-like layout:
	// [Sidebar 25%] | [Request/Response stacked vertically 75%]
	sidebarWidth := v.width * 25 / 100
	if sidebarWidth < 25 {
		sidebarWidth = 25
	}
	if sidebarWidth > 60 {
		sidebarWidth = 60
	}
	rightWidth := v.width - sidebarWidth

	// Reserve 2 lines for help bar + status bar
	totalHeight := v.height - 2
	if totalHeight < 2 {
		totalHeight = 2
	}

	// Split right side vertically: Request on top (45%), Response on bottom (55%)
	requestHeight := totalHeight * 45 / 100
	if requestHeight < 8 {
		requestHeight = 8
	}
	responseHeight := totalHeight - requestHeight

	// Set sizes
	v.tree.SetSize(sidebarWidth, totalHeight)
	v.request.SetSize(rightWidth, requestHeight)
	v.response.SetSize(rightWidth, responseHeight)
}

// View renders the view.
func (v *MainView) View() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Render help overlay if showing
	if v.showHelp {
		return v.renderHelp()
	}

	// Postman-like layout:
	// [Sidebar] | [Request  ]
	//           | [Response ]

	// Render sidebar (Collections)
	sidebar := v.tree.View()

	// Render request and response panels
	requestPane := v.request.View()
	responsePane := v.response.View()

	// Stack request and response vertically on the right
	rightStack := lipgloss.JoinVertical(lipgloss.Left, requestPane, responsePane)

	// Join sidebar with right stack horizontally
	panes := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightStack)

	// Render help bar and status bar
	helpBar := v.renderHelpBar()
	statusBar := v.renderStatusBar()

	// Join panes, help bar, and status bar vertically
	return lipgloss.JoinVertical(lipgloss.Left, panes, helpBar, statusBar)
}

// renderHelpBar renders context-sensitive keyboard shortcuts.
func (v *MainView) renderHelpBar() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	sep := sepStyle.Render(" │ ")

	var hints []string

	// Context-sensitive hints based on focused pane and mode
	switch v.focusedPane {
	case PaneCollections:
		if v.tree.IsSearching() {
			hints = []string{
				keyStyle.Render("Enter") + descStyle.Render(" Apply"),
				keyStyle.Render("Esc") + descStyle.Render(" Cancel"),
				keyStyle.Render("Ctrl+U") + descStyle.Render(" Clear"),
			}
		} else {
			hints = []string{
				keyStyle.Render("j/k") + descStyle.Render(" Navigate"),
				keyStyle.Render("Enter") + descStyle.Render(" Select"),
				keyStyle.Render("/") + descStyle.Render(" Search"),
				keyStyle.Render("Tab") + descStyle.Render(" Next pane"),
			}
		}
	case PaneRequest:
		if v.request.IsEditing() {
			hints = []string{
				keyStyle.Render("Enter") + descStyle.Render(" Save"),
				keyStyle.Render("Esc") + descStyle.Render(" Cancel"),
			}
		} else {
			hints = []string{
				keyStyle.Render("e") + descStyle.Render(" Edit URL"),
				keyStyle.Render("m") + descStyle.Render(" Method"),
				keyStyle.Render("Enter") + descStyle.Render(" Send"),
				keyStyle.Render("[/]") + descStyle.Render(" Switch tab"),
			}
		}
	case PaneResponse:
		hints = []string{
			keyStyle.Render("j/k") + descStyle.Render(" Scroll"),
			keyStyle.Render("y") + descStyle.Render(" Copy body"),
			keyStyle.Render("[/]") + descStyle.Render(" Switch tab"),
		}
	}

	// Always add global hints
	hints = append(hints,
		keyStyle.Render("n")+descStyle.Render(" New"),
		keyStyle.Render("1/2/3")+descStyle.Render(" Pane"),
		keyStyle.Render("?")+descStyle.Render(" Help"),
		keyStyle.Render("q")+descStyle.Render(" Quit"),
	)

	content := strings.Join(hints, sep)

	// Help bar style
	barStyle := lipgloss.NewStyle().
		Width(v.width).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	return barStyle.Render(content)
}

// renderStatusBar renders the bottom status bar with environment info.
func (v *MainView) renderStatusBar() string {
	// Build status items
	var items []string

	// Mode indicator
	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)
	isEditing := v.request.IsEditing() || v.tree.IsSearching()
	if isEditing {
		modeStyle = modeStyle.
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0"))
		items = append(items, modeStyle.Render("INSERT"))
	} else {
		modeStyle = modeStyle.
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("255"))
		items = append(items, modeStyle.Render("NORMAL"))
	}

	// Focused pane indicator
	paneStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
	paneName := "Collections"
	switch v.focusedPane {
	case PaneRequest:
		paneName = "Request"
	case PaneResponse:
		paneName = "Response"
	}
	items = append(items, paneStyle.Render(paneName))

	// Environment indicator
	if v.environment != nil {
		envStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("229")).
			Padding(0, 1).
			Bold(true)
		items = append(items, envStyle.Render("ENV: "+v.environment.Name()))
	} else {
		envStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("250")).
			Padding(0, 1)
		items = append(items, envStyle.Render("No Environment"))
	}

	// Add variable count if environment exists
	if v.environment != nil {
		vars := v.environment.Variables()
		secrets := v.environment.SecretNames()
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)
		items = append(items, countStyle.Render(fmt.Sprintf("%d vars, %d secrets", len(vars), len(secrets))))
	}

	// Add notification if present
	if v.notification != "" {
		notifyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")).
			Bold(true).
			Padding(0, 1)
		if strings.HasPrefix(v.notification, "✗") {
			notifyStyle = notifyStyle.Foreground(lipgloss.Color("160"))
		}
		items = append(items, notifyStyle.Render(v.notification))
	}

	// Help hint on the right
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(0, 1)
	helpHint := helpStyle.Render("? help  q quit")

	// Calculate spacing
	leftContent := strings.Join(items, " ")
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(helpHint)
	spacerWidth := v.width - leftWidth - rightWidth - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := strings.Repeat(" ", spacerWidth)

	// Status bar style
	barStyle := lipgloss.NewStyle().
		Width(v.width).
		Background(lipgloss.Color("236"))

	return barStyle.Render(leftContent + spacer + helpHint)
}

func (v *MainView) renderHelp() string {
	helpContent := []string{
		"╭─────────────────── Currier Help ───────────────────╮",
		"│                                                     │",
		"│  Navigation                                         │",
		"│    Tab / Shift+Tab    Cycle between panes          │",
		"│    [ / ]              Switch tabs within pane      │",
		"│    1 / 2 / 3          Jump to pane                 │",
		"│    j / k              Move down/up                 │",
		"│    h / l              Collapse/Expand              │",
		"│    gg / G             Go to top/bottom             │",
		"│                                                     │",
		"│  Collections Pane                                   │",
		"│    /                  Start search                 │",
		"│    H                  Switch to History view       │",
		"│    C                  Switch to Collections view   │",
		"│    Esc                Clear search filter          │",
		"│                                                     │",
		"│  Request Pane                                       │",
		"│    m                  Change HTTP method           │",
		"│    e                  Edit URL/header/body         │",
		"│    a                  Add header/query param       │",
		"│    d                  Delete header/query param    │",
		"│    Enter              Send request                 │",
		"│                                                     │",
		"│  Response Pane                                      │",
		"│    y                  Copy response body           │",
		"│                                                     │",
		"│  General                                            │",
		"│    ?                  Toggle this help             │",
		"│    q / Ctrl+C         Quit                         │",
		"│                                                     │",
		"│           Press ? or Esc to close                  │",
		"╰─────────────────────────────────────────────────────╯",
	}

	helpStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return helpStyle.Render(strings.Join(helpContent, "\n"))
}

// Name returns the view name.
func (v *MainView) Name() string {
	return "Main"
}

// Title returns the view title.
func (v *MainView) Title() string {
	return "Currier"
}

// Focused returns true if focused.
func (v *MainView) Focused() bool {
	return true // MainView is always focused
}

// Focus sets focus.
func (v *MainView) Focus() {}

// Blur removes focus.
func (v *MainView) Blur() {}

// SetSize sets dimensions.
func (v *MainView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updatePaneSizes()
}

// Width returns the width.
func (v *MainView) Width() int {
	return v.width
}

// Height returns the height.
func (v *MainView) Height() int {
	return v.height
}

// FocusedPane returns the currently focused pane.
func (v *MainView) FocusedPane() Pane {
	return v.focusedPane
}

// FocusPane focuses a specific pane.
func (v *MainView) FocusPane(pane Pane) {
	v.focusPane(pane)
}

// CollectionTree returns the collection tree component.
func (v *MainView) CollectionTree() *components.CollectionTree {
	return v.tree
}

// RequestPanel returns the request panel component.
func (v *MainView) RequestPanel() *components.RequestPanel {
	return v.request
}

// ResponsePanel returns the response panel component.
func (v *MainView) ResponsePanel() *components.ResponsePanel {
	return v.response
}

// SetCollections sets the collections to display and auto-selects the first request.
func (v *MainView) SetCollections(collections []*core.Collection) {
	v.tree.SetCollections(collections)

	// Auto-select the first request from the first collection
	if len(collections) > 0 {
		for _, col := range collections {
			if req := col.FirstRequest(); req != nil {
				v.request.SetRequest(req)
				break
			}
		}
	}
}

// SetEnvironment sets the environment and interpolation engine.
func (v *MainView) SetEnvironment(env *core.Environment, engine *interpolate.Engine) {
	v.environment = env
	v.interpolator = engine
}

// SetHistoryStore sets the history store for browsing request history.
func (v *MainView) SetHistoryStore(store history.Store) {
	v.historyStore = store
	v.tree.SetHistoryStore(store)
}

// saveToHistory saves a request/response pair to history.
func (v *MainView) saveToHistory(req *core.RequestDefinition, resp *core.Response, err error) {
	if v.historyStore == nil || req == nil {
		return
	}

	entry := history.Entry{
		RequestMethod:  req.Method(),
		RequestURL:     req.FullURL(),
		RequestName:    req.Name(),
		RequestBody:    req.Body(),
		RequestHeaders: req.Headers(),
		Timestamp:      time.Now(),
	}

	if resp != nil {
		entry.ResponseStatus = resp.Status().Code()
		entry.ResponseStatusText = resp.Status().Text()
		entry.ResponseBody = resp.Body().String()
		entry.ResponseTime = resp.Timing().Total.Milliseconds()
		entry.ResponseSize = resp.Body().Size()
		entry.ResponseHeaders = make(map[string]string)
		for _, key := range resp.Headers().Keys() {
			entry.ResponseHeaders[key] = resp.Headers().Get(key)
		}
	}

	if err != nil {
		entry.ResponseStatusText = "Error: " + err.Error()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, addErr := v.historyStore.Add(ctx, entry); addErr != nil {
		// Log error but don't crash - history is optional
		// Could add notification here if desired
	}
}

// Environment returns the current environment.
func (v *MainView) Environment() *core.Environment {
	return v.environment
}

// Interpolator returns the interpolation engine.
func (v *MainView) Interpolator() *interpolate.Engine {
	return v.interpolator
}

// ShowingHelp returns true if help is showing.
func (v *MainView) ShowingHelp() bool {
	return v.showHelp
}

// ShowHelp shows the help overlay.
func (v *MainView) ShowHelp() {
	v.showHelp = true
}

// HideHelp hides the help overlay.
func (v *MainView) HideHelp() {
	v.showHelp = false
}

// --- State accessors for E2E testing ---

// Notification returns the current notification message.
func (v *MainView) Notification() string {
	return v.notification
}

// sendRequest creates a tea.Cmd that sends an HTTP request asynchronously.
func sendRequest(reqDef *core.RequestDefinition, engine *interpolate.Engine) tea.Cmd {
	return func() tea.Msg {
		// Early validation of URL
		url := reqDef.FullURL()
		if url == "" {
			return components.RequestErrorMsg{Error: fmt.Errorf("URL is empty. Press 'e' to edit the URL")}
		}

		// Basic URL validation
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return components.RequestErrorMsg{Error: fmt.Errorf("URL must start with http:// or https://")}
		}

		// Convert RequestDefinition to Request (with or without interpolation)
		var req *core.Request
		var err error

		if engine != nil {
			req, err = reqDef.ToRequestWithEnv(engine)
		} else {
			req, err = reqDef.ToRequest()
		}

		if err != nil {
			return components.RequestErrorMsg{Error: err}
		}

		// Create HTTP client with timeout
		client := httpclient.NewClient(
			httpclient.WithTimeout(30 * time.Second),
		)

		// Send the request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Send(ctx, req)
		if err != nil {
			return components.RequestErrorMsg{Error: err}
		}

		return components.ResponseReceivedMsg{Response: resp}
	}
}
