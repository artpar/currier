package vim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMode(t *testing.T) {
	t.Run("mode string representation", func(t *testing.T) {
		assert.Equal(t, "NORMAL", ModeNormal.String())
		assert.Equal(t, "INSERT", ModeInsert.String())
		assert.Equal(t, "COMMAND", ModeCommand.String())
		assert.Equal(t, "VISUAL", ModeVisual.String())
	})
}

func TestNewModeManager(t *testing.T) {
	t.Run("starts in normal mode", func(t *testing.T) {
		m := NewModeManager()
		assert.Equal(t, ModeNormal, m.Current())
	})

	t.Run("command buffer is empty initially", func(t *testing.T) {
		m := NewModeManager()
		assert.Empty(t, m.CommandBuffer())
	})
}

func TestModeManager_ModeTransitions(t *testing.T) {
	t.Run("enters insert mode with i", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeInsert)
		assert.Equal(t, ModeInsert, m.Current())
	})

	t.Run("returns to normal mode with Escape", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeInsert)
		m.SetMode(ModeNormal)
		assert.Equal(t, ModeNormal, m.Current())
	})

	t.Run("enters command mode with colon", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		assert.Equal(t, ModeCommand, m.Current())
	})

	t.Run("enters visual mode with v", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeVisual)
		assert.Equal(t, ModeVisual, m.Current())
	})

	t.Run("tracks previous mode", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeInsert)
		m.SetMode(ModeNormal)
		assert.Equal(t, ModeInsert, m.Previous())
	})
}

func TestModeManager_CommandBuffer(t *testing.T) {
	t.Run("appends to command buffer", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		m.AppendToCommandBuffer("w")
		m.AppendToCommandBuffer("q")
		assert.Equal(t, "wq", m.CommandBuffer())
	})

	t.Run("clears command buffer", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		m.AppendToCommandBuffer("test")
		m.ClearCommandBuffer()
		assert.Empty(t, m.CommandBuffer())
	})

	t.Run("backspace removes last character", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		m.AppendToCommandBuffer("test")
		m.BackspaceCommandBuffer()
		assert.Equal(t, "tes", m.CommandBuffer())
	})

	t.Run("backspace on empty buffer does nothing", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		m.BackspaceCommandBuffer()
		assert.Empty(t, m.CommandBuffer())
	})

	t.Run("clears command buffer when leaving command mode", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeCommand)
		m.AppendToCommandBuffer("test")
		m.SetMode(ModeNormal)
		assert.Empty(t, m.CommandBuffer())
	})
}

func TestModeManager_KeySequence(t *testing.T) {
	t.Run("tracks pending key sequence", func(t *testing.T) {
		m := NewModeManager()
		m.AppendKeySequence("d")
		assert.Equal(t, "d", m.KeySequence())
	})

	t.Run("appends to key sequence", func(t *testing.T) {
		m := NewModeManager()
		m.AppendKeySequence("d")
		m.AppendKeySequence("d")
		assert.Equal(t, "dd", m.KeySequence())
	})

	t.Run("clears key sequence", func(t *testing.T) {
		m := NewModeManager()
		m.AppendKeySequence("dd")
		m.ClearKeySequence()
		assert.Empty(t, m.KeySequence())
	})

	t.Run("has pending key sequence", func(t *testing.T) {
		m := NewModeManager()
		assert.False(t, m.HasPendingSequence())
		m.AppendKeySequence("d")
		assert.True(t, m.HasPendingSequence())
	})
}

func TestModeManager_Count(t *testing.T) {
	t.Run("default count is 1", func(t *testing.T) {
		m := NewModeManager()
		assert.Equal(t, 1, m.Count())
	})

	t.Run("accumulates count digits", func(t *testing.T) {
		m := NewModeManager()
		m.AppendCount(5)
		assert.Equal(t, 5, m.Count())
		m.AppendCount(3)
		assert.Equal(t, 53, m.Count())
	})

	t.Run("resets count", func(t *testing.T) {
		m := NewModeManager()
		m.AppendCount(5)
		m.ResetCount()
		assert.Equal(t, 1, m.Count())
	})

	t.Run("has count returns true when count set", func(t *testing.T) {
		m := NewModeManager()
		assert.False(t, m.HasCount())
		m.AppendCount(5)
		assert.True(t, m.HasCount())
	})
}

func TestModeManager_IsMode(t *testing.T) {
	t.Run("IsNormal", func(t *testing.T) {
		m := NewModeManager()
		assert.True(t, m.IsNormal())
		m.SetMode(ModeInsert)
		assert.False(t, m.IsNormal())
	})

	t.Run("IsInsert", func(t *testing.T) {
		m := NewModeManager()
		assert.False(t, m.IsInsert())
		m.SetMode(ModeInsert)
		assert.True(t, m.IsInsert())
	})

	t.Run("IsCommand", func(t *testing.T) {
		m := NewModeManager()
		assert.False(t, m.IsCommand())
		m.SetMode(ModeCommand)
		assert.True(t, m.IsCommand())
	})

	t.Run("IsVisual", func(t *testing.T) {
		m := NewModeManager()
		assert.False(t, m.IsVisual())
		m.SetMode(ModeVisual)
		assert.True(t, m.IsVisual())
	})
}

func TestModeManager_Reset(t *testing.T) {
	t.Run("reset clears all state", func(t *testing.T) {
		m := NewModeManager()
		m.SetMode(ModeInsert)
		m.AppendKeySequence("dd")
		m.AppendCount(5)

		m.Reset()

		assert.Equal(t, ModeNormal, m.Current())
		assert.Empty(t, m.KeySequence())
		assert.Equal(t, 1, m.Count())
	})
}
