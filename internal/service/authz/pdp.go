package authz

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

// This file centralizes the PDP (Policy Decision Point) composition so the REST
// and gRPC transports share ONE decision path: identity → effective permissions
// → PermissionChecker (role-based + operational-hours time-gate + allow-default
// ABAC).

// UserRolesGetter / RolePermsGetter are the read surface used to resolve a
// user's effective permissions. Satisfied by the concrete admin service
// adapters (adminUserRoleSvc, adminRoleSvc). ctx is interface{} to match those
// adapters' signatures.
type UserRolesGetter interface {
	GetUserRoles(ctx interface{}, userID uuid.UUID) ([]*domain.UserRole, error)
}

type RolePermsGetter interface {
	GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error)
}

// ResolveAuthContext loads the user's effective (org-wide + this-branch) active
// permissions and builds the ABAC AuthContext. Org-wide role assignments
// (branch_id NULL) always apply; branch-scoped ones only for the given branch.
func ResolveAuthContext(ctx context.Context, ur UserRolesGetter, rp RolePermsGetter, userID, orgID, branchID uuid.UUID) (*domain.AuthContext, error) {
	userRoles, err := ur.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	perms := []string{}
	for _, urole := range userRoles {
		if urole == nil || urole.OrgID != orgID || !urole.IsActive() {
			continue
		}
		if urole.BranchID != nil && *urole.BranchID != branchID {
			continue
		}
		rolePerms, perr := rp.GetPermissions(ctx, urole.RoleID)
		if perr != nil {
			return nil, perr
		}
		for _, p := range rolePerms {
			if p == nil || p.Code == "" || seen[p.Code] {
				continue
			}
			seen[p.Code] = true
			perms = append(perms, p.Code)
		}
	}

	return &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		Permissions: perms,
		BranchScope: domain.BranchScopeAssigned,
	}, nil
}

// NewTimeGatedChecker builds a PermissionChecker for the PDP: role-based check +
// (optional) operational-hours time-gate + allow-by-default ABAC. The enforcer,
// ABAC policy store, and role store are no-ops (not exercised on this path:
// permissions are pre-resolved into AuthContext, ABAC has no policies).
func NewTimeGatedChecker(tw *TimeWindowEvaluator, gatedPerms []string) *PermissionChecker {
	checker := NewPermissionChecker(noopEnforcer{}, NewABACEvaluator(noopABACStore{}), noopRoleStore{})
	if tw != nil {
		checker = checker.WithTimeGate(tw, gatedPerms)
	}
	return checker
}

// ── no-op authz stores (satisfy PermissionChecker/ABACEvaluator deps) ─────────

type noopEnforcer struct{}

func (noopEnforcer) UpdateRolePermissions(context.Context, uuid.UUID, uuid.UUID, string, []string) error {
	return nil
}
func (noopEnforcer) GrantUserRole(context.Context, uuid.UUID, uuid.UUID, string, uuid.UUID) error {
	return nil
}
func (noopEnforcer) RevokeUserRole(context.Context, uuid.UUID, uuid.UUID, string, uuid.UUID) error {
	return nil
}
func (noopEnforcer) Enforce(context.Context, *domain.AuthContext, string, interface{}) (*domain.AuthorizationResult, error) {
	return &domain.AuthorizationResult{Allowed: true, Reason: "enforcer disabled"}, nil
}
func (noopEnforcer) LoadPolicies(context.Context, uuid.UUID) error { return nil }

type noopABACStore struct{}

func (noopABACStore) Create(context.Context, *domain.ABACPolicy) error { return nil }
func (noopABACStore) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.ABACPolicy, error) {
	return nil, nil
}
func (noopABACStore) Update(context.Context, *domain.ABACPolicy) error   { return nil }
func (noopABACStore) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (noopABACStore) ListByOrg(context.Context, uuid.UUID) ([]*domain.ABACPolicy, error) {
	return nil, nil
}

type noopRoleStore struct{}

func (noopRoleStore) Create(context.Context, *domain.Role) error { return nil }
func (noopRoleStore) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Role, error) {
	return nil, nil
}
func (noopRoleStore) GetByName(context.Context, uuid.UUID, string) (*domain.Role, error) {
	return nil, nil
}
func (noopRoleStore) Update(context.Context, *domain.Role) error        { return nil }
func (noopRoleStore) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (noopRoleStore) ListByOrg(context.Context, uuid.UUID) ([]*domain.Role, error) {
	return nil, nil
}
