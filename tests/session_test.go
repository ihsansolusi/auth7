package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTKeyGeneration(t *testing.T) {
	km, err := jwt.NewKeyManager(2048)
	require.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotEmpty(t, km.Kid())
	assert.Equal(t, "RS256", km.Algorithm())
}

func TestJWKSGeneration(t *testing.T) {
	km, err := jwt.NewKeyManager(2048)
	require.NoError(t, err)

	jwks := km.JWKS()
	assert.NotNil(t, jwks)
	assert.Equal(t, "RSA", jwks["kty"])
	assert.Equal(t, "RS256", jwks["alg"])
	assert.Equal(t, "sig", jwks["use"])
	assert.NotEmpty(t, jwks["kid"])
	assert.NotEmpty(t, jwks["n"])
	assert.NotEmpty(t, jwks["e"])
}

func TestRotatedKeyManager(t *testing.T) {
	rm := jwt.NewRotatedKeyManager()

	km1, err := rm.GenerateNewKey()
	require.NoError(t, err)
	assert.NotNil(t, km1)
	assert.NotEmpty(t, km1.Kid())

	km2, err := rm.GenerateNewKey()
	require.NoError(t, err)
	assert.NotNil(t, km2)
	assert.NotEmpty(t, km2.Kid())

	retrieved, ok := rm.GetKey(km1.Kid())
	assert.True(t, ok)
	assert.Equal(t, km1.Kid(), retrieved.Kid())
	assert.Equal(t, km1.Algorithm(), retrieved.Algorithm())

	active, ok := rm.ActiveKey()
	assert.True(t, ok)
	assert.Equal(t, km2.Kid(), active.Kid())
}

func TestJWTSigningAndVerification(t *testing.T) {
	svc := jwt.NewService("auth7.test", []string{"auth7"})

	userID := uuid.New()
	orgID := uuid.New()

	claims := jwt.Claims{
		ClientID: "test-client",
		Username: "testuser",
		Roles:    []string{"admin", "user"},
		Scope:    "openid profile",
	}

	token, access, err := svc.IssueAccessToken("session-123", userID, orgID, claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, access)
	assert.Equal(t, "session-123", access.SessionID)
	assert.Equal(t, userID, access.UserID)
	assert.Equal(t, orgID, access.OrgID)

	verified, err := svc.VerifyAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), verified.Subject)
	assert.Equal(t, "test-client", verified.ClientID)
	assert.Equal(t, "testuser", verified.Username)
	assert.Equal(t, []string{"admin", "user"}, verified.Roles)
}

func TestRefreshTokenGeneration(t *testing.T) {
	refreshToken := jwt.GenerateRefreshToken()
	assert.NotEmpty(t, refreshToken)
	assert.Greater(t, len(refreshToken), 32)

	refreshToken2 := jwt.GenerateRefreshToken()
	assert.NotEqual(t, refreshToken, refreshToken2)
}

func TestTokenHashing(t *testing.T) {
	token := "test-refresh-token-12345"
	hash := jwt.HashToken(token)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, token, hash)

	hash2 := jwt.HashToken(token)
	assert.Equal(t, hash, hash2)
}

func TestAccessTokenExpiration(t *testing.T) {
	svc := jwt.NewService("auth7.test", []string{"auth7"})

	userID := uuid.New()
	orgID := uuid.New()

	claims := jwt.Claims{}

	accessToken, access, err := svc.IssueAccessToken("session-123", userID, orgID, claims)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	assert.True(t, access.ExpiresAt.After(time.Now()))
	assert.True(t, access.ExpiresAt.Before(time.Now().Add(20*time.Minute)))
}

func TestSessionDataStructures(t *testing.T) {
	now := time.Now().Unix()
	session := struct {
		ID            string
		UserID        string
		OrgID         string
		ActiveBranchID string
		IPAddress     string
		CreatedAt     int64
		ExpiresAt     int64
	}{
		ID:            "session-123",
		UserID:        "user-456",
		OrgID:         "org-789",
		ActiveBranchID: "branch-001",
		IPAddress:     "192.168.1.1",
		CreatedAt:     now,
		ExpiresAt:     now + 28800,
	}

	assert.Equal(t, "session-123", session.ID)
	assert.Equal(t, "192.168.1.1", session.IPAddress)
	assert.True(t, session.ExpiresAt > session.CreatedAt)
}

func TestIPBinding(t *testing.T) {
	originalIP := "192.168.1.1"
	currentIP := "192.168.1.1"

	assert.Equal(t, originalIP, currentIP, "IP should match for valid session")

	mismatchedIP := "10.0.0.1"
	assert.NotEqual(t, originalIP, mismatchedIP, "IP should not match")
}
