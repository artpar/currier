package interfaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVimModeString(t *testing.T) {
	tests := []struct {
		mode VimMode
		want string
	}{
		{VimModeNormal, "NORMAL"},
		{VimModeInsert, "INSERT"},
		{VimModeVisual, "VISUAL"},
		{VimModeCommand, "COMMAND"},
		{VimModeSearch, "SEARCH"},
		{VimMode(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.String())
		})
	}
}
