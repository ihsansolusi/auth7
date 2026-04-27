package tests

import (
	"testing"

	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/oauth2"
	"github.com/stretchr/testify/assert"
)

func TestOAuth2ClientTypes(t *testing.T) {
	client := &domain.Client{
		ClientType: domain.ClientTypeWeb,
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
	}

	assert.True(t, client.IsConfidential())
	assert.Equal(t, domain.ClientTypeWeb, client.ClientType)
}

func TestOAuth2ClientGrants(t *testing.T) {
	webClient := &domain.Client{ClientType: domain.ClientTypeWeb}
	assert.Contains(t, webClient.AllowedGrants(), domain.GrantTypeAuthorizationCode)
	assert.Contains(t, webClient.AllowedGrants(), domain.GrantTypeRefreshToken)

	machineClient := &domain.Client{ClientType: domain.ClientTypeMachine}
	assert.Contains(t, machineClient.AllowedGrants(), domain.GrantTypeClientCredentials)
	assert.NotContains(t, machineClient.AllowedGrants(), domain.GrantTypeAuthorizationCode)
}

func TestOAuth2ClientRedirectValidation(t *testing.T) {
	client := &domain.Client{
		AllowedRedirectURIs: []string{"https://app.example.com/callback", "http://localhost:3000/callback"},
	}

	assert.True(t, client.ValidateRedirectURI("https://app.example.com/callback"))
	assert.True(t, client.ValidateRedirectURI("http://localhost:3000/callback"))
	assert.False(t, client.ValidateRedirectURI("https://evil.com/callback"))
}

func TestOAuth2ClientScopeValidation(t *testing.T) {
	client := &domain.Client{
		AllowedScopes: []string{"openid", "profile", "email"},
	}

	assert.True(t, client.ValidateScope([]string{"openid", "profile"}))
	assert.True(t, client.ValidateScope([]string{"openid"}))
	assert.False(t, client.ValidateScope([]string{"openid", "admin"}))
	assert.False(t, client.ValidateScope([]string{"unknown"}))
}

func TestPKCEGeneration(t *testing.T) {
	verifier, err := oauth2.GenerateCodeVerifier()
	assert.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.Greater(t, len(verifier), 32)

	challenge := oauth2.GenerateCodeChallenge(verifier)
	assert.NotEmpty(t, challenge)
	assert.NotEqual(t, verifier, challenge)
}

func TestPKCEVerification(t *testing.T) {
	verifier, err := oauth2.GenerateCodeVerifier()
	assert.NoError(t, err)

	challenge := oauth2.GenerateCodeChallenge(verifier)
	assert.True(t, oauth2.VerifyCodeChallenge(verifier, challenge))
	assert.False(t, oauth2.VerifyCodeChallenge(verifier+"x", challenge))
}

func TestOIDCDiscovery(t *testing.T) {
	discovery := oauth2.OIDCDiscovery{
		Issuer:                            "https://auth7.bank.co.id",
		AuthorizationEndpoint:             "https://auth7.bank.co.id/oauth2/authorize",
		TokenEndpoint:                     "https://auth7.bank.co.id/oauth2/token",
		UserInfoEndpoint:                  "https://auth7.bank.co.id/oauth2/userinfo",
		JwksURI:                           "https://auth7.bank.co.id/.well-known/jwks.json",
		ScopesSupported:                   []string{"openid", "profile", "email", "roles"},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}

	assert.Equal(t, "https://auth7.bank.co.id", discovery.Issuer)
	assert.Contains(t, discovery.ScopesSupported, "openid")
	assert.Contains(t, discovery.GrantTypesSupported, "authorization_code")
	assert.Contains(t, discovery.GrantTypesSupported, "client_credentials")
	assert.Contains(t, discovery.CodeChallengeMethodsSupported, "S256")
}

func TestTokenEndpointAuthMethods(t *testing.T) {
	publicClient := &domain.Client{TokenEndpointAuthMethod: domain.AuthMethodNone}
	assert.False(t, publicClient.IsConfidential())

	confidentialClient := &domain.Client{TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic}
	assert.True(t, confidentialClient.IsConfidential())

	privateKeyClient := &domain.Client{TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt}
	assert.True(t, privateKeyClient.IsConfidential())
}

func TestClientCredentialsGrant(t *testing.T) {
	machineClient := &domain.Client{
		ClientType:              domain.ClientTypeMachine,
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		IsActive:                true,
	}

	assert.True(t, machineClient.HasGrant(domain.GrantTypeClientCredentials))
	assert.False(t, machineClient.HasGrant(domain.GrantTypeAuthorizationCode))
	assert.True(t, machineClient.IsConfidential())
}

func TestAuthorizationCodeGrant(t *testing.T) {
	webClient := &domain.Client{
		ClientType:              domain.ClientTypeWeb,
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		IsActive:                true,
	}

	assert.True(t, webClient.HasGrant(domain.GrantTypeAuthorizationCode))
	assert.True(t, webClient.HasGrant(domain.GrantTypeRefreshToken))
	assert.True(t, webClient.IsConfidential())
}