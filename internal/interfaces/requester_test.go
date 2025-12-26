package interfaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{ConnectionStateConnecting, "connecting"},
		{ConnectionStateConnected, "connected"},
		{ConnectionStateDisconnecting, "disconnecting"},
		{ConnectionStateDisconnected, "disconnected"},
		{ConnectionStateError, "error"},
		{ConnectionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}
