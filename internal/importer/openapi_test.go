package importer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIImporter_Name(t *testing.T) {
	imp := NewOpenAPIImporter()
	assert.Equal(t, "OpenAPI 3.x", imp.Name())
}

func TestOpenAPIImporter_Format(t *testing.T) {
	imp := NewOpenAPIImporter()
	assert.Equal(t, FormatOpenAPI, imp.Format())
}

func TestOpenAPIImporter_FileExtensions(t *testing.T) {
	imp := NewOpenAPIImporter()
	exts := imp.FileExtensions()
	assert.Contains(t, exts, ".yaml")
	assert.Contains(t, exts, ".yml")
	assert.Contains(t, exts, ".json")
}

func TestOpenAPIImporter_DetectFormat(t *testing.T) {
	imp := NewOpenAPIImporter()

	t.Run("detects OpenAPI 3.0 JSON", func(t *testing.T) {
		content := []byte(`{"openapi": "3.0.0", "info": {"title": "Test", "version": "1.0"}}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("detects OpenAPI 3.1 JSON", func(t *testing.T) {
		content := []byte(`{"openapi": "3.1.0", "info": {"title": "Test", "version": "1.0"}}`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("detects OpenAPI 3.0 YAML", func(t *testing.T) {
		content := []byte(`openapi: "3.0.0"
info:
  title: Test
  version: "1.0"`)
		assert.True(t, imp.DetectFormat(content))
	})

	t.Run("rejects Swagger 2.0", func(t *testing.T) {
		content := []byte(`{"swagger": "2.0"}`)
		assert.False(t, imp.DetectFormat(content))
	})

	t.Run("rejects non-OpenAPI", func(t *testing.T) {
		content := []byte(`{"name": "not openapi"}`)
		assert.False(t, imp.DetectFormat(content))
	})

	t.Run("rejects invalid content", func(t *testing.T) {
		content := []byte(`not valid json or yaml`)
		assert.False(t, imp.DetectFormat(content))
	})
}

func TestOpenAPIImporter_Import_BasicSpec(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {
			"title": "Pet Store API",
			"description": "A sample API",
			"version": "1.0.0"
		},
		"paths": {}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "Pet Store API", coll.Name())
	assert.Equal(t, "A sample API", coll.Description())
	assert.Equal(t, "1.0.0", coll.Version())
}

func TestOpenAPIImporter_Import_WithServers(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [
			{"url": "https://api.example.com/v1"}
		],
		"paths": {}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "https://api.example.com/v1", coll.GetVariable("base_url"))
}

func TestOpenAPIImporter_Import_ServerVariables(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [
			{
				"url": "https://{environment}.example.com:{port}/v1",
				"variables": {
					"environment": {"default": "api"},
					"port": {"default": "443", "enum": ["443", "8443"]}
				}
			}
		],
		"paths": {}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "https://api.example.com:443/v1", coll.GetVariable("base_url"))
}

func TestOpenAPIImporter_Import_WithPaths(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"operationId": "listUsers"
				},
				"post": {
					"summary": "Create user",
					"operationId": "createUser"
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 2)

	// Check both requests exist (order may vary)
	methods := map[string]bool{}
	for _, req := range requests {
		methods[req.Method()] = true
		assert.Contains(t, req.URL(), "/users")
	}
	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
}

func TestOpenAPIImporter_Import_WithTags(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"tags": [
			{"name": "users", "description": "User operations"},
			{"name": "posts", "description": "Post operations"}
		],
		"paths": {
			"/users": {
				"get": {
					"tags": ["users"],
					"summary": "List users"
				}
			},
			"/posts": {
				"get": {
					"tags": ["posts"],
					"summary": "List posts"
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	folders := coll.Folders()
	require.Len(t, folders, 2)

	// Find folders by name
	var usersFolder, postsFolder *struct {
		name        string
		description string
		reqCount    int
	}

	for _, f := range folders {
		if f.Name() == "users" {
			usersFolder = &struct {
				name        string
				description string
				reqCount    int
			}{f.Name(), f.Description(), len(f.Requests())}
		}
		if f.Name() == "posts" {
			postsFolder = &struct {
				name        string
				description string
				reqCount    int
			}{f.Name(), f.Description(), len(f.Requests())}
		}
	}

	require.NotNil(t, usersFolder)
	require.NotNil(t, postsFolder)

	assert.Equal(t, "User operations", usersFolder.description)
	assert.Equal(t, 1, usersFolder.reqCount)

	assert.Equal(t, "Post operations", postsFolder.description)
	assert.Equal(t, 1, postsFolder.reqCount)
}

func TestOpenAPIImporter_Import_WithParameters(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/users/{id}": {
				"get": {
					"summary": "Get user",
					"parameters": [
						{
							"name": "id",
							"in": "path",
							"required": true,
							"schema": {"type": "integer"},
							"example": 123
						},
						{
							"name": "include",
							"in": "query",
							"schema": {"type": "string"},
							"example": "profile"
						},
						{
							"name": "X-Request-ID",
							"in": "header",
							"schema": {"type": "string"},
							"example": "abc-123"
						}
					]
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)

	req := requests[0]
	assert.Contains(t, req.URL(), "123") // Path parameter replaced
	assert.Contains(t, req.URL(), "include=profile") // Query parameter
	assert.Equal(t, "abc-123", req.GetHeader("X-Request-ID"))
}

func TestOpenAPIImporter_Import_WithRequestBody(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"name": {"type": "string"},
										"email": {"type": "string", "format": "email"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	require.Len(t, requests, 1)

	req := requests[0]
	assert.Equal(t, "application/json", req.GetHeader("Content-Type"))
	assert.NotEmpty(t, req.Body())
	assert.Contains(t, req.Body(), "name")
	assert.Contains(t, req.Body(), "email")
}

func TestOpenAPIImporter_Import_WithSchemaExample(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"content": {
							"application/json": {
								"example": {"name": "John", "email": "john@example.com"}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.Body(), "John")
	assert.Contains(t, req.Body(), "john@example.com")
}

func TestOpenAPIImporter_Import_WithSecuritySchemes(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	t.Run("bearer auth", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"security": [{"bearerAuth": []}],
			"components": {
				"securitySchemes": {
					"bearerAuth": {
						"type": "http",
						"scheme": "bearer"
					}
				}
			},
			"paths": {}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		assert.Equal(t, "bearer", coll.Auth().Type)
	})

	t.Run("basic auth", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"security": [{"basicAuth": []}],
			"components": {
				"securitySchemes": {
					"basicAuth": {
						"type": "http",
						"scheme": "basic"
					}
				}
			},
			"paths": {}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		assert.Equal(t, "basic", coll.Auth().Type)
	})

	t.Run("api key", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"security": [{"apiKey": []}],
			"components": {
				"securitySchemes": {
					"apiKey": {
						"type": "apiKey",
						"name": "X-API-Key",
						"in": "header"
					}
				}
			},
			"paths": {}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		assert.Equal(t, "apikey", coll.Auth().Type)
		assert.Equal(t, "X-API-Key", coll.Auth().Key)
		assert.Equal(t, "header", coll.Auth().In)
	})
}

func TestOpenAPIImporter_Import_WithComponentRefs(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/User"
								}
							}
						}
					}
				}
			}
		},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"id": {"type": "integer"},
						"name": {"type": "string"}
					},
					"example": {"id": 1, "name": "John Doe"}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.Body(), "John Doe")
}

func TestOpenAPIImporter_Import_DeprecatedOperation(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/old-endpoint": {
				"get": {
					"summary": "Old endpoint",
					"deprecated": true
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.Description(), "[DEPRECATED]")
}

func TestOpenAPIImporter_Import_YAML(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`
openapi: "3.0.0"
info:
  title: YAML API
  version: "1.0.0"
servers:
  - url: https://api.example.com
paths:
  /users:
    get:
      summary: List users
      tags:
        - users
`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	assert.Equal(t, "YAML API", coll.Name())
	assert.Equal(t, "https://api.example.com", coll.GetVariable("base_url"))

	folders := coll.Folders()
	require.Len(t, folders, 1)
	assert.Equal(t, "users", folders[0].Name())
}

func TestOpenAPIImporter_Import_AllMethods(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/resource": {
				"get": {"summary": "Get"},
				"post": {"summary": "Create"},
				"put": {"summary": "Replace"},
				"patch": {"summary": "Update"},
				"delete": {"summary": "Delete"},
				"head": {"summary": "Head"},
				"options": {"summary": "Options"}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	requests := coll.Requests()
	assert.Len(t, requests, 7)

	methods := make(map[string]bool)
	for _, req := range requests {
		methods[req.Method()] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["HEAD"])
	assert.True(t, methods["OPTIONS"])
}

func TestOpenAPIImporter_Import_InvalidVersion(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "2.0.0",
		"info": {"title": "Test", "version": "1.0"}
	}`)

	_, err := imp.Import(ctx, content)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)
}

func TestOpenAPIImporter_Import_InvalidJSON(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`not valid json or yaml {{{`)

	_, err := imp.Import(ctx, content)
	assert.ErrorIs(t, err, ErrParseError)
}

func TestOpenAPIImporter_Import_FormURLEncoded(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/login": {
				"post": {
					"summary": "Login",
					"requestBody": {
						"content": {
							"application/x-www-form-urlencoded": {
								"schema": {
									"type": "object",
									"properties": {
										"username": {"type": "string"},
										"password": {"type": "string"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Equal(t, "application/x-www-form-urlencoded", req.GetHeader("Content-Type"))
	assert.Contains(t, req.Body(), "username=")
	assert.Contains(t, req.Body(), "password=")
}

func TestOpenAPIImporter_Import_WithParameterRef(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"parameters": [
						{"$ref": "#/components/parameters/PageParam"},
						{"$ref": "#/components/parameters/LimitParam"}
					]
				}
			}
		},
		"components": {
			"parameters": {
				"PageParam": {
					"name": "page",
					"in": "query",
					"schema": {"type": "integer"},
					"example": 1
				},
				"LimitParam": {
					"name": "limit",
					"in": "query",
					"schema": {"type": "integer"},
					"example": 10
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.URL(), "page=1")
	assert.Contains(t, req.URL(), "limit=10")
}

func TestOpenAPIImporter_Import_WithRequestBodyRef(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"$ref": "#/components/requestBodies/UserBody"
					}
				}
			}
		},
		"components": {
			"requestBodies": {
				"UserBody": {
					"content": {
						"application/json": {
							"schema": {
								"type": "object",
								"properties": {
									"name": {"type": "string"},
									"age": {"type": "integer"}
								}
							},
							"example": {"name": "Jane", "age": 25}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.Body(), "Jane")
	assert.Contains(t, req.Body(), "25")
}

func TestOpenAPIImporter_Import_SchemaTypes(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	t.Run("generates string formats", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"paths": {
				"/test": {
					"post": {
						"requestBody": {
							"content": {
								"application/json": {
									"schema": {
										"type": "object",
										"properties": {
											"date": {"type": "string", "format": "date"},
											"datetime": {"type": "string", "format": "date-time"},
											"email": {"type": "string", "format": "email"},
											"uuid": {"type": "string", "format": "uuid"}
										}
									}
								}
							}
						}
					}
				}
			}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		body := coll.Requests()[0].Body()
		assert.Contains(t, body, "2024-01-01")
		assert.Contains(t, body, "user@example.com")
		assert.Contains(t, body, "550e8400")
	})

	t.Run("generates primitive types", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"paths": {
				"/test": {
					"post": {
						"requestBody": {
							"content": {
								"application/json": {
									"schema": {
										"type": "object",
										"properties": {
											"count": {"type": "integer"},
											"price": {"type": "number"},
											"active": {"type": "boolean"},
											"name": {"type": "string"}
										}
									}
								}
							}
						}
					}
				}
			}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		body := coll.Requests()[0].Body()
		assert.Contains(t, body, "count")
		assert.Contains(t, body, "price")
		assert.Contains(t, body, "active")
	})

	t.Run("generates array types", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"paths": {
				"/test": {
					"post": {
						"requestBody": {
							"content": {
								"application/json": {
									"schema": {
										"type": "object",
										"properties": {
											"tags": {
												"type": "array",
												"items": {"type": "string"}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		body := coll.Requests()[0].Body()
		assert.Contains(t, body, "tags")
	})

	t.Run("generates enum values", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"paths": {
				"/test": {
					"post": {
						"requestBody": {
							"content": {
								"application/json": {
									"schema": {
										"type": "object",
										"properties": {
											"status": {
												"type": "string",
												"enum": ["active", "inactive", "pending"]
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		body := coll.Requests()[0].Body()
		assert.Contains(t, body, "active")
	})

	t.Run("uses default value", func(t *testing.T) {
		content := []byte(`{
			"openapi": "3.0.0",
			"info": {"title": "Test", "version": "1.0"},
			"paths": {
				"/test": {
					"get": {
						"parameters": [
							{
								"name": "sort",
								"in": "query",
								"schema": {"type": "string", "default": "created_at"}
							}
						]
					}
				}
			}
		}`)

		coll, err := imp.Import(ctx, content)
		require.NoError(t, err)

		url := coll.Requests()[0].URL()
		assert.Contains(t, url, "sort=created_at")
	})
}

func TestOpenAPIImporter_Import_NestedObjects(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/test": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"user": {
											"type": "object",
											"properties": {
												"profile": {
													"type": "object",
													"properties": {
														"bio": {"type": "string"}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	body := coll.Requests()[0].Body()
	assert.Contains(t, body, "user")
	assert.Contains(t, body, "profile")
	assert.Contains(t, body, "bio")
}

func TestOpenAPIImporter_Import_PathLevelParameters(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/users/{userId}/posts/{postId}": {
				"parameters": [
					{
						"name": "userId",
						"in": "path",
						"required": true,
						"schema": {"type": "integer"},
						"example": 42
					}
				],
				"get": {
					"summary": "Get post",
					"parameters": [
						{
							"name": "postId",
							"in": "path",
							"required": true,
							"schema": {"type": "integer"},
							"example": 123
						}
					]
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Contains(t, req.URL(), "42")
	assert.Contains(t, req.URL(), "123")
}

func TestOpenAPIImporter_Import_MultipleContentTypes(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/upload": {
				"post": {
					"requestBody": {
						"content": {
							"multipart/form-data": {
								"schema": {
									"type": "object",
									"properties": {
										"file": {"type": "string", "format": "binary"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Equal(t, "multipart/form-data", req.GetHeader("Content-Type"))
}

func TestOpenAPIImporter_Import_SchemaWithArray(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/items": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "integer"},
											"name": {"type": "string"}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	body := req.Body()
	assert.NotEmpty(t, body)
	// Should be an array
	assert.Contains(t, body, "[")
}

func TestOpenAPIImporter_Import_SchemaWithBoolean(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/flags": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"enabled": {"type": "boolean"},
										"active": {"type": "boolean", "example": true}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	body := req.Body()
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "enabled")
}

func TestOpenAPIImporter_Import_SchemaWithNumber(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/metrics": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"count": {"type": "integer"},
										"price": {"type": "number"},
										"ratio": {"type": "number", "format": "float"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	body := req.Body()
	assert.NotEmpty(t, body)
}

func TestOpenAPIImporter_Import_DeeplyNestedObjects(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/user": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"user": {
											"type": "object",
											"properties": {
												"profile": {
													"type": "object",
													"properties": {
														"name": {"type": "string"}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	body := req.Body()
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "user")
	assert.Contains(t, body, "profile")
}

func TestOpenAPIImporter_Import_WithDefaultValue(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/settings": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"theme": {"type": "string", "default": "dark"},
										"limit": {"type": "integer", "default": 100}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	body := req.Body()
	assert.NotEmpty(t, body)
	// Default string value is used when generating examples
	assert.Contains(t, body, "dark")
	assert.Contains(t, body, "limit")
}

func TestOpenAPIImporter_Import_FormURLEncodedLogin(t *testing.T) {
	imp := NewOpenAPIImporter()
	ctx := context.Background()

	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/login": {
				"post": {
					"requestBody": {
						"content": {
							"application/x-www-form-urlencoded": {
								"schema": {
									"type": "object",
									"properties": {
										"username": {"type": "string"},
										"password": {"type": "string"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	coll, err := imp.Import(ctx, content)
	require.NoError(t, err)

	req := coll.Requests()[0]
	assert.Equal(t, "application/x-www-form-urlencoded", req.GetHeader("Content-Type"))
}
