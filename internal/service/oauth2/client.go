package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

const (
	opClientCreate = "oauth2.ClientService.Create"
	opClientUpdate = "oauth2.ClientService.Update"
	opClientDelete = "oauth2.ClientService.Delete"
	opClientGet    = "oauth2.ClientService.Get"
)

type ClientStore interface {
	Create(ctx context.Context, client *domain.Client) error
	Update(ctx context.Context, client *domain.Client) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Client, error)
	GetByClientID(ctx context.Context, clientID string) (*domain.Client, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Client, error)
}

type DCRStore interface {
	CreateClient(ctx context.Context, client *domain.Client) error
	UpdateClient(ctx context.Context, client *domain.Client) error
	GetClient(ctx context.Context, clientID string) (*domain.Client, error)
	DeleteClient(ctx context.Context, clientID string) error
}

type ClientService struct {
	store DCRStore
}

func NewClientService(store DCRStore) *ClientService {
	return &ClientService{store: store}
}

func (s *ClientService) Create(ctx context.Context, orgID uuid.UUID, params CreateClientParams) (*domain.Client, error) {
	clientSecret := ""
	if params.TokenEndpointAuthMethod != domain.AuthMethodNone {
		secret, err := generateClientSecret()
		if err != nil {
			return nil, fmt.Errorf("generate client secret: %w", err)
		}
		clientSecret = secret
	}

	client := &domain.Client{
		ID:                       uuid.New(),
		OrgID:                    orgID,
		Name:                     params.Name,
		Description:              params.Description,
		ClientType:               params.ClientType,
		TokenEndpointAuthMethod:  params.TokenEndpointAuthMethod,
		AllowedScopes:            params.AllowedScopes,
		AllowedRedirectURIs:      params.AllowedRedirectURIs,
		AllowedOrigins:           params.AllowedOrigins,
		TokenExpiration:          params.TokenExpiration,
		RefreshTokenExpiration:   params.RefreshTokenExpiration,
		AllowMultipleTokens:     params.AllowMultipleTokens,
		SkipConsentScreen:       params.SkipConsentScreen,
		IsActive:                true,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if clientSecret != "" {
		client.ClientSecretHash = hashSecret(clientSecret)
	}

	if err := s.store.CreateClient(ctx, client); err != nil {
		return nil, fmt.Errorf("%s: %w", opClientCreate, err)
	}

	return client, nil
}

func (s *ClientService) GetByClientID(ctx context.Context, clientID string) (*domain.Client, error) {
	client, err := s.store.GetClient(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opClientGet, err)
	}
	return client, nil
}

func (s *ClientService) Delete(ctx context.Context, clientID, orgID uuid.UUID) error {
	if err := s.store.DeleteClient(ctx, clientID.String()); err != nil {
		return fmt.Errorf("%s: %w", opClientDelete, err)
	}
	return nil
}

// CreateWithSecretHash creates a client with a pre-computed secret hash (used by DCR).
func (s *ClientService) CreateWithSecretHash(ctx context.Context, orgID uuid.UUID, params CreateClientParams, secretHash string) (*domain.Client, error) {
	allowedOrigins := params.AllowedOrigins
	if allowedOrigins == nil {
		allowedOrigins = []string{}
	}
	allowedScopes := params.AllowedScopes
	if allowedScopes == nil {
		allowedScopes = []string{}
	}
	allowedRedirectURIs := params.AllowedRedirectURIs
	if allowedRedirectURIs == nil {
		allowedRedirectURIs = []string{}
	}
	client := &domain.Client{
		ID:                      uuid.New(),
		OrgID:                   orgID,
		Name:                    params.Name,
		Description:             params.Description,
		ClientType:              params.ClientType,
		TokenEndpointAuthMethod: params.TokenEndpointAuthMethod,
		AllowedScopes:           allowedScopes,
		AllowedRedirectURIs:     allowedRedirectURIs,
		AllowedOrigins:          allowedOrigins,
		TokenExpiration:         params.TokenExpiration,
		RefreshTokenExpiration:  params.RefreshTokenExpiration,
		AllowMultipleTokens:     params.AllowMultipleTokens,
		SkipConsentScreen:       params.SkipConsentScreen,
		IsActive:                true,
		ClientSecretHash:        secretHash,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := s.store.CreateClient(ctx, client); err != nil {
		return nil, fmt.Errorf("%s: %w", opClientCreate, err)
	}

	return client, nil
}

type CreateClientParams struct {
	Name                    string
	Description             string
	ClientType               domain.ClientType
	TokenEndpointAuthMethod  domain.TokenEndpointAuthMethod
	AllowedScopes            []string
	AllowedRedirectURIs      []string
	AllowedOrigins           []string
	TokenExpiration          int
	RefreshTokenExpiration   int
	AllowMultipleTokens      bool
	SkipConsentScreen        bool
}

func generateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(h[:])
}