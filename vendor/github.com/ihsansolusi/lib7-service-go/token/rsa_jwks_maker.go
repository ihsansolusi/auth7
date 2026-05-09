package token

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// RSAJWKSMaker implements Maker by verifying RS256 JWTs using a remote JWKS endpoint.
// It caches the fetched keys and refreshes them on cache miss or expiry.
// CreateToken is not supported (returns ErrInvalidToken).
type RSAJWKSMaker struct {
	jwksURI  string
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	fetchedAt time.Time
	cacheTTL time.Duration
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewRSAJWKSMaker creates a Maker that validates RS256 JWTs by fetching the
// public keys from the given JWKS URI. Keys are cached for cacheTTL duration.
func NewRSAJWKSMaker(jwksURI string, cacheTTL time.Duration) *RSAJWKSMaker {
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}
	return &RSAJWKSMaker{
		jwksURI:  jwksURI,
		keys:     make(map[string]*rsa.PublicKey),
		cacheTTL: cacheTTL,
	}
}

// CreateToken is not supported by RSAJWKSMaker (read-only validator).
func (m *RSAJWKSMaker) CreateToken(_ *Payload, _ time.Duration) (string, error) {
	return "", fmt.Errorf("RSAJWKSMaker: CreateToken not supported")
}

// VerifyToken validates an RS256 JWT, fetching the JWKS if necessary.
// The returned Payload maps auth7 claims: sub→UserID, org_id→BranchID (best effort).
func (m *RSAJWKSMaker) VerifyToken(tokenStr string) (*Payload, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		kid, _ := t.Header["kid"].(string)
		key, err := m.getKey(kid)
		if err != nil {
			return nil, ErrInvalidToken
		}
		return key, nil
	}

	type auth7Claims struct {
		jwt.RegisteredClaims
		OrgID    string `json:"org_id"`
		SessionID string `json:"sid"`
	}

	var claims auth7Claims
	tok, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !tok.Valid {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Map org_id to BranchID for compatibility with lib7 middleware
	branchID := uuid.Nil
	if claims.OrgID != "" {
		if id, err := uuid.Parse(claims.OrgID); err == nil {
			branchID = id
		}
	}

	tokenID := uuid.Nil
	if claims.ID != "" {
		if id, err := uuid.Parse(claims.ID); err == nil {
			tokenID = id
		}
	}

	payload := &Payload{
		ID:        tokenID,
		UserID:    userID,
		BranchID:  branchID,
		Roles:     []string{},
		IssuedAt:  claims.IssuedAt.Time,
		ExpiredAt: claims.ExpiresAt.Time,
	}
	return payload, nil
}

func (m *RSAJWKSMaker) getKey(kid string) (*rsa.PublicKey, error) {
	m.mu.RLock()
	key, ok := m.keys[kid]
	expired := time.Since(m.fetchedAt) > m.cacheTTL
	m.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	if err := m.fetchKeys(); err != nil {
		// On fetch error, return cached key if available
		m.mu.RLock()
		key, ok = m.keys[kid]
		m.mu.RUnlock()
		if ok {
			return key, nil
		}
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}

	m.mu.RLock()
	key, ok = m.keys[kid]
	m.mu.RUnlock()
	if !ok {
		// Try empty kid fallback (single key without kid)
		m.mu.RLock()
		for _, k := range m.keys {
			key = k
			ok = true
			break
		}
		m.mu.RUnlock()
	}
	if !ok {
		return nil, fmt.Errorf("key not found for kid=%q", kid)
	}
	return key, nil
}

func (m *RSAJWKSMaker) fetchKeys() error {
	resp, err := http.Get(m.jwksURI) //nolint:noctx
	if err != nil {
		return fmt.Errorf("GET %s: %w", m.jwksURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	m.mu.Lock()
	m.keys = keys
	m.fetchedAt = time.Now()
	m.mu.Unlock()

	return nil
}

func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
