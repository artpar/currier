package core

import (
	"testing"
	"time"

	"github.com/artpar/currier/internal/interpolate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollection(t *testing.T) {
	t.Run("creates collection with name", func(t *testing.T) {
		c := NewCollection("My API")
		assert.NotEmpty(t, c.ID())
		assert.Equal(t, "My API", c.Name())
		assert.Empty(t, c.Description())
		assert.False(t, c.CreatedAt().IsZero())
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		c1 := NewCollection("API 1")
		c2 := NewCollection("API 2")
		assert.NotEqual(t, c1.ID(), c2.ID())
	})

	t.Run("sets created and updated timestamps", func(t *testing.T) {
		before := time.Now()
		c := NewCollection("Test")
		after := time.Now()

		assert.True(t, c.CreatedAt().After(before) || c.CreatedAt().Equal(before))
		assert.True(t, c.CreatedAt().Before(after) || c.CreatedAt().Equal(after))
		assert.Equal(t, c.CreatedAt(), c.UpdatedAt())
	})
}

func TestCollection_Metadata(t *testing.T) {
	t.Run("sets description", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetDescription("A test API collection")
		assert.Equal(t, "A test API collection", c.Description())
	})

	t.Run("sets version", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetVersion("1.0.0")
		assert.Equal(t, "1.0.0", c.Version())
	})

	t.Run("updates timestamp on modification", func(t *testing.T) {
		c := NewCollection("My API")
		original := c.UpdatedAt()

		time.Sleep(1 * time.Millisecond)
		c.SetDescription("Updated")

		assert.True(t, c.UpdatedAt().After(original))
	})
}

func TestCollection_Variables(t *testing.T) {
	t.Run("starts with empty variables", func(t *testing.T) {
		c := NewCollection("My API")
		assert.Empty(t, c.Variables())
	})

	t.Run("sets and gets variables", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetVariable("base_url", "https://api.example.com")
		c.SetVariable("api_key", "secret123")

		assert.Equal(t, "https://api.example.com", c.GetVariable("base_url"))
		assert.Equal(t, "secret123", c.GetVariable("api_key"))
	})

	t.Run("returns empty string for undefined variable", func(t *testing.T) {
		c := NewCollection("My API")
		assert.Equal(t, "", c.GetVariable("undefined"))
	})

	t.Run("overwrites existing variable", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetVariable("key", "value1")
		c.SetVariable("key", "value2")
		assert.Equal(t, "value2", c.GetVariable("key"))
	})

	t.Run("deletes variable", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetVariable("key", "value")
		c.DeleteVariable("key")
		assert.Equal(t, "", c.GetVariable("key"))
	})

	t.Run("lists all variables", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetVariable("a", "1")
		c.SetVariable("b", "2")
		c.SetVariable("c", "3")

		vars := c.Variables()
		assert.Len(t, vars, 3)
		assert.Equal(t, "1", vars["a"])
		assert.Equal(t, "2", vars["b"])
		assert.Equal(t, "3", vars["c"])
	})
}

func TestCollection_Folders(t *testing.T) {
	t.Run("starts with no folders", func(t *testing.T) {
		c := NewCollection("My API")
		assert.Empty(t, c.Folders())
	})

	t.Run("adds folder", func(t *testing.T) {
		c := NewCollection("My API")
		folder := c.AddFolder("Users")

		assert.NotEmpty(t, folder.ID())
		assert.Equal(t, "Users", folder.Name())
		assert.Len(t, c.Folders(), 1)
	})

	t.Run("adds multiple folders", func(t *testing.T) {
		c := NewCollection("My API")
		c.AddFolder("Users")
		c.AddFolder("Posts")
		c.AddFolder("Comments")

		assert.Len(t, c.Folders(), 3)
	})

	t.Run("gets folder by ID", func(t *testing.T) {
		c := NewCollection("My API")
		folder := c.AddFolder("Users")

		found, ok := c.GetFolder(folder.ID())
		assert.True(t, ok)
		assert.Equal(t, "Users", found.Name())
	})

	t.Run("gets folder by name", func(t *testing.T) {
		c := NewCollection("My API")
		c.AddFolder("Users")

		found, ok := c.GetFolderByName("Users")
		assert.True(t, ok)
		assert.Equal(t, "Users", found.Name())
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		c := NewCollection("My API")
		_, ok := c.GetFolder("non-existent")
		assert.False(t, ok)
	})

	t.Run("removes folder", func(t *testing.T) {
		c := NewCollection("My API")
		folder := c.AddFolder("Users")
		c.RemoveFolder(folder.ID())

		assert.Empty(t, c.Folders())
	})

	t.Run("adds nested folder", func(t *testing.T) {
		c := NewCollection("My API")
		parent := c.AddFolder("Users")
		child := parent.AddFolder("Admin")

		assert.Equal(t, "Admin", child.Name())
		assert.Len(t, parent.Folders(), 1)
	})
}

func TestCollection_Requests(t *testing.T) {
	t.Run("starts with no requests", func(t *testing.T) {
		c := NewCollection("My API")
		assert.Empty(t, c.Requests())
	})

	t.Run("adds request to collection root", func(t *testing.T) {
		c := NewCollection("My API")
		req := NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		c.AddRequest(req)

		assert.Len(t, c.Requests(), 1)
	})

	t.Run("adds request to folder", func(t *testing.T) {
		c := NewCollection("My API")
		folder := c.AddFolder("Users")

		req := NewRequestDefinition("Get User", "GET", "https://api.example.com/users/1")
		folder.AddRequest(req)

		assert.Len(t, folder.Requests(), 1)
		assert.Empty(t, c.Requests()) // Root should be empty
	})

	t.Run("gets request by ID", func(t *testing.T) {
		c := NewCollection("My API")
		req := NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		c.AddRequest(req)

		found, ok := c.GetRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Get Users", found.Name())
	})

	t.Run("finds request in nested folder", func(t *testing.T) {
		c := NewCollection("My API")
		folder := c.AddFolder("Users")
		req := NewRequestDefinition("Get User", "GET", "https://api.example.com/users/1")
		folder.AddRequest(req)

		found, ok := c.FindRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Get User", found.Name())
	})

	t.Run("removes request", func(t *testing.T) {
		c := NewCollection("My API")
		req := NewRequestDefinition("Get Users", "GET", "https://api.example.com/users")
		c.AddRequest(req)
		c.RemoveRequest(req.ID())

		assert.Empty(t, c.Requests())
	})
}

func TestRequestDefinition(t *testing.T) {
	t.Run("creates request definition", func(t *testing.T) {
		req := NewRequestDefinition("Get User", "GET", "https://api.example.com/users/{{user_id}}")

		assert.NotEmpty(t, req.ID())
		assert.Equal(t, "Get User", req.Name())
		assert.Equal(t, "GET", req.Method())
		assert.Equal(t, "https://api.example.com/users/{{user_id}}", req.URL())
	})

	t.Run("sets headers", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("Authorization", "Bearer {{token}}")
		req.SetHeader("Accept", "application/json")

		assert.Equal(t, "Bearer {{token}}", req.GetHeader("Authorization"))
		assert.Equal(t, "application/json", req.GetHeader("Accept"))
	})

	t.Run("sets body", func(t *testing.T) {
		req := NewRequestDefinition("Create User", "POST", "https://example.com/users")
		req.SetBodyJSON(map[string]any{
			"name":  "{{username}}",
			"email": "{{email}}",
		})

		assert.Equal(t, "json", req.BodyType())
		assert.NotEmpty(t, req.BodyContent())
	})

	t.Run("sets pre-request script", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		script := `currier.setVariable("timestamp", Date.now());`
		req.SetPreScript(script)

		assert.Equal(t, script, req.PreScript())
	})

	t.Run("sets post-response script", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		script := `currier.test("Status OK", response.status === 200);`
		req.SetPostScript(script)

		assert.Equal(t, script, req.PostScript())
	})

	t.Run("sets description", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetDescription("Fetches user data by ID")

		assert.Equal(t, "Fetches user data by ID", req.Description())
	})

	t.Run("converts to core.Request", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://api.example.com/users/1")
		def.SetHeader("Accept", "application/json")

		req, err := def.ToRequest()
		require.NoError(t, err)

		assert.Equal(t, "GET", req.Method())
		assert.Equal(t, "https://api.example.com/users/1", req.Endpoint())
		assert.Equal(t, "application/json", req.Headers().Get("Accept"))
	})
}

func TestFolder(t *testing.T) {
	t.Run("creates folder with name", func(t *testing.T) {
		f := NewFolder("Users")
		assert.NotEmpty(t, f.ID())
		assert.Equal(t, "Users", f.Name())
	})

	t.Run("sets description", func(t *testing.T) {
		f := NewFolder("Users")
		f.SetDescription("User management endpoints")
		assert.Equal(t, "User management endpoints", f.Description())
	})

	t.Run("adds nested folders", func(t *testing.T) {
		f := NewFolder("Users")
		admin := f.AddFolder("Admin")
		guest := f.AddFolder("Guest")

		assert.Len(t, f.Folders(), 2)
		assert.Equal(t, "Admin", admin.Name())
		assert.Equal(t, "Guest", guest.Name())
	})

	t.Run("adds requests", func(t *testing.T) {
		f := NewFolder("Users")
		req1 := NewRequestDefinition("List", "GET", "/users")
		req2 := NewRequestDefinition("Create", "POST", "/users")

		f.AddRequest(req1)
		f.AddRequest(req2)

		assert.Len(t, f.Requests(), 2)
	})

	t.Run("finds request recursively", func(t *testing.T) {
		f := NewFolder("Users")
		nested := f.AddFolder("Admin")
		req := NewRequestDefinition("Get Admin", "GET", "/admin")
		nested.AddRequest(req)

		found, ok := f.FindRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Get Admin", found.Name())
	})
}

func TestCollection_Auth(t *testing.T) {
	t.Run("sets collection-level auth", func(t *testing.T) {
		c := NewCollection("My API")
		c.SetAuth(AuthConfig{
			Type:  "bearer",
			Token: "{{access_token}}",
		})

		auth := c.Auth()
		assert.Equal(t, "bearer", auth.Type)
		assert.Equal(t, "{{access_token}}", auth.Token)
	})

	t.Run("supports different auth types", func(t *testing.T) {
		c := NewCollection("My API")

		// Basic auth
		c.SetAuth(AuthConfig{
			Type:     "basic",
			Username: "{{username}}",
			Password: "{{password}}",
		})
		assert.Equal(t, "basic", c.Auth().Type)

		// API Key
		c.SetAuth(AuthConfig{
			Type:   "apikey",
			Key:    "X-API-Key",
			Value:  "{{api_key}}",
			In:     "header",
		})
		assert.Equal(t, "apikey", c.Auth().Type)
	})
}

func TestCollection_Clone(t *testing.T) {
	t.Run("creates deep copy", func(t *testing.T) {
		original := NewCollection("Original")
		original.SetDescription("Original description")
		original.SetVariable("key", "value")
		folder := original.AddFolder("Folder1")
		req := NewRequestDefinition("Req1", "GET", "/test")
		folder.AddRequest(req)

		clone := original.Clone()

		// Verify it's a copy
		assert.NotEqual(t, original.ID(), clone.ID())
		assert.Equal(t, original.Name(), clone.Name())
		assert.Equal(t, original.Description(), clone.Description())
		assert.Equal(t, original.GetVariable("key"), clone.GetVariable("key"))

		// Verify modifications don't affect original
		clone.SetDescription("Modified")
		assert.Equal(t, "Original description", original.Description())
	})
}

func TestCollection_Scripts(t *testing.T) {
	t.Run("sets and gets pre-script", func(t *testing.T) {
		c := NewCollection("My API")
		script := "console.log('pre-request');"
		c.SetPreScript(script)
		assert.Equal(t, script, c.PreScript())
	})

	t.Run("sets and gets post-script", func(t *testing.T) {
		c := NewCollection("My API")
		script := "console.log('post-request');"
		c.SetPostScript(script)
		assert.Equal(t, script, c.PostScript())
	})
}

func TestRequestDefinition_URLAndMethod(t *testing.T) {
	t.Run("sets URL", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://old.example.com")
		req.SetURL("https://new.example.com")
		assert.Equal(t, "https://new.example.com", req.URL())
	})

	t.Run("sets method", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetMethod("POST")
		assert.Equal(t, "POST", req.Method())
	})

	t.Run("removes header", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("X-Custom", "value")
		req.RemoveHeader("X-Custom")
		assert.Equal(t, "", req.GetHeader("X-Custom"))
	})

	t.Run("sets body directly", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBody(`{"key":"value"}`)
		assert.Equal(t, `{"key":"value"}`, req.Body())
	})

	t.Run("returns empty query params", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		assert.Empty(t, req.QueryParams())
	})
}

func TestRequestDefinition_Auth(t *testing.T) {
	t.Run("sets and gets auth", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetAuth(AuthConfig{
			Type:  "bearer",
			Token: "my-token",
		})
		auth := req.Auth()
		require.NotNil(t, auth)
		assert.Equal(t, "bearer", auth.Type)
		assert.Equal(t, "my-token", auth.Token)
	})

	t.Run("returns nil when no auth", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		assert.Nil(t, req.Auth())
	})
}

func TestRequestDefinition_BodyRaw(t *testing.T) {
	t.Run("sets raw body", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "https://example.com")
		req.SetBodyRaw("plain text content", "text/plain")
		assert.Equal(t, "raw", req.BodyType())
		assert.Equal(t, "plain text content", req.BodyContent())
	})
}

func TestRequestDefinition_Clone(t *testing.T) {
	t.Run("clones request definition", func(t *testing.T) {
		original := NewRequestDefinition("Test", "POST", "https://example.com")
		original.SetDescription("Test description")
		original.SetHeader("X-Custom", "value")
		original.SetBody(`{"key":"value"}`)
		original.SetAuth(AuthConfig{Type: "bearer", Token: "token"})
		original.SetPreScript("pre")
		original.SetPostScript("post")

		clone := original.Clone()

		assert.NotEqual(t, original.ID(), clone.ID())
		assert.Equal(t, original.Name(), clone.Name())
		assert.Equal(t, original.Description(), clone.Description())
		assert.Equal(t, original.Method(), clone.Method())
		assert.Equal(t, original.URL(), clone.URL())
		assert.Equal(t, original.GetHeader("X-Custom"), clone.GetHeader("X-Custom"))
		assert.Equal(t, original.Body(), clone.Body())
		assert.Equal(t, original.PreScript(), clone.PreScript())
		assert.Equal(t, original.PostScript(), clone.PostScript())

		// Verify modifications don't affect original
		clone.SetDescription("Modified")
		assert.Equal(t, "Test description", original.Description())
	})
}

func TestFolder_Clone(t *testing.T) {
	t.Run("clones folder", func(t *testing.T) {
		original := NewFolder("Original")
		original.SetDescription("Folder description")
		nested := original.AddFolder("Nested")
		nested.AddRequest(NewRequestDefinition("Test", "GET", "/test"))

		clone := original.Clone()

		assert.NotEqual(t, original.ID(), clone.ID())
		assert.Equal(t, original.Name(), clone.Name())
		assert.Equal(t, original.Description(), clone.Description())
		assert.Len(t, clone.Folders(), 1)
	})
}

func TestFolder_GetRequest(t *testing.T) {
	t.Run("gets request by ID", func(t *testing.T) {
		f := NewFolder("Test")
		req := NewRequestDefinition("Test Request", "GET", "/test")
		f.AddRequest(req)

		found, ok := f.GetRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Test Request", found.Name())
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		f := NewFolder("Test")
		_, ok := f.GetRequest("non-existent")
		assert.False(t, ok)
	})
}

func TestFolder_RemoveRequest(t *testing.T) {
	t.Run("removes request by ID", func(t *testing.T) {
		f := NewFolder("Test")
		req := NewRequestDefinition("Test Request", "GET", "/test")
		f.AddRequest(req)
		f.RemoveRequest(req.ID())
		assert.Empty(t, f.Requests())
	})
}

func TestNewCollectionWithID(t *testing.T) {
	t.Run("creates collection with specific ID", func(t *testing.T) {
		c := NewCollectionWithID("custom-id-123", "Test Collection")
		assert.Equal(t, "custom-id-123", c.ID())
		assert.Equal(t, "Test Collection", c.Name())
	})
}

func TestCollection_SetTimestamps(t *testing.T) {
	t.Run("sets timestamps", func(t *testing.T) {
		c := NewCollection("Test")
		created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		updated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

		c.SetTimestamps(created, updated)

		assert.Equal(t, created, c.CreatedAt())
		assert.Equal(t, updated, c.UpdatedAt())
	})
}

func TestCollection_AddExistingFolder(t *testing.T) {
	t.Run("adds existing folder", func(t *testing.T) {
		c := NewCollection("Test")
		f := NewFolder("Existing")
		c.AddExistingFolder(f)
		assert.Len(t, c.Folders(), 1)
	})
}

func TestNewFolderWithID(t *testing.T) {
	t.Run("creates folder with specific ID", func(t *testing.T) {
		f := NewFolderWithID("folder-123", "Test Folder")
		assert.Equal(t, "folder-123", f.ID())
		assert.Equal(t, "Test Folder", f.Name())
	})
}

func TestFolder_AddExistingFolder(t *testing.T) {
	t.Run("adds existing folder to folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := NewFolder("Child")
		parent.AddExistingFolder(child)
		assert.Len(t, parent.Folders(), 1)
	})
}

func TestNewRequestDefinitionWithID(t *testing.T) {
	t.Run("creates request definition with specific ID", func(t *testing.T) {
		req := NewRequestDefinitionWithID("req-123", "Test Request", "GET", "/test")
		assert.Equal(t, "req-123", req.ID())
		assert.Equal(t, "Test Request", req.Name())
		assert.Equal(t, "GET", req.Method())
		assert.Equal(t, "/test", req.URL())
	})
}

func TestRequestDefinition_ToRequestWithBody(t *testing.T) {
	t.Run("converts request with JSON body", func(t *testing.T) {
		def := NewRequestDefinition("Create User", "POST", "https://example.com/users")
		def.SetBodyJSON(map[string]any{"name": "test"})

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.NotNil(t, req.Body())
	})

	t.Run("converts request with raw body", func(t *testing.T) {
		def := NewRequestDefinition("Send Data", "POST", "https://example.com/data")
		def.SetBodyRaw("plain text", "text/plain")

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.NotNil(t, req.Body())
	})
}

func TestRequestDefinition_ToRequestWithAuth(t *testing.T) {
	t.Run("applies basic auth to request", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewBasicAuth("admin", "secret")
		def.SetAuth(auth)

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.Contains(t, req.Headers().Get("Authorization"), "Basic")
	})

	t.Run("applies bearer auth to request", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewBearerAuth("mytoken123")
		def.SetAuth(auth)

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.Equal(t, "Bearer mytoken123", req.Headers().Get("Authorization"))
	})

	t.Run("applies API key in header", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewAPIKeyAuth("X-API-Key", "secret123", APIKeyInHeader)
		def.SetAuth(auth)

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.Equal(t, "secret123", req.Headers().Get("X-API-Key"))
	})

	t.Run("applies API key in query param", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewAPIKeyAuth("api_key", "secret456", APIKeyInQuery)
		def.SetAuth(auth)

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.Contains(t, req.Endpoint(), "api_key=secret456")
	})

	t.Run("no auth headers when auth not configured", func(t *testing.T) {
		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.Empty(t, req.Headers().Get("Authorization"))
	})
}

func TestRequestDefinition_Headers(t *testing.T) {
	t.Run("returns copy of headers map", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://example.com")
		req.SetHeader("Content-Type", "application/json")
		req.SetHeader("Authorization", "Bearer token")

		headers := req.Headers()
		assert.Len(t, headers, 2)
		assert.Equal(t, "application/json", headers["Content-Type"])
		assert.Equal(t, "Bearer token", headers["Authorization"])

		// Verify it's a copy - modifications shouldn't affect original
		headers["X-New"] = "value"
		assert.Equal(t, "", req.GetHeader("X-New"))
	})
}

func TestCollection_GetFolderByNameEdgeCases(t *testing.T) {
	t.Run("returns false for non-existent folder name", func(t *testing.T) {
		c := NewCollection("My API")
		c.AddFolder("Users")

		_, ok := c.GetFolderByName("NonExistent")
		assert.False(t, ok)
	})

	t.Run("finds folder among multiple", func(t *testing.T) {
		c := NewCollection("My API")
		c.AddFolder("Users")
		c.AddFolder("Products")
		c.AddFolder("Orders")

		found, ok := c.GetFolderByName("Products")
		assert.True(t, ok)
		assert.Equal(t, "Products", found.Name())
	})
}

func TestCollection_FindRequestEdgeCases(t *testing.T) {
	t.Run("returns false for non-existent request", func(t *testing.T) {
		c := NewCollection("My API")
		c.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		_, ok := c.FindRequest("non-existent-id")
		assert.False(t, ok)
	})

	t.Run("finds request in root level", func(t *testing.T) {
		c := NewCollection("My API")
		req := NewRequestDefinition("Root Request", "GET", "/root")
		c.AddRequest(req)

		found, ok := c.FindRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Root Request", found.Name())
	})

	t.Run("finds request in deeply nested folder", func(t *testing.T) {
		c := NewCollection("My API")
		folder1 := c.AddFolder("Level1")
		folder2 := folder1.AddFolder("Level2")
		folder3 := folder2.AddFolder("Level3")
		req := NewRequestDefinition("Deep Request", "GET", "/deep")
		folder3.AddRequest(req)

		found, ok := c.FindRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Deep Request", found.Name())
	})
}

func TestFolder_FindRequestEdgeCases(t *testing.T) {
	t.Run("returns false for non-existent request", func(t *testing.T) {
		f := NewFolder("Test")
		f.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		_, ok := f.FindRequest("non-existent-id")
		assert.False(t, ok)
	})

	t.Run("finds request in direct children", func(t *testing.T) {
		f := NewFolder("Test")
		req := NewRequestDefinition("Direct Request", "GET", "/direct")
		f.AddRequest(req)

		found, ok := f.FindRequest(req.ID())
		assert.True(t, ok)
		assert.Equal(t, "Direct Request", found.Name())
	})
}

func TestRequestDefinition_SetBodyJSONError(t *testing.T) {
	t.Run("returns error for unmarshalable data", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "https://example.com")
		// channels cannot be marshaled to JSON
		err := req.SetBodyJSON(make(chan int))
		assert.Error(t, err)
	})
}

func TestRequestDefinition_ToRequestDefault(t *testing.T) {
	t.Run("handles default body type", func(t *testing.T) {
		def := NewRequestDefinition("Test", "POST", "https://example.com")
		// Set body content without specifying type
		def.SetBody("some content")

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.NotNil(t, req)
	})
}

func TestRequestDefinition_ToRequestWithEnv(t *testing.T) {
	t.Run("interpolates URL with variables", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("host", "api.example.com")
		engine.SetVariable("user_id", "123")

		def := NewRequestDefinition("Get User", "GET", "https://{{host}}/users/{{user_id}}")
		req, err := def.ToRequestWithEnv(engine)

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/users/123", req.Endpoint())
	})

	t.Run("interpolates headers with variables", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("token", "my-secret-token")

		def := NewRequestDefinition("Auth Request", "GET", "https://example.com")
		def.SetHeader("Authorization", "Bearer {{token}}")
		req, err := def.ToRequestWithEnv(engine)

		require.NoError(t, err)
		assert.Equal(t, "Bearer my-secret-token", req.Headers().Get("Authorization"))
	})

	t.Run("interpolates body with variables", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("username", "john")
		engine.SetVariable("email", "john@example.com")

		def := NewRequestDefinition("Create User", "POST", "https://example.com/users")
		def.SetBodyRaw(`{"name": "{{username}}", "email": "{{email}}"}`, "application/json")
		req, err := def.ToRequestWithEnv(engine)

		require.NoError(t, err)
		assert.Contains(t, req.Body().String(), "john")
		assert.Contains(t, req.Body().String(), "john@example.com")
	})

	t.Run("handles empty body", func(t *testing.T) {
		engine := interpolate.NewEngine()
		def := NewRequestDefinition("Get Users", "GET", "https://example.com/users")

		req, err := def.ToRequestWithEnv(engine)

		require.NoError(t, err)
		assert.True(t, req.Body().IsEmpty())
	})

	t.Run("returns error for invalid URL interpolation", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetOption("strict", true)

		def := NewRequestDefinition("Get User", "GET", "https://{{undefined_host}}/users")
		_, err := def.ToRequestWithEnv(engine)

		// Strict mode should fail on undefined variable
		assert.Error(t, err)
	})

	t.Run("interpolates bearer token auth", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("api_token", "secret123")

		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewBearerAuth("{{api_token}}")
		def.SetAuth(auth)

		req, err := def.ToRequestWithEnv(engine)
		require.NoError(t, err)
		assert.Equal(t, "Bearer secret123", req.Headers().Get("Authorization"))
	})

	t.Run("interpolates basic auth credentials", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("user", "admin")
		engine.SetVariable("pass", "secret")

		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewBasicAuth("{{user}}", "{{pass}}")
		def.SetAuth(auth)

		req, err := def.ToRequestWithEnv(engine)
		require.NoError(t, err)
		assert.Contains(t, req.Headers().Get("Authorization"), "Basic")
	})

	t.Run("interpolates API key value", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("key_value", "my-api-key")

		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewAPIKeyAuth("X-API-Key", "{{key_value}}", APIKeyInHeader)
		def.SetAuth(auth)

		req, err := def.ToRequestWithEnv(engine)
		require.NoError(t, err)
		assert.Equal(t, "my-api-key", req.Headers().Get("X-API-Key"))
	})

	t.Run("interpolates API key in query param", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("api_key", "secret-key")

		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		auth := NewAPIKeyAuth("key", "{{api_key}}", APIKeyInQuery)
		def.SetAuth(auth)

		req, err := def.ToRequestWithEnv(engine)
		require.NoError(t, err)
		assert.Contains(t, req.Endpoint(), "key=secret-key")
	})

	t.Run("returns error for invalid header interpolation", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetOption("strict", true)

		def := NewRequestDefinition("Get User", "GET", "https://example.com/users")
		def.SetHeader("X-Custom", "{{undefined_var}}")

		_, err := def.ToRequestWithEnv(engine)
		assert.Error(t, err)
	})

	t.Run("returns error for invalid body interpolation", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetOption("strict", true)

		def := NewRequestDefinition("Create User", "POST", "https://example.com/users")
		def.SetBodyRaw(`{"name": "{{undefined_name}}"}`, "application/json")

		_, err := def.ToRequestWithEnv(engine)
		assert.Error(t, err)
	})
}

func TestCollection_FirstRequest(t *testing.T) {
	t.Run("returns first root request", func(t *testing.T) {
		c := NewCollection("Test")
		req := NewRequestDefinition("First", "GET", "/first")
		c.AddRequest(req)

		first := c.FirstRequest()
		assert.NotNil(t, first)
		assert.Equal(t, "First", first.Name())
	})

	t.Run("returns first request from folder if no root requests", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("Users")
		req := NewRequestDefinition("List Users", "GET", "/users")
		folder.AddRequest(req)

		first := c.FirstRequest()
		assert.NotNil(t, first)
		assert.Equal(t, "List Users", first.Name())
	})

	t.Run("returns nil if no requests", func(t *testing.T) {
		c := NewCollection("Test")
		first := c.FirstRequest()
		assert.Nil(t, first)
	})

	t.Run("returns first request from nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("API")
		nested := folder.AddFolder("Users")
		req := NewRequestDefinition("Nested Request", "GET", "/nested")
		nested.AddRequest(req)

		first := c.FirstRequest()
		assert.NotNil(t, first)
		assert.Equal(t, "Nested Request", first.Name())
	})
}

func TestFolder_FirstRequest(t *testing.T) {
	t.Run("returns first request in folder", func(t *testing.T) {
		f := NewFolder("Users")
		req := NewRequestDefinition("Get User", "GET", "/user")
		f.AddRequest(req)

		first := f.FirstRequest()
		assert.NotNil(t, first)
		assert.Equal(t, "Get User", first.Name())
	})

	t.Run("returns first request from subfolder", func(t *testing.T) {
		f := NewFolder("API")
		sub := f.AddFolder("Users")
		req := NewRequestDefinition("List", "GET", "/list")
		sub.AddRequest(req)

		first := f.FirstRequest()
		assert.NotNil(t, first)
		assert.Equal(t, "List", first.Name())
	})

	t.Run("returns nil if empty", func(t *testing.T) {
		f := NewFolder("Empty")
		assert.Nil(t, f.FirstRequest())
	})
}

func TestRequestDefinition_FullURL(t *testing.T) {
	t.Run("returns URL without params when none set", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://api.example.com/users")
		assert.Equal(t, "https://api.example.com/users", req.FullURL())
	})

	t.Run("appends query params to URL", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "https://api.example.com/users")
		req.SetQueryParam("page", "1")
		req.SetQueryParam("limit", "10")

		fullURL := req.FullURL()
		assert.Contains(t, fullURL, "page=1")
		assert.Contains(t, fullURL, "limit=10")
	})

	t.Run("handles invalid URL gracefully", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "://invalid-url")
		req.SetQueryParam("key", "value")
		// Should return original URL on parse error
		assert.Equal(t, "://invalid-url", req.FullURL())
	})
}

func TestRequestDefinition_QueryParams(t *testing.T) {
	t.Run("GetQueryParam returns empty string for missing key", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "/test")
		assert.Equal(t, "", req.GetQueryParam("missing"))
	})

	t.Run("SetQueryParam and GetQueryParam work together", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "/test")
		req.SetQueryParam("key", "value")
		assert.Equal(t, "value", req.GetQueryParam("key"))
	})

	t.Run("RemoveQueryParam removes parameter", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "/test")
		req.SetQueryParam("key", "value")
		assert.Equal(t, "value", req.GetQueryParam("key"))

		req.RemoveQueryParam("key")
		assert.Equal(t, "", req.GetQueryParam("key"))
	})

	t.Run("RemoveQueryParam on nil map is safe", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "/test")
		// This should not panic
		req.RemoveQueryParam("nonexistent")
		assert.Equal(t, "", req.GetQueryParam("nonexistent"))
	})

	t.Run("QueryParams returns copy of params", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "/test")
		req.SetQueryParam("key1", "value1")
		req.SetQueryParam("key2", "value2")

		params := req.QueryParams()
		assert.Len(t, params, 2)
		assert.Equal(t, "value1", params["key1"])
		assert.Equal(t, "value2", params["key2"])

		// Modifying the returned map should not affect the original
		params["key3"] = "value3"
		assert.Equal(t, "", req.GetQueryParam("key3"))
	})
}

func TestRequestDefinition_SetURL(t *testing.T) {
	t.Run("parses query params from URL", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "")
		req.SetURL("https://api.example.com/users?page=1&limit=10")

		assert.Equal(t, "https://api.example.com/users", req.URL())
		assert.Equal(t, "1", req.GetQueryParam("page"))
		assert.Equal(t, "10", req.GetQueryParam("limit"))
	})

	t.Run("handles URL without query params", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "")
		req.SetURL("https://api.example.com/users")

		assert.Equal(t, "https://api.example.com/users", req.URL())
		assert.Empty(t, req.QueryParams())
	})

	t.Run("handles URL with multiple values for same param", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "")
		req.SetURL("https://api.example.com/search?tag=go&tag=rust")

		assert.Equal(t, "https://api.example.com/search", req.URL())
		// Only first value is used
		assert.Equal(t, "go", req.GetQueryParam("tag"))
	})

	t.Run("handles invalid URL gracefully", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "")
		req.SetURL("://invalid")

		// Should store raw URL when parsing fails
		assert.Equal(t, "://invalid", req.URL())
	})

	t.Run("handles URL with empty query string", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "")
		req.SetURL("https://api.example.com/users?")

		assert.Equal(t, "https://api.example.com/users", req.URL())
	})
}

// Additional comprehensive collection management tests

func TestCollection_SetName(t *testing.T) {
	t.Run("sets collection name", func(t *testing.T) {
		c := NewCollection("Original Name")
		c.SetName("New Name")
		assert.Equal(t, "New Name", c.Name())
	})

	t.Run("updates timestamp on name change", func(t *testing.T) {
		c := NewCollection("Original")
		original := c.UpdatedAt()
		time.Sleep(1 * time.Millisecond)
		c.SetName("Updated")
		assert.True(t, c.UpdatedAt().After(original))
	})
}

func TestCollection_RemoveFolderRecursive(t *testing.T) {
	t.Run("removes folder at root level", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("ToRemove")

		assert.True(t, c.RemoveFolderRecursive(folder.ID()))
		assert.Empty(t, c.Folders())
	})

	t.Run("removes nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		parent := c.AddFolder("Parent")
		child := parent.AddFolder("Child")

		assert.True(t, c.RemoveFolderRecursive(child.ID()))
		assert.Len(t, c.Folders(), 1)
		assert.Empty(t, parent.Folders())
	})

	t.Run("removes deeply nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		level1 := c.AddFolder("Level1")
		level2 := level1.AddFolder("Level2")
		level3 := level2.AddFolder("Level3")

		assert.True(t, c.RemoveFolderRecursive(level3.ID()))
		assert.Empty(t, level2.Folders())
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("Existing")

		assert.False(t, c.RemoveFolderRecursive("non-existent-id"))
	})
}

func TestCollection_FindFolder(t *testing.T) {
	t.Run("finds folder at root level", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("Target")

		found := c.FindFolder(folder.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Target", found.Name())
	})

	t.Run("finds nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		parent := c.AddFolder("Parent")
		child := parent.AddFolder("Child")

		found := c.FindFolder(child.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Child", found.Name())
	})

	t.Run("finds deeply nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		level1 := c.AddFolder("Level1")
		level2 := level1.AddFolder("Level2")
		level3 := level2.AddFolder("Level3")

		found := c.FindFolder(level3.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Level3", found.Name())
	})

	t.Run("returns nil for non-existent folder", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("Existing")

		found := c.FindFolder("non-existent-id")
		assert.Nil(t, found)
	})
}

func TestCollection_RemoveRequestRecursive(t *testing.T) {
	t.Run("removes request at root level", func(t *testing.T) {
		c := NewCollection("Test")
		req := NewRequestDefinition("ToRemove", "GET", "/test")
		c.AddRequest(req)

		assert.True(t, c.RemoveRequestRecursive(req.ID()))
		assert.Empty(t, c.Requests())
	})

	t.Run("removes request from folder", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("Folder")
		req := NewRequestDefinition("InFolder", "GET", "/test")
		folder.AddRequest(req)

		assert.True(t, c.RemoveRequestRecursive(req.ID()))
		assert.Empty(t, folder.Requests())
	})

	t.Run("removes request from nested folder", func(t *testing.T) {
		c := NewCollection("Test")
		parent := c.AddFolder("Parent")
		child := parent.AddFolder("Child")
		req := NewRequestDefinition("Nested", "GET", "/test")
		child.AddRequest(req)

		assert.True(t, c.RemoveRequestRecursive(req.ID()))
		assert.Empty(t, child.Requests())
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, c.RemoveRequestRecursive("non-existent-id"))
	})
}

func TestCollection_MoveRequestUp(t *testing.T) {
	t.Run("moves request up in list", func(t *testing.T) {
		c := NewCollection("Test")
		req1 := NewRequestDefinition("First", "GET", "/first")
		req2 := NewRequestDefinition("Second", "GET", "/second")
		req3 := NewRequestDefinition("Third", "GET", "/third")
		c.AddRequest(req1)
		c.AddRequest(req2)
		c.AddRequest(req3)

		assert.True(t, c.MoveRequestUp(req2.ID()))
		assert.Equal(t, "Second", c.Requests()[0].Name())
		assert.Equal(t, "First", c.Requests()[1].Name())
		assert.Equal(t, "Third", c.Requests()[2].Name())
	})

	t.Run("returns false when already at top", func(t *testing.T) {
		c := NewCollection("Test")
		req := NewRequestDefinition("First", "GET", "/first")
		c.AddRequest(req)
		c.AddRequest(NewRequestDefinition("Second", "GET", "/second"))

		assert.False(t, c.MoveRequestUp(req.ID()))
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, c.MoveRequestUp("non-existent-id"))
	})
}

func TestCollection_MoveRequestDown(t *testing.T) {
	t.Run("moves request down in list", func(t *testing.T) {
		c := NewCollection("Test")
		req1 := NewRequestDefinition("First", "GET", "/first")
		req2 := NewRequestDefinition("Second", "GET", "/second")
		req3 := NewRequestDefinition("Third", "GET", "/third")
		c.AddRequest(req1)
		c.AddRequest(req2)
		c.AddRequest(req3)

		assert.True(t, c.MoveRequestDown(req2.ID()))
		assert.Equal(t, "First", c.Requests()[0].Name())
		assert.Equal(t, "Third", c.Requests()[1].Name())
		assert.Equal(t, "Second", c.Requests()[2].Name())
	})

	t.Run("returns false when already at bottom", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddRequest(NewRequestDefinition("First", "GET", "/first"))
		req := NewRequestDefinition("Last", "GET", "/last")
		c.AddRequest(req)

		assert.False(t, c.MoveRequestDown(req.ID()))
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, c.MoveRequestDown("non-existent-id"))
	})
}

func TestFolder_MoveRequestUp(t *testing.T) {
	t.Run("moves request up in folder", func(t *testing.T) {
		f := NewFolder("Test")
		req1 := NewRequestDefinition("First", "GET", "/first")
		req2 := NewRequestDefinition("Second", "GET", "/second")
		f.AddRequest(req1)
		f.AddRequest(req2)

		assert.True(t, f.MoveRequestUp(req2.ID()))
		assert.Equal(t, "Second", f.Requests()[0].Name())
		assert.Equal(t, "First", f.Requests()[1].Name())
	})

	t.Run("returns false when already at top", func(t *testing.T) {
		f := NewFolder("Test")
		req := NewRequestDefinition("First", "GET", "/first")
		f.AddRequest(req)
		f.AddRequest(NewRequestDefinition("Second", "GET", "/second"))

		assert.False(t, f.MoveRequestUp(req.ID()))
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		f := NewFolder("Test")
		f.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, f.MoveRequestUp("non-existent-id"))
	})
}

func TestFolder_MoveRequestDown(t *testing.T) {
	t.Run("moves request down in folder", func(t *testing.T) {
		f := NewFolder("Test")
		req1 := NewRequestDefinition("First", "GET", "/first")
		req2 := NewRequestDefinition("Second", "GET", "/second")
		f.AddRequest(req1)
		f.AddRequest(req2)

		assert.True(t, f.MoveRequestDown(req1.ID()))
		assert.Equal(t, "Second", f.Requests()[0].Name())
		assert.Equal(t, "First", f.Requests()[1].Name())
	})

	t.Run("returns false when already at bottom", func(t *testing.T) {
		f := NewFolder("Test")
		f.AddRequest(NewRequestDefinition("First", "GET", "/first"))
		req := NewRequestDefinition("Last", "GET", "/last")
		f.AddRequest(req)

		assert.False(t, f.MoveRequestDown(req.ID()))
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		f := NewFolder("Test")
		f.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, f.MoveRequestDown("non-existent-id"))
	})
}

func TestFolder_RemoveFolderRecursive(t *testing.T) {
	t.Run("removes direct child folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := parent.AddFolder("Child")

		assert.True(t, parent.RemoveFolderRecursive(child.ID()))
		assert.Empty(t, parent.Folders())
	})

	t.Run("removes nested folder", func(t *testing.T) {
		level1 := NewFolder("Level1")
		level2 := level1.AddFolder("Level2")
		level3 := level2.AddFolder("Level3")

		assert.True(t, level1.RemoveFolderRecursive(level3.ID()))
		assert.Empty(t, level2.Folders())
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("Child")

		assert.False(t, parent.RemoveFolderRecursive("non-existent-id"))
	})
}

func TestFolder_RemoveRequestRecursive(t *testing.T) {
	t.Run("removes request from direct folder", func(t *testing.T) {
		f := NewFolder("Test")
		req := NewRequestDefinition("ToRemove", "GET", "/test")
		f.AddRequest(req)

		assert.True(t, f.RemoveRequestRecursive(req.ID()))
		assert.Empty(t, f.Requests())
	})

	t.Run("removes request from subfolder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := parent.AddFolder("Child")
		req := NewRequestDefinition("Nested", "GET", "/test")
		child.AddRequest(req)

		assert.True(t, parent.RemoveRequestRecursive(req.ID()))
		assert.Empty(t, child.Requests())
	})

	t.Run("returns false for non-existent request", func(t *testing.T) {
		f := NewFolder("Test")
		f.AddRequest(NewRequestDefinition("Existing", "GET", "/existing"))

		assert.False(t, f.RemoveRequestRecursive("non-existent-id"))
	})
}

func TestFolder_FindFolder(t *testing.T) {
	t.Run("finds direct child folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := parent.AddFolder("Child")

		found := parent.FindFolder(child.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Child", found.Name())
	})

	t.Run("finds nested folder", func(t *testing.T) {
		level1 := NewFolder("Level1")
		level2 := level1.AddFolder("Level2")
		level3 := level2.AddFolder("Level3")

		found := level1.FindFolder(level3.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Level3", found.Name())
	})

	t.Run("returns nil for non-existent folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("Child")

		found := parent.FindFolder("non-existent-id")
		assert.Nil(t, found)
	})
}

func TestCollection_WebSocketsManagement(t *testing.T) {
	t.Run("starts with no websockets", func(t *testing.T) {
		c := NewCollection("Test")
		assert.Empty(t, c.WebSockets())
	})

	t.Run("adds websocket", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)

		assert.Len(t, c.WebSockets(), 1)
	})

	t.Run("gets websocket by ID", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)

		found, ok := c.GetWebSocket("ws-1")
		assert.True(t, ok)
		assert.Equal(t, "Test WebSocket", found.Name)
	})

	t.Run("returns false for non-existent websocket ID", func(t *testing.T) {
		c := NewCollection("Test")
		_, ok := c.GetWebSocket("non-existent")
		assert.False(t, ok)
	})

	t.Run("gets websocket by name", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)

		found, ok := c.GetWebSocketByName("Test WebSocket")
		assert.True(t, ok)
		assert.Equal(t, "ws-1", found.ID)
	})

	t.Run("returns false for non-existent websocket name", func(t *testing.T) {
		c := NewCollection("Test")
		_, ok := c.GetWebSocketByName("Non-existent")
		assert.False(t, ok)
	})

	t.Run("removes websocket", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)
		c.RemoveWebSocket("ws-1")

		assert.Empty(t, c.WebSockets())
	})

	t.Run("remove non-existent websocket does nothing", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)
		c.RemoveWebSocket("non-existent")

		assert.Len(t, c.WebSockets(), 1)
	})

	t.Run("clones websockets", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-1",
			Name:     "Test WebSocket",
			Endpoint: "wss://example.com/ws",
		}
		c.AddWebSocket(ws)

		clone := c.Clone()
		assert.Len(t, clone.WebSockets(), 1)
		// Verify it's a separate copy
		clone.WebSockets()[0].Name = "Modified"
		assert.Equal(t, "Test WebSocket", c.WebSockets()[0].Name)
	})
}

func TestCollection_AddExistingWebSocketMethod(t *testing.T) {
	t.Run("adds existing websocket", func(t *testing.T) {
		c := NewCollection("Test")
		ws := &WebSocketDefinition{
			ID:       "ws-123",
			Name:     "Existing WS",
			Endpoint: "wss://example.com/ws",
		}
		c.AddExistingWebSocket(ws)

		assert.Len(t, c.WebSockets(), 1)
		assert.Equal(t, "ws-123", c.WebSockets()[0].ID)
	})
}

func TestRequestDefinition_FormFields(t *testing.T) {
	t.Run("sets form data body", func(t *testing.T) {
		req := NewRequestDefinition("Upload", "POST", "/upload")
		fields := []FormField{
			{Key: "name", Value: "test"},
			{Key: "description", Value: "A test file"},
		}
		req.SetBodyFormData(fields)

		assert.Equal(t, "form", req.BodyType())
		assert.Len(t, req.FormFields(), 2)
	})

	t.Run("adds form field", func(t *testing.T) {
		req := NewRequestDefinition("Upload", "POST", "/upload")
		req.AddFormField("key1", "value1")
		req.AddFormField("key2", "value2")

		fields := req.FormFields()
		assert.Len(t, fields, 2)
		assert.Equal(t, "key1", fields[0].Key)
		assert.Equal(t, "value1", fields[0].Value)
		assert.False(t, fields[0].IsFile)
	})

	t.Run("adds form file", func(t *testing.T) {
		req := NewRequestDefinition("Upload", "POST", "/upload")
		req.AddFormFile("document", "/path/to/file.pdf")

		fields := req.FormFields()
		assert.Len(t, fields, 1)
		assert.Equal(t, "document", fields[0].Key)
		assert.Equal(t, "/path/to/file.pdf", fields[0].FilePath)
		assert.True(t, fields[0].IsFile)
	})

	t.Run("clears body content when setting form data", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "/test")
		req.SetBody("some raw content")
		req.SetBodyFormData([]FormField{{Key: "key", Value: "value"}})

		assert.Empty(t, req.BodyContent())
		assert.Equal(t, "form", req.BodyType())
	})
}

func TestRequestDefinition_SetBodyType(t *testing.T) {
	t.Run("sets body type to raw", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "/test")
		req.SetBodyType("raw")
		assert.Equal(t, "raw", req.BodyType())
	})

	t.Run("sets body type to json", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "/test")
		req.SetBodyType("json")
		assert.Equal(t, "json", req.BodyType())
	})

	t.Run("sets body type to form", func(t *testing.T) {
		req := NewRequestDefinition("Test", "POST", "/test")
		req.SetBodyType("form")
		assert.Equal(t, "form", req.BodyType())
	})
}

func TestRequestDefinition_ToRequestWithFormBody(t *testing.T) {
	t.Run("converts request with form fields", func(t *testing.T) {
		def := NewRequestDefinition("Upload", "POST", "https://example.com/upload")
		def.SetBodyFormData([]FormField{
			{Key: "name", Value: "test"},
			{Key: "type", Value: "document"},
		})

		req, err := def.ToRequest()
		require.NoError(t, err)
		assert.NotNil(t, req.Body())
		assert.Contains(t, req.Headers().Get("Content-Type"), "multipart/form-data")
	})
}

func TestRequestDefinition_CloneWithFormFields(t *testing.T) {
	t.Run("clones request with form fields", func(t *testing.T) {
		original := NewRequestDefinition("Upload", "POST", "/upload")
		original.AddFormField("key", "value")
		original.AddFormFile("file", "/path/to/file.txt")

		clone := original.Clone()

		assert.Len(t, clone.FormFields(), 2)
		assert.Equal(t, "key", clone.FormFields()[0].Key)
		assert.Equal(t, "file", clone.FormFields()[1].Key)

		// Verify it's a copy
		clone.AddFormField("new", "field")
		assert.Len(t, original.FormFields(), 2)
		assert.Len(t, clone.FormFields(), 3)
	})
}

func TestRequestDefinition_ToRequestWithEnvFormFields(t *testing.T) {
	t.Run("interpolates form field values", func(t *testing.T) {
		engine := interpolate.NewEngine()
		engine.SetVariable("username", "john_doe")
		engine.SetVariable("file_path", "/uploads/test.txt")

		def := NewRequestDefinition("Upload", "POST", "https://example.com/upload")
		def.SetBodyFormData([]FormField{
			{Key: "name", Value: "{{username}}"},
			{Key: "file", Value: "", IsFile: true, FilePath: "{{file_path}}"},
		})

		req, err := def.ToRequestWithEnv(engine)
		require.NoError(t, err)
		assert.NotNil(t, req.Body())
	})
}

func TestCollection_MoveRequestUpDown_Integration(t *testing.T) {
	t.Run("multiple moves maintain order", func(t *testing.T) {
		c := NewCollection("Test")
		req1 := NewRequestDefinition("A", "GET", "/a")
		req2 := NewRequestDefinition("B", "GET", "/b")
		req3 := NewRequestDefinition("C", "GET", "/c")
		req4 := NewRequestDefinition("D", "GET", "/d")
		c.AddRequest(req1)
		c.AddRequest(req2)
		c.AddRequest(req3)
		c.AddRequest(req4)

		// Move D up twice (D -> 2nd position)
		c.MoveRequestUp(req4.ID())
		c.MoveRequestUp(req4.ID())

		assert.Equal(t, "A", c.Requests()[0].Name())
		assert.Equal(t, "D", c.Requests()[1].Name())
		assert.Equal(t, "B", c.Requests()[2].Name())
		assert.Equal(t, "C", c.Requests()[3].Name())

		// Move A down once
		c.MoveRequestDown(req1.ID())

		assert.Equal(t, "D", c.Requests()[0].Name())
		assert.Equal(t, "A", c.Requests()[1].Name())
		assert.Equal(t, "B", c.Requests()[2].Name())
		assert.Equal(t, "C", c.Requests()[3].Name())
	})
}

func TestFolder_GetFolderByID(t *testing.T) {
	t.Run("gets direct subfolder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child1 := parent.AddFolder("Child1")
		parent.AddFolder("Child2")

		// FindFolder is the method to find by ID recursively
		found := parent.FindFolder(child1.ID())
		assert.NotNil(t, found)
		assert.Equal(t, "Child1", found.Name())
	})
}

func TestCollection_RemoveFolder_EdgeCases(t *testing.T) {
	t.Run("remove folder returns true for root folder", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("ToRemove")

		assert.True(t, c.RemoveFolder(folder.ID()))
		assert.Empty(t, c.Folders())
	})

	t.Run("remove folder does not affect nested folders", func(t *testing.T) {
		c := NewCollection("Test")
		parent := c.AddFolder("Parent")
		parent.AddFolder("Child")

		// RemoveFolder only removes from root level
		assert.False(t, c.RemoveFolder("child-id"))
		assert.Len(t, c.Folders(), 1)
	})
}

func TestFolder_RemoveFolder(t *testing.T) {
	t.Run("removes direct child folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := parent.AddFolder("Child")

		assert.True(t, parent.RemoveFolder(child.ID()))
		assert.Empty(t, parent.Folders())
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("Child")

		assert.False(t, parent.RemoveFolder("non-existent"))
		assert.Len(t, parent.Folders(), 1)
	})
}

func TestCollection_MoveFolderUp(t *testing.T) {
	t.Run("moves folder up in list", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("First")
		folder2 := c.AddFolder("Second")
		c.AddFolder("Third")

		assert.True(t, c.MoveFolderUp(folder2.ID()))
		assert.Equal(t, "Second", c.Folders()[0].Name())
		assert.Equal(t, "First", c.Folders()[1].Name())
		assert.Equal(t, "Third", c.Folders()[2].Name())
	})

	t.Run("returns false when already at top", func(t *testing.T) {
		c := NewCollection("Test")
		folder := c.AddFolder("First")
		c.AddFolder("Second")

		assert.False(t, c.MoveFolderUp(folder.ID()))
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("Existing")

		assert.False(t, c.MoveFolderUp("non-existent-id"))
	})
}

func TestCollection_MoveFolderDown(t *testing.T) {
	t.Run("moves folder down in list", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("First")
		folder2 := c.AddFolder("Second")
		c.AddFolder("Third")

		assert.True(t, c.MoveFolderDown(folder2.ID()))
		assert.Equal(t, "First", c.Folders()[0].Name())
		assert.Equal(t, "Third", c.Folders()[1].Name())
		assert.Equal(t, "Second", c.Folders()[2].Name())
	})

	t.Run("returns false when already at bottom", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("First")
		folder := c.AddFolder("Last")

		assert.False(t, c.MoveFolderDown(folder.ID()))
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		c := NewCollection("Test")
		c.AddFolder("Existing")

		assert.False(t, c.MoveFolderDown("non-existent-id"))
	})
}

func TestFolder_MoveFolderUp(t *testing.T) {
	t.Run("moves subfolder up in list", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("First")
		child2 := parent.AddFolder("Second")
		parent.AddFolder("Third")

		assert.True(t, parent.MoveFolderUp(child2.ID()))
		assert.Equal(t, "Second", parent.Folders()[0].Name())
		assert.Equal(t, "First", parent.Folders()[1].Name())
	})

	t.Run("returns false when already at top", func(t *testing.T) {
		parent := NewFolder("Parent")
		child := parent.AddFolder("First")
		parent.AddFolder("Second")

		assert.False(t, parent.MoveFolderUp(child.ID()))
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("Child")

		assert.False(t, parent.MoveFolderUp("non-existent-id"))
	})
}

func TestFolder_MoveFolderDown(t *testing.T) {
	t.Run("moves subfolder down in list", func(t *testing.T) {
		parent := NewFolder("Parent")
		child1 := parent.AddFolder("First")
		parent.AddFolder("Second")
		parent.AddFolder("Third")

		assert.True(t, parent.MoveFolderDown(child1.ID()))
		assert.Equal(t, "Second", parent.Folders()[0].Name())
		assert.Equal(t, "First", parent.Folders()[1].Name())
	})

	t.Run("returns false when already at bottom", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("First")
		child := parent.AddFolder("Last")

		assert.False(t, parent.MoveFolderDown(child.ID()))
	})

	t.Run("returns false for non-existent folder", func(t *testing.T) {
		parent := NewFolder("Parent")
		parent.AddFolder("Child")

		assert.False(t, parent.MoveFolderDown("non-existent-id"))
	})
}

func TestCollection_MoveFolderUpDown_Integration(t *testing.T) {
	t.Run("multiple moves maintain order", func(t *testing.T) {
		c := NewCollection("Test")
		folderA := c.AddFolder("A")
		c.AddFolder("B")
		c.AddFolder("C")
		folderD := c.AddFolder("D")

		// Move D up twice (D -> 2nd position)
		c.MoveFolderUp(folderD.ID())
		c.MoveFolderUp(folderD.ID())

		assert.Equal(t, "A", c.Folders()[0].Name())
		assert.Equal(t, "D", c.Folders()[1].Name())
		assert.Equal(t, "B", c.Folders()[2].Name())
		assert.Equal(t, "C", c.Folders()[3].Name())

		// Move A down once
		c.MoveFolderDown(folderA.ID())

		assert.Equal(t, "D", c.Folders()[0].Name())
		assert.Equal(t, "A", c.Folders()[1].Name())
		assert.Equal(t, "B", c.Folders()[2].Name())
		assert.Equal(t, "C", c.Folders()[3].Name())
	})
}

func TestFolder_SetName(t *testing.T) {
	t.Run("sets folder name", func(t *testing.T) {
		folder := NewFolder("Original")
		assert.Equal(t, "Original", folder.Name())

		folder.SetName("New Name")
		assert.Equal(t, "New Name", folder.Name())
	})
}

func TestRequestDefinition_SetName(t *testing.T) {
	t.Run("sets request name", func(t *testing.T) {
		req := NewRequestDefinition("Original Request", "GET", "http://example.com")
		assert.Equal(t, "Original Request", req.Name())

		req.SetName("Updated Request")
		assert.Equal(t, "Updated Request", req.Name())
	})
}

func TestRequestDefinition_QueryParamsNil(t *testing.T) {
	t.Run("QueryParams returns empty map when nil", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		// New request has nil queryParams
		params := req.QueryParams()
		assert.NotNil(t, params)
		assert.Len(t, params, 0)
	})

	t.Run("GetQueryParam returns empty string when nil", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		// New request has nil queryParams
		value := req.GetQueryParam("nonexistent")
		assert.Equal(t, "", value)
	})

	t.Run("SetQueryParam initializes map when nil", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		// New request has nil queryParams
		req.SetQueryParam("key", "value")
		assert.Equal(t, "value", req.GetQueryParam("key"))
	})

	t.Run("RemoveQueryParam is safe when nil", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		// New request has nil queryParams
		req.RemoveQueryParam("nonexistent")
		// Should not panic
		assert.NotNil(t, req)
	})
}

func TestRequestDefinition_QueryParamsWithValues(t *testing.T) {
	t.Run("QueryParams returns copy of params", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com?page=1&size=10")
		params := req.QueryParams()

		// Modifying returned map should not affect original
		params["extra"] = "value"
		assert.Equal(t, "", req.GetQueryParam("extra"))
	})

	t.Run("SetQueryParam updates existing", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		req.SetQueryParam("key", "value1")
		req.SetQueryParam("key", "value2")
		assert.Equal(t, "value2", req.GetQueryParam("key"))
	})

	t.Run("RemoveQueryParam removes existing", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		req.SetQueryParam("key", "value")
		assert.Equal(t, "value", req.GetQueryParam("key"))

		req.RemoveQueryParam("key")
		assert.Equal(t, "", req.GetQueryParam("key"))
	})

	t.Run("GetQueryParam returns empty for non-existent key", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		assert.Equal(t, "", req.GetQueryParam("nonexistent"))
	})

	t.Run("SetQueryParam with empty value", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		req.SetQueryParam("key", "")
		params := req.QueryParams()
		_, exists := params["key"]
		assert.True(t, exists)
	})

	t.Run("RemoveQueryParam for non-existent key", func(t *testing.T) {
		req := NewRequestDefinition("Test", "GET", "http://example.com")
		req.RemoveQueryParam("nonexistent") // Should not panic
	})
}

func TestRequestDefinition_QueryParamCoverage(t *testing.T) {
	t.Run("GetQueryParam with nil queryParams", func(t *testing.T) {
		req := &RequestDefinition{
			id:     "test-id",
			name:   "Test",
			method: "GET",
			url:    "http://example.com",
			// queryParams is nil
		}
		assert.Equal(t, "", req.GetQueryParam("anykey"))
	})

	t.Run("SetQueryParam initializes nil queryParams", func(t *testing.T) {
		req := &RequestDefinition{
			id:     "test-id",
			name:   "Test",
			method: "GET",
			url:    "http://example.com",
			// queryParams is nil
		}
		req.SetQueryParam("key", "value")
		assert.Equal(t, "value", req.GetQueryParam("key"))
	})

	t.Run("QueryParams with nil returns empty map", func(t *testing.T) {
		req := &RequestDefinition{
			id:     "test-id",
			name:   "Test",
			method: "GET",
			url:    "http://example.com",
			// queryParams is nil
		}
		params := req.QueryParams()
		assert.NotNil(t, params)
		assert.Len(t, params, 0)
	})
}

