package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

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
