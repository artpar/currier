package websocket

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/artpar/currier/internal/interfaces"
)

var (
	// ErrConnectionNotFound is returned when a connection is not found.
	ErrConnectionNotFound = errors.New("connection not found")
	// ErrMaxConnectionsReached is returned when max connections limit is reached.
	ErrMaxConnectionsReached = errors.New("max connections reached")
)

// Config holds WebSocket client configuration.
type Config struct {
	// ConnectTimeout is the timeout for establishing a connection.
	ConnectTimeout time.Duration

	// WriteTimeout is the timeout for write operations.
	WriteTimeout time.Duration

	// PingInterval is the interval between ping messages. 0 disables pings.
	PingInterval time.Duration

	// PongTimeout is the timeout for receiving a pong response.
	PongTimeout time.Duration

	// MaxMessageSize is the maximum size of a message in bytes.
	MaxMessageSize int64

	// ReconnectDelay is the delay before attempting to reconnect.
	ReconnectDelay time.Duration

	// MaxReconnects is the maximum number of reconnection attempts. 0 disables auto-reconnect.
	MaxReconnects int

	// MaxConnections is the maximum number of concurrent connections. 0 means unlimited.
	MaxConnections int

	// TLSInsecure allows insecure TLS connections.
	TLSInsecure bool
}

// DefaultConfig returns the default WebSocket client configuration.
func DefaultConfig() *Config {
	return &Config{
		ConnectTimeout: 30 * time.Second,
		WriteTimeout:   10 * time.Second,
		PingInterval:   30 * time.Second,
		PongTimeout:    60 * time.Second,
		MaxMessageSize: 10 * 1024 * 1024, // 10 MB
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  3,
		MaxConnections: 0, // Unlimited
		TLSInsecure:    false,
	}
}

// Client is the WebSocket client that manages connections.
type Client struct {
	config      *Config
	connections map[string]*Connection
	mu          sync.RWMutex
}

// NewClient creates a new WebSocket client with the given configuration.
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{
		config:      config,
		connections: make(map[string]*Connection),
	}
}

// Protocol returns the protocol identifier.
func (c *Client) Protocol() string {
	return "websocket"
}

// Connect establishes a connection to the given endpoint.
func (c *Client) Connect(ctx context.Context, endpoint string, opts interfaces.ConnectionOptions) (interfaces.Connection, error) {
	c.mu.Lock()

	// Check max connections
	if c.config.MaxConnections > 0 && len(c.connections) >= c.config.MaxConnections {
		c.mu.Unlock()
		return nil, ErrMaxConnectionsReached
	}

	// Generate connection ID
	id := generateConnectionID()

	// Create connection config from client config and options
	connConfig := &Config{
		ConnectTimeout: c.config.ConnectTimeout,
		WriteTimeout:   c.config.WriteTimeout,
		PingInterval:   c.config.PingInterval,
		PongTimeout:    c.config.PongTimeout,
		MaxMessageSize: c.config.MaxMessageSize,
		ReconnectDelay: c.config.ReconnectDelay,
		MaxReconnects:  c.config.MaxReconnects,
		TLSInsecure:    opts.TLSInsecure || c.config.TLSInsecure,
	}

	// Override timeout if specified in options
	if opts.Timeout > 0 {
		connConfig.ConnectTimeout = opts.Timeout
	}

	// Create connection
	conn := NewConnection(id, endpoint, connConfig)

	// Set headers from options
	if opts.Headers != nil {
		conn.SetHeaders(opts.Headers)
	}

	c.connections[id] = conn
	c.mu.Unlock()

	// Connect
	if err := conn.Connect(ctx); err != nil {
		c.mu.Lock()
		delete(c.connections, id)
		c.mu.Unlock()
		return nil, err
	}

	return conn, nil
}

// Disconnect closes the connection with the given ID.
func (c *Client) Disconnect(id string) error {
	c.mu.Lock()
	conn, ok := c.connections[id]
	if !ok {
		c.mu.Unlock()
		return ErrConnectionNotFound
	}
	delete(c.connections, id)
	c.mu.Unlock()

	return conn.Close()
}

// ListConnections returns all active connections.
func (c *Client) ListConnections() []interfaces.ConnectionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	infos := make([]interfaces.ConnectionInfo, 0, len(c.connections))
	for _, conn := range c.connections {
		infos = append(infos, interfaces.ConnectionInfo{
			ID:       conn.ID(),
			Endpoint: conn.Endpoint(),
			State:    conn.State(),
			Protocol: "websocket",
		})
	}
	return infos
}

// GetConnection returns a specific connection by ID.
func (c *Client) GetConnection(id string) (interfaces.Connection, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, ok := c.connections[id]
	if !ok {
		return nil, ErrConnectionNotFound
	}
	return conn, nil
}

// GetWebSocketConnection returns a specific WebSocket connection by ID with full type.
func (c *Client) GetWebSocketConnection(id string) (*Connection, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, ok := c.connections[id]
	if !ok {
		return nil, ErrConnectionNotFound
	}
	return conn, nil
}

// CloseAll closes all connections.
func (c *Client) CloseAll() {
	c.mu.Lock()
	connections := make([]*Connection, 0, len(c.connections))
	for _, conn := range c.connections {
		connections = append(connections, conn)
	}
	c.connections = make(map[string]*Connection)
	c.mu.Unlock()

	for _, conn := range connections {
		conn.Close()
	}
}

// ConnectionCount returns the number of active connections.
func (c *Client) ConnectionCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.connections)
}

// generateConnectionID generates a unique connection ID.
func generateConnectionID() string {
	return fmt.Sprintf("ws-%d", time.Now().UnixNano())
}
