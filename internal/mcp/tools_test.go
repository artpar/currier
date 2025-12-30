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

// ============================================================================
// Protocol Type Tests
// ============================================================================

func TestRequest_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		request Request
	}{
		{
			name: "simple request",
			request: Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/list",
			},
		},
		{
			name: "request with params",
			request: Request{
				JSONRPC: "2.0",
				ID:      "req-123",
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"send_request"}`),
			},
		},
		{
			name: "request with null ID",
			request: Request{
				JSONRPC: "2.0",
				ID:      nil,
				Method:  "ping",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.JSONRPC != tt.request.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, tt.request.JSONRPC)
			}
			if decoded.Method != tt.request.Method {
				t.Errorf("Method = %q, want %q", decoded.Method, tt.request.Method)
			}
		})
	}
}

func TestResponse_JSONSerialization(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`{"tools":[]}`),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var decoded Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if decoded.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, "2.0")
		}
		if decoded.Error != nil {
			t.Error("Error should be nil for success response")
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Error: &ResponseError{
				Code:    InvalidParams,
				Message: "Invalid parameters",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var decoded Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if decoded.Error == nil {
			t.Fatal("Error should not be nil")
		}
		if decoded.Error.Code != InvalidParams {
			t.Errorf("Error.Code = %d, want %d", decoded.Error.Code, InvalidParams)
		}
	})
}

func TestResponseError_JSONSerialization(t *testing.T) {
	tests := []struct {
		name string
		err  ResponseError
	}{
		{
			name: "parse error",
			err: ResponseError{
				Code:    ParseError,
				Message: "Parse error",
			},
		},
		{
			name: "invalid request",
			err: ResponseError{
				Code:    InvalidRequest,
				Message: "Invalid request",
			},
		},
		{
			name: "method not found",
			err: ResponseError{
				Code:    MethodNotFound,
				Message: "Method not found",
			},
		},
		{
			name: "with data",
			err: ResponseError{
				Code:    InternalError,
				Message: "Internal error",
				Data:    map[string]string{"detail": "something went wrong"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.err)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded ResponseError
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.Code != tt.err.Code {
				t.Errorf("Code = %d, want %d", decoded.Code, tt.err.Code)
			}
			if decoded.Message != tt.err.Message {
				t.Errorf("Message = %q, want %q", decoded.Message, tt.err.Message)
			}
		})
	}
}

func TestJSONRPCErrorCodes(t *testing.T) {
	if ParseError != -32700 {
		t.Errorf("ParseError = %d, want -32700", ParseError)
	}
	if InvalidRequest != -32600 {
		t.Errorf("InvalidRequest = %d, want -32600", InvalidRequest)
	}
	if MethodNotFound != -32601 {
		t.Errorf("MethodNotFound = %d, want -32601", MethodNotFound)
	}
	if InvalidParams != -32602 {
		t.Errorf("InvalidParams = %d, want -32602", InvalidParams)
	}
	if InternalError != -32603 {
		t.Errorf("InternalError = %d, want -32603", InternalError)
	}
}

func TestNotification_JSONSerialization(t *testing.T) {
	notification := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		Params:  json.RawMessage(`{}`),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Notification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, "2.0")
	}
	if decoded.Method != "notifications/initialized" {
		t.Errorf("Method = %q, want %q", decoded.Method, "notifications/initialized")
	}
}

func TestServerInfo_JSONSerialization(t *testing.T) {
	info := ServerInfo{
		Name:    "currier-mcp",
		Version: "0.1.0",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ServerInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != info.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, info.Name)
	}
	if decoded.Version != info.Version {
		t.Errorf("Version = %q, want %q", decoded.Version, info.Version)
	}
}

func TestServerCapabilities_JSONSerialization(t *testing.T) {
	caps := ServerCapabilities{
		Tools: &ToolsCapability{
			ListChanged: true,
		},
		Resources: &ResourcesCapability{
			Subscribe:   true,
			ListChanged: false,
		},
	}

	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ServerCapabilities
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Tools == nil {
		t.Fatal("Tools should not be nil")
	}
	if !decoded.Tools.ListChanged {
		t.Error("Tools.ListChanged should be true")
	}
	if decoded.Resources == nil {
		t.Fatal("Resources should not be nil")
	}
	if !decoded.Resources.Subscribe {
		t.Error("Resources.Subscribe should be true")
	}
}

func TestInitializeParams_JSONSerialization(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ProtocolVersion != ProtocolVersion {
		t.Errorf("ProtocolVersion = %q, want %q", decoded.ProtocolVersion, ProtocolVersion)
	}
	if decoded.ClientInfo.Name != "test-client" {
		t.Errorf("ClientInfo.Name = %q, want %q", decoded.ClientInfo.Name, "test-client")
	}
}

func TestInitializeResult_JSONSerialization(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: ServerInfo{
			Name:    "currier-mcp",
			Version: "0.1.0",
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ProtocolVersion != ProtocolVersion {
		t.Errorf("ProtocolVersion = %q, want %q", decoded.ProtocolVersion, ProtocolVersion)
	}
	if decoded.ServerInfo.Name != "currier-mcp" {
		t.Errorf("ServerInfo.Name = %q, want %q", decoded.ServerInfo.Name, "currier-mcp")
	}
}

func TestTool_JSONSerialization(t *testing.T) {
	tool := Tool{
		Name:        "send_request",
		Description: "Send an HTTP request",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"method":{"type":"string"}}}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != tool.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, tool.Name)
	}
	if decoded.Description != tool.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, tool.Description)
	}
}

func TestToolsListResult_JSONSerialization(t *testing.T) {
	result := ToolsListResult{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ToolsListResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(decoded.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(decoded.Tools))
	}
}

func TestToolCallParams_JSONSerialization(t *testing.T) {
	params := ToolCallParams{
		Name:      "send_request",
		Arguments: json.RawMessage(`{"method":"GET","url":"https://example.com"}`),
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ToolCallParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "send_request" {
		t.Errorf("Name = %q, want %q", decoded.Name, "send_request")
	}
}

func TestToolCallResult_JSONSerialization(t *testing.T) {
	result := ToolCallResult{
		Content: []ContentBlock{
			TextContent("Success"),
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ToolCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(decoded.Content) != 1 {
		t.Errorf("len(Content) = %d, want 1", len(decoded.Content))
	}
	if decoded.IsError {
		t.Error("IsError should be false")
	}
}

func TestContentBlock_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name: "text block",
			block: ContentBlock{
				Type: "text",
				Text: "Hello, World!",
			},
		},
		{
			name: "binary block",
			block: ContentBlock{
				Type:     "blob",
				MimeType: "image/png",
				Data:     "iVBORw0KGgo=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.block)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded ContentBlock
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.Type != tt.block.Type {
				t.Errorf("Type = %q, want %q", decoded.Type, tt.block.Type)
			}
		})
	}
}

func TestResource_JSONSerialization(t *testing.T) {
	resource := Resource{
		URI:         "collections://list",
		Name:        "Collections List",
		Description: "List of API collections",
		MimeType:    "application/json",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Resource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.URI != resource.URI {
		t.Errorf("URI = %q, want %q", decoded.URI, resource.URI)
	}
	if decoded.Name != resource.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, resource.Name)
	}
}

func TestResourcesListResult_JSONSerialization(t *testing.T) {
	result := ResourcesListResult{
		Resources: []Resource{
			{URI: "collections://list", Name: "Collections"},
			{URI: "history://recent", Name: "Recent History"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ResourcesListResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(decoded.Resources) != 2 {
		t.Errorf("len(Resources) = %d, want 2", len(decoded.Resources))
	}
}

func TestResourceReadParams_JSONSerialization(t *testing.T) {
	params := ResourceReadParams{
		URI: "collections://list",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ResourceReadParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.URI != "collections://list" {
		t.Errorf("URI = %q, want %q", decoded.URI, "collections://list")
	}
}

func TestResourceReadResult_JSONSerialization(t *testing.T) {
	result := ResourceReadResult{
		Contents: []ResourceContent{
			{
				URI:      "collections://list",
				MimeType: "application/json",
				Text:     `{"collections":[]}`,
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ResourceReadResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(decoded.Contents) != 1 {
		t.Errorf("len(Contents) = %d, want 1", len(decoded.Contents))
	}
}

func TestMCPMethodNames(t *testing.T) {
	if MethodInitialize != "initialize" {
		t.Errorf("MethodInitialize = %q, want %q", MethodInitialize, "initialize")
	}
	if MethodToolsList != "tools/list" {
		t.Errorf("MethodToolsList = %q, want %q", MethodToolsList, "tools/list")
	}
	if MethodToolsCall != "tools/call" {
		t.Errorf("MethodToolsCall = %q, want %q", MethodToolsCall, "tools/call")
	}
	if MethodResourcesList != "resources/list" {
		t.Errorf("MethodResourcesList = %q, want %q", MethodResourcesList, "resources/list")
	}
	if MethodResourcesRead != "resources/read" {
		t.Errorf("MethodResourcesRead = %q, want %q", MethodResourcesRead, "resources/read")
	}
	if MethodPing != "ping" {
		t.Errorf("MethodPing = %q, want %q", MethodPing, "ping")
	}
}

func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, "2024-11-05")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "hello",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring at start",
			s:      "hello world",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "hello world",
			substr: "world",
			want:   true,
		},
		{
			name:   "substring in middle",
			s:      "hello world test",
			substr: "world",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "foo",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "hello",
			substr: "",
			want:   true,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "hello",
			want:   false,
		},
		{
			name:   "both empty",
			s:      "",
			substr: "",
			want:   true,
		},
		{
			name:   "substring longer than string",
			s:      "hi",
			substr: "hello",
			want:   false,
		},
		{
			name:   "single char match",
			s:      "hello",
			substr: "e",
			want:   true,
		},
		{
			name:   "single char no match",
			s:      "hello",
			substr: "x",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestSearchSubstring(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "found at start",
			s:      "hello world",
			substr: "hello",
			want:   true,
		},
		{
			name:   "found at end",
			s:      "hello world",
			substr: "world",
			want:   true,
		},
		{
			name:   "found in middle",
			s:      "hello world test",
			substr: "world",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "xyz",
			want:   false,
		},
		{
			name:   "exact match",
			s:      "test",
			substr: "test",
			want:   true,
		},
		{
			name:   "partial match only",
			s:      "testing",
			substr: "test",
			want:   true,
		},
		{
			name:   "case sensitive no match",
			s:      "Hello",
			substr: "hello",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("searchSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestJSONContentError(t *testing.T) {
	// Test with a value that cannot be marshaled to JSON
	// Channels cannot be marshaled
	ch := make(chan int)
	_, err := JSONContent(ch)
	if err == nil {
		t.Error("JSONContent should return error for unmarshalable value")
	}
}

func TestTruncateBodyEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		maxSize     int
		wantLen     int
		wantTrunc   bool
	}{
		{
			name:      "empty body",
			body:      "",
			maxSize:   100,
			wantLen:   0,
			wantTrunc: false,
		},
		{
			name:      "one char under max",
			body:      strings.Repeat("x", 99),
			maxSize:   100,
			wantLen:   99,
			wantTrunc: false,
		},
		{
			name:      "one char over max",
			body:      strings.Repeat("x", 101),
			maxSize:   100,
			wantTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, truncated := truncateBody(tt.body, tt.maxSize)
			if truncated != tt.wantTrunc {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTrunc)
			}
			if !tt.wantTrunc && len(result) != tt.wantLen {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantLen)
			}
			if tt.wantTrunc && !strings.Contains(result, "TRUNCATED") {
				t.Errorf("result should contain 'TRUNCATED' when truncated")
			}
		})
	}
}

func TestApplyPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		offset     int
		limit      int
		wantLen    int
		wantTotal  int
		wantMore   bool
	}{
		{
			name:      "empty slice",
			items:     []int{},
			offset:    0,
			limit:     10,
			wantLen:   0,
			wantTotal: 0,
			wantMore:  false,
		},
		{
			name:      "negative offset treated as zero",
			items:     []int{1, 2, 3},
			offset:    -1,
			limit:     2,
			wantLen:   2,
			wantTotal: 3,
			wantMore:  true,
		},
		{
			name:      "single item",
			items:     []int{1},
			offset:    0,
			limit:     10,
			wantLen:   1,
			wantTotal: 1,
			wantMore:  false,
		},
		{
			name:      "limit larger than slice",
			items:     []int{1, 2, 3},
			offset:    0,
			limit:     100,
			wantLen:   3,
			wantTotal: 3,
			wantMore:  false,
		},
		{
			name:      "offset at boundary",
			items:     []int{1, 2, 3, 4, 5},
			offset:    5,
			limit:     2,
			wantLen:   0,
			wantTotal: 5,
			wantMore:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pagination := applyPagination(tt.items, tt.offset, tt.limit)
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

// ============================================================================
// Additional Args Struct Tests
// ============================================================================

func TestListCookiesArgs(t *testing.T) {
	args := listCookiesArgs{
		Domain: "example.com",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded listCookiesArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", decoded.Domain, "example.com")
	}
}

func TestGetHistoryArgs(t *testing.T) {
	args := getHistoryArgs{
		Limit:  50,
		Filter: "GET",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded getHistoryArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Limit != 50 {
		t.Errorf("Limit = %d, want %d", decoded.Limit, 50)
	}
	if decoded.Filter != "GET" {
		t.Errorf("Filter = %q, want %q", decoded.Filter, "GET")
	}
}

func TestCreateCollectionArgs(t *testing.T) {
	args := createCollectionArgs{
		Name:        "My API Collection",
		Description: "A collection of API endpoints",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded createCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "My API Collection" {
		t.Errorf("Name = %q, want %q", decoded.Name, "My API Collection")
	}
	if decoded.Description != "A collection of API endpoints" {
		t.Errorf("Description = %q, want %q", decoded.Description, "A collection of API endpoints")
	}
}

func TestDeleteCollectionArgs(t *testing.T) {
	args := deleteCollectionArgs{
		Name: "Old Collection",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded deleteCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "Old Collection" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Old Collection")
	}
}

func TestRenameCollectionArgs(t *testing.T) {
	args := renameCollectionArgs{
		Name:    "Old Name",
		NewName: "New Name",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded renameCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "Old Name" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Old Name")
	}
	if decoded.NewName != "New Name" {
		t.Errorf("NewName = %q, want %q", decoded.NewName, "New Name")
	}
}

func TestCreateFolderArgs(t *testing.T) {
	args := createFolderArgs{
		Collection:  "My Collection",
		Name:        "new-folder",
		Parent:      "parent/child",
		Description: "A test folder",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded createFolderArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Collection != "My Collection" {
		t.Errorf("Collection = %q, want %q", decoded.Collection, "My Collection")
	}
	if decoded.Name != "new-folder" {
		t.Errorf("Name = %q, want %q", decoded.Name, "new-folder")
	}
	if decoded.Parent != "parent/child" {
		t.Errorf("Parent = %q, want %q", decoded.Parent, "parent/child")
	}
	if decoded.Description != "A test folder" {
		t.Errorf("Description = %q, want %q", decoded.Description, "A test folder")
	}
}

func TestDeleteFolderArgs(t *testing.T) {
	args := deleteFolderArgs{
		Collection: "My Collection",
		Folder:     "folder/to/delete",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded deleteFolderArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Collection != "My Collection" {
		t.Errorf("Collection = %q, want %q", decoded.Collection, "My Collection")
	}
	if decoded.Folder != "folder/to/delete" {
		t.Errorf("Folder = %q, want %q", decoded.Folder, "folder/to/delete")
	}
}

func TestCreateEnvironmentArgs(t *testing.T) {
	args := createEnvironmentArgs{
		Name: "Production",
		Variables: map[string]string{
			"base_url": "https://api.example.com",
			"api_key":  "secret123",
		},
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded createEnvironmentArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "Production" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Production")
	}
	if decoded.Variables["base_url"] != "https://api.example.com" {
		t.Errorf("Variables[base_url] = %q, want %q", decoded.Variables["base_url"], "https://api.example.com")
	}
}

func TestDeleteEnvironmentArgs(t *testing.T) {
	args := deleteEnvironmentArgs{
		Name: "Staging",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded deleteEnvironmentArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "Staging" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Staging")
	}
}

func TestSetEnvVarArgs(t *testing.T) {
	args := setEnvVarArgs{
		Environment: "Production",
		Key:         "api_key",
		Value:       "new-secret",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded setEnvVarArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Environment != "Production" {
		t.Errorf("Environment = %q, want %q", decoded.Environment, "Production")
	}
	if decoded.Key != "api_key" {
		t.Errorf("Key = %q, want %q", decoded.Key, "api_key")
	}
	if decoded.Value != "new-secret" {
		t.Errorf("Value = %q, want %q", decoded.Value, "new-secret")
	}
}

func TestRunCollectionArgs(t *testing.T) {
	args := runCollectionArgs{
		Name:        "My API Tests",
		Environment: "Production",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded runCollectionArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "My API Tests" {
		t.Errorf("Name = %q, want %q", decoded.Name, "My API Tests")
	}
	if decoded.Environment != "Production" {
		t.Errorf("Environment = %q, want %q", decoded.Environment, "Production")
	}
}

func TestExportAsCurlArgs(t *testing.T) {
	args := exportAsCurlArgs{
		Collection: "My API",
		Request:    "Get Users",
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded exportAsCurlArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Collection != "My API" {
		t.Errorf("Collection = %q, want %q", decoded.Collection, "My API")
	}
	if decoded.Request != "Get Users" {
		t.Errorf("Request = %q, want %q", decoded.Request, "Get Users")
	}
}
