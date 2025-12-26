package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketDefinition(t *testing.T) {
	t.Run("creates definition with name and endpoint", func(t *testing.T) {
		def := NewWebSocketDefinition("My WebSocket", "wss://api.example.com/ws")
		assert.NotEmpty(t, def.ID)
		assert.Equal(t, "My WebSocket", def.Name)
		assert.Equal(t, "wss://api.example.com/ws", def.Endpoint)
	})

	t.Run("initializes with default values", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost:8080")
		assert.NotNil(t, def.Headers)
		assert.NotNil(t, def.Subprotocols)
		assert.NotNil(t, def.AutoResponseRules)
		assert.Equal(t, 30, def.PingInterval)
		assert.True(t, def.ReconnectEnabled)
		assert.Equal(t, 3, def.MaxReconnectAttempts)
	})

	t.Run("sets timestamps", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		assert.False(t, def.CreatedAt.IsZero())
		assert.False(t, def.UpdatedAt.IsZero())
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		def1 := NewWebSocketDefinition("WS1", "ws://localhost:8080")
		def2 := NewWebSocketDefinition("WS2", "ws://localhost:8081")
		assert.NotEqual(t, def1.ID, def2.ID)
	})
}

func TestWebSocketDefinition_Clone(t *testing.T) {
	t.Run("creates deep copy with new ID", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "wss://api.example.com/ws")
		original.Headers["Authorization"] = "Bearer token123"
		original.Subprotocols = []string{"graphql-ws"}
		original.PingInterval = 60

		clone := original.Clone()

		assert.NotEqual(t, original.ID, clone.ID)
		assert.Contains(t, clone.Name, "(copy)")
		assert.Equal(t, original.Endpoint, clone.Endpoint)
		assert.Equal(t, original.PingInterval, clone.PingInterval)
	})

	t.Run("copies headers independently", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "ws://localhost")
		original.Headers["X-Custom"] = "value"

		clone := original.Clone()
		clone.Headers["X-Custom"] = "modified"

		assert.Equal(t, "value", original.Headers["X-Custom"])
		assert.Equal(t, "modified", clone.Headers["X-Custom"])
	})

	t.Run("copies subprotocols independently", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "ws://localhost")
		original.Subprotocols = []string{"proto1", "proto2"}

		clone := original.Clone()
		assert.Equal(t, len(original.Subprotocols), len(clone.Subprotocols))
	})

	t.Run("copies auth config", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "ws://localhost")
		original.Auth = &AuthConfig{
			Type:     "bearer",
			Token:    "secret-token",
			Username: "user",
		}

		clone := original.Clone()

		assert.NotNil(t, clone.Auth)
		assert.Equal(t, original.Auth.Type, clone.Auth.Type)
		assert.Equal(t, original.Auth.Token, clone.Auth.Token)

		// Modify clone - should not affect original
		clone.Auth.Token = "modified"
		assert.Equal(t, "secret-token", original.Auth.Token)
	})

	t.Run("copies auto response rules with new IDs", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "ws://localhost")
		rule := NewAutoResponseRule("Pong", "msg => msg === 'ping'", "pong")
		original.AutoResponseRules = append(original.AutoResponseRules, *rule)

		clone := original.Clone()

		assert.Equal(t, 1, len(clone.AutoResponseRules))
		assert.NotEqual(t, original.AutoResponseRules[0].ID, clone.AutoResponseRules[0].ID)
		assert.Equal(t, original.AutoResponseRules[0].Name, clone.AutoResponseRules[0].Name)
	})

	t.Run("handles nil auth", func(t *testing.T) {
		original := NewWebSocketDefinition("Original", "ws://localhost")
		original.Auth = nil

		clone := original.Clone()
		assert.Nil(t, clone.Auth)
	})
}

func TestNewAutoResponseRule(t *testing.T) {
	t.Run("creates rule with required fields", func(t *testing.T) {
		rule := NewAutoResponseRule("Heartbeat", "msg => msg === 'ping'", "pong")

		assert.NotEmpty(t, rule.ID)
		assert.Equal(t, "Heartbeat", rule.Name)
		assert.Equal(t, "msg => msg === 'ping'", rule.MatchScript)
		assert.Equal(t, "pong", rule.Response)
		assert.True(t, rule.Enabled)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		rule1 := NewAutoResponseRule("Rule1", "match1", "response1")
		rule2 := NewAutoResponseRule("Rule2", "match2", "response2")
		assert.NotEqual(t, rule1.ID, rule2.ID)
	})
}

func TestNewWebSocketMessage(t *testing.T) {
	t.Run("creates message with required fields", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-123", "Hello, World!", "sent")

		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, "conn-123", msg.ConnectionID)
		assert.Equal(t, "Hello, World!", msg.Content)
		assert.Equal(t, "sent", msg.Direction)
		assert.Equal(t, "text", msg.Type)
		assert.False(t, msg.Timestamp.IsZero())
	})

	t.Run("creates received message", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-456", "Response data", "received")
		assert.Equal(t, "received", msg.Direction)
	})
}

func TestWebSocketMessage_IsSent(t *testing.T) {
	t.Run("returns true for sent messages", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-1", "Hello", "sent")
		assert.True(t, msg.IsSent())
		assert.False(t, msg.IsReceived())
	})

	t.Run("returns false for received messages", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-1", "Hello", "received")
		assert.False(t, msg.IsSent())
	})
}

func TestWebSocketMessage_IsReceived(t *testing.T) {
	t.Run("returns true for received messages", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-1", "Response", "received")
		assert.True(t, msg.IsReceived())
		assert.False(t, msg.IsSent())
	})

	t.Run("returns false for sent messages", func(t *testing.T) {
		msg := NewWebSocketMessage("conn-1", "Request", "sent")
		assert.False(t, msg.IsReceived())
	})
}

func TestNewWebSocketSession(t *testing.T) {
	t.Run("creates session with definition", func(t *testing.T) {
		def := NewWebSocketDefinition("Test WS", "ws://localhost")
		session := NewWebSocketSession(def)

		assert.Equal(t, def, session.Definition)
		assert.Empty(t, session.ConnectionID)
		assert.NotNil(t, session.Messages)
		assert.Empty(t, session.Messages)
		assert.False(t, session.StartedAt.IsZero())
		assert.True(t, session.EndedAt.IsZero())
	})
}

func TestWebSocketSession_AddMessage(t *testing.T) {
	t.Run("adds messages to session", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)

		msg1 := NewWebSocketMessage("conn-1", "Hello", "sent")
		msg2 := NewWebSocketMessage("conn-1", "Hi there", "received")

		session.AddMessage(msg1)
		assert.Equal(t, 1, len(session.Messages))

		session.AddMessage(msg2)
		assert.Equal(t, 2, len(session.Messages))
	})
}

func TestWebSocketSession_MessageCount(t *testing.T) {
	t.Run("returns correct count", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)

		assert.Equal(t, 0, session.MessageCount())

		session.AddMessage(NewWebSocketMessage("conn-1", "msg1", "sent"))
		assert.Equal(t, 1, session.MessageCount())

		session.AddMessage(NewWebSocketMessage("conn-1", "msg2", "received"))
		session.AddMessage(NewWebSocketMessage("conn-1", "msg3", "sent"))
		assert.Equal(t, 3, session.MessageCount())
	})
}

func TestWebSocketSession_LastMessage(t *testing.T) {
	t.Run("returns nil for empty session", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)

		assert.Nil(t, session.LastMessage())
	})

	t.Run("returns last message", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)

		msg1 := NewWebSocketMessage("conn-1", "First", "sent")
		msg2 := NewWebSocketMessage("conn-1", "Second", "received")
		msg3 := NewWebSocketMessage("conn-1", "Third", "sent")

		session.AddMessage(msg1)
		session.AddMessage(msg2)
		session.AddMessage(msg3)

		last := session.LastMessage()
		assert.NotNil(t, last)
		assert.Equal(t, "Third", last.Content)
	})
}

func TestWebSocketSession_IsActive(t *testing.T) {
	t.Run("returns false for new session without connection", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)

		assert.False(t, session.IsActive())
	})

	t.Run("returns true when connection ID is set", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)
		session.ConnectionID = "conn-123"

		assert.True(t, session.IsActive())
	})

	t.Run("returns false after session ended", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)
		session.ConnectionID = "conn-123"

		session.End()

		assert.False(t, session.IsActive())
	})
}

func TestWebSocketSession_End(t *testing.T) {
	t.Run("marks session as ended", func(t *testing.T) {
		def := NewWebSocketDefinition("Test", "ws://localhost")
		session := NewWebSocketSession(def)
		session.ConnectionID = "conn-123"

		assert.True(t, session.EndedAt.IsZero())

		session.End()

		assert.False(t, session.EndedAt.IsZero())
		assert.Empty(t, session.ConnectionID)
	})
}

func TestCollection_WebSockets(t *testing.T) {
	t.Run("returns empty list initially", func(t *testing.T) {
		c := NewCollection("Test API")
		assert.Empty(t, c.WebSockets())
	})

	t.Run("AddWebSocket adds to collection", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("WS1", "ws://localhost")

		c.AddWebSocket(ws)

		assert.Equal(t, 1, len(c.WebSockets()))
	})

	t.Run("GetWebSocket retrieves by ID", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("WS1", "ws://localhost")
		c.AddWebSocket(ws)

		found, ok := c.GetWebSocket(ws.ID)
		assert.True(t, ok)
		assert.NotNil(t, found)
		assert.Equal(t, ws.ID, found.ID)
	})

	t.Run("GetWebSocket returns false for unknown ID", func(t *testing.T) {
		c := NewCollection("Test API")
		_, ok := c.GetWebSocket("unknown-id")
		assert.False(t, ok)
	})

	t.Run("GetWebSocketByName retrieves by name", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("My WebSocket", "ws://localhost")
		c.AddWebSocket(ws)

		found, ok := c.GetWebSocketByName("My WebSocket")
		assert.True(t, ok)
		assert.NotNil(t, found)
		assert.Equal(t, "My WebSocket", found.Name)
	})

	t.Run("GetWebSocketByName returns false for unknown name", func(t *testing.T) {
		c := NewCollection("Test API")
		_, ok := c.GetWebSocketByName("Unknown")
		assert.False(t, ok)
	})

	t.Run("RemoveWebSocket removes from collection", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("WS1", "ws://localhost")
		c.AddWebSocket(ws)

		assert.Equal(t, 1, len(c.WebSockets()))

		c.RemoveWebSocket(ws.ID)

		assert.Equal(t, 0, len(c.WebSockets()))
	})

	t.Run("RemoveWebSocket handles unknown ID", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("WS1", "ws://localhost")
		c.AddWebSocket(ws)

		c.RemoveWebSocket("unknown-id")

		assert.Equal(t, 1, len(c.WebSockets()))
	})

	t.Run("AddExistingWebSocket adds without generating new ID", func(t *testing.T) {
		c := NewCollection("Test API")
		ws := NewWebSocketDefinition("WS1", "ws://localhost")
		originalID := ws.ID

		c.AddExistingWebSocket(ws)

		found, ok := c.GetWebSocket(originalID)
		assert.True(t, ok)
		assert.NotNil(t, found)
		assert.Equal(t, originalID, found.ID)
	})
}
