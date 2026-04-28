package authz

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

type CasbinAdapter struct {
	roleStore RoleStore
}

func NewCasbinAdapter(roleStore RoleStore) *CasbinAdapter {
	return &CasbinAdapter{
		roleStore: roleStore,
	}
}

type CasbinRule struct {
	ID   uint   `json:"id"`
	P0   string `json:"p0"`
	P1   string `json:"p1"`
	P2   string `json:"p2"`
	P3   string `json:"p3"`
	P4   string `json:"p4"`
	P5   string `json:"p5"`
}

func (c *CasbinAdapter) LoadPolicy(orgID uuid.UUID) ([]*CasbinRule, error) {
	return nil, nil
}

func (c *CasbinAdapter) SavePolicy(orgID uuid.UUID, rules []*CasbinRule) error {
	return nil
}

func (c *CasbinAdapter) UpdatePolicy(orgID uuid.UUID, oldRules, newRules []*CasbinRule) error {
	return nil
}

func (c *CasbinAdapter) AddPolicy(orgID uuid.UUID, rule *CasbinRule) error {
	return nil
}

func (c *CasbinAdapter) RemovePolicy(orgID uuid.UUID, rule *CasbinRule) error {
	return nil
}

func (c *CasbinAdapter) RemoveFilteredPolicy(orgID uuid.UUID, fieldIndex int, fieldValues ...string) error {
	return nil
}

type CasbinEnforcer struct {
	adapter     *CasbinAdapter
	roleStore   RoleStore
	userRoleStore UserRoleStore
	policyStore ABACPolicyStore
}

func NewCasbinEnforcer(
	adapter *CasbinAdapter,
	roleStore RoleStore,
	userRoleStore UserRoleStore,
	policyStore ABACPolicyStore,
) *CasbinEnforcer {
	return &CasbinEnforcer{
		adapter:      adapter,
		roleStore:    roleStore,
		userRoleStore: userRoleStore,
		policyStore:  policyStore,
	}
}

func (e *CasbinEnforcer) UpdateRolePermissions(ctx context.Context, orgID, roleID uuid.UUID, roleName string, permissions []string) error {
	return nil
}

func (e *CasbinEnforcer) GrantUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error {
	return nil
}

func (e *CasbinEnforcer) RevokeUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error {
	return nil
}

func (e *CasbinEnforcer) Enforce(ctx context.Context, authCtx *domain.AuthContext, permission string, resource interface{}) (*domain.AuthorizationResult, error) {
	hasPermission := false
	for _, p := range authCtx.Permissions {
		if p == permission || p == "*" {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return &domain.AuthorizationResult{
			Allowed: false,
			Reason:  "permission denied",
		}, nil
	}

	return &domain.AuthorizationResult{
		Allowed: true,
		Reason:  "permission granted",
	}, nil
}

func (e *CasbinEnforcer) LoadPolicies(ctx context.Context, orgID uuid.UUID) error {
	return nil
}