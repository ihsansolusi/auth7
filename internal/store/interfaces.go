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
	MFAConfigs() MFAConfigStore
	EmailOTPCodes() EmailOTPCodeStore
	Roles() RoleStore
	Permissions() PermissionStore
	RolePermissions() RolePermissionStore
	UserRoles() UserRoleStore
	AuditLogs() AuditLogStore
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

type MFAConfigStore interface {
	Create(ctx context.Context, cfg *domain.MFAConfig) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error)
	Update(ctx context.Context, cfg *domain.MFAConfig) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

type EmailOTPCodeStore interface {
	Create(ctx context.Context, code *domain.EmailOTPCode) error
	GetByUserIDAndPurpose(ctx context.Context, userID uuid.UUID, purpose string) (*domain.EmailOTPCode, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailOTPCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type RoleStore interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Role, error)
	GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.Role, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
}

type PermissionStore interface {
	Create(ctx context.Context, perm *domain.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error)
	GetByCode(ctx context.Context, code string) (*domain.Permission, error)
	List(ctx context.Context) ([]*domain.Permission, error)
	ListByResourceType(ctx context.Context, resourceType string) ([]*domain.Permission, error)
}

type RolePermissionStore interface {
	Assign(ctx context.Context, roleID, permissionID uuid.UUID) error
	Revoke(ctx context.Context, roleID, permissionID uuid.UUID) error
	GetByRole(ctx context.Context, roleID uuid.UUID) ([]*domain.Permission, error)
	DeleteByRole(ctx context.Context, roleID uuid.UUID) error
}

type UserRoleStore interface {
	Create(ctx context.Context, ur *domain.UserRole) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.UserRole, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error)
	GetByBranch(ctx context.Context, branchID uuid.UUID) ([]*domain.UserRole, error)
	Revoke(ctx context.Context, id, orgID, revokedBy uuid.UUID) error
	RevokeByUserAndRole(ctx context.Context, userID, roleID, orgID, revokedBy uuid.UUID) error
}

type AuditLogStore interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error)
}
