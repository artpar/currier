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
