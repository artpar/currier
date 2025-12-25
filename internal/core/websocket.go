package core

import (
	"time"

	"github.com/google/uuid"
)

// WebSocketDefinition defines a WebSocket connection configuration.
type WebSocketDefinition struct {
	// ID is the unique identifier.
	ID string `yaml:"id" json:"id"`

	// Name is the human-readable name.
	Name string `yaml:"name" json:"name"`

	// Endpoint is the WebSocket URL (ws:// or wss://).
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Headers are custom headers to send during handshake.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Subprotocols are the WebSocket subprotocols to request.
	Subprotocols []string `yaml:"subprotocols,omitempty" json:"subprotocols,omitempty"`

	// Auth is the authentication configuration.
	Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`

	// PreConnectScript runs before connecting.
	PreConnectScript string `yaml:"preConnectScript,omitempty" json:"preConnectScript,omitempty"`

	// PreMessageScript runs before sending a message.
	PreMessageScript string `yaml:"preMessageScript,omitempty" json:"preMessageScript,omitempty"`

	// PostMessageScript runs after receiving a message.
	PostMessageScript string `yaml:"postMessageScript,omitempty" json:"postMessageScript,omitempty"`

	// FilterScript determines whether to show/hide a message.
	FilterScript string `yaml:"filterScript,omitempty" json:"filterScript,omitempty"`

	// AutoResponseRules defines automatic responses to incoming messages.
	AutoResponseRules []AutoResponseRule `yaml:"autoResponseRules,omitempty" json:"autoResponseRules,omitempty"`

	// PingInterval is the interval for sending ping messages (in seconds).
	PingInterval int `yaml:"pingInterval,omitempty" json:"pingInterval,omitempty"`

	// ReconnectEnabled enables auto-reconnect on disconnect.
	ReconnectEnabled bool `yaml:"reconnectEnabled,omitempty" json:"reconnectEnabled,omitempty"`

	// MaxReconnectAttempts is the max number of reconnect attempts.
	MaxReconnectAttempts int `yaml:"maxReconnectAttempts,omitempty" json:"maxReconnectAttempts,omitempty"`

	// CreatedAt is when this definition was created.
	CreatedAt time.Time `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`

	// UpdatedAt is when this definition was last updated.
	UpdatedAt time.Time `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

// NewWebSocketDefinition creates a new WebSocket definition.
func NewWebSocketDefinition(name, endpoint string) *WebSocketDefinition {
	now := time.Now()
	return &WebSocketDefinition{
		ID:                   uuid.New().String(),
		Name:                 name,
		Endpoint:             endpoint,
		Headers:              make(map[string]string),
		Subprotocols:         []string{},
		AutoResponseRules:    []AutoResponseRule{},
		PingInterval:         30,
		ReconnectEnabled:     true,
		MaxReconnectAttempts: 3,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// Clone creates a deep copy of the WebSocket definition.
func (w *WebSocketDefinition) Clone() *WebSocketDefinition {
	clone := &WebSocketDefinition{
		ID:                   uuid.New().String(),
		Name:                 w.Name + " (copy)",
		Endpoint:             w.Endpoint,
		Headers:              make(map[string]string),
		Subprotocols:         make([]string, len(w.Subprotocols)),
		PreConnectScript:     w.PreConnectScript,
		PreMessageScript:     w.PreMessageScript,
		PostMessageScript:    w.PostMessageScript,
		FilterScript:         w.FilterScript,
		AutoResponseRules:    make([]AutoResponseRule, len(w.AutoResponseRules)),
		PingInterval:         w.PingInterval,
		ReconnectEnabled:     w.ReconnectEnabled,
		MaxReconnectAttempts: w.MaxReconnectAttempts,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	for k, v := range w.Headers {
		clone.Headers[k] = v
	}
	copy(clone.Subprotocols, w.Subprotocols)

	if w.Auth != nil {
		clone.Auth = &AuthConfig{
			Type:     w.Auth.Type,
			Username: w.Auth.Username,
			Password: w.Auth.Password,
			Token:    w.Auth.Token,
		}
	}

	for i, rule := range w.AutoResponseRules {
		clone.AutoResponseRules[i] = AutoResponseRule{
			ID:          uuid.New().String(),
			Name:        rule.Name,
			MatchScript: rule.MatchScript,
			Response:    rule.Response,
			Enabled:     rule.Enabled,
		}
	}

	return clone
}

// AutoResponseRule defines an automatic response rule.
type AutoResponseRule struct {
	// ID is the unique identifier.
	ID string `yaml:"id" json:"id"`

	// Name is the human-readable name.
	Name string `yaml:"name" json:"name"`

	// MatchScript is a JavaScript function that returns true if message matches.
	// Function signature: (message: string) => boolean
	MatchScript string `yaml:"matchScript" json:"matchScript"`

	// Response is the message to send when matched.
	// Can be a string or a JavaScript function that returns a string.
	Response string `yaml:"response" json:"response"`

	// Enabled indicates if this rule is active.
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// NewAutoResponseRule creates a new auto-response rule.
func NewAutoResponseRule(name, matchScript, response string) *AutoResponseRule {
	return &AutoResponseRule{
		ID:          uuid.New().String(),
		Name:        name,
		MatchScript: matchScript,
		Response:    response,
		Enabled:     true,
	}
}

// WebSocketMessage represents a message in a WebSocket conversation.
type WebSocketMessage struct {
	// ID is the unique message identifier.
	ID string `yaml:"id" json:"id"`

	// ConnectionID is the ID of the connection this message belongs to.
	ConnectionID string `yaml:"connectionId" json:"connectionId"`

	// Content is the message content.
	Content string `yaml:"content" json:"content"`

	// Direction is "sent" or "received".
	Direction string `yaml:"direction" json:"direction"`

	// Timestamp is when the message was sent/received.
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`

	// Type is "text" or "binary".
	Type string `yaml:"type" json:"type"`

	// Filtered indicates if this message was hidden by a filter script.
	Filtered bool `yaml:"filtered,omitempty" json:"filtered,omitempty"`

	// AutoResponse indicates if this was an automatic response.
	AutoResponse bool `yaml:"autoResponse,omitempty" json:"autoResponse,omitempty"`

	// Error contains any error message associated with this message.
	Error string `yaml:"error,omitempty" json:"error,omitempty"`
}

// NewWebSocketMessage creates a new WebSocket message.
func NewWebSocketMessage(connectionID, content, direction string) *WebSocketMessage {
	return &WebSocketMessage{
		ID:           uuid.New().String(),
		ConnectionID: connectionID,
		Content:      content,
		Direction:    direction,
		Timestamp:    time.Now(),
		Type:         "text",
	}
}

// IsSent returns true if this message was sent by the client.
func (m *WebSocketMessage) IsSent() bool {
	return m.Direction == "sent"
}

// IsReceived returns true if this message was received from the server.
func (m *WebSocketMessage) IsReceived() bool {
	return m.Direction == "received"
}

// WebSocketSession represents an active WebSocket session with message history.
type WebSocketSession struct {
	// Definition is the WebSocket definition.
	Definition *WebSocketDefinition

	// ConnectionID is the active connection ID (empty if disconnected).
	ConnectionID string

	// Messages is the message history for this session.
	Messages []*WebSocketMessage

	// StartedAt is when this session was started.
	StartedAt time.Time

	// EndedAt is when this session ended (zero if still active).
	EndedAt time.Time
}

// NewWebSocketSession creates a new WebSocket session.
func NewWebSocketSession(definition *WebSocketDefinition) *WebSocketSession {
	return &WebSocketSession{
		Definition: definition,
		Messages:   make([]*WebSocketMessage, 0),
		StartedAt:  time.Now(),
	}
}

// AddMessage adds a message to the session history.
func (s *WebSocketSession) AddMessage(msg *WebSocketMessage) {
	s.Messages = append(s.Messages, msg)
}

// MessageCount returns the number of messages in this session.
func (s *WebSocketSession) MessageCount() int {
	return len(s.Messages)
}

// LastMessage returns the most recent message, or nil if none.
func (s *WebSocketSession) LastMessage() *WebSocketMessage {
	if len(s.Messages) == 0 {
		return nil
	}
	return s.Messages[len(s.Messages)-1]
}

// IsActive returns true if this session is still active.
func (s *WebSocketSession) IsActive() bool {
	return s.EndedAt.IsZero() && s.ConnectionID != ""
}

// End marks this session as ended.
func (s *WebSocketSession) End() {
	s.EndedAt = time.Now()
	s.ConnectionID = ""
}
