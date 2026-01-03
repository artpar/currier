package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Component is the interface for all TUI components.
type Component interface {
	// Init initializes the component.
	Init() tea.Cmd

	// Update handles messages and returns the updated component.
	Update(msg tea.Msg) (Component, tea.Cmd)

	// View renders the component.
	View() string

	// Title returns the component title.
	Title() string

	// Focused returns true if the component is focused.
	Focused() bool

	// Focus sets the component as focused.
	Focus()

	// Blur removes focus from the component.
	Blur()

	// SetSize sets the component dimensions.
	SetSize(width, height int)

	// Width returns the component width.
	Width() int

	// Height returns the component height.
	Height() int
}

// NavDirection represents a navigation direction.
type NavDirection int

const (
	NavUp NavDirection = iota
	NavDown
	NavLeft
	NavRight
)

// Panel padding constants for comfortable spacing
const (
	PanelPaddingV = 0 // Vertical padding (lines) - kept minimal to preserve content
	PanelPaddingH = 1 // Horizontal padding (chars)
	ContentPadH   = 1 // Content area horizontal padding
)

// Messages

// FocusMsg is sent when a component should gain focus.
type FocusMsg struct{}

// BlurMsg is sent when a component should lose focus.
type BlurMsg struct{}

// NavigateMsg is sent for navigation within a component.
type NavigateMsg struct {
	Direction NavDirection
	Count     int
}

// SelectMsg is sent when an item is selected.
type SelectMsg struct {
	ID   string
	Item interface{}
}

// RefreshMsg is sent to refresh component data.
type RefreshMsg struct{}

// BaseComponent provides common functionality for components.
type BaseComponent struct {
	title   string
	focused bool
	width   int
	height  int
}

// NewBaseComponent creates a new base component.
func NewBaseComponent(title string) *BaseComponent {
	return &BaseComponent{
		title: title,
	}
}

// Init initializes the component.
func (c *BaseComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (c *BaseComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
	case FocusMsg:
		c.focused = true
	case BlurMsg:
		c.focused = false
	}
	return c, nil
}

// View renders the component.
func (c *BaseComponent) View() string {
	style := lipgloss.NewStyle().
		Width(c.width).
		Height(c.height).
		Align(lipgloss.Center, lipgloss.Center)

	if c.focused {
		style = style.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	} else {
		style = style.BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	}

	content := fmt.Sprintf("[ %s ]", c.title)
	return style.Render(content)
}

// Title returns the component title.
func (c *BaseComponent) Title() string {
	return c.title
}

// Focused returns true if focused.
func (c *BaseComponent) Focused() bool {
	return c.focused
}

// Focus sets the component as focused.
func (c *BaseComponent) Focus() {
	c.focused = true
}

// Blur removes focus.
func (c *BaseComponent) Blur() {
	c.focused = false
}

// SetSize sets dimensions.
func (c *BaseComponent) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Width returns the width.
func (c *BaseComponent) Width() int {
	return c.width
}

// Height returns the height.
func (c *BaseComponent) Height() int {
	return c.height
}

// ComponentList manages a list of components with focus cycling.
type ComponentList struct {
	components []Component
	focusIndex int
}

// NewComponentList creates a new component list.
func NewComponentList() *ComponentList {
	return &ComponentList{
		components: make([]Component, 0),
		focusIndex: -1,
	}
}

// Add adds a component to the list.
func (cl *ComponentList) Add(c Component) {
	cl.components = append(cl.components, c)
}

// Len returns the number of components.
func (cl *ComponentList) Len() int {
	return len(cl.components)
}

// Get returns a component by index.
func (cl *ComponentList) Get(index int) Component {
	if index < 0 || index >= len(cl.components) {
		return nil
	}
	return cl.components[index]
}

// FocusFirst focuses the first component.
func (cl *ComponentList) FocusFirst() {
	if len(cl.components) == 0 {
		return
	}
	cl.setFocus(0)
}

// FocusNext cycles focus to the next component.
func (cl *ComponentList) FocusNext() {
	if len(cl.components) == 0 {
		return
	}
	next := (cl.focusIndex + 1) % len(cl.components)
	cl.setFocus(next)
}

// FocusPrev cycles focus to the previous component.
func (cl *ComponentList) FocusPrev() {
	if len(cl.components) == 0 {
		return
	}
	prev := cl.focusIndex - 1
	if prev < 0 {
		prev = len(cl.components) - 1
	}
	cl.setFocus(prev)
}

// FocusIndex returns the current focus index.
func (cl *ComponentList) FocusIndex() int {
	return cl.focusIndex
}

// SetFocusIndex sets focus to a specific index.
func (cl *ComponentList) SetFocusIndex(index int) {
	if index < 0 || index >= len(cl.components) {
		return
	}
	cl.setFocus(index)
}

// Focused returns the currently focused component.
func (cl *ComponentList) Focused() Component {
	if cl.focusIndex < 0 || cl.focusIndex >= len(cl.components) {
		return nil
	}
	return cl.components[cl.focusIndex]
}

func (cl *ComponentList) setFocus(index int) {
	// Blur current
	if cl.focusIndex >= 0 && cl.focusIndex < len(cl.components) {
		cl.components[cl.focusIndex].Blur()
	}
	// Focus new
	cl.focusIndex = index
	if index >= 0 && index < len(cl.components) {
		cl.components[index].Focus()
	}
}

// Styles

// DefaultStyles returns the default component styles.
type Styles struct {
	Focused   lipgloss.Style
	Unfocused lipgloss.Style
	Title     lipgloss.Style
	Border    lipgloss.Style
}

// DefaultStyles returns default styling.
func DefaultStyles() Styles {
	return Styles{
		Focused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")),
		Unfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")),
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()),
	}
}

// RenderTitle renders a title bar.
func RenderTitle(title string, width int, focused bool) string {
	style := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Bold(true)

	if focused {
		style = style.Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62"))
	} else {
		style = style.Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238"))
	}

	return style.Render(title)
}

// RenderBorder renders content with a border.
func RenderBorder(content string, width, height int, focused bool) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder())

	if focused {
		style = style.BorderForeground(lipgloss.Color("62"))
	} else {
		style = style.BorderForeground(lipgloss.Color("240"))
	}

	return style.Render(content)
}

// Truncate truncates a string to fit within a width.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// PadRight pads a string to a given width.
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
