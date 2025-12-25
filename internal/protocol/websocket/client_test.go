package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/internal/interfaces"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWSUpgrader upgrades HTTP connections to WebSocket for testing.
var testWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newTestWSServer creates a test WebSocket server.
func newTestWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()
		if handler != nil {
			handler(conn)
		} else {
			// Default: echo messages back
			for {
				mt, msg, err := conn.ReadMessage()
				if err != nil {
					break
				}
				conn.WriteMessage(mt, msg)
			}
		}
	}))
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 30*time.Second, cfg.ConnectTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.PingInterval)
	assert.Equal(t, 60*time.Second, cfg.PongTimeout)
	assert.Equal(t, int64(10*1024*1024), cfg.MaxMessageSize)
	assert.Equal(t, 5*time.Second, cfg.ReconnectDelay)
	assert.Equal(t, 3, cfg.MaxReconnects)
	assert.Equal(t, 0, cfg.MaxConnections)
	assert.False(t, cfg.TLSInsecure)
}

func TestNewClient(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		client := NewClient(nil)

		assert.NotNil(t, client)
		assert.NotNil(t, client.config)
		assert.Equal(t, 30*time.Second, client.config.ConnectTimeout)
		assert.Equal(t, 0, client.ConnectionCount())
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			ConnectTimeout: 5 * time.Second,
			MaxConnections: 10,
		}
		client := NewClient(cfg)

		assert.NotNil(t, client)
		assert.Equal(t, 5*time.Second, client.config.ConnectTimeout)
		assert.Equal(t, 10, client.config.MaxConnections)
	})
}

func TestClient_Protocol(t *testing.T) {
	client := NewClient(nil)
	assert.Equal(t, "websocket", client.Protocol())
}

func TestClient_Connect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		server := newTestWSServer(t, nil)
		defer server.Close()

		client := NewClient(&Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0, // Disable pings for test
		})

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, 1, client.ConnectionCount())
		assert.Equal(t, interfaces.ConnectionStateConnected, conn.State())

		conn.Close()
	})

	t.Run("connection failure", func(t *testing.T) {
		client := NewClient(&Config{
			ConnectTimeout: 100 * time.Millisecond,
			PingInterval:   0,
		})

		_, err := client.Connect(context.Background(), "ws://localhost:59999/nonexistent", interfaces.ConnectionOptions{})

		assert.Error(t, err)
		assert.Equal(t, 0, client.ConnectionCount())
	})

	t.Run("max connections limit", func(t *testing.T) {
		server := newTestWSServer(t, nil)
		defer server.Close()

		client := NewClient(&Config{
			ConnectTimeout: 5 * time.Second,
			MaxConnections: 1,
			PingInterval:   0,
		})

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// First connection succeeds
		conn1, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
		require.NoError(t, err)
		defer conn1.Close()

		// Second connection fails due to limit
		_, err = client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
		assert.Equal(t, ErrMaxConnectionsReached, err)
	})

	t.Run("with custom headers", func(t *testing.T) {
		var receivedAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			conn, err := testWSUpgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			conn.Close()
		}))
		defer server.Close()

		client := NewClient(&Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{
			Headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "Bearer test-token", receivedAuth)
		conn.Close()
	})

	t.Run("with custom timeout", func(t *testing.T) {
		client := NewClient(&Config{
			ConnectTimeout: 30 * time.Second,
			PingInterval:   0,
		})

		ctx := context.Background()
		_, err := client.Connect(ctx, "ws://localhost:59999/slow", interfaces.ConnectionOptions{
			Timeout: 50 * time.Millisecond,
		})

		assert.Error(t, err)
	})
}

func TestClient_Disconnect(t *testing.T) {
	t.Run("disconnect existing connection", func(t *testing.T) {
		server := newTestWSServer(t, nil)
		defer server.Close()

		client := NewClient(&Config{
			ConnectTimeout: 5 * time.Second,
			PingInterval:   0,
		})

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
		require.NoError(t, err)

		err = client.Disconnect(conn.ID())
		assert.NoError(t, err)
		assert.Equal(t, 0, client.ConnectionCount())
	})

	t.Run("disconnect nonexistent connection", func(t *testing.T) {
		client := NewClient(nil)

		err := client.Disconnect("nonexistent-id")
		assert.Equal(t, ErrConnectionNotFound, err)
	})
}

func TestClient_ListConnections(t *testing.T) {
	server := newTestWSServer(t, nil)
	defer server.Close()

	client := NewClient(&Config{
		ConnectTimeout: 5 * time.Second,
		PingInterval:   0,
	})

	// Initially empty
	assert.Empty(t, client.ListConnections())

	// Add a connection
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
	require.NoError(t, err)
	defer conn.Close()

	// Should list one connection
	infos := client.ListConnections()
	assert.Len(t, infos, 1)
	assert.Equal(t, conn.ID(), infos[0].ID)
	assert.Equal(t, wsURL, infos[0].Endpoint)
	assert.Equal(t, interfaces.ConnectionStateConnected, infos[0].State)
	assert.Equal(t, "websocket", infos[0].Protocol)
}

func TestClient_GetConnection(t *testing.T) {
	server := newTestWSServer(t, nil)
	defer server.Close()

	client := NewClient(&Config{
		ConnectTimeout: 5 * time.Second,
		PingInterval:   0,
	})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
	require.NoError(t, err)
	defer conn.Close()

	t.Run("existing connection", func(t *testing.T) {
		found, err := client.GetConnection(conn.ID())
		assert.NoError(t, err)
		assert.Equal(t, conn.ID(), found.ID())
	})

	t.Run("nonexistent connection", func(t *testing.T) {
		_, err := client.GetConnection("nonexistent")
		assert.Equal(t, ErrConnectionNotFound, err)
	})
}

func TestClient_GetWebSocketConnection(t *testing.T) {
	server := newTestWSServer(t, nil)
	defer server.Close()

	client := NewClient(&Config{
		ConnectTimeout: 5 * time.Second,
		PingInterval:   0,
	})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
	require.NoError(t, err)
	defer conn.Close()

	t.Run("existing connection", func(t *testing.T) {
		wsConn, err := client.GetWebSocketConnection(conn.ID())
		assert.NoError(t, err)
		assert.NotNil(t, wsConn)
	})

	t.Run("nonexistent connection", func(t *testing.T) {
		_, err := client.GetWebSocketConnection("nonexistent")
		assert.Equal(t, ErrConnectionNotFound, err)
	})
}

func TestClient_CloseAll(t *testing.T) {
	server := newTestWSServer(t, nil)
	defer server.Close()

	client := NewClient(&Config{
		ConnectTimeout: 5 * time.Second,
		MaxConnections: 10,
		PingInterval:   0,
	})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create multiple connections
	for i := 0; i < 3; i++ {
		_, err := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
		require.NoError(t, err)
	}
	assert.Equal(t, 3, client.ConnectionCount())

	// Close all
	client.CloseAll()
	assert.Equal(t, 0, client.ConnectionCount())
}

func TestClient_ConnectionCount(t *testing.T) {
	server := newTestWSServer(t, nil)
	defer server.Close()

	client := NewClient(&Config{
		ConnectTimeout: 5 * time.Second,
		PingInterval:   0,
	})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	assert.Equal(t, 0, client.ConnectionCount())

	conn1, _ := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
	assert.Equal(t, 1, client.ConnectionCount())

	conn2, _ := client.Connect(context.Background(), wsURL, interfaces.ConnectionOptions{})
	assert.Equal(t, 2, client.ConnectionCount())

	client.Disconnect(conn1.ID())
	assert.Equal(t, 1, client.ConnectionCount())

	client.Disconnect(conn2.ID())
	assert.Equal(t, 0, client.ConnectionCount())
}

func TestGenerateConnectionID(t *testing.T) {
	id1 := generateConnectionID()

	assert.True(t, strings.HasPrefix(id1, "ws-"))
	assert.NotEmpty(t, strings.TrimPrefix(id1, "ws-"))
}
