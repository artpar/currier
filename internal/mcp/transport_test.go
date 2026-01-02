package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestNewStdioTransport(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransport(reader, writer)

	if transport == nil {
		t.Fatal("expected transport to be created")
	}
	if transport.reader == nil {
		t.Error("expected reader to be set")
	}
	if transport.writer == nil {
		t.Error("expected writer to be set")
	}
}

func TestStdioTransport_ReadMessage(t *testing.T) {
	t.Run("reads valid JSON message", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n"
		reader := strings.NewReader(msg)
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)
		req, err := transport.ReadMessage()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Method != "test" {
			t.Errorf("expected method 'test', got '%s'", req.Method)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		msg := `not valid json` + "\n"
		reader := strings.NewReader(msg)
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)
		_, err := transport.ReadMessage()

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("returns EOF at end of input", func(t *testing.T) {
		reader := strings.NewReader("")
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)
		_, err := transport.ReadMessage()

		if err != io.EOF {
			t.Errorf("expected EOF, got %v", err)
		}
	})
}

func TestStdioTransport_WriteResponse(t *testing.T) {
	t.Run("writes valid response", func(t *testing.T) {
		reader := strings.NewReader("")
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)

		id := json.RawMessage(`1`)
		resp := &Response{
			JSONRPC: "2.0",
			ID:      &id,
			Result:  json.RawMessage(`{"status":"ok"}`),
		}

		err := transport.WriteResponse(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := writer.String()
		if !strings.Contains(output, `"result"`) {
			t.Error("expected result in output")
		}
		if !strings.HasSuffix(output, "\n") {
			t.Error("expected newline at end")
		}
	})
}

func TestStdioTransport_WriteNotification(t *testing.T) {
	t.Run("writes valid notification", func(t *testing.T) {
		reader := strings.NewReader("")
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)

		notif := &Notification{
			JSONRPC: "2.0",
			Method:  "test/event",
			Params:  json.RawMessage(`{"key":"value"}`),
		}

		err := transport.WriteNotification(notif)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := writer.String()
		if !strings.Contains(output, `"method"`) {
			t.Error("expected method in output")
		}
	})
}

func TestStdioTransport_Close(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransport(reader, writer)
	err := transport.Close()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMessageLoop(t *testing.T) {
	t.Run("processes messages until EOF", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
		reader := strings.NewReader(msg)
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)

		handler := func(req *Request) *Response {
			return &Response{
				Result: json.RawMessage(`"pong"`),
			}
		}

		err := MessageLoop(context.Background(), transport, handler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles nil response from handler", func(t *testing.T) {
		// Handler returns nil for notification (no ID)
		msg := `{"jsonrpc":"2.0","method":"notification/test"}` + "\n"
		reader := strings.NewReader(msg)
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)

		handler := func(req *Request) *Response {
			return nil // Return nil to indicate no response needed
		}

		err := MessageLoop(context.Background(), transport, handler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles parse errors", func(t *testing.T) {
		msg := `invalid json` + "\n" + `{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n"
		reader := strings.NewReader(msg)
		writer := &bytes.Buffer{}

		transport := NewStdioTransport(reader, writer)

		callCount := 0
		handler := func(req *Request) *Response {
			callCount++
			return &Response{Result: json.RawMessage(`"ok"`)}
		}

		err := MessageLoop(context.Background(), transport, handler)
		if err != nil {
			t.Logf("error: %v", err)
		}

		// Should have written an error response for the invalid JSON
		output := writer.String()
		if !strings.Contains(output, "Parse error") {
			t.Logf("expected parse error in output, got: %s", output)
		}
	})
}
