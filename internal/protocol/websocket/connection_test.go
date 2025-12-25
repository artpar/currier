package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/artpar/currier/internal/interfaces"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// echoWSServer creates a WebSocket server that echoes messages.
func echoWSServer(t *testing.T) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			conn.WriteMessage(mt, msg)
		}
	}))
}

func TestNewConnection(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		conn := NewConnection("test-id", "ws://localhost:8080", nil)

		assert.Equal(t, "test-id", conn.ID())
		assert.Equal(t, "ws://localhost:8080", conn.Endpoint())
		assert.Equal(t, interfaces.ConnectionStateDisconnected, conn.State())
		assert.NotNil(t, conn.config)
		assert.Equal(t, 30*time.Second, conn.config.ConnectTimeout)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			ConnectTimeout: 10 * time.Second,
			PingInterval:   5 * time.Second,
		}
		conn := NewConnection("custom-id", "ws://example.com/ws", cfg)

		assert.Equal(t, "custom-id", conn.ID())
		assert.Equal(t, "ws://example.com/ws", conn.Endpoint())
		assert.Equal(t, 10*time.Second, conn.config.ConnectTimeout)
	})
}

func TestConnection_IDEndpointState(t *testing.T) {
	conn := NewConnection("conn-123", "ws://test.local/socket", nil)

	assert.Equal(t, "conn-123", conn.ID())
	assert.Equal(t, "ws://test.local/socket", conn.Endpoint())
	assert.Equal(t, interfaces.ConnectionStateDisconnected, conn.State())
}

func TestConnection_SetHeader(t *testing.T) {
	conn := NewConnection("test", "ws://localhost", nil)

	conn.SetHeader("Authorization", "Bearer token123")
	conn.SetHeader("X-Custom", "value")

	// Headers are internal, verify via connect if needed
	assert.NotNil(t, conn.headers)
}

func TestConnection_SetHeaders(t *testing.T) {
	conn := NewConnection("test", "ws://localhost", nil)

	conn.SetHeaders(map[string]string{
		"Authorization": "Bearer token",
		"X-Api-Key":     "key123",
	})

	assert.NotNil(t, conn.headers)
}

func TestConnection_Connect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
			PongTimeout:    60 * time.Second,
		})

		err := conn.Connect(context.Background())
		require.NoError(t, err)
		assert.Equal(t, interfaces.ConnectionStateConnected, conn.State())

		conn.Close()
	})

	t.Run("connection failure", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost:59999/invalid", &Config{
			ConnectTimeout: 100 * time.Millisecond,
			PingInterval:   0,
		})

		err := conn.Connect(context.Background())
		assert.Error(t, err)
		assert.Equal(t, interfaces.ConnectionStateError, conn.State())
	})

	t.Run("already connected returns nil", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		err := conn.Connect(context.Background())
		require.NoError(t, err)

		// Second connect should return nil (already connected)
		err = conn.Connect(context.Background())
		assert.NoError(t, err)

		conn.Close()
	})
}

func TestConnection_Send(t *testing.T) {
	t.Run("send when connected", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			WriteTimeout:   5 * time.Second,
			PingInterval:   0,
			PongTimeout:    60 * time.Second,
		})

		require.NoError(t, conn.Connect(context.Background()))
		defer conn.Close()

		err := conn.Send(context.Background(), []byte("hello"))
		assert.NoError(t, err)
	})

	t.Run("send when not connected", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost", nil)

		err := conn.Send(context.Background(), []byte("hello"))
		assert.Equal(t, ErrConnectionNotConnected, err)
	})
}

func TestConnection_SendBinary(t *testing.T) {
	t.Run("send binary when connected", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			WriteTimeout:   5 * time.Second,
			PingInterval:   0,
			PongTimeout:    60 * time.Second,
		})

		require.NoError(t, conn.Connect(context.Background()))
		defer conn.Close()

		err := conn.SendBinary(context.Background(), []byte{0x00, 0x01, 0x02})
		assert.NoError(t, err)
	})

	t.Run("send binary when not connected", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost", nil)

		err := conn.SendBinary(context.Background(), []byte{0x00})
		assert.Equal(t, ErrConnectionNotConnected, err)
	})
}

func TestConnection_Receive(t *testing.T) {
	t.Run("receive when not connected", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := conn.Receive(ctx)
		assert.Equal(t, ErrConnectionNotConnected, err)
	})

	t.Run("receive with context cancellation", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
			PongTimeout:    60 * time.Second,
		})

		require.NoError(t, conn.Connect(context.Background()))
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := conn.Receive(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestConnection_Close(t *testing.T) {
	t.Run("close connected connection", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		require.NoError(t, conn.Connect(context.Background()))
		assert.Equal(t, interfaces.ConnectionStateConnected, conn.State())

		err := conn.Close()
		assert.NoError(t, err)
		assert.Equal(t, interfaces.ConnectionStateDisconnected, conn.State())
	})

	t.Run("close disconnected connection", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost", nil)

		err := conn.Close()
		assert.NoError(t, err)
	})

	t.Run("double close", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		require.NoError(t, conn.Connect(context.Background()))

		err := conn.Close()
		assert.NoError(t, err)

		err = conn.Close()
		assert.NoError(t, err)
	})
}

func TestConnection_Callbacks(t *testing.T) {
	t.Run("OnMessage callback", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			WriteTimeout:   5 * time.Second,
			PingInterval:   0,
			PongTimeout:    60 * time.Second,
		})

		var receivedMessages []*Message
		var mu sync.Mutex

		conn.OnMessage(func(msg *Message) {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()
		})

		require.NoError(t, conn.Connect(context.Background()))
		defer conn.Close()

		// Send a message (will be echoed back)
		err := conn.Send(context.Background(), []byte("test message"))
		require.NoError(t, err)

		// Wait for echo
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		// Should have sent message and received echo
		assert.GreaterOrEqual(t, len(receivedMessages), 1)
	})

	t.Run("OnStateChange callback", func(t *testing.T) {
		server := echoWSServer(t)
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn := NewConnection("test", wsURL, &Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		var states []interfaces.ConnectionState
		var mu sync.Mutex

		conn.OnStateChange(func(state interfaces.ConnectionState) {
			mu.Lock()
			states = append(states, state)
			mu.Unlock()
		})

		require.NoError(t, conn.Connect(context.Background()))

		// Wait for callbacks
		time.Sleep(50 * time.Millisecond)

		conn.Close()

		// Wait for close callback
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		// Should transition through: connecting -> connected -> disconnecting -> disconnected
		assert.Contains(t, states, interfaces.ConnectionStateConnecting)
		assert.Contains(t, states, interfaces.ConnectionStateConnected)
	})

	t.Run("OnError callback", func(t *testing.T) {
		conn := NewConnection("test", "ws://localhost:59999/invalid", &Config{
			ConnectTimeout: 100 * time.Millisecond,
			PingInterval:   0,
		})

		var receivedError error
		var mu sync.Mutex

		conn.OnError(func(err error) {
			mu.Lock()
			receivedError = err
			mu.Unlock()
		})

		_ = conn.Connect(context.Background())

		// Wait for error callback
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.NotNil(t, receivedError)
	})
}

func TestConnection_LastPingPong(t *testing.T) {
	conn := NewConnection("test", "ws://localhost", nil)

	// Initially zero
	assert.True(t, conn.LastPing().IsZero())
	assert.True(t, conn.LastPong().IsZero())
}

func TestConnection_SendWithEcho(t *testing.T) {
	server := echoWSServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn := NewConnection("test", wsURL, &Config{
		ConnectTimeout: 5 * time.Second,
		WriteTimeout:   5 * time.Second,
		PingInterval:   0,
		PongTimeout:    60 * time.Second,
	})

	var messages []*Message
	var mu sync.Mutex

	conn.OnMessage(func(msg *Message) {
		mu.Lock()
		messages = append(messages, msg)
		mu.Unlock()
	})

	require.NoError(t, conn.Connect(context.Background()))
	defer conn.Close()

	// Send message
	testData := []byte(`{"action":"ping"}`)
	err := conn.Send(context.Background(), testData)
	require.NoError(t, err)

	// Wait for echo
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Find received echo
	var echoReceived bool
	for _, msg := range messages {
		if msg.Direction == DirectionReceived && string(msg.Data) == string(testData) {
			echoReceived = true
			break
		}
	}
	assert.True(t, echoReceived, "should receive echoed message")
}
