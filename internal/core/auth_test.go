package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonAuthTypes(t *testing.T) {
	types := CommonAuthTypes()
	assert.Len(t, types, 5)
	assert.Contains(t, types, AuthTypeNone)
	assert.Contains(t, types, AuthTypeBasic)
	assert.Contains(t, types, AuthTypeBearer)
	assert.Contains(t, types, AuthTypeAPIKey)
	assert.Contains(t, types, AuthTypeOAuth2)
}

func TestAuthConfig_IsConfigured(t *testing.T) {
	t.Run("nil config returns false", func(t *testing.T) {
		var a *AuthConfig
		assert.False(t, a.IsConfigured())
	})

	t.Run("empty type returns false", func(t *testing.T) {
		a := &AuthConfig{}
		assert.False(t, a.IsConfigured())
	})

	t.Run("none type returns false", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeNone)}
		assert.False(t, a.IsConfigured())
	})

	t.Run("basic type returns true", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBasic)}
		assert.True(t, a.IsConfigured())
	})
}

func TestAuthConfig_GetAuthType(t *testing.T) {
	t.Run("nil config returns none", func(t *testing.T) {
		var a *AuthConfig
		assert.Equal(t, AuthTypeNone, a.GetAuthType())
	})

	t.Run("empty type returns none", func(t *testing.T) {
		a := &AuthConfig{}
		assert.Equal(t, AuthTypeNone, a.GetAuthType())
	})

	t.Run("basic type", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBasic)}
		assert.Equal(t, AuthTypeBasic, a.GetAuthType())
	})
}

func TestAuthConfig_SetAuthType(t *testing.T) {
	a := &AuthConfig{}
	a.SetAuthType(AuthTypeBearer)
	assert.Equal(t, string(AuthTypeBearer), a.Type)
}

func TestAuthConfig_Validate(t *testing.T) {
	t.Run("nil config is valid", func(t *testing.T) {
		var a *AuthConfig
		assert.NoError(t, a.Validate())
	})

	t.Run("none type is valid", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeNone)}
		assert.NoError(t, a.Validate())
	})

	t.Run("basic auth requires username", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBasic)}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("basic auth valid with username", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBasic), Username: "user"}
		assert.NoError(t, a.Validate())
	})

	t.Run("bearer auth requires token", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBearer)}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("bearer auth valid with token", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBearer), Token: "abc123"}
		assert.NoError(t, a.Validate())
	})

	t.Run("api key requires key name", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeAPIKey)}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key name")
	})

	t.Run("api key requires key value", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeAPIKey), Key: "X-API-Key"}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key value")
	})

	t.Run("api key defaults location to header", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeAPIKey), Key: "X-API-Key", Value: "secret"}
		assert.NoError(t, a.Validate())
		assert.Equal(t, string(APIKeyInHeader), a.In)
	})

	t.Run("oauth2 requires config", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeOAuth2)}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration")
	})

	t.Run("oauth2 requires token or client ID", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeOAuth2), OAuth2: &OAuth2Config{}}
		err := a.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access token or client credentials")
	})

	t.Run("oauth2 valid with access token", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeOAuth2), OAuth2: &OAuth2Config{AccessToken: "token123"}}
		assert.NoError(t, a.Validate())
	})

	t.Run("oauth2 valid with client ID", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeOAuth2), OAuth2: &OAuth2Config{ClientID: "client123"}}
		assert.NoError(t, a.Validate())
	})
}

func TestAuthConfig_ApplyToHeaders(t *testing.T) {
	t.Run("nil config returns empty", func(t *testing.T) {
		var a *AuthConfig
		headers := make(map[string]string)
		result := a.ApplyToHeaders(headers)
		assert.Empty(t, result)
		assert.Empty(t, headers)
	})

	t.Run("unconfigured returns empty", func(t *testing.T) {
		a := &AuthConfig{}
		headers := make(map[string]string)
		result := a.ApplyToHeaders(headers)
		assert.Empty(t, result)
	})

	t.Run("basic auth adds authorization header", func(t *testing.T) {
		a := NewBasicAuth("user", "pass")
		headers := make(map[string]string)
		a.ApplyToHeaders(headers)
		assert.Contains(t, headers["Authorization"], "Basic ")
	})

	t.Run("bearer auth adds authorization header", func(t *testing.T) {
		a := NewBearerAuth("mytoken")
		headers := make(map[string]string)
		a.ApplyToHeaders(headers)
		assert.Equal(t, "Bearer mytoken", headers["Authorization"])
	})

	t.Run("api key in header", func(t *testing.T) {
		a := NewAPIKeyAuth("X-API-Key", "secret123", APIKeyInHeader)
		headers := make(map[string]string)
		result := a.ApplyToHeaders(headers)
		assert.Equal(t, "secret123", headers["X-API-Key"])
		assert.Empty(t, result)
	})

	t.Run("api key in query", func(t *testing.T) {
		a := NewAPIKeyAuth("api_key", "secret456", APIKeyInQuery)
		headers := make(map[string]string)
		result := a.ApplyToHeaders(headers)
		assert.Equal(t, "secret456", result["api_key"])
		assert.NotContains(t, headers, "api_key")
	})

	t.Run("oauth2 adds bearer header", func(t *testing.T) {
		a := NewOAuth2Auth(OAuth2Config{AccessToken: "oauth_token"})
		headers := make(map[string]string)
		a.ApplyToHeaders(headers)
		assert.Equal(t, "Bearer oauth_token", headers["Authorization"])
	})

	t.Run("oauth2 custom prefix", func(t *testing.T) {
		a := NewOAuth2Auth(OAuth2Config{AccessToken: "token", HeaderPrefix: "Token"})
		headers := make(map[string]string)
		a.ApplyToHeaders(headers)
		assert.Equal(t, "Token token", headers["Authorization"])
	})

	t.Run("oauth2 in query", func(t *testing.T) {
		a := NewOAuth2Auth(OAuth2Config{AccessToken: "token", AddTokenTo: "query"})
		headers := make(map[string]string)
		result := a.ApplyToHeaders(headers)
		assert.Equal(t, "token", result["access_token"])
	})
}

func TestAuthConfig_ApplyToURL(t *testing.T) {
	t.Run("no query params returns original", func(t *testing.T) {
		a := &AuthConfig{}
		url, err := a.ApplyToURL("https://example.com/api")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/api", url)
	})

	t.Run("api key in query adds param", func(t *testing.T) {
		a := NewAPIKeyAuth("key", "val", APIKeyInQuery)
		url, err := a.ApplyToURL("https://example.com/api")
		assert.NoError(t, err)
		assert.Contains(t, url, "key=val")
	})

	t.Run("invalid url returns error", func(t *testing.T) {
		a := NewAPIKeyAuth("key", "val", APIKeyInQuery)
		_, err := a.ApplyToURL("://invalid")
		assert.Error(t, err)
	})
}

func TestAuthConfig_Clone(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		var a *AuthConfig
		assert.Nil(t, a.Clone())
	})

	t.Run("clones basic config", func(t *testing.T) {
		a := &AuthConfig{
			Type:     string(AuthTypeBasic),
			Username: "user",
			Password: "pass",
		}
		clone := a.Clone()
		assert.Equal(t, a.Type, clone.Type)
		assert.Equal(t, a.Username, clone.Username)
		assert.Equal(t, a.Password, clone.Password)
		// Ensure it's a different pointer
		clone.Username = "changed"
		assert.NotEqual(t, a.Username, clone.Username)
	})

	t.Run("clones oauth2 config", func(t *testing.T) {
		a := &AuthConfig{
			Type: string(AuthTypeOAuth2),
			OAuth2: &OAuth2Config{
				ClientID:     "client",
				ClientSecret: "secret",
				AccessToken:  "token",
			},
		}
		clone := a.Clone()
		assert.NotNil(t, clone.OAuth2)
		assert.Equal(t, a.OAuth2.ClientID, clone.OAuth2.ClientID)
		// Ensure OAuth2 is a different pointer
		clone.OAuth2.ClientID = "changed"
		assert.NotEqual(t, a.OAuth2.ClientID, clone.OAuth2.ClientID)
	})

	t.Run("clones aws config", func(t *testing.T) {
		a := &AuthConfig{
			Type: string(AuthTypeAWSV4),
			AWS: &AWSAuthConfig{
				AccessKeyID:     "AKID",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
			},
		}
		clone := a.Clone()
		assert.NotNil(t, clone.AWS)
		assert.Equal(t, a.AWS.AccessKeyID, clone.AWS.AccessKeyID)
	})
}

func TestNewBasicAuth(t *testing.T) {
	auth := NewBasicAuth("user", "pass")
	assert.Equal(t, string(AuthTypeBasic), auth.Type)
	assert.Equal(t, "user", auth.Username)
	assert.Equal(t, "pass", auth.Password)
}

func TestNewBearerAuth(t *testing.T) {
	auth := NewBearerAuth("mytoken")
	assert.Equal(t, string(AuthTypeBearer), auth.Type)
	assert.Equal(t, "mytoken", auth.Token)
}

func TestNewAPIKeyAuth(t *testing.T) {
	auth := NewAPIKeyAuth("X-Key", "value", APIKeyInHeader)
	assert.Equal(t, string(AuthTypeAPIKey), auth.Type)
	assert.Equal(t, "X-Key", auth.Key)
	assert.Equal(t, "value", auth.Value)
	assert.Equal(t, string(APIKeyInHeader), auth.In)
}

func TestNewOAuth2Auth(t *testing.T) {
	config := OAuth2Config{
		GrantType:   OAuth2GrantClientCredentials,
		ClientID:    "client",
		AccessToken: "token",
	}
	auth := NewOAuth2Auth(config)
	assert.Equal(t, string(AuthTypeOAuth2), auth.Type)
	assert.NotNil(t, auth.OAuth2)
	assert.Equal(t, "client", auth.OAuth2.ClientID)
}

func TestAuthConfig_DisplayName(t *testing.T) {
	t.Run("nil config returns No Auth", func(t *testing.T) {
		var a *AuthConfig
		assert.Equal(t, "No Auth", a.DisplayName())
	})

	t.Run("empty type returns No Auth", func(t *testing.T) {
		a := &AuthConfig{}
		assert.Equal(t, "No Auth", a.DisplayName())
	})

	t.Run("basic returns Basic Auth", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeBasic)}
		assert.Equal(t, "Basic Auth", a.DisplayName())
	})

	t.Run("unknown type returns raw type", func(t *testing.T) {
		a := &AuthConfig{Type: "custom_auth"}
		assert.Equal(t, "custom_auth", a.DisplayName())
	})
}

func TestAuthConfig_Summary(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		var a *AuthConfig
		assert.Equal(t, "No authentication", a.Summary())
	})

	t.Run("unconfigured", func(t *testing.T) {
		a := &AuthConfig{}
		assert.Equal(t, "No authentication", a.Summary())
	})

	t.Run("basic auth shows username", func(t *testing.T) {
		a := NewBasicAuth("admin", "secret")
		assert.Equal(t, "Basic: admin", a.Summary())
	})

	t.Run("bearer with long token shows truncated", func(t *testing.T) {
		a := NewBearerAuth("abcdefghijklmnopqrstuvwxyz")
		summary := a.Summary()
		assert.Contains(t, summary, "Bearer:")
		assert.Contains(t, summary, "...")
	})

	t.Run("bearer with short token shows masked", func(t *testing.T) {
		a := NewBearerAuth("short")
		assert.Equal(t, "Bearer: ****", a.Summary())
	})

	t.Run("api key shows key and location", func(t *testing.T) {
		a := NewAPIKeyAuth("X-Key", "val", APIKeyInQuery)
		assert.Contains(t, a.Summary(), "API Key:")
		assert.Contains(t, a.Summary(), "X-Key")
		assert.Contains(t, a.Summary(), "query")
	})

	t.Run("api key defaults to header", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeAPIKey), Key: "X-Key", Value: "val"}
		assert.Contains(t, a.Summary(), "header")
	})

	t.Run("oauth2 with config", func(t *testing.T) {
		a := NewOAuth2Auth(OAuth2Config{GrantType: OAuth2GrantClientCredentials})
		assert.Contains(t, a.Summary(), "OAuth 2.0")
		assert.Contains(t, a.Summary(), "client_credentials")
	})

	t.Run("oauth2 without config", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeOAuth2)}
		assert.Equal(t, "OAuth 2.0", a.Summary())
	})

	t.Run("unknown type shows display name", func(t *testing.T) {
		a := &AuthConfig{Type: string(AuthTypeDigest)}
		assert.Equal(t, "Digest Auth", a.Summary())
	})
}
