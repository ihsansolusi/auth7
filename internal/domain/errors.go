package domain

import "errors"

var (
	ErrNotFound           = errors.New("entity not found")
	ErrAlreadyExists     = errors.New("entity already exists")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrAccountLocked     = errors.New("account locked")
	ErrAccountDisabled   = errors.New("account disabled")
	ErrSessionExpired    = errors.New("session expired")
	ErrTokenRevoked      = errors.New("token revoked")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrMFARequired       = errors.New("mfa required")
	ErrInvalidToken      = errors.New("invalid token")
)
