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
}
