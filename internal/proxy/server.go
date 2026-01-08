package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Server is an HTTP/HTTPS proxy server that captures traffic.
type Server struct {
	httpServer *http.Server
	tlsConfig  *TLSConfig
	store      *CaptureStore
	handler    *ProxyHandler
	config     Config
	running    bool
	listener   net.Listener
	mu         sync.RWMutex
}

// NewServer creates a new proxy server with the given configuration.
func NewServer(opts ...ConfigOption) (*Server, error) {
	config := NewConfig(opts...)

	// Create capture store
	store := NewCaptureStore(config.BufferSize)

	// Initialize TLS config if HTTPS is enabled
	var tlsConfig *TLSConfig
	if config.EnableHTTPS {
		var err error
		tlsConfig, err = NewTLSConfig(config.CACertPath, config.CAKeyPath, config.AutoGenerateCA)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize TLS: %w", err)
		}
	}

	// Create handler
	handler := NewProxyHandler(store, tlsConfig, config)

	return &Server{
		store:     store,
		tlsConfig: tlsConfig,
		handler:   handler,
		config:    config,
	}, nil
}

// Start starts the proxy server.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("proxy server is already running")
	}

	// Create listener
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}
	s.listener = listener

	// Create HTTP server
	s.httpServer = &http.Server{
		Handler:      s.handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.running = true
	s.mu.Unlock()

	// Start serving
	go func() {
		if err := s.httpServer.Serve(listener); err != http.ErrServerClosed {
			// Log error if not a normal shutdown
			if s.config.Verbose {
				fmt.Printf("Proxy server error: %v\n", err)
			}
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop stops the proxy server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		// Force close if graceful shutdown fails
		s.httpServer.Close()
	}

	s.running = false
	return nil
}

// IsRunning returns true if the proxy server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ListenAddr returns the actual address the server is listening on.
// Useful when using port 0 to get an available port.
func (s *Server) ListenAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.ListenAddr
}

// Store returns the capture store.
func (s *Server) Store() *CaptureStore {
	return s.store
}

// TLSConfig returns the TLS configuration (for CA export).
func (s *Server) TLSConfig() *TLSConfig {
	return s.tlsConfig
}

// Config returns the server configuration.
func (s *Server) Config() Config {
	return s.config
}

// AddListener adds a capture listener for real-time events.
func (s *Server) AddListener(listener CaptureListener) {
	s.store.AddListener(listener)
}

// RemoveListener removes a capture listener.
func (s *Server) RemoveListener(listener CaptureListener) {
	s.store.RemoveListener(listener)
}

// ClearCaptures clears all captured requests.
func (s *Server) ClearCaptures() {
	s.store.Clear()
}

// GetCaptures returns captured requests matching the filter.
func (s *Server) GetCaptures(opts FilterOptions) []*CapturedRequest {
	return s.store.List(opts)
}

// GetCapture returns a single capture by ID.
func (s *Server) GetCapture(id string) *CapturedRequest {
	return s.store.Get(id)
}

// Stats returns capture statistics.
func (s *Server) Stats() CaptureStats {
	return s.store.Stats()
}

// ExportCACert exports the CA certificate to a file.
func (s *Server) ExportCACert(path string) error {
	if s.tlsConfig == nil {
		return fmt.Errorf("HTTPS not enabled")
	}
	return s.tlsConfig.ExportCACert(path)
}

// CACertPEM returns the CA certificate in PEM format.
func (s *Server) CACertPEM() []byte {
	if s.tlsConfig == nil {
		return nil
	}
	return s.tlsConfig.CACertPEM()
}
