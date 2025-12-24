package interfaces

import (
	tea "github.com/charmbracelet/bubbletea"
)

// View represents a TUI view/screen.
type View interface {
	// Init initializes the view.
	Init() tea.Cmd

	// Update handles messages and returns the updated view.
	Update(msg tea.Msg) (View, tea.Cmd)

	// View renders the view to a string.
	View() string

	// Name returns the view name.
	Name() string

	// Focused returns true if the view is focused.
	Focused() bool

	// Focus gives focus to the view.
	Focus() View

	// Blur removes focus from the view.
	Blur() View
}

// Component represents a reusable TUI component.
type Component interface {
	// Init initializes the component.
	Init() tea.Cmd

	// Update handles messages and returns the updated component.
	Update(msg tea.Msg) (Component, tea.Cmd)

	// View renders the component to a string.
	View() string

	// Focus gives focus to the component.
	Focus() Component

	// Blur removes focus from the component.
	Blur() Component

	// Focused returns true if the component is focused.
	Focused() bool

	// SetSize sets the component dimensions.
	SetSize(width, height int) Component

	// Width returns the component width.
	Width() int

	// Height returns the component height.
	Height() int
}

// KeyHandler processes keyboard input.
type KeyHandler interface {
	// HandleKey processes a key press.
	HandleKey(key tea.KeyMsg) (tea.Cmd, bool)

	// SetMode sets the current vim mode.
	SetMode(mode VimMode)

	// GetMode returns the current vim mode.
	GetMode() VimMode
}

// VimMode represents vim editing modes.
type VimMode int

const (
	VimModeNormal VimMode = iota
	VimModeInsert
	VimModeVisual
	VimModeCommand
	VimModeSearch
)

func (m VimMode) String() string {
	switch m {
	case VimModeNormal:
		return "NORMAL"
	case VimModeInsert:
		return "INSERT"
	case VimModeVisual:
		return "VISUAL"
	case VimModeCommand:
		return "COMMAND"
	case VimModeSearch:
		return "SEARCH"
	default:
		return "UNKNOWN"
	}
}

// CommandExecutor executes : commands.
type CommandExecutor interface {
	// Execute runs a command.
	Execute(command string) tea.Cmd

	// Complete returns completions for a partial command.
	Complete(partial string) []string

	// Register registers a custom command.
	Register(name string, handler CommandHandler)
}

// CommandHandler handles a custom command.
type CommandHandler func(args []string) tea.Cmd

// Theme defines UI colors and styles.
type Theme interface {
	// Name returns the theme name.
	Name() string

	// Background returns the background color.
	Background() string

	// Foreground returns the foreground color.
	Foreground() string

	// Primary returns the primary accent color.
	Primary() string

	// Secondary returns the secondary accent color.
	Secondary() string

	// Success returns the success color.
	Success() string

	// Warning returns the warning color.
	Warning() string

	// Error returns the error color.
	Error() string

	// Border returns the border color.
	Border() string

	// Muted returns the muted text color.
	Muted() string

	// Highlight returns the highlight color.
	Highlight() string
}

// Layout manages component arrangement.
type Layout interface {
	// AddComponent adds a component to the layout.
	AddComponent(name string, c Component, region LayoutRegion)

	// RemoveComponent removes a component.
	RemoveComponent(name string)

	// GetComponent retrieves a component by name.
	GetComponent(name string) (Component, bool)

	// FocusNext moves focus to the next component.
	FocusNext() Component

	// FocusPrev moves focus to the previous component.
	FocusPrev() Component

	// SetSize sets the layout dimensions.
	SetSize(width, height int)

	// View renders the layout.
	View() string
}

// LayoutRegion identifies a layout region.
type LayoutRegion int

const (
	LayoutRegionLeft LayoutRegion = iota
	LayoutRegionCenter
	LayoutRegionRight
	LayoutRegionTop
	LayoutRegionBottom
	LayoutRegionOverlay
)
