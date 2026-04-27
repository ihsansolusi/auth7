package jwt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
	SessionID   string   `json:"sid,omitempty"`
	OrgID       string   `json:"org_id,omitempty"`
	ClientID    string   `json:"client_id,omitempty"`
	Username    string   `json:"preferred_username,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	BranchID    string   `json:"branch_id,omitempty"`
}

type AccessToken struct {
	TokenID   string
	SessionID string
	UserID    uuid.UUID
	OrgID     uuid.UUID
	ClientID  string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type Service struct {
	keyManager *RotatedKeyManager
	issuer    string
	audience  []string
}

func NewService(issuer string, audience []string) *Service {
	km := NewRotatedKeyManager()
	km.GenerateNewKey()

	return &Service{
		keyManager: km,
		issuer:     issuer,
		audience:   audience,
	}
}

func (s *Service) IssueAccessToken(sessionID string, userID, orgID uuid.UUID, claims Claims) (string, *AccessToken, error) {
	km, ok := s.keyManager.ActiveKey()
	if !ok {
		return "", nil, fmt.Errorf("no active key")
	}

	now := time.Now()
	tokenID := GenerateTokenID()

	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			Audience:  s.audience,
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		SessionID: sessionID,
		OrgID:     orgID.String(),
		ClientID:  claims.ClientID,
		Username:  claims.Username,
		Email:     claims.Email,
		Roles:     claims.Roles,
		Scope:     claims.Scope,
		BranchID:  claims.BranchID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	token.Header["kid"] = km.Kid()
	signedToken, err := token.SignedString(km.PrivateKey())
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	access := &AccessToken{
		TokenID:   tokenID,
		SessionID: sessionID,
		UserID:    userID,
		OrgID:     orgID,
		ClientID:  claims.ClientID,
		ExpiresAt: now.Add(15 * time.Minute),
		IssuedAt:  now,
	}

	return signedToken, access, nil
}

func (s *Service) VerifyAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		km, ok := s.keyManager.GetKey(kid)
		if !ok {
			return nil, fmt.Errorf("key not found: %s", kid)
		}

		return km.PublicKey(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *Service) GetJWKS() []map[string]interface{} {
	return s.keyManager.JWKS()
}

func (s *Service) GetActiveKid() string {
	km, _ := s.keyManager.ActiveKey()
	if km == nil {
		return ""
	}
	return km.Kid()
}

func (s *Service) RotateKey() error {
	_, err := s.keyManager.GenerateNewKey()
	return err
}

type RefreshToken struct {
	TokenID   string
	FamilyID  string
	SessionID string
	UserID    uuid.UUID
	OrgID     uuid.UUID
	ClientID  string
	Scopes    []string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

func GenerateTokenID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

func GenerateRefreshToken() string {
	bytes := make([]byte, 32)
	io.ReadFull(rand.Reader, bytes)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sum[:])
}

type RefreshTokenService struct {
	keyManager *RotatedKeyManager
	issuer    string
	audience  []string
}

func NewRefreshTokenService(issuer string, audience []string) *RefreshTokenService {
	return &RefreshTokenService{
		keyManager: NewRotatedKeyManager(),
		issuer:     issuer,
		audience:   audience,
	}
}

func (s *RefreshTokenService) IssueRefreshToken(sessionID string, userID, orgID uuid.UUID, clientID string, scopes []string) (*RefreshToken, string, error) {
	now := time.Now()
	tokenID := GenerateTokenID()
	familyID := GenerateTokenID()
	plainToken := GenerateRefreshToken()

	rt := &RefreshToken{
		TokenID:   tokenID,
		FamilyID:  familyID,
		SessionID: sessionID,
		UserID:    userID,
		OrgID:     orgID,
		ClientID:  clientID,
		Scopes:    scopes,
		ExpiresAt: now.Add(8 * time.Hour),
		IssuedAt:  now,
	}

	return rt, plainToken, nil
}
