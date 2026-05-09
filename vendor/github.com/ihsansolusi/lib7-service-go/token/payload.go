// Package token provides JWT and PASETO token creation and verification.
package token

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Payload carries the claims embedded in every token.
type Payload struct {
	ID        uuid.UUID `json:"jti"`
	UserID    uuid.UUID `json:"sub"`
	BranchID  uuid.UUID `json:"branch_id"`
	Roles     []string  `json:"roles"`
	IssuedAt  time.Time `json:"iat"`
	ExpiredAt time.Time `json:"exp"`
}

// NewPayload creates a Payload with a new random ID and the given duration.
func NewPayload(userID, branchID uuid.UUID, roles []string, duration time.Duration) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate token ID: %w", err)
	}

	now := time.Now()
	return &Payload{
		ID:        tokenID,
		UserID:    userID,
		BranchID:  branchID,
		Roles:     roles,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}, nil
}

// Valid returns ErrExpiredToken if the token has expired.
func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
