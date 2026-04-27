package oauth2

import (
	"context"
	"fmt"

	"github.com/ihsansolusi/auth7/internal/service/jwt"
)

type OIDCService struct {
	jwtSvc     *jwt.Service
	clientSvc  *ClientService
	identitySvc any
}

func NewOIDCService(jwtSvc *jwt.Service, clientSvc *ClientService, identitySvc any) *OIDCService {
	return &OIDCService{
		jwtSvc:     jwtSvc,
		clientSvc:  clientSvc,
		identitySvc: identitySvc,
	}
}

func (s *OIDCService) Discovery() OIDCDiscovery {
	return OIDCDiscovery{
		Issuer:                            "https://auth7.bank.co.id",
		AuthorizationEndpoint:             "https://auth7.bank.co.id/oauth2/authorize",
		TokenEndpoint:                     "https://auth7.bank.co.id/oauth2/token",
		UserInfoEndpoint:                  "https://auth7.bank.co.id/oauth2/userinfo",
		JwksURI:                           "https://auth7.bank.co.id/.well-known/jwks.json",
		RegistrationEndpoint:              "https://auth7.bank.co.id/oauth2/register",
		ScopesSupported:                   []string{"openid", "profile", "email", "roles"},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}
}

func (s *OIDCService) UserInfo(ctx context.Context, token string) (*UserInfoResponse, error) {
	verified, err := s.jwtSvc.VerifyAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	return &UserInfoResponse{
		Sub:     verified.Subject,
		Name:    verified.Username,
		Email:   verified.Email,
		Scope:   verified.Scope,
	}, nil
}

type OIDCDiscovery struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

type UserInfoResponse struct {
	Sub         string `json:"sub"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	EmailVerified bool `json:"email_verified,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Gender      string `json:"gender,omitempty"`
	Birthdate   string `json:"birthdate,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	PhoneVerified bool `json:"phone_number_verified,omitempty"`
	Address     string `json:"address,omitempty"`
	Scope       string `json:"scope,omitempty"`
}