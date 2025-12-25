package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessageType_String(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"text message", MessageTypeText, "text"},
		{"binary message", MessageTypeBinary, "binary"},
		{"ping message", MessageTypePing, "ping"},
		{"pong message", MessageTypePong, "pong"},
		{"close message", MessageTypeClose, "close"},
		{"unknown message", MessageType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.msgType.String())
		})
	}
}

func TestDirection_String(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		expected  string
	}{
		{"sent direction", DirectionSent, "sent"},
		{"received direction", DirectionReceived, "received"},
		{"unknown direction", Direction(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.direction.String())
		})
	}
}

func TestNewTextMessage(t *testing.T) {
	connID := "test-conn-1"
	data := []byte("hello world")

	t.Run("sent message", func(t *testing.T) {
		msg := NewTextMessage(connID, data, DirectionSent)

		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, MessageTypeText, msg.Type)
		assert.Equal(t, data, msg.Data)
		assert.Equal(t, DirectionSent, msg.Direction)
		assert.Equal(t, connID, msg.ConnectionID)
		assert.False(t, msg.Filtered)
		assert.Empty(t, msg.Error)
		assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
	})

	t.Run("received message", func(t *testing.T) {
		msg := NewTextMessage(connID, data, DirectionReceived)

		assert.Equal(t, DirectionReceived, msg.Direction)
		assert.Equal(t, MessageTypeText, msg.Type)
	})
}

func TestNewBinaryMessage(t *testing.T) {
	connID := "test-conn-2"
	data := []byte{0x00, 0x01, 0x02, 0x03}

	t.Run("sent binary", func(t *testing.T) {
		msg := NewBinaryMessage(connID, data, DirectionSent)

		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, MessageTypeBinary, msg.Type)
		assert.Equal(t, data, msg.Data)
		assert.Equal(t, DirectionSent, msg.Direction)
		assert.Equal(t, connID, msg.ConnectionID)
	})

	t.Run("received binary", func(t *testing.T) {
		msg := NewBinaryMessage(connID, data, DirectionReceived)

		assert.Equal(t, DirectionReceived, msg.Direction)
		assert.Equal(t, MessageTypeBinary, msg.Type)
	})
}

func TestMessage_Content(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"simple text", []byte("hello"), "hello"},
		{"empty data", []byte{}, ""},
		{"unicode text", []byte("日本語"), "日本語"},
		{"json data", []byte(`{"key":"value"}`), `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Data: tt.data}
			assert.Equal(t, tt.expected, msg.Content())
		})
	}
}

func TestMessage_IsControl(t *testing.T) {
	tests := []struct {
		name      string
		msgType   MessageType
		isControl bool
	}{
		{"text is not control", MessageTypeText, false},
		{"binary is not control", MessageTypeBinary, false},
		{"ping is control", MessageTypePing, true},
		{"pong is control", MessageTypePong, true},
		{"close is control", MessageTypeClose, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Type: tt.msgType}
			assert.Equal(t, tt.isControl, msg.IsControl())
		})
	}
}

func TestGenerateMessageID(t *testing.T) {
	id1 := generateMessageID()
	time.Sleep(time.Millisecond)
	id2 := generateMessageID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "sequential IDs should be unique")
}
