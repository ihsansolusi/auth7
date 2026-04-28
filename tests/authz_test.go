package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoleStore struct {
	roles map[uuid.UUID]*domain.Role
}

func newMockRoleStore() *mockRoleStore {
	return &mockRoleStore{
		roles: make(map[uuid.UUID]*domain.Role),
	}
}

func (s *mockRoleStore) Create(ctx context.Context, role *domain.Role) error {
	s.roles[role.ID] = role
	return nil
}

func (s *mockRoleStore) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Role, error) {
	role, ok := s.roles[id]
	if !ok {
		return nil, authz.ErrRoleNotFound
	}
	return role, nil
}

func (s *mockRoleStore) GetByName(ctx context.Context, orgID uuid.UUID, name string) (*domain.Role, error) {
	for _, role := range s.roles {
		if role.OrgID == orgID && role.Name == name {
			return role, nil
		}
	}
	return nil, nil
}

func (s *mockRoleStore) Update(ctx context.Context, role *domain.Role) error {
	s.roles[role.ID] = role
	return nil
}

func (s *mockRoleStore) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	delete(s.roles, id)
	return nil
}

func (s *mockRoleStore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	var result []*domain.Role
	for _, role := range s.roles {
		if role.OrgID == orgID {
			result = append(result, role)
		}
	}
	return result, nil
}

type mockPermissionStore struct {
	perms map[uuid.UUID]*domain.Permission
}

func newMockPermissionStore() *mockPermissionStore {
	return &mockPermissionStore{
		perms: make(map[uuid.UUID]*domain.Permission),
	}
}

func (s *mockPermissionStore) Create(ctx context.Context, perm *domain.Permission) error {
	s.perms[perm.ID] = perm
	return nil
}

func (s *mockPermissionStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	perm, ok := s.perms[id]
	if !ok {
		return nil, authz.ErrPermissionNotFound
	}
	return perm, nil
}

func (s *mockPermissionStore) GetByCode(ctx context.Context, code string) (*domain.Permission, error) {
	for _, perm := range s.perms {
		if perm.Code == code {
			return perm, nil
		}
	}
	return nil, nil
}

func (s *mockPermissionStore) ListByCategory(ctx context.Context, category string) ([]*domain.Permission, error) {
	return nil, nil
}

func (s *mockPermissionStore) ListAll(ctx context.Context) ([]*domain.Permission, error) {
	var result []*domain.Permission
	for _, perm := range s.perms {
		result = append(result, perm)
	}
	return result, nil
}

type mockUserRoleStore struct {
	userRoles map[uuid.UUID]*domain.UserRole
}

func newMockUserRoleStore() *mockUserRoleStore {
	return &mockUserRoleStore{
		userRoles: make(map[uuid.UUID]*domain.UserRole),
	}
}

func (s *mockUserRoleStore) Create(ctx context.Context, ur *domain.UserRole) error {
	s.userRoles[ur.ID] = ur
	return nil
}

func (s *mockUserRoleStore) GetByUser(ctx context.Context, userID, orgID uuid.UUID) ([]*domain.UserRole, error) {
	var result []*domain.UserRole
	for _, ur := range s.userRoles {
		if ur.UserID == userID && ur.OrgID == orgID {
			result = append(result, ur)
		}
	}
	return result, nil
}

func (s *mockUserRoleStore) GetByUserBranch(ctx context.Context, userID, branchID, orgID uuid.UUID) ([]*domain.UserRole, error) {
	var result []*domain.UserRole
	for _, ur := range s.userRoles {
		if ur.UserID == userID && ur.OrgID == orgID && ur.BranchID != nil && *ur.BranchID == branchID {
			result = append(result, ur)
		}
	}
	return result, nil
}

func (s *mockUserRoleStore) Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	if ur, ok := s.userRoles[id]; ok {
		now := time.Now()
		ur.RevokedAt = &now
		ur.RevokedBy = &revokedBy
	}
	return nil
}

type mockABACPolicyStore struct {
	policies map[uuid.UUID]*domain.ABACPolicy
}

func newMockABACPolicyStore() *mockABACPolicyStore {
	return &mockABACPolicyStore{
		policies: make(map[uuid.UUID]*domain.ABACPolicy),
	}
}

func (s *mockABACPolicyStore) Create(ctx context.Context, policy *domain.ABACPolicy) error {
	s.policies[policy.ID] = policy
	return nil
}

func (s *mockABACPolicyStore) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.ABACPolicy, error) {
	policy, ok := s.policies[id]
	if !ok {
		return nil, authz.ErrPolicyNotFound
	}
	return policy, nil
}

func (s *mockABACPolicyStore) Update(ctx context.Context, policy *domain.ABACPolicy) error {
	s.policies[policy.ID] = policy
	return nil
}

func (s *mockABACPolicyStore) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	delete(s.policies, id)
	return nil
}

func (s *mockABACPolicyStore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.ABACPolicy, error) {
	var result []*domain.ABACPolicy
	for _, p := range s.policies {
		if p.OrgID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

type mockEnforcer struct{}

func (e *mockEnforcer) UpdateRolePermissions(ctx context.Context, orgID, roleID uuid.UUID, roleName string, permissions []string) error {
	return nil
}

func (e *mockEnforcer) GrantUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error {
	return nil
}

func (e *mockEnforcer) RevokeUserRole(ctx context.Context, orgID, userID uuid.UUID, roleName string, branchID uuid.UUID) error {
	return nil
}

func (e *mockEnforcer) Enforce(ctx context.Context, authCtx *domain.AuthContext, permission string, resource interface{}) (*domain.AuthorizationResult, error) {
	for _, p := range authCtx.Permissions {
		if p == permission || p == "*" {
			return &domain.AuthorizationResult{
				Allowed: true,
				Reason:  "granted",
			}, nil
		}
	}
	return &domain.AuthorizationResult{
		Allowed: false,
		Reason:  "denied",
	}, nil
}

func (e *mockEnforcer) LoadPolicies(ctx context.Context, orgID uuid.UUID) error {
	return nil
}

func TestAuthzService_CreateRole(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	roleStore := newMockRoleStore()
	permStore := newMockPermissionStore()
	userRoleStore := newMockUserRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	svc := authz.NewService(roleStore, permStore, userRoleStore, policyStore, enforcer)

	params := authz.RoleParams{
		Name:        "teller",
		Description: "Teller role",
		IsDefault:   true,
		Permissions: []string{"transaction:read", "transaction:write"},
	}

	role, err := svc.CreateRole(ctx, orgID, params)
	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "teller", role.Name)
	assert.True(t, role.IsDefault)
	assert.Equal(t, params.Permissions, role.Permissions)
}

func TestAuthzService_AssignRole(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	branchID := uuid.New()
	assignedBy := uuid.New()

	roleStore := newMockRoleStore()
	permStore := newMockPermissionStore()
	userRoleStore := newMockUserRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	svc := authz.NewService(roleStore, permStore, userRoleStore, policyStore, enforcer)

	role := &domain.Role{
		ID:          roleID,
		OrgID:       orgID,
		Name:        "teller",
		Permissions: []string{"transaction:read"},
	}
	_ = roleStore.Create(ctx, role)

	ur, err := svc.AssignRole(ctx, userID, roleID, branchID, orgID, assignedBy)
	require.NoError(t, err)
	assert.NotNil(t, ur)
	assert.Equal(t, userID, ur.UserID)
	assert.Equal(t, roleID, ur.RoleID)
	assert.Equal(t, branchID, ur.BranchID)
	assert.True(t, ur.IsActive())
}

func TestAuthzService_CheckPermission(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	branchID := uuid.New()

	roleStore := newMockRoleStore()
	permStore := newMockPermissionStore()
	userRoleStore := newMockUserRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	svc := authz.NewService(roleStore, permStore, userRoleStore, policyStore, enforcer)

	authCtx := &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		Permissions: []string{"transaction:read", "account:read"},
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := svc.CheckPermission(ctx, authCtx, "transaction:read", nil)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestAuthzService_CheckPermission_Denied(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	branchID := uuid.New()

	roleStore := newMockRoleStore()
	permStore := newMockPermissionStore()
	userRoleStore := newMockUserRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	svc := authz.NewService(roleStore, permStore, userRoleStore, policyStore, enforcer)

	authCtx := &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		Permissions: []string{"transaction:read"},
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := svc.CheckPermission(ctx, authCtx, "admin:write", nil)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "denied", result.Reason)
}

func TestPermissionChecker_CheckBranchScope(t *testing.T) {
	ctx := context.Background()

	roleStore := newMockRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	abac := authz.NewABACEvaluator(policyStore)
	checker := authz.NewPermissionChecker(enforcer, abac, roleStore)

	authCtx := &domain.AuthContext{
		UserID:      uuid.New(),
		OrgID:       uuid.New(),
		BranchID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Permissions: []string{"transaction:read"},
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := checker.CheckBranchScope(ctx, authCtx, uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestABACEvaluator_Evaluate(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	policyStore := newMockABACPolicyStore()
	abac := authz.NewABACEvaluator(policyStore)

	policy := &domain.ABACPolicy{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        "sensitive_data_policy",
		Description: "Mask sensitive fields",
		Priority:    1,
		Effect:      domain.ABACEffectAllow,
		Conditions: map[string]interface{}{
			"permission": "transaction:read",
			"branch_scope": "assigned",
		},
		Fields: []string{"account_number", "balance"},
	}
	_ = policyStore.Create(ctx, policy)

	authCtx := &domain.AuthContext{
		UserID:      uuid.New(),
		OrgID:       orgID,
		BranchID:    uuid.New(),
		Permissions: []string{"transaction:read"},
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := abac.Evaluate(ctx, authCtx, "transaction:read", nil)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.NotEmpty(t, result.FieldMasks)
}

func TestFourLayerAuth_Authorize(t *testing.T) {
	ctx := context.Background()

	roleStore := newMockRoleStore()
	policyStore := newMockABACPolicyStore()
	enforcer := &mockEnforcer{}

	abac := authz.NewABACEvaluator(policyStore)
	checker := authz.NewPermissionChecker(enforcer, abac, roleStore)
	fourLayer := authz.NewFourLayerAuth(checker)

	authCtx := &domain.AuthContext{
		UserID:      uuid.New(),
		OrgID:       uuid.New(),
		BranchID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Permissions: []string{"menu:view", "transaction:read"},
		BranchScope: domain.BranchScopeAssigned,
	}

	params := authz.AuthParams{
		PagePermission: "menu:view",
		DataPermission: "transaction:read",
		ResourceType:   "transaction",
		ResourceID:     uuid.New(),
		TargetBranchID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		SkipPageCheck:  false,
	}

	result, err := fourLayer.Authorize(ctx, authCtx, params)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}