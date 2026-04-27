package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClientType string

const (
	ClientTypeWeb         ClientType = "web"
	ClientTypeSPASpa      ClientType = "spa"
	ClientTypeNative      ClientType = "native"
	ClientTypeMachine     ClientType = "machine"
)

type TokenEndpointAuthMethod string

const (
	AuthMethodNone             TokenEndpointAuthMethod = "none"
	AuthMethodClientSecretPost TokenEndpointAuthMethod = "client_secret_post"
	AuthMethodClientSecretBasic TokenEndpointAuthMethod = "client_secret_basic"
	AuthMethodClientSecretJwt  TokenEndpointAuthMethod = "client_secret_jwt"
	AuthMethodPrivateKeyJwt    TokenEndpointAuthMethod = "private_key_jwt"
)

type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeRefreshToken      GrantType = "refresh_token"
	GrantTypeClientCredentials GrantType = "client_credentials"
	GrantTypeDeviceCode        GrantType = "urn:ietf:params:oauth:grant-type:device_code"
)

type Client struct {
	ID                       uuid.UUID                `json:"id"`
	OrgID                    uuid.UUID                `json:"org_id"`
	Name                     string                   `json:"name"`
	Description              string                   `json:"description"`
	ClientType               ClientType               `json:"client_type"`
	TokenEndpointAuthMethod  TokenEndpointAuthMethod  `json:"token_endpoint_auth_method"`
	AllowedScopes            []string                 `json:"allowed_scopes"`
	AllowedRedirectURIs      []string                 `json:"allowed_redirect_uris"`
	AllowedOrigins           []string                 `json:"allowed_origins"`
	ClientSecretHash         string                   `json:"-"`
	PublicKeyJWK             string                   `json:"-"`
	TokenExpiration          int                      `json:"token_expiration"`
	RefreshTokenExpiration   int                      `json:"refresh_token_expiration"`
	AllowMultipleTokens      bool                     `json:"allow_multiple_tokens"`
	SkipConsentScreen        bool                     `json:"skip_consent_screen"`
	IsActive                 bool                     `json:"is_active"`
	CreatedAt                time.Time                `json:"created_at"`
	UpdatedAt                time.Time                `json:"updated_at"`
}

func (c *Client) IsConfidential() bool {
	return c.TokenEndpointAuthMethod != AuthMethodNone
}

func (c *Client) HasGrant(grant GrantType) bool {
	for _, g := range c.AllowedGrants() {
		if g == grant {
			return true
		}
	}
	return false
}

func (c *Client) AllowedGrants() []GrantType {
	switch c.ClientType {
	case ClientTypeMachine:
		return []GrantType{GrantTypeClientCredentials}
	case ClientTypeWeb, ClientTypeSPASpa, ClientTypeNative:
		return []GrantType{GrantTypeAuthorizationCode, GrantTypeRefreshToken}
	default:
		return nil
	}
}

func (c *Client) ValidateRedirectURI(uri string) bool {
	for _, allowed := range c.AllowedRedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

func (c *Client) ValidateScope(requested []string) bool {
	allowed := make(map[string]bool)
	for _, s := range c.AllowedScopes {
		allowed[s] = true
	}
	for _, r := range requested {
		if !allowed[r] {
			return false
		}
	}
	return true
}