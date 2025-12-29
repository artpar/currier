package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with defaults", func(t *testing.T) {
		client := NewClient()
		assert.NotNil(t, client)
		assert.Equal(t, "http", client.Protocol())
	})

	t.Run("creates client with custom timeout", func(t *testing.T) {
		client := NewClient(WithTimeout(5 * time.Second))
		assert.NotNil(t, client)
	})

	t.Run("creates client with custom transport", func(t *testing.T) {
		transport := &http.Transport{
			MaxIdleConns: 100,
		}
		client := NewClient(WithTransport(transport))
		assert.NotNil(t, client)
	})
}

func TestClient_Protocol(t *testing.T) {
	client := NewClient()
	assert.Equal(t, "http", client.Protocol())
}

func TestClient_Send_GET(t *testing.T) {
	t.Run("sends GET request and receives response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/users", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "John"})
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/users")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "application/json", resp.Headers().Get("Content-Type"))

		body, _ := resp.Body().JSON()
		m := body.(map[string]any)
		assert.Equal(t, "John", m["name"])
	})

	t.Run("sends GET request with headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/users")
		req.SetHeader("Authorization", "Bearer token123")
		req.SetHeader("Accept", "application/json")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
	})

	t.Run("sends GET request with query parameters in URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "admin", r.URL.Query().Get("role"))
			assert.Equal(t, "10", r.URL.Query().Get("limit"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/users?role=admin&limit=10")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
	})
}

func TestClient_Send_POST(t *testing.T) {
	t.Run("sends POST request with JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			var data map[string]string
			json.Unmarshal(body, &data)
			assert.Equal(t, "John", data["name"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "John"})
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "POST", server.URL+"/users")
		req.SetBody(core.NewJSONBody(map[string]string{"name": "John"}))
		req.SetHeader("Content-Type", "application/json")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 201, resp.Status().Code())
	})

	t.Run("sends POST request with raw body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, "raw text content", string(body))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "POST", server.URL+"/data")
		req.SetBody(core.NewRawBody([]byte("raw text content"), "text/plain"))

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
	})
}

func TestClient_Send_PUT(t *testing.T) {
	t.Run("sends PUT request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "PUT", server.URL+"/users/1")
		req.SetBody(core.NewJSONBody(map[string]string{"name": "Updated"}))

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
	})
}

func TestClient_Send_PATCH(t *testing.T) {
	t.Run("sends PATCH request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PATCH", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "PATCH", server.URL+"/users/1")
		req.SetBody(core.NewJSONBody(map[string]string{"name": "Patched"}))

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
	})
}

func TestClient_Send_DELETE(t *testing.T) {
	t.Run("sends DELETE request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "DELETE", server.URL+"/users/1")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 204, resp.Status().Code())
	})
}

func TestClient_Send_HEAD(t *testing.T) {
	t.Run("sends HEAD request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "HEAD", r.Method)
			w.Header().Set("X-Custom-Header", "custom-value")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "HEAD", server.URL+"/resource")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "custom-value", resp.Headers().Get("X-Custom-Header"))
	})
}

func TestClient_Send_OPTIONS(t *testing.T) {
	t.Run("sends OPTIONS request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "OPTIONS", r.Method)
			w.Header().Set("Allow", "GET, POST, PUT, DELETE")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "OPTIONS", server.URL+"/resource")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "GET, POST, PUT, DELETE", resp.Headers().Get("Allow"))
	})
}

func TestClient_Send_ErrorResponses(t *testing.T) {
	t.Run("handles 400 Bad Request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid input"})
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "POST", server.URL+"/users")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err) // HTTP errors are not Go errors
		assert.Equal(t, 400, resp.Status().Code())
		assert.True(t, resp.Status().IsError())
	})

	t.Run("handles 500 Internal Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/error")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 500, resp.Status().Code())
		assert.True(t, resp.Status().IsError())
	})

	t.Run("handles 404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/notfound")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 404, resp.Status().Code())
	})
}

func TestClient_Send_NetworkErrors(t *testing.T) {
	t.Run("returns error for invalid URL", func(t *testing.T) {
		client := NewClient()
		req, _ := core.NewRequest("http", "GET", "://invalid-url")

		ctx := context.Background()
		_, err := client.Send(ctx, req)

		assert.Error(t, err)
	})

	t.Run("returns error for connection refused", func(t *testing.T) {
		client := NewClient(WithTimeout(1 * time.Second))
		req, _ := core.NewRequest("http", "GET", "http://localhost:59999/nowhere")

		ctx := context.Background()
		_, err := client.Send(ctx, req)

		assert.Error(t, err)
	})
}

func TestClient_Send_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/slow")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := client.Send(ctx, req)
		assert.Error(t, err)
	})
}

func TestClient_Send_Timing(t *testing.T) {
	t.Run("records timing information", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/timed")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		timing := resp.Timing()
		assert.False(t, timing.StartTime.IsZero())
		assert.False(t, timing.EndTime.IsZero())
		assert.True(t, timing.Total >= 10*time.Millisecond)
	})
}

func TestClient_Send_ResponseMetadata(t *testing.T) {
	t.Run("includes request ID in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/test")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, req.ID(), resp.RequestID())
		assert.Equal(t, "http", resp.Protocol())
	})
}

func TestClient_Send_LargeResponse(t *testing.T) {
	t.Run("handles large response body", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte('a' + (i % 26))
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(largeData)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/large")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, int64(1024*1024), resp.Body().Size())
	})
}

func TestClient_Send_EmptyBody(t *testing.T) {
	t.Run("handles empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "DELETE", server.URL+"/empty")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 204, resp.Status().Code())
		assert.True(t, resp.Body().IsEmpty())
	})
}

func TestClient_Send_Redirect(t *testing.T) {
	t.Run("follows redirects by default", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/final", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("final destination"))
		}))
		defer server.Close()

		client := NewClient()
		req, _ := core.NewRequest("http", "GET", server.URL+"/redirect")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 200, resp.Status().Code())
		assert.Equal(t, "final destination", resp.Body().String())
	})

	t.Run("respects no redirect option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/final", http.StatusFound)
		}))
		defer server.Close()

		client := NewClient(WithNoRedirects())
		req, _ := core.NewRequest("http", "GET", server.URL+"/redirect")

		ctx := context.Background()
		resp, err := client.Send(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 302, resp.Status().Code())
	})
}

func TestClient_WithCookieJar(t *testing.T) {
	t.Run("uses cookie jar for requests", func(t *testing.T) {
		var receivedCookie string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/set-cookie" {
				http.SetCookie(w, &http.Cookie{
					Name:  "session",
					Value: "abc123",
					Path:  "/",
				})
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.URL.Path == "/check-cookie" {
				cookie, err := r.Cookie("session")
				if err == nil {
					receivedCookie = cookie.Value
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}))
		defer server.Close()

		jar, _ := cookiejar.New(nil)
		client := NewClient(WithCookieJar(jar))

		// Set cookie
		req1, _ := core.NewRequest("http", "GET", server.URL+"/set-cookie")
		ctx := context.Background()
		_, err := client.Send(ctx, req1)
		require.NoError(t, err)

		// Check cookie is sent
		req2, _ := core.NewRequest("http", "GET", server.URL+"/check-cookie")
		_, err = client.Send(ctx, req2)
		require.NoError(t, err)

		assert.Equal(t, "abc123", receivedCookie)
	})
}

func TestClient_WithProxy(t *testing.T) {
	t.Run("sets proxy configuration", func(t *testing.T) {
		client := NewClient(WithProxy("http://proxy.example.com:8080"))
		assert.NotNil(t, client)
		assert.Equal(t, "http://proxy.example.com:8080", client.config.ProxyURL)
	})

	t.Run("supports socks5 proxy", func(t *testing.T) {
		client := NewClient(WithProxy("socks5://proxy.example.com:1080"))
		assert.NotNil(t, client)
		assert.Equal(t, "socks5://proxy.example.com:1080", client.config.ProxyURL)
	})
}

func TestClient_WithClientCert(t *testing.T) {
	t.Run("sets client certificate configuration", func(t *testing.T) {
		client := NewClient(WithClientCert("/path/to/cert.pem", "/path/to/key.pem"))
		assert.NotNil(t, client)
		assert.NotNil(t, client.config.TLS)
		assert.Equal(t, "/path/to/cert.pem", client.config.TLS.CertFile)
		assert.Equal(t, "/path/to/key.pem", client.config.TLS.KeyFile)
	})

	t.Run("creates TLS config if nil", func(t *testing.T) {
		client := NewClient()
		assert.Nil(t, client.config.TLS)

		client = NewClient(WithClientCert("/cert.pem", "/key.pem"))
		assert.NotNil(t, client.config.TLS)
	})
}

func TestClient_WithCACert(t *testing.T) {
	t.Run("sets CA certificate configuration", func(t *testing.T) {
		client := NewClient(WithCACert("/path/to/ca.pem"))
		assert.NotNil(t, client)
		assert.NotNil(t, client.config.TLS)
		assert.Equal(t, "/path/to/ca.pem", client.config.TLS.CAFile)
	})

	t.Run("creates TLS config if nil", func(t *testing.T) {
		client := NewClient()
		assert.Nil(t, client.config.TLS)

		client = NewClient(WithCACert("/ca.pem"))
		assert.NotNil(t, client.config.TLS)
	})
}

func TestClient_WithInsecureSkipVerify(t *testing.T) {
	t.Run("sets insecure skip verify", func(t *testing.T) {
		client := NewClient(WithInsecureSkipVerify())
		assert.NotNil(t, client)
		assert.NotNil(t, client.config.TLS)
		assert.True(t, client.config.TLS.InsecureSkipVerify)
	})

	t.Run("creates TLS config if nil", func(t *testing.T) {
		client := NewClient()
		assert.Nil(t, client.config.TLS)

		client = NewClient(WithInsecureSkipVerify())
		assert.NotNil(t, client.config.TLS)
		assert.True(t, client.config.TLS.InsecureSkipVerify)
	})
}

func TestClient_buildTLSConfig(t *testing.T) {
	t.Run("returns nil when no TLS config", func(t *testing.T) {
		client := NewClient()
		tlsConfig := client.buildTLSConfig()
		assert.Nil(t, tlsConfig)
	})

	t.Run("sets InsecureSkipVerify", func(t *testing.T) {
		client := NewClient(WithInsecureSkipVerify())
		tlsConfig := client.buildTLSConfig()
		assert.NotNil(t, tlsConfig)
		assert.True(t, tlsConfig.InsecureSkipVerify)
	})

	t.Run("handles missing certificate files gracefully", func(t *testing.T) {
		client := NewClient(WithClientCert("/nonexistent/cert.pem", "/nonexistent/key.pem"))
		tlsConfig := client.buildTLSConfig()
		assert.NotNil(t, tlsConfig)
		// Certificate loading failed silently, so no certificates
		assert.Empty(t, tlsConfig.Certificates)
	})

	t.Run("handles missing CA file gracefully", func(t *testing.T) {
		client := NewClient(WithCACert("/nonexistent/ca.pem"))
		tlsConfig := client.buildTLSConfig()
		assert.NotNil(t, tlsConfig)
		// CA loading failed silently, so no custom CA
		assert.Nil(t, tlsConfig.RootCAs)
	})
}

func TestClient_CombinedTLSOptions(t *testing.T) {
	t.Run("combines multiple TLS options", func(t *testing.T) {
		client := NewClient(
			WithInsecureSkipVerify(),
			WithCACert("/path/to/ca.pem"),
			WithClientCert("/path/to/cert.pem", "/path/to/key.pem"),
		)

		assert.NotNil(t, client.config.TLS)
		assert.True(t, client.config.TLS.InsecureSkipVerify)
		assert.Equal(t, "/path/to/ca.pem", client.config.TLS.CAFile)
		assert.Equal(t, "/path/to/cert.pem", client.config.TLS.CertFile)
		assert.Equal(t, "/path/to/key.pem", client.config.TLS.KeyFile)
	})
}
