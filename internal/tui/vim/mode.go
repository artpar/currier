package vim

// Mode represents the current vim editing mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeCommand
	ModeVisual
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	case ModeVisual:
		return "VISUAL"
	default:
		return "UNKNOWN"
	}
}

// ModeManager handles vim mode state and transitions.
type ModeManager struct {
	current       Mode
	previous      Mode
	commandBuffer string
	keySequence   string
	count         int
	hasCount      bool
}

// NewModeManager creates a new mode manager starting in normal mode.
func NewModeManager() *ModeManager {
	return &ModeManager{
		current:  ModeNormal,
		previous: ModeNormal,
		count:    1,
	}
}

// Current returns the current mode.
func (m *ModeManager) Current() Mode {
	return m.current
}

// Previous returns the previous mode.
func (m *ModeManager) Previous() Mode {
	return m.previous
}

// SetMode changes the current mode.
func (m *ModeManager) SetMode(mode Mode) {
	if m.current == ModeCommand && mode != ModeCommand {
		m.ClearCommandBuffer()
	}
	m.previous = m.current
	m.current = mode
}

// IsNormal returns true if in normal mode.
func (m *ModeManager) IsNormal() bool {
	return m.current == ModeNormal
}

// IsInsert returns true if in insert mode.
func (m *ModeManager) IsInsert() bool {
	return m.current == ModeInsert
}

// IsCommand returns true if in command mode.
func (m *ModeManager) IsCommand() bool {
	return m.current == ModeCommand
}

// IsVisual returns true if in visual mode.
func (m *ModeManager) IsVisual() bool {
	return m.current == ModeVisual
}

// CommandBuffer returns the current command buffer content.
func (m *ModeManager) CommandBuffer() string {
	return m.commandBuffer
}

// AppendToCommandBuffer adds a character to the command buffer.
func (m *ModeManager) AppendToCommandBuffer(s string) {
	m.commandBuffer += s
}

// ClearCommandBuffer clears the command buffer.
func (m *ModeManager) ClearCommandBuffer() {
	m.commandBuffer = ""
}

// BackspaceCommandBuffer removes the last character from the command buffer.
func (m *ModeManager) BackspaceCommandBuffer() {
	if len(m.commandBuffer) > 0 {
		m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
	}
}

// KeySequence returns the pending key sequence (e.g., "dd", "yy").
func (m *ModeManager) KeySequence() string {
	return m.keySequence
}

// AppendKeySequence adds a key to the pending sequence.
func (m *ModeManager) AppendKeySequence(key string) {
	m.keySequence += key
}

// ClearKeySequence clears the pending key sequence.
func (m *ModeManager) ClearKeySequence() {
	m.keySequence = ""
}

// HasPendingSequence returns true if there's a pending key sequence.
func (m *ModeManager) HasPendingSequence() bool {
	return len(m.keySequence) > 0
}

// Count returns the current count (default 1).
func (m *ModeManager) Count() int {
	return m.count
}

// AppendCount adds a digit to the count.
func (m *ModeManager) AppendCount(digit int) {
	if !m.hasCount {
		m.count = digit
		m.hasCount = true
	} else {
		m.count = m.count*10 + digit
	}
}

// ResetCount resets the count to default (1).
func (m *ModeManager) ResetCount() {
	m.count = 1
	m.hasCount = false
}

// HasCount returns true if a count was explicitly set.
func (m *ModeManager) HasCount() bool {
	return m.hasCount
}

// Reset resets all mode state to defaults.
func (m *ModeManager) Reset() {
	m.current = ModeNormal
	m.previous = ModeNormal
	m.commandBuffer = ""
	m.keySequence = ""
	m.count = 1
	m.hasCount = false
}
