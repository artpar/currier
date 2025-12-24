package app

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRequester implements Requester interface for testing.
type MockRequester struct {
	protocol     string
	sendFunc     func(ctx context.Context, req *core.Request) (*core.Response, error)
	sendCalled   bool
	lastRequest  *core.Request
}

func NewMockRequester(protocol string) *MockRequester {
	return &MockRequester{
		protocol: protocol,
		sendFunc: func(ctx context.Context, req *core.Request) (*core.Response, error) {
			return core.NewResponse(req.ID(), protocol, core.NewStatus(200, "OK")), nil
		},
	}
}

func (m *MockRequester) Send(ctx context.Context, req *core.Request) (*core.Response, error) {
	m.sendCalled = true
	m.lastRequest = req
	return m.sendFunc(ctx, req)
}

func (m *MockRequester) Protocol() string {
	return m.protocol
}

func TestNewApp(t *testing.T) {
	t.Run("creates app with defaults", func(t *testing.T) {
		app := New()
		assert.NotNil(t, app)
	})

	t.Run("creates app with HTTP protocol", func(t *testing.T) {
		mock := NewMockRequester("http")
		app := New(WithProtocol("http", mock))

		requester, exists := app.GetProtocol("http")
		assert.True(t, exists)
		assert.Equal(t, "http", requester.Protocol())
	})

	t.Run("creates app with multiple protocols", func(t *testing.T) {
		httpMock := NewMockRequester("http")
		wsMock := NewMockRequester("websocket")

		app := New(
			WithProtocol("http", httpMock),
			WithProtocol("websocket", wsMock),
		)

		http, _ := app.GetProtocol("http")
		ws, _ := app.GetProtocol("websocket")

		assert.Equal(t, "http", http.Protocol())
		assert.Equal(t, "websocket", ws.Protocol())
	})

	t.Run("creates app with config", func(t *testing.T) {
		cfg := Config{
			Timeout: 30 * time.Second,
			DataDir: "/tmp/currier",
		}
		app := New(WithConfig(cfg))
		assert.Equal(t, cfg.Timeout, app.Config().Timeout)
		assert.Equal(t, cfg.DataDir, app.Config().DataDir)
	})
}

func TestApp_GetProtocol(t *testing.T) {
	t.Run("returns protocol if registered", func(t *testing.T) {
		mock := NewMockRequester("http")
		app := New(WithProtocol("http", mock))

		requester, exists := app.GetProtocol("http")
		assert.True(t, exists)
		assert.NotNil(t, requester)
	})

	t.Run("returns false for unregistered protocol", func(t *testing.T) {
		app := New()
		_, exists := app.GetProtocol("unknown")
		assert.False(t, exists)
	})
}

func TestApp_ListProtocols(t *testing.T) {
	t.Run("returns empty list for new app", func(t *testing.T) {
		app := New()
		protocols := app.ListProtocols()
		assert.Empty(t, protocols)
	})

	t.Run("returns all registered protocols", func(t *testing.T) {
		app := New(
			WithProtocol("http", NewMockRequester("http")),
			WithProtocol("websocket", NewMockRequester("websocket")),
			WithProtocol("grpc", NewMockRequester("grpc")),
		)

		protocols := app.ListProtocols()
		assert.Len(t, protocols, 3)
		assert.Contains(t, protocols, "http")
		assert.Contains(t, protocols, "websocket")
		assert.Contains(t, protocols, "grpc")
	})
}

func TestApp_Send(t *testing.T) {
	t.Run("sends request using registered protocol", func(t *testing.T) {
		mock := NewMockRequester("http")
		app := New(WithProtocol("http", mock))

		req, _ := core.NewRequest("http", "GET", "https://example.com")
		ctx := context.Background()

		resp, err := app.Send(ctx, req)

		require.NoError(t, err)
		assert.True(t, mock.sendCalled)
		assert.Equal(t, 200, resp.Status().Code())
	})

	t.Run("returns error for unregistered protocol", func(t *testing.T) {
		app := New()

		req, _ := core.NewRequest("http", "GET", "https://example.com")
		ctx := context.Background()

		_, err := app.Send(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "protocol")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		mock := NewMockRequester("http")
		mock.sendFunc = func(ctx context.Context, req *core.Request) (*core.Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return core.NewResponse(req.ID(), "http", core.NewStatus(200, "OK")), nil
			}
		}

		app := New(WithProtocol("http", mock))
		req, _ := core.NewRequest("http", "GET", "https://example.com")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := app.Send(ctx, req)
		assert.Error(t, err)
	})
}

func TestApp_Config(t *testing.T) {
	t.Run("returns default config", func(t *testing.T) {
		app := New()
		cfg := app.Config()
		assert.NotZero(t, cfg.Timeout)
	})

	t.Run("returns custom config", func(t *testing.T) {
		cfg := Config{
			Timeout:         60 * time.Second,
			DataDir:         "/custom/path",
			FollowRedirects: false,
		}
		app := New(WithConfig(cfg))

		assert.Equal(t, 60*time.Second, app.Config().Timeout)
		assert.Equal(t, "/custom/path", app.Config().DataDir)
		assert.False(t, app.Config().FollowRedirects)
	})
}

// Ensure MockRequester implements Requester
var _ Requester = (*MockRequester)(nil)

func TestApp_RegisterHook(t *testing.T) {
	t.Run("registers pre-request hook", func(t *testing.T) {
		app := New()

		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			return data, nil
		})

		// Hooks should be stored
		hooks := app.GetHooks(interfaces.HookPreRequest)
		assert.Len(t, hooks, 1)
	})

	t.Run("registers multiple hooks for same event", func(t *testing.T) {
		app := New()

		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			return data, nil
		})
		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			return data, nil
		})

		hooks := app.GetHooks(interfaces.HookPreRequest)
		assert.Len(t, hooks, 2)
	})
}

func TestApp_ExecuteHooks(t *testing.T) {
	t.Run("executes hooks in order", func(t *testing.T) {
		var order []int
		app := New()

		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			order = append(order, 1)
			return data, nil
		})
		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			order = append(order, 2)
			return data, nil
		})

		ctx := context.Background()
		_, err := app.ExecuteHooks(ctx, interfaces.HookPreRequest, nil)

		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, order)
	})

	t.Run("passes data through hook chain", func(t *testing.T) {
		app := New()

		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			n := data.(int)
			return n + 1, nil
		})
		app.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			n := data.(int)
			return n * 2, nil
		})

		ctx := context.Background()
		result, err := app.ExecuteHooks(ctx, interfaces.HookPreRequest, 5)

		require.NoError(t, err)
		assert.Equal(t, 12, result) // (5 + 1) * 2 = 12
	})
}
