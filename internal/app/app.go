package app

import (
	"context"
	"fmt"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
)

// Requester is the interface for protocol adapters.
type Requester interface {
	Send(ctx context.Context, req *core.Request) (*core.Response, error)
	Protocol() string
}

// HookHandler is a function that handles a hook event.
type HookHandler func(ctx context.Context, data any) (any, error)

// Config holds application configuration.
type Config struct {
	Timeout         time.Duration
	DataDir         string
	FollowRedirects bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:         30 * time.Second,
		DataDir:         "~/.currier",
		FollowRedirects: true,
	}
}

// App is the main application container with dependency injection.
type App struct {
	config    Config
	protocols map[string]Requester
	hooks     map[string][]HookHandler
}

// Option is a function that configures the App.
type Option func(*App)

// New creates a new App with the given options.
func New(opts ...Option) *App {
	app := &App{
		config:    DefaultConfig(),
		protocols: make(map[string]Requester),
		hooks:     make(map[string][]HookHandler),
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// WithProtocol registers a protocol adapter.
func WithProtocol(name string, requester Requester) Option {
	return func(a *App) {
		a.protocols[name] = requester
	}
}

// WithConfig sets the application configuration.
func WithConfig(cfg Config) Option {
	return func(a *App) {
		a.config = cfg
	}
}

// Config returns the application configuration.
func (a *App) Config() Config {
	return a.config
}

// GetProtocol returns the requester for the given protocol.
func (a *App) GetProtocol(name string) (Requester, bool) {
	r, ok := a.protocols[name]
	return r, ok
}

// ListProtocols returns all registered protocol names.
func (a *App) ListProtocols() []string {
	protocols := make([]string, 0, len(a.protocols))
	for name := range a.protocols {
		protocols = append(protocols, name)
	}
	return protocols
}

// Send sends a request using the appropriate protocol adapter.
func (a *App) Send(ctx context.Context, req *core.Request) (*core.Response, error) {
	protocol := req.Protocol()
	requester, ok := a.protocols[protocol]
	if !ok {
		return nil, fmt.Errorf("protocol not registered: %s", protocol)
	}

	return requester.Send(ctx, req)
}

// RegisterHook registers a hook handler for the given hook name.
func (a *App) RegisterHook(hook string, handler HookHandler) {
	if a.hooks[hook] == nil {
		a.hooks[hook] = make([]HookHandler, 0)
	}
	a.hooks[hook] = append(a.hooks[hook], handler)
}

// GetHooks returns all handlers for the given hook.
func (a *App) GetHooks(hook string) []HookHandler {
	return a.hooks[hook]
}

// ExecuteHooks executes all handlers for the given hook in order.
func (a *App) ExecuteHooks(ctx context.Context, hook string, data any) (any, error) {
	handlers := a.hooks[hook]
	result := data

	for _, handler := range handlers {
		var err error
		result, err = handler(ctx, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Ensure App uses the hook constants from interfaces
var _ = interfaces.HookPreRequest
