package authz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

const (
	opRoleCreate          = "authz.Service.CreateRole"
	opRoleGet             = "authz.Service.GetRole"
	opRoleUpdate          = "authz.Service.UpdateRole"
	opRoleDelete          = "authz.Service.DeleteRole"
	opRoleList            = "authz.Service.ListRoles"

	opPermissionCreate    = "authz.Service.CreatePermission"
	opPermissionGet       = "authz.Service.GetPermission"
	opPermissionList      = "authz.Service.ListPermissions"

	opAssignRole          = "authz.Service.AssignRole"
	opRevokeRole          = "authz.Service.RevokeRole"
	opGetUserRoles        = "authz.Service.GetUserRoles"

	opCheckPermission     = "authz.Service.CheckPermission"
	opGetUserPermissions  = "authz.Service.GetUserPermissions"
)

type RoleStore interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Role, error)
	GetByName(ctx context.Context, orgID uuid.UUID, name string) (*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error)
}

type PermissionStore interface {
	Create(ctx context.Context, perm *domain.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error)
	GetByCode(ctx context.Context, code string) (*domain.Permission, error)
	ListByCategory(ctx context.Context, category string) ([]*domain.Permission, error)
	ListAll(ctx context.Context) ([]*domain.Permission, error)
}

type UserRoleStore interface {
	Create(ctx context.Context, ur *domain.UserRole) error
	GetByUser(ctx context.Context, userID, orgID uuid.UUID) ([]*domain.UserRole, error)
	GetByUserBranch(ctx context.Context, userID, branchID, orgID uuid.UUID) ([]*domain.UserRole, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error
}

type ABACPolicyStore interface {
	Create(ctx context.Context, policy *domain.ABACPolicy) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.ABACPolicy, error)
	Update(ctx context.Context, policy *domain.ABACPolicy) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.ABACPolicy, error)
}

type Service struct {
	roleStore      RoleStore
	permStore      PermissionStore
	userRoleStore  UserRoleStore
	policyStore    ABACPolicyStore
	enforcer       Enforcer
}

func NewService(
	roleStore RoleStore,
	permStore PermissionStore,
	userRoleStore UserRoleStore,
	policyStore ABACPolicyStore,
	enforcer Enforcer,
) *Service {
	return &Service{
		roleStore:     roleStore,
		permStore:     permStore,
		userRoleStore: userRoleStore,
		policyStore:   policyStore,
		enforcer:      enforcer,
	}
}

func (s *Service) CreateRole(ctx context.Context, orgID uuid.UUID, params RoleParams) (*domain.Role, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	existing, _ := s.roleStore.GetByName(ctx, orgID, params.Name)
	if existing != nil {
		return nil, ErrRoleExists
	}

	role := &domain.Role{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        params.Name,
		Description: params.Description,
		IsDefault:   params.IsDefault,
		Permissions: params.Permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.roleStore.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("%s: %w", opRoleCreate, err)
	}

	if err := s.enforcer.UpdateRolePermissions(ctx, orgID, role.ID, role.Name, role.Permissions); err != nil {
		return nil, fmt.Errorf("%s: sync casbin: %w", opRoleCreate, err)
	}

	return role, nil
}

func (s *Service) GetRole(ctx context.Context, id, orgID uuid.UUID) (*domain.Role, error) {
	role, err := s.roleStore.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opRoleGet, err)
	}
	return role, nil
}

func (s *Service) ListRoles(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	roles, err := s.roleStore.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opRoleList, err)
	}
	return roles, nil
}

func (s *Service) CreatePermission(ctx context.Context, params PermParams) (*domain.Permission, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	existing, _ := s.permStore.GetByCode(ctx, params.Code)
	if existing != nil {
		return nil, ErrPermissionExists
	}

	perm := &domain.Permission{
		ID:          uuid.New(),
		Code:        params.Code,
		Name:        params.Name,
		Description: params.Description,
		ResourceType: params.Category,
		CreatedAt:   time.Now(),
	}

	if err := s.permStore.Create(ctx, perm); err != nil {
		return nil, fmt.Errorf("%s: %w", opPermissionCreate, err)
	}

	return perm, nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]*domain.Permission, error) {
	perms, err := s.permStore.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opPermissionList, err)
	}
	return perms, nil
}

func (s *Service) AssignRole(ctx context.Context, userID, roleID, branchID uuid.UUID, orgID, assignedBy uuid.UUID) (*domain.UserRole, error) {
	role, err := s.roleStore.GetByID(ctx, roleID, orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid role: %w", err)
	}

	ur := &domain.UserRole{
		ID:        uuid.New(),
		UserID:    userID,
		RoleID:    roleID,
		BranchID:  &branchID,
		OrgID:     orgID,
		GrantedBy: assignedBy,
		GrantedAt: time.Now(),
	}

	if err := s.userRoleStore.Create(ctx, ur); err != nil {
		return nil, fmt.Errorf("%s: %w", opAssignRole, err)
	}

	if err := s.enforcer.GrantUserRole(ctx, orgID, userID, role.Name, branchID); err != nil {
		return nil, fmt.Errorf("%s: sync casbin: %w", opAssignRole, err)
	}

	return ur, nil
}

func (s *Service) RevokeRole(ctx context.Context, userRoleID, revokedBy uuid.UUID) error {
	if err := s.userRoleStore.Revoke(ctx, userRoleID, revokedBy); err != nil {
		return fmt.Errorf("%s: %w", opRevokeRole, err)
	}
	return nil
}

func (s *Service) GetUserRoles(ctx context.Context, userID, orgID uuid.UUID) ([]*domain.UserRole, error) {
	roles, err := s.userRoleStore.GetByUser(ctx, userID, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opGetUserRoles, err)
	}

	active := make([]*domain.UserRole, 0)
	for _, r := range roles {
		if r.IsActive() {
			active = append(active, r)
		}
	}
	return active, nil
}

func (s *Service) CheckPermission(ctx context.Context, authCtx *domain.AuthContext, permission string, resource interface{}) (*domain.AuthorizationResult, error) {
	result, err := s.enforcer.Enforce(ctx, authCtx, permission, resource)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opCheckPermission, err)
	}
	return result, nil
}

func (s *Service) GetUserPermissions(ctx context.Context, userID, orgID, branchID uuid.UUID) ([]string, error) {
	roles, err := s.userRoleStore.GetByUserBranch(ctx, userID, branchID, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opGetUserPermissions, err)
	}

	perms := make([]string, 0)
	for _, ur := range roles {
		if ur.IsActive() {
			role, err := s.roleStore.GetByID(ctx, ur.RoleID, orgID)
			if err == nil {
				perms = append(perms, role.Permissions...)
			}
		}
	}
	return perms, nil
}

type RoleParams struct {
	Name        string
	Description string
	IsDefault   bool
	Permissions []string
}

func (p RoleParams) Validate() error {
	if len(p.Name) < 2 || len(p.Name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	return nil
}

type PermParams struct {
	Code        string
	Name        string
	Description string
	Category    string
}

func (p PermParams) Validate() error {
	if len(p.Code) < 2 || len(p.Code) > 100 {
		return fmt.Errorf("code must be between 2 and 100 characters")
	}
	if len(p.Category) < 1 || len(p.Category) > 50 {
		return fmt.Errorf("category must be between 1 and 50 characters")
	}
	return nil
}

type Enforcer interface {
	UpdateRolePermissions(ctx context.Context, orgID, roleID uuid.UUID, roleName string, permissions []string) error
	GrantUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error
	RevokeUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error
	Enforce(ctx context.Context, authCtx *domain.AuthContext, permission string, resource interface{}) (*domain.AuthorizationResult, error)
	LoadPolicies(ctx context.Context, orgID uuid.UUID) error
}