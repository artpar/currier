package script

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests simulate real-world script execution scenarios.

func TestIntegration_PreRequestScript(t *testing.T) {
	t.Run("pre-request script adds auth header", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestMethod("GET")
		scope.SetRequestURL("https://api.example.com/users")
		scope.SetVariable("access_token", "my-secret-token")

		script := `
			var token = currier.getVariable("access_token");
			currier.request.setHeader("Authorization", "Bearer " + token);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		headers := scope.GetRequestHeaders()
		assert.Equal(t, "Bearer my-secret-token", headers["Authorization"])
	})

	t.Run("pre-request script generates timestamp", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestMethod("POST")
		scope.SetRequestURL("https://api.example.com/events")

		script := `
			var timestamp = Date.now();
			currier.setVariable("request_timestamp", String(timestamp));
			currier.request.setHeader("X-Timestamp", String(timestamp));
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		ts := scope.GetVariable("request_timestamp")
		assert.NotEmpty(t, ts)
		headers := scope.GetRequestHeaders()
		assert.Equal(t, ts, headers["X-Timestamp"])
	})

	t.Run("pre-request script signs request body", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestMethod("POST")
		scope.SetRequestBody(`{"user": "john", "action": "login"}`)
		scope.SetVariable("secret_key", "my-secret")

		script := `
			var body = currier.request.body;
			var secret = currier.getVariable("secret_key");
			var signature = currier.crypto.hmac("sha256", secret, body);
			currier.request.setHeader("X-Signature", signature);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		headers := scope.GetRequestHeaders()
		assert.NotEmpty(t, headers["X-Signature"])
		// Verify it's a valid hex string (64 chars for SHA256)
		assert.Len(t, headers["X-Signature"], 64)
	})

	t.Run("pre-request script modifies URL with variable", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestMethod("GET")
		scope.SetRequestURL("https://api.example.com/users/{id}")
		scope.SetVariable("user_id", "12345")

		script := `
			var url = currier.request.url;
			var userId = currier.getVariable("user_id");
			url = url.replace("{id}", userId);
			currier.request.setUrl(url);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "https://api.example.com/users/12345", scope.GetRequestURL())
	})
}

func TestIntegration_PostResponseScript(t *testing.T) {
	t.Run("post-response script extracts token from response", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{"token": "abc123xyz", "expires_in": 3600}`)

		script := `
			var data = currier.response.json();
			currier.setVariable("auth_token", data.token);
			currier.setVariable("token_expires", String(data.expires_in));
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "abc123xyz", scope.GetVariable("auth_token"))
		assert.Equal(t, "3600", scope.GetVariable("token_expires"))
	})

	t.Run("post-response script handles nested JSON", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{
			"user": {
				"id": 1,
				"profile": {
					"email": "john@example.com",
					"name": "John Doe"
				}
			}
		}`)

		script := `
			var data = currier.response.json();
			currier.setVariable("user_email", data.user.profile.email);
			currier.setVariable("user_name", data.user.profile.name);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "john@example.com", scope.GetVariable("user_email"))
		assert.Equal(t, "John Doe", scope.GetVariable("user_name"))
	})

	t.Run("post-response script extracts from array response", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`[
			{"id": 1, "name": "Item 1"},
			{"id": 2, "name": "Item 2"},
			{"id": 3, "name": "Item 3"}
		]`)

		script := `
			var items = currier.response.json();
			currier.setVariable("item_count", String(items.length));
			currier.setVariable("first_item_name", items[0].name);
			currier.setVariable("last_item_id", String(items[items.length - 1].id));
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "3", scope.GetVariable("item_count"))
		assert.Equal(t, "Item 1", scope.GetVariable("first_item_name"))
		assert.Equal(t, "3", scope.GetVariable("last_item_id"))
	})

	t.Run("post-response script handles error response", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(401)
		scope.SetResponseBody(`{"error": "Unauthorized", "message": "Invalid token"}`)

		script := `
			if (currier.response.status === 401) {
				var error = currier.response.json();
				currier.setVariable("error_type", error.error);
				currier.setVariable("error_message", error.message);
			}
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "Unauthorized", scope.GetVariable("error_type"))
		assert.Equal(t, "Invalid token", scope.GetVariable("error_message"))
	})
}

func TestIntegration_TestAssertions(t *testing.T) {
	t.Run("complete test suite for API response", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)
		scope.SetResponseTime(150)
		scope.SetResponseBody(`{
			"success": true,
			"data": {
				"users": [
					{"id": 1, "name": "Alice", "email": "alice@example.com"},
					{"id": 2, "name": "Bob", "email": "bob@example.com"}
				],
				"total": 2,
				"page": 1
			}
		}`)
		scope.SetResponseHeaders(map[string]string{
			"Content-Type":  "application/json",
			"X-RateLimit":   "100",
			"X-Request-ID":  "abc-123",
		})

		script := `
			currier.test("Status is 200", function() {
				currier.expect(currier.response.status).toBe(200);
			});

			currier.test("Response time is under 500ms", function() {
				currier.expect(currier.response.time).toBeLessThan(500);
			});

			currier.test("Response is successful", function() {
				var data = currier.response.json();
				currier.expect(data.success).toBe(true);
			});

			currier.test("Has correct number of users", function() {
				var data = currier.response.json();
				currier.expect(data.data.users).toHaveLength(2);
				currier.expect(data.data.total).toBe(2);
			});

			currier.test("First user is Alice", function() {
				var data = currier.response.json();
				currier.expect(data.data.users[0].name).toBe("Alice");
				currier.expect(data.data.users[0].email).toContain("@example.com");
			});

			currier.test("Content-Type is JSON", function() {
				currier.expect(currier.response.headers["Content-Type"]).toBe("application/json");
			});

			currier.test("Has rate limit header", function() {
				currier.expect(currier.response.headers["X-RateLimit"]).toBeDefined();
			});
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 7)
		for _, r := range results {
			assert.True(t, r.Passed, "Test '%s' should pass", r.Name)
		}
		assert.True(t, scope.AllTestsPassed())
	})

	t.Run("test suite with some failures", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(404)
		scope.SetResponseBody(`{"error": "Not found"}`)

		script := `
			currier.test("Status is 200", function() {
				currier.expect(currier.response.status).toBe(200);
			});

			currier.test("Status is not 500", function() {
				currier.expect(currier.response.status).not.toBe(500);
			});

			currier.test("Has error message", function() {
				var data = currier.response.json();
				currier.expect(data.error).toBe("Not found");
			});
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 3)

		// First test should fail (404 != 200)
		assert.False(t, results[0].Passed)
		assert.Contains(t, results[0].Error, "200")

		// Second and third tests should pass
		assert.True(t, results[1].Passed)
		assert.True(t, results[2].Passed)

		summary := scope.GetTestSummary()
		assert.Equal(t, 3, summary.Total)
		assert.Equal(t, 2, summary.Passed)
		assert.Equal(t, 1, summary.Failed)
	})
}

func TestIntegration_EnvironmentVariables(t *testing.T) {
	t.Run("script accesses environment variables", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetEnvironmentName("production")
		scope.SetEnvironmentVariable("API_URL", "https://api.prod.example.com")
		scope.SetEnvironmentVariable("API_KEY", "prod-secret-key")

		script := `
			var env = currier.environment.name;
			var apiUrl = currier.environment.get("API_URL");
			var apiKey = currier.environment.get("API_KEY");

			currier.setVariable("env_name", env);
			currier.setVariable("base_url", apiUrl);

			currier.request.setHeader("X-API-Key", apiKey);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "production", scope.GetVariable("env_name"))
		assert.Equal(t, "https://api.prod.example.com", scope.GetVariable("base_url"))
		headers := scope.GetRequestHeaders()
		assert.Equal(t, "prod-secret-key", headers["X-API-Key"])
	})

	t.Run("script modifies environment variables", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetEnvironmentName("test")

		script := `
			currier.environment.set("NEW_VAR", "new-value");
			currier.environment.set("COUNTER", "1");
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		assert.Equal(t, "new-value", scope.GetEnvironmentVariable("NEW_VAR"))
		assert.Equal(t, "1", scope.GetEnvironmentVariable("COUNTER"))
	})
}

func TestIntegration_UtilityFunctions(t *testing.T) {
	t.Run("base64 encoding in script", func(t *testing.T) {
		scope := NewScopeWithAssertions()

		script := `
			var credentials = "user:password";
			var encoded = currier.base64.encode(credentials);
			currier.request.setHeader("Authorization", "Basic " + encoded);

			var decoded = currier.base64.decode(encoded);
			currier.setVariable("decoded_creds", decoded);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		headers := scope.GetRequestHeaders()
		assert.Equal(t, "Basic dXNlcjpwYXNzd29yZA==", headers["Authorization"])
		assert.Equal(t, "user:password", scope.GetVariable("decoded_creds"))
	})

	t.Run("crypto hashing in script", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetRequestBody(`{"data": "test"}`)

		script := `
			var body = currier.request.body;
			var md5Hash = currier.crypto.md5(body);
			var sha256Hash = currier.crypto.sha256(body);

			currier.request.setHeader("Content-MD5", md5Hash);
			currier.request.setHeader("X-Content-SHA256", sha256Hash);
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		headers := scope.GetRequestHeaders()
		assert.Len(t, headers["Content-MD5"], 32)   // MD5 is 32 hex chars
		assert.Len(t, headers["X-Content-SHA256"], 64) // SHA256 is 64 hex chars
	})
}

func TestIntegration_Logging(t *testing.T) {
	t.Run("script logs messages", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		var logs []string
		scope.SetLogHandler(func(msg string) {
			logs = append(logs, msg)
		})
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{"id": 123}`)

		script := `
			currier.log("Starting post-response script");
			var data = currier.response.json();
			currier.log("Received data with id:", data.id);
			currier.log("Script completed");
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		require.Len(t, logs, 3)
		assert.Equal(t, "Starting post-response script", logs[0])
		assert.Equal(t, "Received data with id: 123", logs[1])
		assert.Equal(t, "Script completed", logs[2])
	})
}

func TestIntegration_SandboxedExecution(t *testing.T) {
	t.Run("sandboxed scope prevents dangerous operations", func(t *testing.T) {
		scope := NewSandboxedScope()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{"test": true}`)

		// Try to access dangerous globals
		script := `
			currier.test("require is not available", function() {
				currier.expect(typeof require).toBe("undefined");
			});

			currier.test("process is not available", function() {
				currier.expect(typeof process).toBe("undefined");
			});

			currier.test("Response is accessible", function() {
				currier.expect(currier.response.status).toBe(200);
			});
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 3)
		for _, r := range results {
			assert.True(t, r.Passed, "Test '%s' should pass", r.Name)
		}
	})

	t.Run("sandboxed scope times out on infinite loop", func(t *testing.T) {
		scope := NewSandboxedScope()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := scope.Execute(ctx, `while(true) {}`)

		assert.Error(t, err)
	})

	t.Run("sandboxed scope blocks eval", func(t *testing.T) {
		scope := NewSandboxedScope()
		scope.DisableEval()

		_, err := scope.Execute(context.Background(), `eval("1+1")`)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "eval")
	})
}

func TestIntegration_CompleteRequestFlow(t *testing.T) {
	t.Run("simulates complete pre-request and post-response flow", func(t *testing.T) {
		// Simulate a login request flow
		scope := NewScopeWithAssertions()

		// Set up initial state
		scope.SetEnvironmentName("staging")
		scope.SetEnvironmentVariable("BASE_URL", "https://api.staging.example.com")
		scope.SetVariable("username", "testuser")
		scope.SetVariable("password", "testpass123")

		// Pre-request script
		preRequestScript := `
			var baseUrl = currier.environment.get("BASE_URL");
			var username = currier.getVariable("username");
			var password = currier.getVariable("password");

			// Set request URL
			currier.request.setUrl(baseUrl + "/auth/login");

			// Create auth header
			var creds = username + ":" + password;
			var encoded = currier.base64.encode(creds);
			currier.request.setHeader("Authorization", "Basic " + encoded);

			// Add request metadata
			currier.request.setHeader("X-Request-Time", String(Date.now()));
			currier.request.setHeader("Content-Type", "application/json");

			currier.log("Pre-request: Setting up login request");
		`

		var logs []string
		scope.SetLogHandler(func(msg string) {
			logs = append(logs, msg)
		})

		_, err := scope.Execute(context.Background(), preRequestScript)
		require.NoError(t, err)

		// Verify pre-request modifications
		assert.Equal(t, "https://api.staging.example.com/auth/login", scope.GetRequestURL())
		headers := scope.GetRequestHeaders()
		assert.Contains(t, headers["Authorization"], "Basic ")
		assert.Equal(t, "application/json", headers["Content-Type"])

		// Simulate response (in real usage, this would come from HTTP client)
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`{
			"success": true,
			"token": "jwt-token-12345",
			"user": {
				"id": 1,
				"username": "testuser",
				"roles": ["user", "admin"]
			},
			"expires_in": 3600
		}`)
		scope.SetResponseTime(234)
		scope.SetResponseHeaders(map[string]string{
			"Content-Type":    "application/json",
			"X-RateLimit-Remaining": "99",
		})

		// Post-response script with tests
		postResponseScript := `
			currier.log("Post-response: Processing login response");

			// Run tests
			currier.test("Login successful", function() {
				currier.expect(currier.response.status).toBe(200);
				currier.expect(currier.response.json().success).toBe(true);
			});

			currier.test("Response time acceptable", function() {
				currier.expect(currier.response.time).toBeLessThan(500);
			});

			currier.test("Token received", function() {
				var data = currier.response.json();
				currier.expect(data.token).toBeDefined();
				currier.expect(data.token).toMatch(/^jwt-/);
			});

			currier.test("User has admin role", function() {
				var roles = currier.response.json().user.roles;
				currier.expect(roles).toContain("admin");
			});

			// Extract and store token
			var data = currier.response.json();
			currier.setVariable("auth_token", data.token);
			currier.setVariable("user_id", String(data.user.id));
			currier.environment.set("TOKEN_EXPIRES", String(Date.now() + data.expires_in * 1000));

			currier.log("Post-response: Token stored for future requests");
		`

		_, err = scope.Execute(context.Background(), postResponseScript)
		require.NoError(t, err)

		// Verify all tests passed
		results := scope.GetTestResults()
		require.Len(t, results, 4)
		for _, r := range results {
			assert.True(t, r.Passed, "Test '%s' should pass: %s", r.Name, r.Error)
		}

		// Verify extracted data
		assert.Equal(t, "jwt-token-12345", scope.GetVariable("auth_token"))
		assert.Equal(t, "1", scope.GetVariable("user_id"))
		assert.NotEmpty(t, scope.GetEnvironmentVariable("TOKEN_EXPIRES"))

		// Verify logging
		assert.Contains(t, logs, "Pre-request: Setting up login request")
		assert.Contains(t, logs, "Post-response: Processing login response")
		assert.Contains(t, logs, "Post-response: Token stored for future requests")
	})
}

func TestIntegration_ChainedRequests(t *testing.T) {
	t.Run("simulates chained request flow with variable sharing", func(t *testing.T) {
		// First request: Get auth token
		scope1 := NewScopeWithAssertions()
		scope1.SetResponseStatus(200)
		scope1.SetResponseBody(`{"token": "first-token-123"}`)

		_, err := scope1.Execute(context.Background(), `
			var token = currier.response.json().token;
			currier.setVariable("auth_token", token);
		`)
		require.NoError(t, err)

		// Second request: Use token from first request
		scope2 := NewScopeWithAssertions()
		scope2.SetVariable("auth_token", scope1.GetVariable("auth_token"))
		scope2.SetRequestMethod("GET")
		scope2.SetRequestURL("https://api.example.com/protected")

		_, err = scope2.Execute(context.Background(), `
			var token = currier.getVariable("auth_token");
			currier.request.setHeader("Authorization", "Bearer " + token);
		`)
		require.NoError(t, err)

		headers := scope2.GetRequestHeaders()
		assert.Equal(t, "Bearer first-token-123", headers["Authorization"])
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	t.Run("script handles JSON parse error gracefully", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(200)
		scope.SetResponseBody(`not valid json`)

		script := `
			currier.test("Response parsing", function() {
				var data = currier.response.json();
				if (data === null) {
					throw new Error("Failed to parse JSON response");
				}
			});
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 1)
		assert.False(t, results[0].Passed)
		assert.Contains(t, results[0].Error, "parse")
	})

	t.Run("script continues after test failure", func(t *testing.T) {
		scope := NewScopeWithAssertions()
		scope.SetResponseStatus(500)

		script := `
			currier.test("First test fails", function() {
				currier.expect(currier.response.status).toBe(200);
			});

			// This should still run
			currier.setVariable("continued", "yes");

			currier.test("Second test passes", function() {
				currier.expect(true).toBe(true);
			});
		`

		_, err := scope.Execute(context.Background(), script)

		require.NoError(t, err)
		results := scope.GetTestResults()
		require.Len(t, results, 2)
		assert.False(t, results[0].Passed)
		assert.True(t, results[1].Passed)
		assert.Equal(t, "yes", scope.GetVariable("continued"))
	})
}

// Benchmark tests
func BenchmarkIntegration_SimpleExecution(b *testing.B) {
	scope := NewScopeWithAssertions()
	scope.SetResponseStatus(200)
	scope.SetResponseBody(`{"id": 1}`)

	script := `
		var data = currier.response.json();
		currier.setVariable("id", String(data.id));
	`

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope.Execute(ctx, script)
	}
}

func BenchmarkIntegration_TestAssertions(b *testing.B) {
	scope := NewScopeWithAssertions()
	scope.SetResponseStatus(200)
	scope.SetResponseBody(`{"success": true, "data": [1, 2, 3]}`)

	script := `
		currier.test("Status check", function() {
			currier.expect(currier.response.status).toBe(200);
		});
		currier.test("Data check", function() {
			currier.expect(currier.response.json().success).toBe(true);
		});
	`

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope.ClearTestResults()
		scope.Execute(ctx, script)
	}
}

func BenchmarkIntegration_SandboxedExecution(b *testing.B) {
	scope := NewSandboxedScope()
	scope.SetResponseStatus(200)
	scope.SetResponseBody(`{"id": 1}`)

	script := `
		var data = currier.response.json();
		currier.setVariable("id", String(data.id));
	`

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope.Execute(ctx, script)
	}
}

// Helper to convert interface{} to JSON string for debugging
func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
