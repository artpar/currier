package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/artpar/currier/internal/interfaces"
	"github.com/gorilla/websocket"
)

var (
	// ErrConnectionClosed is returned when the connection is closed.
	ErrConnectionClosed = errors.New("connection closed")
	// ErrConnectionNotConnected is returned when trying to use a disconnected connection.
	ErrConnectionNotConnected = errors.New("connection not connected")
	// ErrSendTimeout is returned when a send operation times out.
	ErrSendTimeout = errors.New("send timeout")
)

// Connection represents a WebSocket connection.
type Connection struct {
	id        string
	endpoint  string
	state     interfaces.ConnectionState
	conn      *websocket.Conn
	config    *Config
	headers   http.Header
	mu        sync.RWMutex
	closeChan chan struct{}
	msgChan   chan *Message
	errChan   chan error

	// Callbacks
	onMessage     func(*Message)
	onStateChange func(interfaces.ConnectionState)
	onError       func(error)

	// Ping/pong handling
	lastPing time.Time
	lastPong time.Time
}

// NewConnection creates a new WebSocket connection.
func NewConnection(id, endpoint string, config *Config) *Connection {
	if config == nil {
		config = DefaultConfig()
	}
	return &Connection{
		id:        id,
		endpoint:  endpoint,
		state:     interfaces.ConnectionStateDisconnected,
		config:    config,
		headers:   make(http.Header),
		closeChan: make(chan struct{}),
		msgChan:   make(chan *Message, 100),
		errChan:   make(chan error, 10),
	}
}

// ID returns the unique connection identifier.
func (c *Connection) ID() string {
	return c.id
}

// Endpoint returns the connection endpoint.
func (c *Connection) Endpoint() string {
	return c.endpoint
}

// State returns the current connection state.
func (c *Connection) State() interfaces.ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetHeader sets a connection header.
func (c *Connection) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers.Set(key, value)
}

// SetHeaders sets multiple connection headers.
func (c *Connection) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range headers {
		c.headers.Set(k, v)
	}
}

// OnMessage sets the message callback.
func (c *Connection) OnMessage(fn func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = fn
}

// OnStateChange sets the state change callback.
func (c *Connection) OnStateChange(fn func(interfaces.ConnectionState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = fn
}

// OnError sets the error callback.
func (c *Connection) OnError(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = fn
}

// Connect establishes the WebSocket connection.
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.state == interfaces.ConnectionStateConnected || c.state == interfaces.ConnectionStateConnecting {
		c.mu.Unlock()
		return nil
	}
	c.setState(interfaces.ConnectionStateConnecting)
	c.mu.Unlock()

	// Create dialer with config
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectTimeout,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	// Handle TLS options
	if c.config.TLSInsecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Create context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	// Connect
	c.mu.RLock()
	headers := c.headers.Clone()
	c.mu.RUnlock()

	conn, resp, err := dialer.DialContext(connectCtx, c.endpoint, headers)
	if err != nil {
		c.setState(interfaces.ConnectionStateError)
		c.notifyError(fmt.Errorf("failed to connect: %w", err))
		return err
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.conn = conn
	c.closeChan = make(chan struct{})
	c.setState(interfaces.ConnectionStateConnected)
	c.mu.Unlock()

	// Set up ping/pong handlers
	c.setupPingPong()

	// Start read loop
	go c.readLoop()

	// Start ping loop
	if c.config.PingInterval > 0 {
		go c.pingLoop()
	}

	return nil
}

// setupPingPong sets up ping/pong handlers.
func (c *Connection) setupPingPong() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}

	c.conn.SetPongHandler(func(appData string) error {
		c.mu.Lock()
		c.lastPong = time.Now()
		c.mu.Unlock()
		return nil
	})

	c.conn.SetPingHandler(func(appData string) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conn != nil {
			return c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		}
		return nil
	})
}

// readLoop continuously reads messages from the connection.
func (c *Connection) readLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		// Set read deadline
		if c.config.PongTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout))
		}

		msgType, data, err := conn.ReadMessage()
		if err != nil {
			// Check if connection was intentionally closed
			select {
			case <-c.closeChan:
				return
			default:
			}

			// Handle close errors
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.handleDisconnect()
				return
			}

			c.notifyError(err)
			c.handleDisconnect()
			return
		}

		// Convert gorilla message type to our type
		var mt MessageType
		switch msgType {
		case websocket.TextMessage:
			mt = MessageTypeText
		case websocket.BinaryMessage:
			mt = MessageTypeBinary
		case websocket.PingMessage:
			mt = MessageTypePing
		case websocket.PongMessage:
			mt = MessageTypePong
		case websocket.CloseMessage:
			mt = MessageTypeClose
		}

		msg := &Message{
			ID:           generateMessageID(),
			Type:         mt,
			Data:         data,
			Timestamp:    time.Now(),
			Direction:    DirectionReceived,
			ConnectionID: c.id,
		}

		c.notifyMessage(msg)
	}
}

// pingLoop sends periodic ping messages.
func (c *Connection) pingLoop() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeChan:
			return
		case <-ticker.C:
			c.mu.Lock()
			conn := c.conn
			c.lastPing = time.Now()
			c.mu.Unlock()

			if conn != nil {
				err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second))
				if err != nil {
					c.notifyError(fmt.Errorf("ping failed: %w", err))
				}
			}
		}
	}
}

// Send sends a message on this connection.
func (c *Connection) Send(ctx context.Context, data []byte) error {
	c.mu.RLock()
	conn := c.conn
	state := c.state
	c.mu.RUnlock()

	if state != interfaces.ConnectionStateConnected || conn == nil {
		return ErrConnectionNotConnected
	}

	// Set write deadline from context or config
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.config.WriteTimeout)
	}
	conn.SetWriteDeadline(deadline)

	err := conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		c.notifyError(fmt.Errorf("send failed: %w", err))
		return err
	}

	// Create sent message for notification
	msg := NewTextMessage(c.id, data, DirectionSent)
	c.notifyMessage(msg)

	return nil
}

// SendBinary sends a binary message on this connection.
func (c *Connection) SendBinary(ctx context.Context, data []byte) error {
	c.mu.RLock()
	conn := c.conn
	state := c.state
	c.mu.RUnlock()

	if state != interfaces.ConnectionStateConnected || conn == nil {
		return ErrConnectionNotConnected
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.config.WriteTimeout)
	}
	conn.SetWriteDeadline(deadline)

	err := conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		c.notifyError(fmt.Errorf("send binary failed: %w", err))
		return err
	}

	msg := NewBinaryMessage(c.id, data, DirectionSent)
	c.notifyMessage(msg)

	return nil
}

// Receive receives a message from this connection.
// This is a blocking call that waits for the next message.
func (c *Connection) Receive(ctx context.Context) ([]byte, error) {
	c.mu.RLock()
	state := c.state
	c.mu.RUnlock()

	if state != interfaces.ConnectionStateConnected {
		return nil, ErrConnectionNotConnected
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	case msg := <-c.msgChan:
		return msg.Data, nil
	case err := <-c.errChan:
		return nil, err
	}
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == interfaces.ConnectionStateDisconnected || c.state == interfaces.ConnectionStateDisconnecting {
		return nil
	}

	c.setState(interfaces.ConnectionStateDisconnecting)

	// Signal close to goroutines
	select {
	case <-c.closeChan:
		// Already closed
	default:
		close(c.closeChan)
	}

	if c.conn != nil {
		// Send close message
		c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		err := c.conn.Close()
		c.conn = nil
		c.setState(interfaces.ConnectionStateDisconnected)
		return err
	}

	c.setState(interfaces.ConnectionStateDisconnected)
	return nil
}

// handleDisconnect handles an unexpected disconnection.
func (c *Connection) handleDisconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == interfaces.ConnectionStateDisconnected {
		return
	}

	c.setState(interfaces.ConnectionStateDisconnected)

	select {
	case <-c.closeChan:
	default:
		close(c.closeChan)
	}

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// setState sets the connection state and notifies listeners.
// Must be called with mu held.
func (c *Connection) setState(state interfaces.ConnectionState) {
	if c.state == state {
		return
	}
	c.state = state

	if c.onStateChange != nil {
		go c.onStateChange(state)
	}
}

// notifyMessage notifies message callback.
func (c *Connection) notifyMessage(msg *Message) {
	c.mu.RLock()
	fn := c.onMessage
	c.mu.RUnlock()

	if fn != nil {
		fn(msg)
	}

	// Also push to channel for Receive()
	select {
	case c.msgChan <- msg:
	default:
		// Channel full, drop oldest
		select {
		case <-c.msgChan:
		default:
		}
		c.msgChan <- msg
	}
}

// notifyError notifies error callback.
func (c *Connection) notifyError(err error) {
	c.mu.RLock()
	fn := c.onError
	c.mu.RUnlock()

	if fn != nil {
		fn(err)
	}

	// Also push to error channel
	select {
	case c.errChan <- err:
	default:
	}
}

// LastPing returns the time of the last ping sent.
func (c *Connection) LastPing() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPing
}

// LastPong returns the time of the last pong received.
func (c *Connection) LastPong() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPong
}
