package oauth2

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
)

const (
	opTokenExchange = "oauth2.TokenService.Exchange"
	opTokenRefresh  = "oauth2.TokenService.Refresh"
	opTokenClientCreds = "oauth2.TokenService.ClientCredentials"
)

type TokenService struct {
	clientSvc   *ClientService
	authCodeSvc *AuthorizationCodeService
	sessionSvc  any
	jwtSvc      *jwt.Service
}

func NewTokenService(clientSvc *ClientService, authCodeSvc *AuthorizationCodeService, sessionSvc any, jwtSvc *jwt.Service) *TokenService {
	return &TokenService{
		clientSvc:   clientSvc,
		authCodeSvc: authCodeSvc,
		sessionSvc:  sessionSvc,
		jwtSvc:      jwtSvc,
	}
}

func (s *TokenService) ExchangeCodeForTokens(ctx context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	authCode, err := s.authCodeSvc.ExchangeAuthCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opTokenExchange, err)
	}

	if authCode.RedirectURI != redirectURI {
		return nil, ErrInvalidRedirectURI
	}

	client, err := s.clientSvc.GetByClientID(ctx, authCode.ClientID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opTokenExchange, err)
	}

	if !client.IsActive {
		return nil, ErrUnauthorizedClient
	}

	sessionID := uuid.New().String()
	token, access, err := s.jwtSvc.IssueAccessToken(sessionID, authCode.UserID, authCode.OrgID, jwt.Claims{
		ClientID: client.ID.String(),
		Scope:    authCode.Scope,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opTokenExchange, err)
	}

	refreshToken := jwt.GenerateRefreshToken()

	return &TokenResponse{
		AccessToken:  token,
		TokenType:    "Bearer",
		ExpiresIn:    int(access.ExpiresAt.Sub(time.Now()).Seconds()),
		RefreshToken: refreshToken,
		Scope:        authCode.Scope,
	}, nil
}

func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken, scope string) (*TokenResponse, error) {
	return nil, nil
}

func (s *TokenService) ClientCredentials(ctx context.Context, clientID string, scope string) (*TokenResponse, error) {
	client, err := s.clientSvc.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opTokenClientCreds, err)
	}

	if !client.IsConfidential() {
		return nil, ErrUnauthorizedClient
	}

	if !client.HasGrant(domain.GrantTypeClientCredentials) {
		return nil, ErrInvalidGrant
	}

	userID := uuid.New()
	token, access, err := s.jwtSvc.IssueAccessToken("m2m-"+clientID, userID, client.OrgID, jwt.Claims{
		ClientID: clientID,
		Scope:    scope,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opTokenClientCreds, err)
	}

	return &TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(access.ExpiresAt.Sub(time.Now()).Seconds()),
		Scope:       scope,
	}, nil
}

func (s *TokenService) IntrospectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	verified, err := s.jwtSvc.VerifyAccessToken(token)
	if err != nil {
		return &IntrospectionResponse{Active: false}, nil
	}

	return &IntrospectionResponse{
		Active:    true,
		ClientID:  verified.ClientID,
		Username:  verified.Username,
		TokenType: "access_token",
		Exp:       verified.ExpiresAt.Unix(),
		Iat:       verified.IssuedAt.Unix(),
		Sub:       verified.Subject,
		Scope:     verified.Scope,
	}, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
}

type IntrospectionResponse struct {
	Active    bool   `json:"active"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Scope     string `json:"scope,omitempty"`
}