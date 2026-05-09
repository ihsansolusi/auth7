package token

import (
	"fmt"
	"time"

	"github.com/o1egl/paseto"
)

// PASETOMaker implements Maker using PASETO v2 local (symmetric) encryption.
// The symmetric key must be exactly 32 bytes.
type PASETOMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// NewPASETOMaker returns a PASETOMaker. Returns an error if the key is not 32 chars.
func NewPASETOMaker(symmetricKey string) (*PASETOMaker, error) {
	if len(symmetricKey) != 32 {
		return nil, fmt.Errorf("invalid key size: must be exactly 32 characters")
	}

	maker := &PASETOMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}
	return maker, nil
}

// CreateToken encrypts a new PASETO v2 local token.
// The payload's IssuedAt and ExpiredAt are overwritten by the provided duration.
func (m *PASETOMaker) CreateToken(payload *Payload, duration time.Duration) (string, error) {
	now := time.Now()
	payload.IssuedAt = now
	payload.ExpiredAt = now.Add(duration)

	encrypted, err := m.paseto.Encrypt(m.symmetricKey, payload, nil)
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}
	return encrypted, nil
}

// VerifyToken decrypts and validates the PASETO token, returning the Payload.
// Returns ErrExpiredToken if expired, ErrInvalidToken if the token is malformed.
func (m *PASETOMaker) VerifyToken(tokenStr string) (*Payload, error) {
	payload := &Payload{}

	if err := m.paseto.Decrypt(tokenStr, m.symmetricKey, payload, nil); err != nil {
		return nil, ErrInvalidToken
	}

	if err := payload.Valid(); err != nil {
		return nil, err
	}

	return payload, nil
}
