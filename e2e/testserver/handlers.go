package testserver

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handlers provides reusable response handlers.
type Handlers struct{}

// JSON returns a handler that responds with JSON.
func (Handlers) JSON(code int, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(data)
	}
}

// Text returns a handler that responds with plain text.
func (Handlers) Text(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

// Delayed returns a handler with simulated latency.
func (Handlers) Delayed(delay time.Duration, code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

// Echo returns a handler that echoes request details.
func (Handlers) Echo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.Query(),
			"headers": r.Header,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Error returns a handler that responds with an error.
func (Handlers) Error(code int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
}

// Status returns a handler that responds with just a status code.
func (Handlers) Status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}
}

// Headers returns a handler that responds with custom headers.
func (Handlers) Headers(code int, headers map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(code)
	}
}
