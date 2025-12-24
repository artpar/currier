package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/core"
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
}

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

	case components.SendRequestMsg:
		v.response.SetLoading(true)
		v.focusPane(PaneResponse)
		return v, sendRequest(msg.Request, v.interpolator)

	case components.ResponseReceivedMsg:
		v.response.SetLoading(false)
		v.response.SetResponse(msg.Response)
		return v, nil

	case components.RequestErrorMsg:
		v.response.SetLoading(false)
		v.response.SetError(msg.Error)
		return v, nil
	}

	// Forward messages to focused pane
	return v.forwardToFocusedPane(msg)
}

func (v *MainView) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return v, tea.Quit

	case tea.KeyTab:
		v.cycleFocusForward()
		return v, nil

	case tea.KeyShiftTab:
		v.cycleFocusBackward()
		return v, nil

	case tea.KeyEsc:
		// Could be used for mode switching
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
		}
	}

	// Forward to focused pane
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

	// Calculate pane widths (25% / 37% / 38%)
	leftWidth := v.width * 25 / 100
	middleWidth := v.width * 37 / 100
	rightWidth := v.width - leftWidth - middleWidth

	// Reserve 1 line for status bar
	paneHeight := v.height - 1
	if paneHeight < 1 {
		paneHeight = 1
	}

	// Set sizes
	v.tree.SetSize(leftWidth, paneHeight)
	v.request.SetSize(middleWidth, paneHeight)
	v.response.SetSize(rightWidth, paneHeight)
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

	// Render three panes side by side
	leftPane := v.tree.View()
	middlePane := v.request.View()
	rightPane := v.response.View()

	// Join panes horizontally
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, middlePane, rightPane)

	// Render status bar
	statusBar := v.renderStatusBar()

	// Join panes and status bar vertically
	return lipgloss.JoinVertical(lipgloss.Left, panes, statusBar)
}

// renderStatusBar renders the bottom status bar with environment info.
func (v *MainView) renderStatusBar() string {
	// Build status items
	var items []string

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
		"│    1 / 2 / 3          Jump to pane                 │",
		"│    j / k              Move down/up                 │",
		"│    h / l              Collapse/Expand              │",
		"│    gg / G             Go to top/bottom             │",
		"│                                                     │",
		"│  Search (Collections pane)                         │",
		"│    /                  Start search                 │",
		"│    Esc                Clear search filter          │",
		"│                                                     │",
		"│  Actions                                            │",
		"│    Enter              Select / Send request        │",
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

// SetCollections sets the collections to display.
func (v *MainView) SetCollections(collections []*core.Collection) {
	v.tree.SetCollections(collections)
}

// SetEnvironment sets the environment and interpolation engine.
func (v *MainView) SetEnvironment(env *core.Environment, engine *interpolate.Engine) {
	v.environment = env
	v.interpolator = engine
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

// sendRequest creates a tea.Cmd that sends an HTTP request asynchronously.
func sendRequest(reqDef *core.RequestDefinition, engine *interpolate.Engine) tea.Cmd {
	return func() tea.Msg {
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
