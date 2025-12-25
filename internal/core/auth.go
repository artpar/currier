package core

import (
	"encoding/base64"
	"fmt"
	"net/url"
)

// AuthType represents the type of authentication.
type AuthType string

const (
	AuthTypeNone      AuthType = "none"
	AuthTypeBasic     AuthType = "basic"
	AuthTypeBearer    AuthType = "bearer"
	AuthTypeAPIKey    AuthType = "apikey"
	AuthTypeOAuth2    AuthType = "oauth2"
	AuthTypeDigest    AuthType = "digest"
	AuthTypeAWSV4     AuthType = "awsv4"
	AuthTypeNTLM      AuthType = "ntlm"
)

// AuthTypeNames returns display names for auth types.
var AuthTypeNames = map[AuthType]string{
	AuthTypeNone:   "No Auth",
	AuthTypeBasic:  "Basic Auth",
	AuthTypeBearer: "Bearer Token",
	AuthTypeAPIKey: "API Key",
	AuthTypeOAuth2: "OAuth 2.0",
	AuthTypeDigest: "Digest Auth",
	AuthTypeAWSV4:  "AWS Signature v4",
	AuthTypeNTLM:   "NTLM",
}

// CommonAuthTypes returns the most commonly used auth types for UI.
func CommonAuthTypes() []AuthType {
	return []AuthType{
		AuthTypeNone,
		AuthTypeBasic,
		AuthTypeBearer,
		AuthTypeAPIKey,
		AuthTypeOAuth2,
	}
}

// APIKeyLocation specifies where to add the API key.
type APIKeyLocation string

const (
	APIKeyInHeader APIKeyLocation = "header"
	APIKeyInQuery  APIKeyLocation = "query"
)

// OAuth2GrantType specifies the OAuth 2.0 grant type.
type OAuth2GrantType string

const (
	OAuth2GrantAuthorizationCode OAuth2GrantType = "authorization_code"
	OAuth2GrantClientCredentials OAuth2GrantType = "client_credentials"
	OAuth2GrantPassword          OAuth2GrantType = "password"
	OAuth2GrantImplicit          OAuth2GrantType = "implicit"
)

// OAuth2Config holds OAuth 2.0 specific configuration.
type OAuth2Config struct {
	GrantType       OAuth2GrantType `json:"grantType" yaml:"grantType"`
	AuthURL         string          `json:"authUrl,omitempty" yaml:"authUrl,omitempty"`
	TokenURL        string          `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	ClientID        string          `json:"clientId,omitempty" yaml:"clientId,omitempty"`
	ClientSecret    string          `json:"clientSecret,omitempty" yaml:"clientSecret,omitempty"`
	Scope           string          `json:"scope,omitempty" yaml:"scope,omitempty"`
	State           string          `json:"state,omitempty" yaml:"state,omitempty"`
	RedirectURI     string          `json:"redirectUri,omitempty" yaml:"redirectUri,omitempty"`
	AccessToken     string          `json:"accessToken,omitempty" yaml:"accessToken,omitempty"`
	RefreshToken    string          `json:"refreshToken,omitempty" yaml:"refreshToken,omitempty"`
	TokenType       string          `json:"tokenType,omitempty" yaml:"tokenType,omitempty"`
	ExpiresIn       int64           `json:"expiresIn,omitempty" yaml:"expiresIn,omitempty"`
	HeaderPrefix    string          `json:"headerPrefix,omitempty" yaml:"headerPrefix,omitempty"` // Default: "Bearer"
	AddTokenTo      string          `json:"addTokenTo,omitempty" yaml:"addTokenTo,omitempty"`     // header or query
	UsePKCE         bool            `json:"usePkce,omitempty" yaml:"usePkce,omitempty"`
	PKCECodeVerifier string         `json:"pkceCodeVerifier,omitempty" yaml:"pkceCodeVerifier,omitempty"`
}

// AWSAuthConfig holds AWS Signature v4 configuration.
type AWSAuthConfig struct {
	AccessKeyID     string `json:"accessKeyId,omitempty" yaml:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`
	SessionToken    string `json:"sessionToken,omitempty" yaml:"sessionToken,omitempty"`
	Region          string `json:"region,omitempty" yaml:"region,omitempty"`
	Service         string `json:"service,omitempty" yaml:"service,omitempty"`
}

// IsConfigured returns true if authentication is configured (not none/empty).
func (a *AuthConfig) IsConfigured() bool {
	if a == nil {
		return false
	}
	return a.Type != "" && AuthType(a.Type) != AuthTypeNone
}

// GetAuthType returns the auth type as AuthType enum.
func (a *AuthConfig) GetAuthType() AuthType {
	if a == nil || a.Type == "" {
		return AuthTypeNone
	}
	return AuthType(a.Type)
}

// SetAuthType sets the auth type.
func (a *AuthConfig) SetAuthType(t AuthType) {
	a.Type = string(t)
}

// Validate checks if the auth configuration is valid.
func (a *AuthConfig) Validate() error {
	if a == nil {
		return nil
	}

	switch a.GetAuthType() {
	case AuthTypeNone:
		return nil

	case AuthTypeBasic:
		if a.Username == "" {
			return fmt.Errorf("basic auth requires username")
		}
		// Password can be empty

	case AuthTypeBearer:
		if a.Token == "" {
			return fmt.Errorf("bearer auth requires token")
		}

	case AuthTypeAPIKey:
		if a.Key == "" {
			return fmt.Errorf("API key auth requires key name")
		}
		if a.Value == "" {
			return fmt.Errorf("API key auth requires key value")
		}
		if a.In == "" {
			a.In = string(APIKeyInHeader) // Default to header
		}

	case AuthTypeOAuth2:
		if a.OAuth2 == nil {
			return fmt.Errorf("OAuth 2.0 requires configuration")
		}
		if a.OAuth2.AccessToken == "" && a.OAuth2.ClientID == "" {
			return fmt.Errorf("OAuth 2.0 requires access token or client credentials")
		}
	}

	return nil
}

// ApplyToHeaders adds authentication headers to the provided map.
// Returns any query parameters that should be added (for API key in query).
func (a *AuthConfig) ApplyToHeaders(headers map[string]string) map[string]string {
	queryParams := make(map[string]string)

	if a == nil || !a.IsConfigured() {
		return queryParams
	}

	switch a.GetAuthType() {
	case AuthTypeBasic:
		credentials := base64.StdEncoding.EncodeToString(
			[]byte(a.Username + ":" + a.Password),
		)
		headers["Authorization"] = "Basic " + credentials

	case AuthTypeBearer:
		headers["Authorization"] = "Bearer " + a.Token

	case AuthTypeAPIKey:
		if APIKeyLocation(a.In) == APIKeyInQuery {
			queryParams[a.Key] = a.Value
		} else {
			headers[a.Key] = a.Value
		}

	case AuthTypeOAuth2:
		if a.OAuth2 != nil && a.OAuth2.AccessToken != "" {
			prefix := a.OAuth2.HeaderPrefix
			if prefix == "" {
				prefix = "Bearer"
			}
			if a.OAuth2.AddTokenTo == "query" {
				queryParams["access_token"] = a.OAuth2.AccessToken
			} else {
				headers["Authorization"] = prefix + " " + a.OAuth2.AccessToken
			}
		}
	}

	return queryParams
}

// ApplyToURL adds authentication query parameters to the URL.
func (a *AuthConfig) ApplyToURL(rawURL string) (string, error) {
	queryParams := a.ApplyToHeaders(make(map[string]string))
	if len(queryParams) == 0 {
		return rawURL, nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, err
	}

	q := parsed.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
}

// Clone creates a deep copy of the auth config.
func (a *AuthConfig) Clone() *AuthConfig {
	if a == nil {
		return nil
	}

	clone := &AuthConfig{
		Type:     a.Type,
		Token:    a.Token,
		Username: a.Username,
		Password: a.Password,
		Key:      a.Key,
		Value:    a.Value,
		In:       a.In,
	}

	if a.OAuth2 != nil {
		clone.OAuth2 = &OAuth2Config{
			GrantType:        a.OAuth2.GrantType,
			AuthURL:          a.OAuth2.AuthURL,
			TokenURL:         a.OAuth2.TokenURL,
			ClientID:         a.OAuth2.ClientID,
			ClientSecret:     a.OAuth2.ClientSecret,
			Scope:            a.OAuth2.Scope,
			State:            a.OAuth2.State,
			RedirectURI:      a.OAuth2.RedirectURI,
			AccessToken:      a.OAuth2.AccessToken,
			RefreshToken:     a.OAuth2.RefreshToken,
			TokenType:        a.OAuth2.TokenType,
			ExpiresIn:        a.OAuth2.ExpiresIn,
			HeaderPrefix:     a.OAuth2.HeaderPrefix,
			AddTokenTo:       a.OAuth2.AddTokenTo,
			UsePKCE:          a.OAuth2.UsePKCE,
			PKCECodeVerifier: a.OAuth2.PKCECodeVerifier,
		}
	}

	if a.AWS != nil {
		clone.AWS = &AWSAuthConfig{
			AccessKeyID:     a.AWS.AccessKeyID,
			SecretAccessKey: a.AWS.SecretAccessKey,
			SessionToken:    a.AWS.SessionToken,
			Region:          a.AWS.Region,
			Service:         a.AWS.Service,
		}
	}

	return clone
}

// NewBasicAuth creates a new basic auth configuration.
func NewBasicAuth(username, password string) AuthConfig {
	return AuthConfig{
		Type:     string(AuthTypeBasic),
		Username: username,
		Password: password,
	}
}

// NewBearerAuth creates a new bearer token auth configuration.
func NewBearerAuth(token string) AuthConfig {
	return AuthConfig{
		Type:  string(AuthTypeBearer),
		Token: token,
	}
}

// NewAPIKeyAuth creates a new API key auth configuration.
func NewAPIKeyAuth(key, value string, location APIKeyLocation) AuthConfig {
	return AuthConfig{
		Type:  string(AuthTypeAPIKey),
		Key:   key,
		Value: value,
		In:    string(location),
	}
}

// NewOAuth2Auth creates a new OAuth 2.0 auth configuration.
func NewOAuth2Auth(config OAuth2Config) AuthConfig {
	return AuthConfig{
		Type:   string(AuthTypeOAuth2),
		OAuth2: &config,
	}
}

// DisplayName returns a human-readable name for the auth type.
func (a *AuthConfig) DisplayName() string {
	if a == nil || a.Type == "" {
		return AuthTypeNames[AuthTypeNone]
	}
	if name, ok := AuthTypeNames[AuthType(a.Type)]; ok {
		return name
	}
	return a.Type
}

// Summary returns a brief summary of the auth configuration.
func (a *AuthConfig) Summary() string {
	if a == nil || !a.IsConfigured() {
		return "No authentication"
	}

	switch a.GetAuthType() {
	case AuthTypeBasic:
		return fmt.Sprintf("Basic: %s", a.Username)
	case AuthTypeBearer:
		if len(a.Token) > 20 {
			return fmt.Sprintf("Bearer: %s...%s", a.Token[:8], a.Token[len(a.Token)-4:])
		}
		return "Bearer: ****"
	case AuthTypeAPIKey:
		loc := "header"
		if a.In != "" {
			loc = a.In
		}
		return fmt.Sprintf("API Key: %s (in %s)", a.Key, loc)
	case AuthTypeOAuth2:
		if a.OAuth2 != nil {
			return fmt.Sprintf("OAuth 2.0: %s", a.OAuth2.GrantType)
		}
		return "OAuth 2.0"
	default:
		return a.DisplayName()
	}
}
