// Package websocket provides WebSocket protocol support for Currier.
package websocket

import (
	"time"
)

// MessageType represents the type of WebSocket message.
type MessageType int

const (
	// MessageTypeText is a text message.
	MessageTypeText MessageType = iota
	// MessageTypeBinary is a binary message.
	MessageTypeBinary
	// MessageTypePing is a ping control message.
	MessageTypePing
	// MessageTypePong is a pong control message.
	MessageTypePong
	// MessageTypeClose is a close control message.
	MessageTypeClose
)

// String returns the string representation of the message type.
func (t MessageType) String() string {
	switch t {
	case MessageTypeText:
		return "text"
	case MessageTypeBinary:
		return "binary"
	case MessageTypePing:
		return "ping"
	case MessageTypePong:
		return "pong"
	case MessageTypeClose:
		return "close"
	default:
		return "unknown"
	}
}

// Direction represents the direction of a message.
type Direction int

const (
	// DirectionSent indicates a message was sent by the client.
	DirectionSent Direction = iota
	// DirectionReceived indicates a message was received from the server.
	DirectionReceived
)

// String returns the string representation of the direction.
func (d Direction) String() string {
	switch d {
	case DirectionSent:
		return "sent"
	case DirectionReceived:
		return "received"
	default:
		return "unknown"
	}
}

// Message represents a WebSocket message.
type Message struct {
	// ID is the unique message identifier.
	ID string

	// Type is the message type (text, binary, ping, pong, close).
	Type MessageType

	// Data is the raw message data.
	Data []byte

	// Timestamp is when the message was sent/received.
	Timestamp time.Time

	// Direction indicates if this message was sent or received.
	Direction Direction

	// ConnectionID is the ID of the connection this message belongs to.
	ConnectionID string

	// Filtered indicates if this message was filtered by a script.
	Filtered bool

	// Error contains any error associated with this message.
	Error string
}

// NewTextMessage creates a new text message.
func NewTextMessage(connectionID string, data []byte, direction Direction) *Message {
	return &Message{
		ID:           generateMessageID(),
		Type:         MessageTypeText,
		Data:         data,
		Timestamp:    time.Now(),
		Direction:    direction,
		ConnectionID: connectionID,
	}
}

// NewBinaryMessage creates a new binary message.
func NewBinaryMessage(connectionID string, data []byte, direction Direction) *Message {
	return &Message{
		ID:           generateMessageID(),
		Type:         MessageTypeBinary,
		Data:         data,
		Timestamp:    time.Now(),
		Direction:    direction,
		ConnectionID: connectionID,
	}
}

// Content returns the message content as a string.
func (m *Message) Content() string {
	return string(m.Data)
}

// IsControl returns true if this is a control message (ping, pong, close).
func (m *Message) IsControl() bool {
	return m.Type == MessageTypePing || m.Type == MessageTypePong || m.Type == MessageTypeClose
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return time.Now().Format("20060102150405.000000000")
}
