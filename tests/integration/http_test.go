package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/artpar/currier/internal/app"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/interfaces"
	httpclient "github.com/artpar/currier/internal/protocol/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPIntegration tests the full flow from App -> HTTP Client -> Server
func TestHTTPIntegration(t *testing.T) {
	t.Run("full GET request flow", func(t *testing.T) {
		// Setup test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/users", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", "test-123")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"users": []map[string]any{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
			})
		}))
		defer server.Close()

		// Create app with real HTTP client
		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		// Create and send request
		req, err := core.NewRequest("http", "GET", server.URL+"/api/users")
		require.NoError(t, err)
		req.SetHeader("Accept", "application/json")

		ctx := context.Background()
		resp, err := application.Send(ctx, req)

		// Verify response
		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "application/json", resp.Headers().Get("Content-Type"))
		assert.Equal(t, "test-123", resp.Headers().Get("X-Request-ID"))

		// Verify body
		body, err := resp.Body().JSON()
		require.NoError(t, err)
		data := body.(map[string]any)
		users := data["users"].([]any)
		assert.Len(t, users, 2)
	})

	t.Run("full POST request flow with JSON body", func(t *testing.T) {
		var receivedBody map[string]any

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			bodyBytes, _ := io.ReadAll(r.Body)
			json.Unmarshal(bodyBytes, &receivedBody)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"id":      123,
				"name":    receivedBody["name"],
				"created": true,
			})
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "POST", server.URL+"/api/users")
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(core.NewJSONBody(map[string]any{
			"name":  "Charlie",
			"email": "charlie@example.com",
		}))

		ctx := context.Background()
		resp, err := application.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 201, resp.Status().Code())
		assert.Equal(t, "Charlie", receivedBody["name"])

		body, _ := resp.Body().JSON()
		data := body.(map[string]any)
		assert.Equal(t, float64(123), data["id"])
		assert.Equal(t, true, data["created"])
	})

	t.Run("request with all HTTP methods", func(t *testing.T) {
		methods := []struct {
			method         string
			expectedStatus int
		}{
			{"GET", http.StatusOK},
			{"POST", http.StatusCreated},
			{"PUT", http.StatusOK},
			{"PATCH", http.StatusOK},
			{"DELETE", http.StatusNoContent},
			{"HEAD", http.StatusOK},
			{"OPTIONS", http.StatusOK},
		}

		for _, tc := range methods {
			t.Run(tc.method, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, tc.method, r.Method)
					w.WriteHeader(tc.expectedStatus)
				}))
				defer server.Close()

				application := app.New(
					app.WithProtocol("http", httpclient.NewClient()),
				)

				req, _ := core.NewRequest("http", tc.method, server.URL+"/resource")
				resp, err := application.Send(context.Background(), req)

				require.NoError(t, err)
				assert.Equal(t, tc.expectedStatus, resp.Status().Code())
			})
		}
	})
}

// TestHTTPIntegration_Hooks tests hook execution in request flow
func TestHTTPIntegration_Hooks(t *testing.T) {
	t.Run("pre-request hook modifies request", func(t *testing.T) {
		var receivedAuth string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		// Register hook that adds auth header
		application.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			req := data.(*core.Request)
			req.SetHeader("Authorization", "Bearer injected-token")
			return req, nil
		})

		req, _ := core.NewRequest("http", "GET", server.URL+"/secure")

		// Execute pre-request hooks manually (simulating full flow)
		ctx := context.Background()
		hookResult, err := application.ExecuteHooks(ctx, interfaces.HookPreRequest, req)
		require.NoError(t, err)
		modifiedReq := hookResult.(*core.Request)

		resp, err := application.Send(ctx, modifiedReq)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "Bearer injected-token", receivedAuth)
	})

	t.Run("hook chain executes in order", func(t *testing.T) {
		var executionOrder []string

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		application.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			executionOrder = append(executionOrder, "hook1")
			return data, nil
		})

		application.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			executionOrder = append(executionOrder, "hook2")
			return data, nil
		})

		application.RegisterHook(interfaces.HookPreRequest, func(ctx context.Context, data any) (any, error) {
			executionOrder = append(executionOrder, "hook3")
			return data, nil
		})

		req, _ := core.NewRequest("http", "GET", "http://example.com")
		_, err := application.ExecuteHooks(context.Background(), interfaces.HookPreRequest, req)

		require.NoError(t, err)
		assert.Equal(t, []string{"hook1", "hook2", "hook3"}, executionOrder)
	})
}

// TestHTTPIntegration_ErrorHandling tests error scenarios
func TestHTTPIntegration_ErrorHandling(t *testing.T) {
	t.Run("handles server error responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "internal server error",
			})
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/error")
		resp, err := application.Send(context.Background(), req)

		// HTTP errors are not Go errors
		require.NoError(t, err)
		assert.Equal(t, 500, resp.Status().Code())
		assert.True(t, resp.Status().IsError())

		body, _ := resp.Body().JSON()
		data := body.(map[string]any)
		assert.Equal(t, "internal server error", data["error"])
	})

	t.Run("handles connection timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient(
				httpclient.WithTimeout(100*time.Millisecond),
			)),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/slow")
		_, err := application.Send(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/slow")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := application.Send(ctx, req)
		assert.Error(t, err)
	})

	t.Run("handles unregistered protocol", func(t *testing.T) {
		application := app.New() // No protocols registered

		req, _ := core.NewRequest("http", "GET", "http://example.com")
		_, err := application.Send(context.Background(), req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "protocol")
	})
}

// TestHTTPIntegration_Timing tests timing information
func TestHTTPIntegration_Timing(t *testing.T) {
	t.Run("captures timing information", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("response"))
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/timed")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)

		timing := resp.Timing()
		assert.False(t, timing.StartTime.IsZero())
		assert.False(t, timing.EndTime.IsZero())
		assert.True(t, timing.Total >= 50*time.Millisecond)
		assert.True(t, timing.EndTime.After(timing.StartTime))
	})
}

// TestHTTPIntegration_Headers tests header handling
func TestHTTPIntegration_Headers(t *testing.T) {
	t.Run("sends and receives multiple headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify received headers
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

			// Send response headers
			w.Header().Set("X-Response-ID", "resp-456")
			w.Header().Set("X-Rate-Limit", "100")
			w.Header().Add("Set-Cookie", "session=abc")
			w.Header().Add("Set-Cookie", "token=xyz")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "POST", server.URL+"/headers")
		req.SetHeader("Content-Type", "application/json")
		req.SetHeader("Authorization", "Bearer token123")
		req.SetHeader("X-Custom-Header", "custom-value")

		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "resp-456", resp.Headers().Get("X-Response-ID"))
		assert.Equal(t, "100", resp.Headers().Get("X-Rate-Limit"))

		// Multiple values for same header
		cookies := resp.Headers().GetAll("Set-Cookie")
		assert.Len(t, cookies, 2)
	})

	t.Run("header case insensitivity", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL)
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)

		// Should work with any case
		assert.Equal(t, "application/json", resp.Headers().Get("content-type"))
		assert.Equal(t, "application/json", resp.Headers().Get("Content-Type"))
		assert.Equal(t, "application/json", resp.Headers().Get("CONTENT-TYPE"))
	})
}

// TestHTTPIntegration_Body tests body handling
func TestHTTPIntegration_Body(t *testing.T) {
	t.Run("handles large response body", func(t *testing.T) {
		largeData := strings.Repeat("x", 1024*1024) // 1MB

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(largeData))
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/large")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, int64(1024*1024), resp.Body().Size())
		assert.Equal(t, largeData, resp.Body().String())
	})

	t.Run("handles empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "DELETE", server.URL+"/resource")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 204, resp.Status().Code())
		assert.True(t, resp.Body().IsEmpty())
	})

	t.Run("handles JSON body parsing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"nested": map[string]any{
					"array": []int{1, 2, 3},
					"object": map[string]string{
						"key": "value",
					},
				},
			})
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/json")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)

		body, err := resp.Body().JSON()
		require.NoError(t, err)

		data := body.(map[string]any)
		nested := data["nested"].(map[string]any)
		array := nested["array"].([]any)
		assert.Len(t, array, 3)
	})
}

// TestHTTPIntegration_Redirects tests redirect handling
func TestHTTPIntegration_Redirects(t *testing.T) {
	t.Run("follows redirects by default", func(t *testing.T) {
		redirectCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/start" {
				redirectCount++
				http.Redirect(w, r, "/middle", http.StatusFound)
				return
			}
			if r.URL.Path == "/middle" {
				redirectCount++
				http.Redirect(w, r, "/end", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("final destination"))
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient()),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/start")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "final destination", resp.Body().String())
		assert.Equal(t, 2, redirectCount)
	})

	t.Run("respects no redirect option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/other", http.StatusFound)
		}))
		defer server.Close()

		application := app.New(
			app.WithProtocol("http", httpclient.NewClient(httpclient.WithNoRedirects())),
		)

		req, _ := core.NewRequest("http", "GET", server.URL+"/redirect")
		resp, err := application.Send(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, 302, resp.Status().Code())
	})
}
