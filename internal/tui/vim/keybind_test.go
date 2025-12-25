package vim

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestKeyBinding(t *testing.T) {
	t.Run("creates simple key binding", func(t *testing.T) {
		kb := NewKeyBinding("j", "move down")
		assert.Equal(t, "j", kb.Key())
		assert.Equal(t, "move down", kb.Description())
	})

	t.Run("matches simple key", func(t *testing.T) {
		kb := NewKeyBinding("j", "move down")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}))
		assert.False(t, kb.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}))
	})

	t.Run("matches special keys", func(t *testing.T) {
		kb := NewKeyBinding("enter", "confirm")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyEnter}))
		assert.False(t, kb.Matches(tea.KeyMsg{Type: tea.KeyEsc}))
	})

	t.Run("matches escape key", func(t *testing.T) {
		kb := NewKeyBinding("esc", "cancel")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyEsc}))
	})

	t.Run("matches ctrl combinations", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+c", "quit")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlC}))
	})

	t.Run("matches space", func(t *testing.T) {
		kb := NewKeyBinding("space", "select")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeySpace}))
	})

	t.Run("matches tab", func(t *testing.T) {
		kb := NewKeyBinding("tab", "next")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyTab}))
	})

	t.Run("matches backspace", func(t *testing.T) {
		kb := NewKeyBinding("backspace", "delete")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyBackspace}))
	})
}

func TestKeyMap(t *testing.T) {
	t.Run("creates empty keymap", func(t *testing.T) {
		km := NewKeyMap()
		assert.NotNil(t, km)
	})

	t.Run("registers binding for mode", func(t *testing.T) {
		km := NewKeyMap()
		km.Register(ModeNormal, "j", "move down", func() tea.Cmd { return nil })

		bindings := km.GetBindings(ModeNormal)
		assert.Len(t, bindings, 1)
		assert.Equal(t, "j", bindings[0].Key())
	})

	t.Run("registers multiple bindings", func(t *testing.T) {
		km := NewKeyMap()
		km.Register(ModeNormal, "j", "move down", func() tea.Cmd { return nil })
		km.Register(ModeNormal, "k", "move up", func() tea.Cmd { return nil })
		km.Register(ModeNormal, "h", "move left", func() tea.Cmd { return nil })
		km.Register(ModeNormal, "l", "move right", func() tea.Cmd { return nil })

		bindings := km.GetBindings(ModeNormal)
		assert.Len(t, bindings, 4)
	})

	t.Run("separates bindings by mode", func(t *testing.T) {
		km := NewKeyMap()
		km.Register(ModeNormal, "j", "move down", func() tea.Cmd { return nil })
		km.Register(ModeInsert, "j", "insert j", func() tea.Cmd { return nil })

		normalBindings := km.GetBindings(ModeNormal)
		insertBindings := km.GetBindings(ModeInsert)

		assert.Len(t, normalBindings, 1)
		assert.Len(t, insertBindings, 1)
		assert.Equal(t, "move down", normalBindings[0].Description())
		assert.Equal(t, "insert j", insertBindings[0].Description())
	})

	t.Run("finds matching binding", func(t *testing.T) {
		km := NewKeyMap()
		called := false
		km.Register(ModeNormal, "j", "move down", func() tea.Cmd {
			called = true
			return nil
		})

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		binding, found := km.FindBinding(ModeNormal, msg)

		assert.True(t, found)
		assert.NotNil(t, binding)
		binding.Execute()
		assert.True(t, called)
	})

	t.Run("returns false for no match", func(t *testing.T) {
		km := NewKeyMap()
		km.Register(ModeNormal, "j", "move down", func() tea.Cmd { return nil })

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		_, found := km.FindBinding(ModeNormal, msg)

		assert.False(t, found)
	})
}

func TestDefaultKeyMap(t *testing.T) {
	t.Run("has navigation keys", func(t *testing.T) {
		km := DefaultKeyMap()

		// Check hjkl navigation exists
		keys := []string{"j", "k", "h", "l"}
		for _, key := range keys {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			_, found := km.FindBinding(ModeNormal, msg)
			assert.True(t, found, "expected binding for key: %s", key)
		}
	})

	t.Run("has mode switching keys", func(t *testing.T) {
		km := DefaultKeyMap()

		// i for insert mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		_, found := km.FindBinding(ModeNormal, msg)
		assert.True(t, found)

		// : for command mode
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
		_, found = km.FindBinding(ModeNormal, msg)
		assert.True(t, found)
	})

	t.Run("has gg and G for top/bottom", func(t *testing.T) {
		km := DefaultKeyMap()

		// G for bottom
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		_, found := km.FindBinding(ModeNormal, msg)
		assert.True(t, found)

		// g is a prefix key (gg)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		_, found = km.FindBinding(ModeNormal, msg)
		assert.True(t, found)
	})

	t.Run("escape returns to normal mode", func(t *testing.T) {
		km := DefaultKeyMap()

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, found := km.FindBinding(ModeInsert, msg)
		assert.True(t, found)

		_, found = km.FindBinding(ModeCommand, msg)
		assert.True(t, found)
	})
}

func TestKeySequenceHandler(t *testing.T) {
	t.Run("handles single key commands", func(t *testing.T) {
		h := NewKeySequenceHandler()
		called := false
		h.Register("j", func() tea.Cmd {
			called = true
			return nil
		})

		result := h.Handle("j")
		assert.Equal(t, SequenceComplete, result.Status)
		result.Execute()
		assert.True(t, called)
	})

	t.Run("handles multi-key sequences", func(t *testing.T) {
		h := NewKeySequenceHandler()
		called := false
		h.Register("dd", func() tea.Cmd {
			called = true
			return nil
		})

		result := h.Handle("d")
		assert.Equal(t, SequencePending, result.Status)

		result = h.Handle("d")
		assert.Equal(t, SequenceComplete, result.Status)
		result.Execute()
		assert.True(t, called)
	})

	t.Run("handles gg sequence", func(t *testing.T) {
		h := NewKeySequenceHandler()
		called := false
		h.Register("gg", func() tea.Cmd {
			called = true
			return nil
		})

		result := h.Handle("g")
		assert.Equal(t, SequencePending, result.Status)

		result = h.Handle("g")
		assert.Equal(t, SequenceComplete, result.Status)
		result.Execute()
		assert.True(t, called)
	})

	t.Run("resets on invalid sequence", func(t *testing.T) {
		h := NewKeySequenceHandler()
		h.Register("dd", func() tea.Cmd { return nil })

		result := h.Handle("d")
		assert.Equal(t, SequencePending, result.Status)

		result = h.Handle("x") // Invalid, should reset
		assert.Equal(t, SequenceInvalid, result.Status)
	})

	t.Run("handles mixed single and multi-key", func(t *testing.T) {
		h := NewKeySequenceHandler()
		singleCalled := false
		multiCalled := false

		h.Register("g", func() tea.Cmd {
			singleCalled = true
			return nil
		})
		h.Register("gg", func() tea.Cmd {
			multiCalled = true
			return nil
		})

		// First g should be pending (could be gg)
		result := h.Handle("g")
		assert.Equal(t, SequencePending, result.Status)

		// Second g completes gg
		result = h.Handle("g")
		assert.Equal(t, SequenceComplete, result.Status)
		result.Execute()
		assert.True(t, multiCalled)
		assert.False(t, singleCalled)
	})

	t.Run("Reset clears buffer", func(t *testing.T) {
		h := NewKeySequenceHandler()
		h.Register("dd", func() tea.Cmd { return nil })

		// Start a sequence
		result := h.Handle("d")
		assert.Equal(t, SequencePending, result.Status)

		// Reset
		h.Reset()

		// Buffer should be empty, so 'd' starts fresh
		result = h.Handle("d")
		assert.Equal(t, SequencePending, result.Status)
	})

	t.Run("Buffer returns current buffer", func(t *testing.T) {
		h := NewKeySequenceHandler()
		h.Register("dd", func() tea.Cmd { return nil })

		assert.Equal(t, "", h.Buffer())

		h.Handle("d")
		assert.Equal(t, "d", h.Buffer())

		h.Handle("d")
		assert.Equal(t, "", h.Buffer()) // Completed, should be empty
	})
}

func TestKeyBinding_MatchKey(t *testing.T) {
	t.Run("matches up arrow", func(t *testing.T) {
		kb := NewKeyBinding("up", "up")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyUp}))
	})

	t.Run("matches down arrow", func(t *testing.T) {
		kb := NewKeyBinding("down", "down")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyDown}))
	})

	t.Run("matches left arrow", func(t *testing.T) {
		kb := NewKeyBinding("left", "left")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyLeft}))
	})

	t.Run("matches right arrow", func(t *testing.T) {
		kb := NewKeyBinding("right", "right")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyRight}))
	})

	t.Run("matches ctrl+d", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+d", "page down")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlD}))
	})

	t.Run("matches ctrl+u", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+u", "page up")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlU}))
	})

	t.Run("matches ctrl+f", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+f", "forward")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlF}))
	})

	t.Run("matches ctrl+b", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+b", "backward")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlB}))
	})

	t.Run("matches ctrl+r", func(t *testing.T) {
		kb := NewKeyBinding("ctrl+r", "redo")
		assert.True(t, kb.Matches(tea.KeyMsg{Type: tea.KeyCtrlR}))
	})
}
