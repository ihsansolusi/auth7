package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

type Store interface {
	Organizations() OrganizationStore
	Users() UserStore
	Sessions() SessionStore
	Credentials() CredentialStore
	VerificationTokens() VerificationTokenStore
}

type OrganizationStore interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	GetByCode(ctx context.Context, code string) (*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type UserStore interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, orgID uuid.UUID, username string) (*domain.User, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.User, int, error)
}

type CredentialStore interface {
	Create(ctx context.Context, cred *domain.UserCredential) error
	GetCurrentByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserCredential, error)
	GetHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UserCredential, error)
	Update(ctx context.Context, cred *domain.UserCredential) error
	 RetireOldCredentials(ctx context.Context, userID uuid.UUID, keepCount int) error
}

type VerificationTokenStore interface {
	Create(ctx context.Context, token *domain.VerificationToken) error
	GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type SessionStore interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	Update(ctx context.Context, session *domain.Session) error
	Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID, reason string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}
