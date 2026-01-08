package proxy

import (
	"os"
	"path/filepath"
)

// Config holds the proxy server configuration.
type Config struct {
	// ListenAddr is the address to listen on (e.g., ":8080", "127.0.0.1:8080")
	ListenAddr string

	// EnableHTTPS enables HTTPS interception via MITM
	EnableHTTPS bool

	// CACertPath is the path to the CA certificate for MITM
	CACertPath string

	// CAKeyPath is the path to the CA private key for MITM
	CAKeyPath string

	// AutoGenerateCA generates CA certificate if not found
	AutoGenerateCA bool

	// MaxBodySize is the maximum request/response body size to capture (bytes)
	// Bodies larger than this will be truncated
	MaxBodySize int64

	// BufferSize is the maximum number of captures to keep in memory
	BufferSize int

	// ExcludeHosts is a list of hosts to exclude from capture
	ExcludeHosts []string

	// IncludeHosts is a list of hosts to include (if empty, all hosts are included)
	IncludeHosts []string

	// ExcludeContentTypes is a list of content-types to exclude from body capture
	ExcludeContentTypes []string

	// Verbose enables verbose logging
	Verbose bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	configDir, _ := os.UserConfigDir()
	proxyDir := filepath.Join(configDir, "currier", "proxy")

	return Config{
		ListenAddr:     ":8080",
		EnableHTTPS:    true,
		CACertPath:     filepath.Join(proxyDir, "ca.crt"),
		CAKeyPath:      filepath.Join(proxyDir, "ca.key"),
		AutoGenerateCA: true,
		MaxBodySize:    10 * 1024 * 1024, // 10MB
		BufferSize:     1000,             // Keep last 1000 captures
		ExcludeContentTypes: []string{
			"image/",
			"video/",
			"audio/",
			"font/",
		},
	}
}

// ConfigOption is a function that modifies the Config.
type ConfigOption func(*Config)

// WithListenAddr sets the listen address.
func WithListenAddr(addr string) ConfigOption {
	return func(c *Config) {
		c.ListenAddr = addr
	}
}

// WithHTTPS enables or disables HTTPS interception.
func WithHTTPS(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableHTTPS = enable
	}
}

// WithCACert sets the CA certificate and key paths.
func WithCACert(certPath, keyPath string) ConfigOption {
	return func(c *Config) {
		c.CACertPath = certPath
		c.CAKeyPath = keyPath
	}
}

// WithAutoGenerateCA enables or disables auto CA generation.
func WithAutoGenerateCA(auto bool) ConfigOption {
	return func(c *Config) {
		c.AutoGenerateCA = auto
	}
}

// WithMaxBodySize sets the maximum body size to capture.
func WithMaxBodySize(size int64) ConfigOption {
	return func(c *Config) {
		c.MaxBodySize = size
	}
}

// WithBufferSize sets the capture buffer size.
func WithBufferSize(size int) ConfigOption {
	return func(c *Config) {
		c.BufferSize = size
	}
}

// WithExcludeHosts sets hosts to exclude from capture.
func WithExcludeHosts(hosts ...string) ConfigOption {
	return func(c *Config) {
		c.ExcludeHosts = hosts
	}
}

// WithIncludeHosts sets hosts to include in capture (whitelist mode).
func WithIncludeHosts(hosts ...string) ConfigOption {
	return func(c *Config) {
		c.IncludeHosts = hosts
	}
}

// WithVerbose enables verbose logging.
func WithVerbose(verbose bool) ConfigOption {
	return func(c *Config) {
		c.Verbose = verbose
	}
}

// NewConfig creates a new Config with the given options applied to defaults.
func NewConfig(opts ...ConfigOption) Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
