package importer

import (
	"context"
	"testing"

	"github.com/artpar/currier/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHARImporter_Name(t *testing.T) {
	imp := NewHARImporter()
	assert.Equal(t, "HTTP Archive (HAR)", imp.Name())
}

func TestHARImporter_Format(t *testing.T) {
	imp := NewHARImporter()
	assert.Equal(t, FormatHAR, imp.Format())
}

func TestHARImporter_FileExtensions(t *testing.T) {
	imp := NewHARImporter()
	exts := imp.FileExtensions()
	assert.Contains(t, exts, ".har")
}

func TestHARImporter_DetectFormat(t *testing.T) {
	imp := NewHARImporter()

	t.Run("detects HAR with version", func(t *testing.T) {
		content := []byte(`{
			"log": {
				"version": "1.2",
				"entries": []
			}
		}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("detects HAR with creator", func(t *testing.T) {
		content := []byte(`{
			"log": {
				"creator": {
					"name": "Chrome DevTools"
				},
				"entries": []
			}
		}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("rejects non-HAR JSON", func(t *testing.T) {
		content := []byte(`{"openapi": "3.0.0"}`)
		assert.False(t, imp.DetectFormat(content))
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		content := []byte(`not valid json`)
		assert.False(t, imp.DetectFormat(content))
	})
}

func TestHARImporter_Import_BasicHAR(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {
				"name": "Chrome DevTools",
				"version": "100.0"
			},
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {
						"status": 200,
						"statusText": "OK",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"cookies": [],
						"content": {},
						"redirectURL": "",
						"headersSize": -1,
						"bodySize": 0
					},
					"cache": {},
					"timings": {
						"blocked": 0,
						"dns": 0,
						"connect": 0,
						"send": 0,
						"wait": 100,
						"receive": 50,
						"ssl": 0
					}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "HAR from Chrome DevTools", coll.Name())
	assert.Equal(t, "1.2", coll.Version())

	folders := coll.Folders()
	require.Len(t, folders, 1)
	assert.Equal(t, "api.example.com", folders[0].Name())

	requests := folders[0].Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "GET", requests[0].Method())
	assert.Equal(t, "https://api.example.com/users", requests[0].URL())
}

func TestHARImporter_Import_MultipleEntries(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				},
				{
					"request": {
						"method": "POST",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 201, "statusText": "Created", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				},
				{
					"request": {
						"method": "GET",
						"url": "https://cdn.example.com/assets/style.css",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	folders := coll.Folders()
	require.Len(t, folders, 2) // Two different domains

	// Check api.example.com folder
	var apiFolder *core.Folder
	var cdnFolder *core.Folder
	for _, f := range folders {
		if f.Name() == "api.example.com" {
			apiFolder = f
		} else if f.Name() == "cdn.example.com" {
			cdnFolder = f
		}
	}

	require.NotNil(t, apiFolder)
	require.NotNil(t, cdnFolder)

	assert.Len(t, apiFolder.Requests(), 2)
	assert.Len(t, cdnFolder.Requests(), 1)
}

func TestHARImporter_Import_WithHeaders(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/data",
						"httpVersion": "HTTP/2",
						"headers": [
							{"name": ":authority", "value": "api.example.com"},
							{"name": ":method", "value": "GET"},
							{"name": "Accept", "value": "application/json"},
							{"name": "Authorization", "value": "Bearer token123"}
						],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/2", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Folders()[0].Requests()[0]

	// Should include regular headers
	assert.Equal(t, "application/json", req.GetHeader("Accept"))
	assert.Equal(t, "Bearer token123", req.GetHeader("Authorization"))

	// Should skip pseudo-headers (starting with :)
	assert.Empty(t, req.GetHeader(":authority"))
	assert.Empty(t, req.GetHeader(":method"))
}

func TestHARImporter_Import_WithCookies(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/data",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [
							{"name": "session", "value": "abc123"},
							{"name": "user", "value": "john"}
						],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Folders()[0].Requests()[0]
	cookie := req.GetHeader("Cookie")
	assert.Contains(t, cookie, "session=abc123")
	assert.Contains(t, cookie, "user=john")
}

func TestHARImporter_Import_WithPostData(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": [
				{
					"request": {
						"method": "POST",
						"url": "https://api.example.com/users",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{"name": "Content-Type", "value": "application/json"}
						],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 50,
						"postData": {
							"mimeType": "application/json",
							"text": "{\"name\": \"John\", \"email\": \"john@example.com\"}"
						}
					},
					"response": {"status": 201, "statusText": "Created", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Folders()[0].Requests()[0]
	assert.Equal(t, "POST", req.Method())
	assert.Equal(t, "{\"name\": \"John\", \"email\": \"john@example.com\"}", req.Body())
}

func TestHARImporter_Import_RequestNaming(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": [
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/users/123",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				},
				{
					"request": {
						"method": "GET",
						"url": "https://api.example.com/",
						"httpVersion": "HTTP/1.1",
						"headers": [],
						"queryString": [],
						"cookies": [],
						"headersSize": -1,
						"bodySize": 0
					},
					"response": {"status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": {}, "redirectURL": "", "headersSize": -1, "bodySize": 0},
					"cache": {},
					"timings": {"blocked": 0, "dns": 0, "connect": 0, "send": 0, "wait": 0, "receive": 0, "ssl": 0}
				}
			]
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Folders()[0].Requests()
	require.Len(t, requests, 2)

	assert.Equal(t, "123", requests[0].Name()) // Uses last path segment
	assert.Equal(t, "Request 2", requests[1].Name()) // Falls back to generic name
}

func TestHARImporter_Import_InvalidJSON(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`not valid json`)

	_, err := imp.Import(ctx, content)
	assert.ErrorIs(t, err, ErrParseError)
}

func TestHARImporter_Import_EmptyEntries(t *testing.T) {
	imp := NewHARImporter()
	ctx := context.Background()

	content := []byte(`{
		"log": {
			"version": "1.2",
			"creator": {"name": "Test"},
			"entries": []
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Empty(t, coll.Folders())
	assert.Empty(t, coll.Requests())
}
