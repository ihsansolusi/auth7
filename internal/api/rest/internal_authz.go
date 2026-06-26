package rest

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"github.com/rs/zerolog"
)

// authz PDP (Policy Decision Point) — M2M endpoints under /internal/v1 that let
// other Core7 services (PEPs) ask auth7 whether a user may perform a permission,
// with the operational-hours time-gate applied. auth7 resolves the user's
// effective permissions from its own role/permission data (authoritative), so
// callers pass only identity + the permission being checked.
//
// Transport is REST (the proto AuthCheckService is not yet codegen'd); the
// decision logic is authz.PermissionChecker, shared with the (future) gRPC path.

// userRolesGetter / rolePermsGetter are the read surface used to resolve a
// user's effective permissions. Satisfied by the concrete admin service
// adapters (adminUserRoleSvc, adminRoleSvc).
type userRolesGetter interface {
	GetUserRoles(ctx interface{}, userID uuid.UUID) ([]*domain.UserRole, error)
}

type rolePermsGetter interface {
	GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error)
}

type authzPDPHandler struct {
	userRoleSvc userRolesGetter
	roleSvc     rolePermsGetter
	checker     *authz.PermissionChecker
	logger      zerolog.Logger
}

func newAuthzPDPHandler(userRoleSvc userRolesGetter, roleSvc rolePermsGetter, checker *authz.PermissionChecker, logger zerolog.Logger) *authzPDPHandler {
	return &authzPDPHandler{userRoleSvc: userRoleSvc, roleSvc: roleSvc, checker: checker, logger: logger}
}

func (h *authzPDPHandler) registerRoutes(g *gin.RouterGroup) {
	authzGroup := g.Group("/authz")
	{
		authzGroup.POST("/check", h.handleCheck)
		authzGroup.POST("/check-data-access", h.handleCheckDataAccess)
	}
}

type pdpCheckRequest struct {
	UserID       string `json:"user_id"`
	OrgID        string `json:"org_id"`
	BranchID     string `json:"branch_id"`
	Permission   string `json:"permission"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

// parseIdentity validates the common identity fields shared by both endpoints.
func (h *authzPDPHandler) parseIdentity(c *gin.Context, req *pdpCheckRequest) (userID, orgID, branchID uuid.UUID, ok bool) {
	var err error
	if userID, err = uuid.Parse(req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if orgID, err = uuid.Parse(req.OrgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if branchID, err = uuid.Parse(req.BranchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if req.Permission == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "permission required"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, branchID, true
}

// resolveAuthContext loads the user's effective (org-wide + this-branch) active
// permissions and builds the ABAC AuthContext.
func (h *authzPDPHandler) resolveAuthContext(ctx context.Context, userID, orgID, branchID uuid.UUID) (*domain.AuthContext, error) {
	userRoles, err := h.userRoleSvc.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	perms := []string{}
	for _, ur := range userRoles {
		if ur == nil || ur.OrgID != orgID || !ur.IsActive() {
			continue
		}
		// org-wide (branch_id NULL) always applies; branch-scoped only for this branch.
		if ur.BranchID != nil && *ur.BranchID != branchID {
			continue
		}
		rolePerms, perr := h.roleSvc.GetPermissions(ctx, ur.RoleID)
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

func (h *authzPDPHandler) handleCheck(c *gin.Context) {
	var req pdpCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	userID, orgID, branchID, ok := h.parseIdentity(c, &req)
	if !ok {
		return
	}

	authCtx, err := h.resolveAuthContext(c.Request.Context(), userID, orgID, branchID)
	if err != nil {
		h.logger.Error().Err(err).Str("user", req.UserID).Msg("pdp resolve auth context failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	result, err := h.checker.CheckPermission(c.Request.Context(), authCtx, req.Permission)
	if err != nil {
		h.logger.Error().Err(err).Str("permission", req.Permission).Msg("pdp check permission failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": result.Allowed, "reason": result.Reason})
}

func (h *authzPDPHandler) handleCheckDataAccess(c *gin.Context) {
	var req pdpCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	userID, orgID, branchID, ok := h.parseIdentity(c, &req)
	if !ok {
		return
	}
	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resource_id"})
		return
	}

	authCtx, err := h.resolveAuthContext(c.Request.Context(), userID, orgID, branchID)
	if err != nil {
		h.logger.Error().Err(err).Str("user", req.UserID).Msg("pdp resolve auth context failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	result, err := h.checker.CheckDataAccess(c.Request.Context(), authCtx, req.Permission, req.ResourceType, resourceID)
	if err != nil {
		h.logger.Error().Err(err).Str("permission", req.Permission).Msg("pdp check data access failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"allowed":     result.Allowed,
		"reason":      result.Reason,
		"field_masks": result.FieldMasks,
	})
}

// ── no-op authz stores ──────────────────────────────────────────────────────
// The PDP path pre-populates AuthContext.Permissions and relies on the
// role-based check + time-gate (+ empty-ABAC = allow-by-default). The enforcer,
// ABAC policy store, and role store are not exercised, but PermissionChecker /
// ABACEvaluator require non-nil deps — these satisfy the interfaces.

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

func (noopABACStore) Create(context.Context, *domain.ABACPolicy) error            { return nil }
func (noopABACStore) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.ABACPolicy, error) {
	return nil, nil
}
func (noopABACStore) Update(context.Context, *domain.ABACPolicy) error            { return nil }
func (noopABACStore) Delete(context.Context, uuid.UUID, uuid.UUID) error          { return nil }
func (noopABACStore) ListByOrg(context.Context, uuid.UUID) ([]*domain.ABACPolicy, error) {
	return nil, nil
}

type noopRoleStore struct{}

func (noopRoleStore) Create(context.Context, *domain.Role) error                       { return nil }
func (noopRoleStore) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Role, error) {
	return nil, nil
}
func (noopRoleStore) GetByName(context.Context, uuid.UUID, string) (*domain.Role, error) {
	return nil, nil
}
func (noopRoleStore) Update(context.Context, *domain.Role) error              { return nil }
func (noopRoleStore) Delete(context.Context, uuid.UUID, uuid.UUID) error      { return nil }
func (noopRoleStore) ListByOrg(context.Context, uuid.UUID) ([]*domain.Role, error) {
	return nil, nil
}

// newTimeGatedChecker builds a PermissionChecker for the PDP: role-based check +
// (optional) operational-hours time-gate + allow-by-default ABAC.
func newTimeGatedChecker(tw *authz.TimeWindowEvaluator, gatedPerms []string) *authz.PermissionChecker {
	checker := authz.NewPermissionChecker(noopEnforcer{}, authz.NewABACEvaluator(noopABACStore{}), noopRoleStore{})
	if tw != nil {
		checker = checker.WithTimeGate(tw, gatedPerms)
	}
	return checker
}
