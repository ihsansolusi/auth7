package oauth2

import "errors"

var (
	ErrCodeAlreadyUsed    = errors.New("authorization code already used")
	ErrCodeExpired        = errors.New("authorization code expired")
	ErrInvalidCodeVerifier = errors.New("invalid code verifier")
	ErrInvalidClient      = errors.New("invalid client")
	ErrInvalidRedirectURI = errors.New("invalid redirect URI")
	ErrInvalidScope       = errors.New("invalid scope")
	ErrInvalidGrant       = errors.New("invalid grant")
	ErrUnauthorizedClient = errors.New("unauthorized client")
	ErrUnsupportedGrantType = errors.New("unsupported grant type")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidRequest     = errors.New("invalid request")
)