package token

import (
	"errors"
	"time"
)

var (
	ErrExpiredToken = errors.New("token has expired")
	ErrInvalidToken = errors.New("token is invalid")
)

// Maker is the interface for creating and verifying tokens.
type Maker interface {
	// CreateToken signs a token for the given payload with the given duration.
	// The payload's IssuedAt and ExpiredAt fields are set by the implementation.
	CreateToken(payload *Payload, duration time.Duration) (string, error)

	// VerifyToken parses and validates the token string, returning the payload.
	VerifyToken(token string) (*Payload, error)
}
