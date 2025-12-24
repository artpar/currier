package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestBaseComponent(t *testing.T) {
	t.Run("creates with title", func(t *testing.T) {
		c := NewBaseComponent("Test Component")
		assert.Equal(t, "Test Component", c.Title())
	})

	t.Run("starts unfocused", func(t *testing.T) {
		c := NewBaseComponent("Test")
		assert.False(t, c.Focused())
	})

	t.Run("can be focused", func(t *testing.T) {
		c := NewBaseComponent("Test")
		c.Focus()
		assert.True(t, c.Focused())
	})

	t.Run("can be blurred", func(t *testing.T) {
		c := NewBaseComponent("Test")
		c.Focus()
		c.Blur()
		assert.False(t, c.Focused())
	})

	t.Run("tracks dimensions", func(t *testing.T) {
		c := NewBaseComponent("Test")
		c.SetSize(80, 24)
		assert.Equal(t, 80, c.Width())
		assert.Equal(t, 24, c.Height())
	})

	t.Run("has default dimensions", func(t *testing.T) {
		c := NewBaseComponent("Test")
		assert.Equal(t, 0, c.Width())
		assert.Equal(t, 0, c.Height())
	})
}

func TestBaseComponent_Update(t *testing.T) {
	t.Run("handles window size message", func(t *testing.T) {
		c := NewBaseComponent("Test")
		msg := tea.WindowSizeMsg{Width: 120, Height: 40}

		updated, _ := c.Update(msg)
		base := updated.(*BaseComponent)

		assert.Equal(t, 120, base.Width())
		assert.Equal(t, 40, base.Height())
	})

	t.Run("handles focus message", func(t *testing.T) {
		c := NewBaseComponent("Test")
		msg := FocusMsg{}

		updated, _ := c.Update(msg)
		base := updated.(*BaseComponent)

		assert.True(t, base.Focused())
	})

	t.Run("handles blur message", func(t *testing.T) {
		c := NewBaseComponent("Test")
		c.Focus()
		msg := BlurMsg{}

		updated, _ := c.Update(msg)
		base := updated.(*BaseComponent)

		assert.False(t, base.Focused())
	})
}

func TestBaseComponent_View(t *testing.T) {
	t.Run("renders placeholder when empty", func(t *testing.T) {
		c := NewBaseComponent("Test Panel")
		c.SetSize(40, 10)

		view := c.View()
		assert.Contains(t, view, "Test Panel")
	})
}

func TestComponentList(t *testing.T) {
	t.Run("creates empty list", func(t *testing.T) {
		cl := NewComponentList()
		assert.Equal(t, 0, cl.Len())
	})

	t.Run("adds components", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))

		assert.Equal(t, 2, cl.Len())
	})

	t.Run("gets component by index", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))

		c := cl.Get(1)
		assert.Equal(t, "Second", c.Title())
	})

	t.Run("focuses first component initially", func(t *testing.T) {
		cl := NewComponentList()
		c1 := NewBaseComponent("First")
		c2 := NewBaseComponent("Second")

		cl.Add(c1)
		cl.Add(c2)
		cl.FocusFirst()

		assert.True(t, cl.Get(0).Focused())
		assert.False(t, cl.Get(1).Focused())
	})

	t.Run("cycles focus forward", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))
		cl.Add(NewBaseComponent("Third"))
		cl.FocusFirst()

		cl.FocusNext()
		assert.Equal(t, 1, cl.FocusIndex())
		assert.True(t, cl.Get(1).Focused())
		assert.False(t, cl.Get(0).Focused())

		cl.FocusNext()
		assert.Equal(t, 2, cl.FocusIndex())

		// Wraps around
		cl.FocusNext()
		assert.Equal(t, 0, cl.FocusIndex())
	})

	t.Run("cycles focus backward", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))
		cl.Add(NewBaseComponent("Third"))
		cl.FocusFirst()

		// Wraps around from first to last
		cl.FocusPrev()
		assert.Equal(t, 2, cl.FocusIndex())
	})

	t.Run("focuses by index", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))
		cl.Add(NewBaseComponent("Third"))

		cl.SetFocusIndex(1)
		assert.Equal(t, 1, cl.FocusIndex())
		assert.True(t, cl.Get(1).Focused())
	})

	t.Run("returns focused component", func(t *testing.T) {
		cl := NewComponentList()
		cl.Add(NewBaseComponent("First"))
		cl.Add(NewBaseComponent("Second"))
		cl.FocusFirst()
		cl.FocusNext()

		focused := cl.Focused()
		assert.Equal(t, "Second", focused.Title())
	})
}

func TestMessages(t *testing.T) {
	t.Run("FocusMsg is a message", func(t *testing.T) {
		msg := FocusMsg{}
		assert.NotNil(t, msg)
	})

	t.Run("BlurMsg is a message", func(t *testing.T) {
		msg := BlurMsg{}
		assert.NotNil(t, msg)
	})

	t.Run("NavigateMsg carries direction", func(t *testing.T) {
		msg := NavigateMsg{Direction: NavDown}
		assert.Equal(t, NavDown, msg.Direction)
	})

	t.Run("SelectMsg carries item", func(t *testing.T) {
		msg := SelectMsg{ID: "item-123"}
		assert.Equal(t, "item-123", msg.ID)
	})
}

func TestNavigationDirection(t *testing.T) {
	t.Run("direction constants", func(t *testing.T) {
		assert.Equal(t, NavDirection(0), NavUp)
		assert.Equal(t, NavDirection(1), NavDown)
		assert.Equal(t, NavDirection(2), NavLeft)
		assert.Equal(t, NavDirection(3), NavRight)
	})
}
