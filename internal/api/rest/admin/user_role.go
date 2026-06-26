package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// UserRoleService is the read surface for the admin HTTP API. Assign/revoke flow
// through workflow7 → the M2M /internal/v1 wf-callbacks; the concrete adapter
// still implements them for those callbacks.
type UserRoleService interface {
	GetUserRoles(ctx interface{}, userID uuid.UUID) ([]*domain.UserRole, error)
	GetBranchRoles(ctx interface{}, branchID uuid.UUID) ([]*domain.UserRole, error)
}

type UserRoleHandler struct {
	userRoleSvc UserRoleService
	auditSvc    *audit.Service
	logger      zerolog.Logger
}

func NewUserRoleHandler(userRoleSvc UserRoleService, auditSvc *audit.Service, logger zerolog.Logger) *UserRoleHandler {
	return &UserRoleHandler{
		userRoleSvc: userRoleSvc,
		auditSvc:    auditSvc,
		logger:      logger,
	}
}

func (h *UserRoleHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/users/:id/roles", h.handleGetUserRoles)
	r.GET("/branches/:id/roles", h.handleGetBranchRoles)
}

func (h *UserRoleHandler) handleGetUserRoles(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	roles, err := h.userRoleSvc.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get user roles failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *UserRoleHandler) handleGetBranchRoles(c *gin.Context) {
	branchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}

	roles, err := h.userRoleSvc.GetBranchRoles(c.Request.Context(), branchID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get branch roles failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}
