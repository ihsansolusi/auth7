package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/api/grpc/authcheck"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/rs/zerolog"
)

// NewAuthCheckGRPCServer builds the gRPC AuthCheckService server (lib7
// auth7grpc contract) sharing the same decision core as the REST PDP. Returns
// nil if the store is not a *postgres.Store. Started by cmd/server/start.go.
func NewAuthCheckGRPCServer(deps ServerDeps) *authcheck.Server {
	store, ok := deps.Store.(*postgres.Store)
	if !ok {
		return nil
	}
	checker := authz.NewTimeGatedChecker(deps.TimeWindow, deps.TimeGatedPermissions)
	return authcheck.NewServer(newAdminUserRoleSvc(store), newAdminRoleSvc(store), checker, deps.Logger)
}

// authz PDP (Policy Decision Point) — M2M REST endpoints under /internal/v1 that
// let other Core7 services (PEPs) ask auth7 whether a user may perform a
// permission, with the operational-hours time-gate applied. auth7 resolves the
// user's effective permissions from its own role data (authoritative).
//
// This is the REST surface; the gRPC surface (lib7 auth7grpc contract) lives in
// internal/api/grpc/authcheck and shares the same authz.PermissionChecker +
// authz.ResolveAuthContext decision core.

type authzPDPHandler struct {
	userRoleSvc authz.UserRolesGetter
	roleSvc     authz.RolePermsGetter
	checker     *authz.PermissionChecker
	logger      zerolog.Logger
}

func newAuthzPDPHandler(userRoleSvc authz.UserRolesGetter, roleSvc authz.RolePermsGetter, checker *authz.PermissionChecker, logger zerolog.Logger) *authzPDPHandler {
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

	authCtx, err := authz.ResolveAuthContext(c.Request.Context(), h.userRoleSvc, h.roleSvc, userID, orgID, branchID)
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

	authCtx, err := authz.ResolveAuthContext(c.Request.Context(), h.userRoleSvc, h.roleSvc, userID, orgID, branchID)
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
