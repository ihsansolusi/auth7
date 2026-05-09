package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

// JWTMaker implements Maker using HMAC-SHA256 (HS256).
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker returns a JWTMaker. Returns an error if the secret is too short.
func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

// jwtClaims maps Payload fields onto JWT registered claims to avoid
// embedding conflicts between Payload and jwt.RegisteredClaims.
type jwtClaims struct {
	BranchID string   `json:"branch_id"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// CreateToken signs a new HS256 JWT. The payload's IssuedAt and ExpiredAt
// are overwritten by the provided duration relative to now.
func (m *JWTMaker) CreateToken(payload *Payload, duration time.Duration) (string, error) {
	now := time.Now()
	payload.IssuedAt = now
	payload.ExpiredAt = now.Add(duration)

	claims := jwtClaims{
		BranchID: payload.BranchID.String(),
		Roles:    payload.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        payload.ID.String(),
			Subject:   payload.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// VerifyToken parses and validates the JWT, returning the Payload.
// Returns ErrExpiredToken if the token has expired, ErrInvalidToken otherwise.
func (m *JWTMaker) VerifyToken(tokenStr string) (*Payload, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.secretKey), nil
	}

	var claims jwtClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	tokenID, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}
	branchID, err := uuid.Parse(claims.BranchID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	payload := &Payload{
		ID:        tokenID,
		UserID:    userID,
		BranchID:  branchID,
		Roles:     claims.Roles,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiredAt: claims.ExpiresAt.Time,
	}
	return payload, nil
}
