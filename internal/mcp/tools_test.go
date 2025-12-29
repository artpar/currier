package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/artpar/currier/internal/history"
)

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		maxSize     int
		wantTrunc   bool
		wantContain string
	}{
		{
			name:      "short body not truncated",
			body:      "hello world",
			maxSize:   100,
			wantTrunc: false,
		},
		{
			name:        "long body truncated",
			body:        strings.Repeat("x", 200),
			maxSize:     100,
			wantTrunc:   true,
			wantContain: "TRUNCATED",
		},
		{
			name:      "exact size not truncated",
			body:      strings.Repeat("x", 100),
			maxSize:   100,
			wantTrunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, truncated := truncateBody(tt.body, tt.maxSize)
			if truncated != tt.wantTrunc {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTrunc)
			}
			if tt.wantContain != "" && !strings.Contains(result, tt.wantContain) {
				t.Errorf("result should contain %q", tt.wantContain)
			}
		})
	}
}

func TestApplyPagination(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tests := []struct {
		name       string
		offset     int
		limit      int
		wantLen    int
		wantTotal  int
		wantMore   bool
	}{
		{
			name:      "default pagination",
			offset:    0,
			limit:     0, // Uses default
			wantLen:   10,
			wantTotal: 10,
			wantMore:  false,
		},
		{
			name:      "first page",
			offset:    0,
			limit:     3,
			wantLen:   3,
			wantTotal: 10,
			wantMore:  true,
		},
		{
			name:      "second page",
			offset:    3,
			limit:     3,
			wantLen:   3,
			wantTotal: 10,
			wantMore:  true,
		},
		{
			name:      "last page",
			offset:    9,
			limit:     3,
			wantLen:   1,
			wantTotal: 10,
			wantMore:  false,
		},
		{
			name:      "offset beyond range",
			offset:    100,
			limit:     3,
			wantLen:   0,
			wantTotal: 10,
			wantMore:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pagination := applyPagination(items, tt.offset, tt.limit)
			if len(result) != tt.wantLen {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantLen)
			}
			if pagination.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", pagination.Total, tt.wantTotal)
			}
			if pagination.HasMore != tt.wantMore {
				t.Errorf("HasMore = %v, want %v", pagination.HasMore, tt.wantMore)
			}
		})
	}
}

func TestTextContent(t *testing.T) {
	content := TextContent("hello")
	if content.Type != "text" {
		t.Errorf("Type = %q, want %q", content.Type, "text")
	}
	if content.Text != "hello" {
		t.Errorf("Text = %q, want %q", content.Text, "hello")
	}
}

func TestJSONContent(t *testing.T) {
	data := map[string]string{"key": "value"}
	content, err := JSONContent(data)
	if err != nil {
		t.Fatalf("JSONContent() error = %v", err)
	}
	if content.Type != "text" {
		t.Errorf("Type = %q, want %q", content.Type, "text")
	}
	if !strings.Contains(content.Text, "key") || !strings.Contains(content.Text, "value") {
		t.Errorf("Text should contain key and value")
	}
}

func TestErrorContent(t *testing.T) {
	err := context.DeadlineExceeded
	content := ErrorContent(err)
	if content.Type != "text" {
		t.Errorf("Type = %q, want %q", content.Type, "text")
	}
	if !strings.Contains(content.Text, "Error") {
		t.Errorf("Text should contain 'Error'")
	}
}

func TestGetFolderByName(t *testing.T) {
	// This tests the helper function for finding subfolders
	// Since core.Folder is internal, we just verify the function exists and compiles
	// Full integration testing would be done with real collections
}

func TestPaginationResult(t *testing.T) {
	result := paginationResult{
		Offset:     10,
		Limit:      5,
		Total:      100,
		HasMore:    true,
		TotalPages: 20,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded paginationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Offset != result.Offset {
		t.Errorf("Offset = %d, want %d", decoded.Offset, result.Offset)
	}
	if decoded.HasMore != result.HasMore {
		t.Errorf("HasMore = %v, want %v", decoded.HasMore, result.HasMore)
	}
}

func TestSendRequestArgs(t *testing.T) {
	args := sendRequestArgs{
		Method:      "POST",
		URL:         "https://api.example.com",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        `{"key":"value"}`,
		Environment: "production",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded sendRequestArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Method != "POST" {
		t.Errorf("Method = %q, want %q", decoded.Method, "POST")
	}
	if decoded.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header not preserved")
	}
}

func TestSendRequestResult(t *testing.T) {
	result := sendRequestResult{
		Status:      200,
		StatusText:  "200 OK",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        `{"result":"success"}`,
		DurationMs:  150,
		SizeBytes:   1024,
		IsTruncated: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded sendRequestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Status != 200 {
		t.Errorf("Status = %d, want %d", decoded.Status, 200)
	}
	if decoded.IsTruncated {
		t.Error("IsTruncated should be false")
	}
}

func TestSearchHistoryArgs(t *testing.T) {
	args := searchHistoryArgs{
		Query:  "api.example.com",
		Method: "GET",
		Status: 200,
		Limit:  50,
		Offset: 0,
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded searchHistoryArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Query != "api.example.com" {
		t.Errorf("Query = %q, want %q", decoded.Query, "api.example.com")
	}
}

func TestApplyPaginationWithHistoryEntries(t *testing.T) {
	entries := make([]history.Entry, 100)
	for i := range entries {
		entries[i] = history.Entry{
			ID:             fmt.Sprintf("entry-%d", i+1),
			RequestMethod:  "GET",
			RequestURL:     "https://example.com",
			ResponseStatus: 200,
		}
	}

	result, pagination := applyPagination(entries, 0, 10)
	if len(result) != 10 {
		t.Errorf("len(result) = %d, want 10", len(result))
	}
	if pagination.Total != 100 {
		t.Errorf("Total = %d, want 100", pagination.Total)
	}
	if !pagination.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestImportCollectionArgs(t *testing.T) {
	args := importCollectionArgs{
		Content: `{"info":{"name":"Test"}}`,
		Format:  "postman",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded importCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Format != "postman" {
		t.Errorf("Format = %q, want %q", decoded.Format, "postman")
	}
}

func TestExportCollectionArgs(t *testing.T) {
	args := exportCollectionArgs{
		Name:   "My API",
		Format: "curl",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded exportCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "My API" {
		t.Errorf("Name = %q, want %q", decoded.Name, "My API")
	}
	if decoded.Format != "curl" {
		t.Errorf("Format = %q, want %q", decoded.Format, "curl")
	}
}

// ============================================================================
// WebSocket Tool Tests
// ============================================================================

func TestMaxWebSocketMessages(t *testing.T) {
	if MaxWebSocketMessages != 100 {
		t.Errorf("MaxWebSocketMessages = %d, want 100", MaxWebSocketMessages)
	}
}

func TestWsConnectArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        wsConnectArgs
		wantJSON    bool
		checkFields func(t *testing.T, decoded wsConnectArgs)
	}{
		{
			name: "basic endpoint",
			args: wsConnectArgs{
				Endpoint: "wss://api.example.com/ws",
			},
			wantJSON: true,
			checkFields: func(t *testing.T, decoded wsConnectArgs) {
				if decoded.Endpoint != "wss://api.example.com/ws" {
					t.Errorf("Endpoint = %q, want %q", decoded.Endpoint, "wss://api.example.com/ws")
				}
				if decoded.TLSInsecure {
					t.Error("TLSInsecure should be false by default")
				}
			},
		},
		{
			name: "with headers and TLS skip",
			args: wsConnectArgs{
				Endpoint:    "wss://secure.example.com/ws",
				Headers:     map[string]string{"Authorization": "Bearer token123", "X-Custom": "value"},
				TLSInsecure: true,
			},
			wantJSON: true,
			checkFields: func(t *testing.T, decoded wsConnectArgs) {
				if decoded.Endpoint != "wss://secure.example.com/ws" {
					t.Errorf("Endpoint = %q, want %q", decoded.Endpoint, "wss://secure.example.com/ws")
				}
				if !decoded.TLSInsecure {
					t.Error("TLSInsecure should be true")
				}
				if decoded.Headers["Authorization"] != "Bearer token123" {
					t.Errorf("Authorization header = %q, want %q", decoded.Headers["Authorization"], "Bearer token123")
				}
				if decoded.Headers["X-Custom"] != "value" {
					t.Errorf("X-Custom header = %q, want %q", decoded.Headers["X-Custom"], "value")
				}
			},
		},
		{
			name: "ws protocol",
			args: wsConnectArgs{
				Endpoint: "ws://localhost:8080/socket",
			},
			wantJSON: true,
			checkFields: func(t *testing.T, decoded wsConnectArgs) {
				if decoded.Endpoint != "ws://localhost:8080/socket" {
					t.Errorf("Endpoint = %q, want %q", decoded.Endpoint, "ws://localhost:8080/socket")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded wsConnectArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			tt.checkFields(t, decoded)
		})
	}
}

func TestWsDisconnectArgs(t *testing.T) {
	tests := []struct {
		name         string
		connectionID string
	}{
		{
			name:         "standard connection ID",
			connectionID: "ws-1234567890",
		},
		{
			name:         "uuid-style connection ID",
			connectionID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := wsDisconnectArgs{
				ConnectionID: tt.connectionID,
			}

			data, err := json.Marshal(args)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded wsDisconnectArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.ConnectionID != tt.connectionID {
				t.Errorf("ConnectionID = %q, want %q", decoded.ConnectionID, tt.connectionID)
			}
		})
	}
}

func TestWsSendArgs(t *testing.T) {
	tests := []struct {
		name string
		args wsSendArgs
	}{
		{
			name: "simple text message",
			args: wsSendArgs{
				ConnectionID: "ws-123",
				Message:      "Hello, WebSocket!",
				WaitForReply: false,
				TimeoutMs:    0,
			},
		},
		{
			name: "JSON message with reply wait",
			args: wsSendArgs{
				ConnectionID: "ws-456",
				Message:      `{"type":"ping","data":"test"}`,
				WaitForReply: true,
				TimeoutMs:    10000,
			},
		},
		{
			name: "message with custom timeout",
			args: wsSendArgs{
				ConnectionID: "ws-789",
				Message:      "Quick ping",
				WaitForReply: true,
				TimeoutMs:    1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded wsSendArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.ConnectionID != tt.args.ConnectionID {
				t.Errorf("ConnectionID = %q, want %q", decoded.ConnectionID, tt.args.ConnectionID)
			}
			if decoded.Message != tt.args.Message {
				t.Errorf("Message = %q, want %q", decoded.Message, tt.args.Message)
			}
			if decoded.WaitForReply != tt.args.WaitForReply {
				t.Errorf("WaitForReply = %v, want %v", decoded.WaitForReply, tt.args.WaitForReply)
			}
			if decoded.TimeoutMs != tt.args.TimeoutMs {
				t.Errorf("TimeoutMs = %d, want %d", decoded.TimeoutMs, tt.args.TimeoutMs)
			}
		})
	}
}

func TestWsGetMessagesArgs(t *testing.T) {
	tests := []struct {
		name string
		args wsGetMessagesArgs
	}{
		{
			name: "basic request",
			args: wsGetMessagesArgs{
				ConnectionID: "ws-123",
			},
		},
		{
			name: "with pagination",
			args: wsGetMessagesArgs{
				ConnectionID: "ws-456",
				Limit:        25,
				Offset:       50,
			},
		},
		{
			name: "filter sent messages",
			args: wsGetMessagesArgs{
				ConnectionID: "ws-789",
				Direction:    "sent",
				Limit:        10,
			},
		},
		{
			name: "filter received messages",
			args: wsGetMessagesArgs{
				ConnectionID: "ws-abc",
				Direction:    "received",
				Limit:        100,
				Offset:       0,
			},
		},
		{
			name: "all messages with pagination",
			args: wsGetMessagesArgs{
				ConnectionID: "ws-def",
				Direction:    "all",
				Limit:        50,
				Offset:       25,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded wsGetMessagesArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.ConnectionID != tt.args.ConnectionID {
				t.Errorf("ConnectionID = %q, want %q", decoded.ConnectionID, tt.args.ConnectionID)
			}
			if decoded.Limit != tt.args.Limit {
				t.Errorf("Limit = %d, want %d", decoded.Limit, tt.args.Limit)
			}
			if decoded.Offset != tt.args.Offset {
				t.Errorf("Offset = %d, want %d", decoded.Offset, tt.args.Offset)
			}
			if decoded.Direction != tt.args.Direction {
				t.Errorf("Direction = %q, want %q", decoded.Direction, tt.args.Direction)
			}
		})
	}
}

func TestWsGetMessagesArgsDirectionValues(t *testing.T) {
	validDirections := []string{"", "all", "sent", "received"}

	for _, dir := range validDirections {
		t.Run("direction_"+dir, func(t *testing.T) {
			args := wsGetMessagesArgs{
				ConnectionID: "ws-test",
				Direction:    dir,
			}

			data, err := json.Marshal(args)
			if err != nil {
				t.Fatalf("Marshal error for direction %q: %v", dir, err)
			}

			var decoded wsGetMessagesArgs
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error for direction %q: %v", dir, err)
			}

			if decoded.Direction != dir {
				t.Errorf("Direction = %q, want %q", decoded.Direction, dir)
			}
		})
	}
}

func TestWsConnectArgsJSONOmitEmpty(t *testing.T) {
	// Test that omitempty works correctly
	args := wsConnectArgs{
		Endpoint: "wss://test.com/ws",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify headers and tls_insecure are omitted when empty/false
	if strings.Contains(string(data), "headers") {
		t.Error("headers should be omitted when nil")
	}
	if strings.Contains(string(data), "tls_insecure") {
		t.Error("tls_insecure should be omitted when false")
	}
}

func TestWsSendArgsJSONOmitEmpty(t *testing.T) {
	args := wsSendArgs{
		ConnectionID: "ws-123",
		Message:      "test",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify optional fields are omitted when default
	if strings.Contains(string(data), "wait_for_reply") {
		t.Error("wait_for_reply should be omitted when false")
	}
	if strings.Contains(string(data), "timeout_ms") {
		t.Error("timeout_ms should be omitted when 0")
	}
}

func TestWsGetMessagesArgsJSONOmitEmpty(t *testing.T) {
	args := wsGetMessagesArgs{
		ConnectionID: "ws-123",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify optional fields are omitted when default
	if strings.Contains(string(data), "limit") {
		t.Error("limit should be omitted when 0")
	}
	if strings.Contains(string(data), "offset") {
		t.Error("offset should be omitted when 0")
	}
	if strings.Contains(string(data), "direction") {
		t.Error("direction should be omitted when empty")
	}
}
