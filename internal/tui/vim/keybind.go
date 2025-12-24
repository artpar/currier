package vim

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyBinding represents a single key binding.
type KeyBinding struct {
	key         string
	description string
	action      func() tea.Cmd
}

// NewKeyBinding creates a new key binding.
func NewKeyBinding(key, description string) *KeyBinding {
	return &KeyBinding{
		key:         key,
		description: description,
	}
}

// Key returns the key string.
func (kb *KeyBinding) Key() string {
	return kb.key
}

// Description returns the description.
func (kb *KeyBinding) Description() string {
	return kb.description
}

// Matches returns true if the key message matches this binding.
func (kb *KeyBinding) Matches(msg tea.KeyMsg) bool {
	return matchKey(kb.key, msg)
}

// Execute runs the action and returns the command.
func (kb *KeyBinding) Execute() tea.Cmd {
	if kb.action != nil {
		return kb.action()
	}
	return nil
}

// SetAction sets the action for this binding.
func (kb *KeyBinding) SetAction(action func() tea.Cmd) {
	kb.action = action
}

// matchKey checks if a key string matches a tea.KeyMsg.
func matchKey(key string, msg tea.KeyMsg) bool {
	key = strings.ToLower(key)

	switch key {
	case "enter":
		return msg.Type == tea.KeyEnter
	case "esc", "escape":
		return msg.Type == tea.KeyEsc
	case "space":
		return msg.Type == tea.KeySpace
	case "tab":
		return msg.Type == tea.KeyTab
	case "backspace":
		return msg.Type == tea.KeyBackspace
	case "up":
		return msg.Type == tea.KeyUp
	case "down":
		return msg.Type == tea.KeyDown
	case "left":
		return msg.Type == tea.KeyLeft
	case "right":
		return msg.Type == tea.KeyRight
	case "ctrl+c":
		return msg.Type == tea.KeyCtrlC
	case "ctrl+d":
		return msg.Type == tea.KeyCtrlD
	case "ctrl+u":
		return msg.Type == tea.KeyCtrlU
	case "ctrl+f":
		return msg.Type == tea.KeyCtrlF
	case "ctrl+b":
		return msg.Type == tea.KeyCtrlB
	case "ctrl+r":
		return msg.Type == tea.KeyCtrlR
	default:
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			return strings.ToLower(string(msg.Runes)) == key || string(msg.Runes) == key
		}
		return false
	}
}

// KeyMap holds key bindings organized by mode.
type KeyMap struct {
	bindings map[Mode][]*KeyBinding
}

// NewKeyMap creates a new empty key map.
func NewKeyMap() *KeyMap {
	return &KeyMap{
		bindings: make(map[Mode][]*KeyBinding),
	}
}

// Register adds a key binding for a mode.
func (km *KeyMap) Register(mode Mode, key, description string, action func() tea.Cmd) {
	kb := NewKeyBinding(key, description)
	kb.SetAction(action)
	km.bindings[mode] = append(km.bindings[mode], kb)
}

// GetBindings returns all bindings for a mode.
func (km *KeyMap) GetBindings(mode Mode) []*KeyBinding {
	return km.bindings[mode]
}

// FindBinding finds a matching binding for the given mode and key message.
func (km *KeyMap) FindBinding(mode Mode, msg tea.KeyMsg) (*KeyBinding, bool) {
	for _, kb := range km.bindings[mode] {
		if kb.Matches(msg) {
			return kb, true
		}
	}
	return nil, false
}

// DefaultKeyMap returns a key map with default vim-like bindings.
func DefaultKeyMap() *KeyMap {
	km := NewKeyMap()

	// Normal mode navigation
	km.Register(ModeNormal, "j", "move down", nil)
	km.Register(ModeNormal, "k", "move up", nil)
	km.Register(ModeNormal, "h", "move left", nil)
	km.Register(ModeNormal, "l", "move right", nil)
	km.Register(ModeNormal, "G", "go to bottom", nil)
	km.Register(ModeNormal, "g", "go prefix", nil)

	// Mode switching
	km.Register(ModeNormal, "i", "enter insert mode", nil)
	km.Register(ModeNormal, ":", "enter command mode", nil)
	km.Register(ModeNormal, "v", "enter visual mode", nil)

	// Exit insert/command mode
	km.Register(ModeInsert, "esc", "exit insert mode", nil)
	km.Register(ModeCommand, "esc", "exit command mode", nil)
	km.Register(ModeVisual, "esc", "exit visual mode", nil)

	return km
}

// SequenceStatus represents the state of a key sequence.
type SequenceStatus int

const (
	SequenceNone SequenceStatus = iota
	SequencePending
	SequenceComplete
	SequenceInvalid
)

// SequenceResult holds the result of handling a key in a sequence.
type SequenceResult struct {
	Status  SequenceStatus
	action  func() tea.Cmd
}

// Execute runs the action if the sequence is complete.
func (r *SequenceResult) Execute() tea.Cmd {
	if r.action != nil {
		return r.action()
	}
	return nil
}

// KeySequenceHandler handles multi-key sequences like "dd", "gg", etc.
type KeySequenceHandler struct {
	sequences map[string]func() tea.Cmd
	buffer    string
}

// NewKeySequenceHandler creates a new sequence handler.
func NewKeySequenceHandler() *KeySequenceHandler {
	return &KeySequenceHandler{
		sequences: make(map[string]func() tea.Cmd),
	}
}

// Register adds a sequence handler.
func (h *KeySequenceHandler) Register(sequence string, action func() tea.Cmd) {
	h.sequences[sequence] = action
}

// Handle processes a key and returns the sequence status.
func (h *KeySequenceHandler) Handle(key string) *SequenceResult {
	h.buffer += key

	// Check for exact match
	if action, ok := h.sequences[h.buffer]; ok {
		// Check if there's a longer sequence that starts with this
		hasLonger := false
		for seq := range h.sequences {
			if len(seq) > len(h.buffer) && strings.HasPrefix(seq, h.buffer) {
				hasLonger = true
				break
			}
		}

		if !hasLonger {
			// No longer sequence, this is complete
			h.buffer = ""
			return &SequenceResult{Status: SequenceComplete, action: action}
		}
		// There's a longer sequence, keep pending
		return &SequenceResult{Status: SequencePending}
	}

	// Check if this could be a prefix of any sequence
	for seq := range h.sequences {
		if strings.HasPrefix(seq, h.buffer) {
			return &SequenceResult{Status: SequencePending}
		}
	}

	// Invalid sequence
	h.buffer = ""
	return &SequenceResult{Status: SequenceInvalid}
}

// Reset clears the sequence buffer.
func (h *KeySequenceHandler) Reset() {
	h.buffer = ""
}

// Buffer returns the current sequence buffer.
func (h *KeySequenceHandler) Buffer() string {
	return h.buffer
}
