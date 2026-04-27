package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

type AuthCode struct {
	Code        string
	ClientID    string
	RedirectURI string
	Scope       string
	UserID      uuid.UUID
	OrgID       uuid.UUID
	CodeChallenge string
	CodeChallengeMethod string
	ExpiresAt   time.Time
	CodeUsed    bool
}

type AuthCodeStore interface {
	Create(ctx context.Context, code *AuthCode) error
	GetByCode(ctx context.Context, code string) (*AuthCode, error)
	MarkUsed(ctx context.Context, code string) error
	Delete(ctx context.Context, code string) error
}

type PKCEVerifier struct {
	codeVerifier string
	codeChallenge string
}

func GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base6464URLNoPadding(bytes), nil
}

func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base6464URLNoPadding(h[:])
}

func VerifyCodeChallenge(verifier, challenge string) bool {
	expected := GenerateCodeChallenge(verifier)
	return expected == challenge
}

func base6464URLNoPadding(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

type AuthorizationCodeService struct {
	store AuthCodeStore
}

func NewAuthorizationCodeService(store AuthCodeStore) *AuthorizationCodeService {
	return &AuthorizationCodeService{store: store}
}

func (s *AuthorizationCodeService) CreateAuthCode(ctx context.Context, params AuthCodeParams) (*AuthCode, error) {
	code, err := generateCode(32)
	if err != nil {
		return nil, err
	}

	authCode := &AuthCode{
		Code:               code,
		ClientID:           params.ClientID,
		RedirectURI:        params.RedirectURI,
		Scope:              params.Scope,
		UserID:             params.UserID,
		OrgID:              params.OrgID,
		CodeChallenge:      params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		ExpiresAt:          time.Now().Add(5 * time.Minute),
		CodeUsed:           false,
	}

	if err := s.store.Create(ctx, authCode); err != nil {
		return nil, err
	}

	return authCode, nil
}

func (s *AuthorizationCodeService) ExchangeAuthCode(ctx context.Context, code, codeVerifier string) (*AuthCode, error) {
	authCode, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if authCode.CodeUsed {
		return nil, ErrCodeAlreadyUsed
	}

	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrCodeExpired
	}

	if authCode.CodeChallenge != "" {
		if !VerifyCodeChallenge(codeVerifier, authCode.CodeChallenge) {
			return nil, ErrInvalidCodeVerifier
		}
	}

	s.store.MarkUsed(ctx, code)
	return authCode, nil
}

type AuthCodeParams struct {
	ClientID           string
	RedirectURI        string
	Scope              string
	UserID             uuid.UUID
	OrgID              uuid.UUID
	CodeChallenge      string
	CodeChallengeMethod string
}

func generateCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base6464URLNoPadding(bytes), nil
}