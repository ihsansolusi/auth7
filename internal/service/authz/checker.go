package authz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

type PermissionChecker struct {
	enforcer  Enforcer
	abac      *ABACEvaluator
	roleStore RoleStore
}

func NewPermissionChecker(
	enforcer Enforcer,
	abac *ABACEvaluator,
	roleStore RoleStore,
) *PermissionChecker {
	return &PermissionChecker{
		enforcer:  enforcer,
		abac:      abac,
		roleStore: roleStore,
	}
}

func (c *PermissionChecker) CheckPermission(ctx context.Context, authCtx *domain.AuthContext, permission string) (*domain.AuthorizationResult, error) {
	if authCtx == nil {
		return &domain.AuthorizationResult{
			Allowed: false,
			Reason:  "empty auth context",
		}, nil
	}

	if len(authCtx.Roles) == 0 && len(authCtx.Permissions) == 0 {
		perms, err := c.getUserPermissionsFromStore(ctx, authCtx.UserID, authCtx.OrgID, authCtx.BranchID)
		if err == nil {
			authCtx.Permissions = perms
		}
	}

	if c.hasPermission(authCtx, permission) {
		return &domain.AuthorizationResult{
			Allowed: true,
			Reason:  "permission granted",
		}, nil
	}

	return &domain.AuthorizationResult{
		Allowed: false,
		Reason:  fmt.Sprintf("permission '%s' denied", permission),
	}, nil
}

func (c *PermissionChecker) CheckDataAccess(ctx context.Context, authCtx *domain.AuthContext, permission, resourceType string, resourceID uuid.UUID) (*domain.AuthorizationResult, error) {
	result, err := c.CheckPermission(ctx, authCtx, permission)
	if err != nil || !result.Allowed {
		return result, err
	}

	if c.needsABACCheck(permission) {
		return c.abac.Evaluate(ctx, authCtx, permission, map[string]interface{}{
			"type": resourceType,
			"id":   resourceID.String(),
		})
	}

	return result, nil
}

func (c *PermissionChecker) CheckBranchScope(ctx context.Context, authCtx *domain.AuthContext, targetBranchID uuid.UUID) (*domain.AuthorizationResult, error) {
	if authCtx.BranchScope == domain.BranchScopeAll {
		return &domain.AuthorizationResult{
			Allowed: true,
			Reason:  "branch scope: all",
		}, nil
	}

	if authCtx.BranchScope == domain.BranchScopeAssigned {
		if authCtx.BranchID == targetBranchID {
			return &domain.AuthorizationResult{
				Allowed: true,
				Reason:  "branch scope: assigned",
			}, nil
		}
		return &domain.AuthorizationResult{
			Allowed: false,
			Reason:  "branch scope: not in assigned branch",
		}, nil
	}

	if authCtx.BranchScope == domain.BranchScopeOwn {
		return &domain.AuthorizationResult{
			Allowed: authCtx.BranchID == targetBranchID,
			Reason:  "branch scope: own data only",
		}, nil
	}

	return &domain.AuthorizationResult{
		Allowed: false,
		Reason:  "unknown branch scope",
	}, nil
}

func (c *PermissionChecker) CheckFieldAccess(ctx context.Context, authCtx *domain.AuthContext, permission string, fields []string) (map[string]bool, error) {
	allowed := make(map[string]bool)

	for _, field := range fields {
		allowed[field] = c.hasFieldPermission(authCtx, permission, field)
	}

	return allowed, nil
}

func (c *PermissionChecker) ApplyFieldMasks(ctx context.Context, authCtx *domain.AuthContext, data map[string]interface{}, fields []string) (map[string]interface{}, error) {
	if len(authCtx.FieldMasks) == 0 {
		return data, nil
	}

	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}

	for _, field := range fields {
		for _, mask := range authCtx.FieldMasks {
			if mask.Field == field {
				result[field] = mask.MaskValue
			}
		}
	}

	return result, nil
}

func (c *PermissionChecker) hasPermission(authCtx *domain.AuthContext, permission string) bool {
	for _, p := range authCtx.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

func (c *PermissionChecker) hasFieldPermission(authCtx *domain.AuthContext, permission, field string) bool {
	if len(authCtx.FieldMasks) == 0 {
		return true
	}

	for _, m := range authCtx.FieldMasks {
		if m.Field == field {
			return false
		}
	}

	return c.hasPermission(authCtx, permission)
}

func (c *PermissionChecker) needsABACCheck(permission string) bool {
	sensitive := []string{
		"transaction:read",
		"account:read",
		"customer:read",
		"report:generate",
	}
	for _, s := range sensitive {
		if s == permission {
			return true
		}
	}
	return false
}

func (c *PermissionChecker) getUserPermissionsFromStore(ctx context.Context, userID, orgID, branchID uuid.UUID) ([]string, error) {
	return nil, nil
}

type FourLayerAuth struct {
	checker *PermissionChecker
}

func NewFourLayerAuth(checker *PermissionChecker) *FourLayerAuth {
	return &FourLayerAuth{
		checker: checker,
	}
}

func (f *FourLayerAuth) Authorize(ctx context.Context, authCtx *domain.AuthContext, params AuthParams) (*domain.AuthorizationResult, error) {
	if !params.SkipPageCheck {
		pageResult, err := f.checker.CheckPermission(ctx, authCtx, params.PagePermission)
		if err != nil || !pageResult.Allowed {
			return pageResult, err
		}
	}

	dataResult, err := f.checker.CheckDataAccess(ctx, authCtx, params.DataPermission, params.ResourceType, params.ResourceID)
	if err != nil || !dataResult.Allowed {
		return dataResult, err
	}

	branchResult, err := f.checker.CheckBranchScope(ctx, authCtx, params.TargetBranchID)
	if err != nil || !branchResult.Allowed {
		return branchResult, err
	}

	return &domain.AuthorizationResult{
		Allowed:    true,
		Reason:     "all layers passed",
		FieldMasks: dataResult.FieldMasks,
	}, nil
}

type AuthParams struct {
	PagePermission   string
	DataPermission   string
	ResourceType     string
	ResourceID       uuid.UUID
	TargetBranchID   uuid.UUID
	SkipPageCheck    bool
}