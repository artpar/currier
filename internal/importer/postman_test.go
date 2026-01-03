package importer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostmanImporter_Name(t *testing.T) {
	imp := NewPostmanImporter()
	assert.Equal(t, "Postman Collection", imp.Name())
}

func TestPostmanImporter_Format(t *testing.T) {
	imp := NewPostmanImporter()
	assert.Equal(t, FormatPostman, imp.Format())
}

func TestPostmanImporter_FileExtensions(t *testing.T) {
	imp := NewPostmanImporter()
	exts := imp.FileExtensions()
	assert.Contains(t, exts, ".json")
	assert.Contains(t, exts, ".postman_collection.json")
}

func TestPostmanImporter_DetectFormat(t *testing.T) {
	imp := NewPostmanImporter()

	t.Run("detects v2.0 schema", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.0.0/collection.json"
			}
		}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("detects v2.1 schema", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			}
		}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("rejects non-postman JSON", func(t *testing.T) {
		content := []byte(`{"openapi": "3.0.0"}`)
		assert.False(t, imp.DetectFormat(content))
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		content := []byte(`not valid json`)
		assert.False(t, imp.DetectFormat(content))
	})
}

func TestPostmanImporter_Import_BasicCollection(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "My API",
			"description": "API Description",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": []
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)
	assert.Equal(t, "My API", coll.Name())
	assert.Equal(t, "API Description", coll.Description())
}

func TestPostmanImporter_Import_WithVariables(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"variable": [
			{"key": "base_url", "value": "https://api.example.com"},
			{"key": "api_key", "value": "secret123"}
		],
		"item": []
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", coll.GetVariable("base_url"))
	assert.Equal(t, "secret123", coll.GetVariable("api_key"))
}

func TestPostmanImporter_Import_WithRequests(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Get Users",
				"request": {
					"method": "GET",
					"url": "https://api.example.com/users",
					"header": [
						{"key": "Accept", "value": "application/json"}
					]
				}
			},
			{
				"name": "Create User",
				"request": {
					"method": "POST",
					"url": "https://api.example.com/users",
					"header": [
						{"key": "Content-Type", "value": "application/json"}
					],
					"body": {
						"mode": "raw",
						"raw": "{\"name\": \"John\"}"
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 2)

	assert.Equal(t, "Get Users", requests[0].Name())
	assert.Equal(t, "GET", requests[0].Method())
	assert.Equal(t, "https://api.example.com/users", requests[0].URL())
	assert.Equal(t, "application/json", requests[0].GetHeader("Accept"))

	assert.Equal(t, "Create User", requests[1].Name())
	assert.Equal(t, "POST", requests[1].Method())
	assert.Equal(t, "{\"name\": \"John\"}", requests[1].Body())
}

func TestPostmanImporter_Import_WithFolders(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Users",
				"description": "User endpoints",
				"item": [
					{
						"name": "Get Users",
						"request": {
							"method": "GET",
							"url": "https://api.example.com/users"
						}
					}
				]
			},
			{
				"name": "Posts",
				"item": [
					{
						"name": "Get Posts",
						"request": {
							"method": "GET",
							"url": "https://api.example.com/posts"
						}
					}
				]
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	folders := coll.Folders()
	require.Len(t, folders, 2)

	assert.Equal(t, "Users", folders[0].Name())
	assert.Equal(t, "User endpoints", folders[0].Description())
	require.Len(t, folders[0].Requests(), 1)
	assert.Equal(t, "Get Users", folders[0].Requests()[0].Name())

	assert.Equal(t, "Posts", folders[1].Name())
	require.Len(t, folders[1].Requests(), 1)
}

func TestPostmanImporter_Import_NestedFolders(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "API",
				"item": [
					{
						"name": "v1",
						"item": [
							{
								"name": "Get Data",
								"request": {
									"method": "GET",
									"url": "https://api.example.com/v1/data"
								}
							}
						]
					}
				]
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	folders := coll.Folders()
	require.Len(t, folders, 1)
	assert.Equal(t, "API", folders[0].Name())

	subFolders := folders[0].Folders()
	require.Len(t, subFolders, 1)
	assert.Equal(t, "v1", subFolders[0].Name())

	requests := subFolders[0].Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "Get Data", requests[0].Name())
}

func TestPostmanImporter_Import_WithAuth(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	t.Run("bearer auth", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"auth": {
				"type": "bearer",
				"bearer": [
					{"key": "token", "value": "my-token"}
				]
			},
			"item": []
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)
		assert.Equal(t, "bearer", coll.Auth().Type)
		assert.Equal(t, "my-token", coll.Auth().Token)
	})

	t.Run("basic auth", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"auth": {
				"type": "basic",
				"basic": [
					{"key": "username", "value": "user"},
					{"key": "password", "value": "pass"}
				]
			},
			"item": []
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)
		assert.Equal(t, "basic", coll.Auth().Type)
		assert.Equal(t, "user", coll.Auth().Username)
		assert.Equal(t, "pass", coll.Auth().Password)
	})

	t.Run("apikey auth", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"auth": {
				"type": "apikey",
				"apikey": [
					{"key": "key", "value": "X-API-Key"},
					{"key": "value", "value": "secret123"},
					{"key": "in", "value": "header"}
				]
			},
			"item": []
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)
		assert.Equal(t, "apikey", coll.Auth().Type)
		assert.Equal(t, "X-API-Key", coll.Auth().Key)
		assert.Equal(t, "secret123", coll.Auth().Value)
		assert.Equal(t, "header", coll.Auth().In)
	})
}

func TestPostmanImporter_Import_WithScripts(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"event": [
			{
				"listen": "prerequest",
				"script": {
					"exec": ["console.log('pre-request');", "pm.variables.set('x', 1);"]
				}
			},
			{
				"listen": "test",
				"script": {
					"exec": ["pm.test('status', function() {", "  pm.response.to.have.status(200);", "});"]
				}
			}
		],
		"item": []
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)
	assert.Contains(t, coll.PreScript(), "console.log('pre-request');")
	assert.Contains(t, coll.PostScript(), "pm.test('status'")
}

func TestPostmanImporter_Import_RequestWithScripts(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test Request",
				"event": [
					{
						"listen": "prerequest",
						"script": {
							"exec": ["console.log('before');"]
						}
					},
					{
						"listen": "test",
						"script": {
							"exec": ["console.log('after');"]
						}
					}
				],
				"request": {
					"method": "GET",
					"url": "https://example.com"
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].PreScript(), "console.log('before');")
	assert.Contains(t, requests[0].PostScript(), "console.log('after');")
}

func TestPostmanImporter_Import_URLObject(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test",
				"request": {
					"method": "GET",
					"url": {
						"raw": "https://api.example.com/users/1",
						"protocol": "https",
						"host": ["api", "example", "com"],
						"path": ["users", "1"]
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "https://api.example.com/users/1", requests[0].URL())
}

func TestPostmanImporter_Import_DisabledHeaders(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test",
				"request": {
					"method": "GET",
					"url": "https://example.com",
					"header": [
						{"key": "Active", "value": "yes"},
						{"key": "Disabled", "value": "no", "disabled": true}
					]
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "yes", requests[0].GetHeader("Active"))
	assert.Empty(t, requests[0].GetHeader("Disabled"))
}

func TestPostmanImporter_Import_URLEncodedBody(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test",
				"request": {
					"method": "POST",
					"url": "https://example.com",
					"body": {
						"mode": "urlencoded",
						"urlencoded": [
							{"key": "username", "value": "john"},
							{"key": "password", "value": "secret"},
							{"key": "disabled", "value": "skip", "disabled": true}
						]
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "username=john&password=secret", requests[0].Body())
}

func TestPostmanImporter_Import_InvalidJSON(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`not valid json`)

	_, err := imp.Import(ctx, content)
	assert.ErrorIs(t, err, ErrParseError)
}

func TestPostmanImporter_Import_URLObjectWithParts(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	t.Run("builds URL from parts without raw", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"item": [
				{
					"name": "Test",
					"request": {
						"method": "GET",
						"url": {
							"protocol": "https",
							"host": ["api", "example", "com"],
							"path": ["users", "1"]
						}
					}
				}
			]
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		requests := coll.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "https://api.example.com/users/1", requests[0].URL())
	})

	t.Run("builds URL with port", func(t *testing.T) {
		content := []byte(`{
			"info": {
				"name": "Test",
				"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
			},
			"item": [
				{
					"name": "Test",
					"request": {
						"method": "GET",
						"url": {
							"protocol": "http",
							"host": ["localhost"],
							"port": "8080",
							"path": ["api", "users"]
						}
					}
				}
			]
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		requests := coll.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, "http://localhost:8080/api/users", requests[0].URL())
	})
}

func TestPostmanImporter_Import_RequestAuth(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Test Request",
				"request": {
					"method": "GET",
					"url": "https://example.com",
					"auth": {
						"type": "bearer",
						"bearer": [
							{"key": "token", "value": "request-token"}
						]
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "bearer", requests[0].Auth().Type)
	assert.Equal(t, "request-token", requests[0].Auth().Token)
}

func TestPostmanImporter_Import_OAuth2Auth(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"auth": {
			"type": "oauth2",
			"oauth2": [
				{"key": "grant_type", "value": "client_credentials"},
				{"key": "accessToken", "value": "access-token-123"},
				{"key": "accessTokenUrl", "value": "https://auth.example.com/token"},
				{"key": "clientId", "value": "my-client-id"},
				{"key": "clientSecret", "value": "my-client-secret"},
				{"key": "scope", "value": "read write"},
				{"key": "addTokenTo", "value": "header"}
			]
		},
		"item": []
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "oauth2", coll.Auth().Type)
	require.NotNil(t, coll.Auth().OAuth2)
	assert.Equal(t, "client_credentials", string(coll.Auth().OAuth2.GrantType))
	assert.Equal(t, "access-token-123", coll.Auth().OAuth2.AccessToken)
	assert.Equal(t, "https://auth.example.com/token", coll.Auth().OAuth2.TokenURL)
	assert.Equal(t, "my-client-id", coll.Auth().OAuth2.ClientID)
	assert.Equal(t, "my-client-secret", coll.Auth().OAuth2.ClientSecret)
	assert.Equal(t, "read write", coll.Auth().OAuth2.Scope)
	assert.Equal(t, "header", coll.Auth().OAuth2.AddTokenTo)
}

func TestPostmanImporter_Import_AWSAuth(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"auth": {
			"type": "awsv4",
			"awsv4": [
				{"key": "accessKey", "value": "AKIAIOSFODNN7EXAMPLE"},
				{"key": "secretKey", "value": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
				{"key": "region", "value": "us-east-1"},
				{"key": "service", "value": "s3"},
				{"key": "sessionToken", "value": "session-token-123"}
			]
		},
		"item": []
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "awsv4", coll.Auth().Type)
	require.NotNil(t, coll.Auth().AWS)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", coll.Auth().AWS.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", coll.Auth().AWS.SecretAccessKey)
	assert.Equal(t, "us-east-1", coll.Auth().AWS.Region)
	assert.Equal(t, "s3", coll.Auth().AWS.Service)
	assert.Equal(t, "session-token-123", coll.Auth().AWS.SessionToken)
}

func TestPostmanImporter_Import_RequestOAuth2Auth(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "OAuth Request",
				"request": {
					"method": "GET",
					"url": "https://api.example.com/data",
					"auth": {
						"type": "oauth2",
						"oauth2": [
							{"key": "accessToken", "value": "request-oauth-token"},
							{"key": "tokenType", "value": "Bearer"}
						]
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "oauth2", requests[0].Auth().Type)
	require.NotNil(t, requests[0].Auth().OAuth2)
	assert.Equal(t, "request-oauth-token", requests[0].Auth().OAuth2.AccessToken)
	assert.Equal(t, "Bearer", requests[0].Auth().OAuth2.TokenType)
}

func TestPostmanImporter_Import_FormDataBody(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Upload File",
				"request": {
					"method": "POST",
					"url": "https://example.com/upload",
					"body": {
						"mode": "formdata",
						"formdata": [
							{"key": "file", "type": "file", "src": "/path/to/file.txt"},
							{"key": "name", "type": "text", "value": "document.txt"}
						]
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "POST", requests[0].Method())
	// Body should contain the formdata fields as JSON
	assert.Contains(t, requests[0].Body(), "file")
}

func TestPostmanImporter_Import_GraphQLBody(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "GraphQL Query",
				"request": {
					"method": "POST",
					"url": "https://example.com/graphql",
					"body": {
						"mode": "graphql",
						"graphql": {
							"query": "query GetUser { user(id: 1) { name email } }",
							"variables": "{\"id\": 123}"
						}
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "POST", requests[0].Method())
	// Body should contain the GraphQL query
	assert.Contains(t, requests[0].Body(), "query")
	assert.Contains(t, requests[0].Body(), "GetUser")
}

func TestPostmanImporter_Import_GraphQLBodyWithoutVariables(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Simple GraphQL",
				"request": {
					"method": "POST",
					"url": "https://example.com/graphql",
					"body": {
						"mode": "graphql",
						"graphql": {
							"query": "{ users { id name } }"
						}
					}
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body(), "users")
}

func TestPostmanImporter_Import_WithPreAndPostScripts(t *testing.T) {
	imp := NewPostmanImporter()
	ctx := context.Background()

	content := []byte(`{
		"info": {
			"name": "Test",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Request with Scripts",
				"event": [
					{
						"listen": "prerequest",
						"script": {
							"type": "text/javascript",
							"exec": ["console.log('pre-request');", "pm.environment.set('var', 'value');"]
						}
					},
					{
						"listen": "test",
						"script": {
							"type": "text/javascript",
							"exec": ["pm.test('status', function() {", "  pm.response.to.have.status(200);", "});"]
						}
					}
				],
				"request": {
					"method": "GET",
					"url": "https://example.com/api"
				}
			}
		]
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].PreScript(), "pre-request")
	assert.Contains(t, requests[0].PostScript(), "pm.test")
}
