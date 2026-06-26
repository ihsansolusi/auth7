package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// RoleService is the read surface for the admin HTTP API. Mutations flow through
// workflow7 → the M2M /internal/v1 wf-callbacks; the concrete adapter still
// implements create/update/delete/assign for those callbacks.
type RoleService interface {
	ListRoles(ctx interface{}, orgID uuid.UUID) ([]*domain.Role, error)
	GetRole(ctx interface{}, id, orgID uuid.UUID) (*domain.Role, error)
	GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error)
	ListPermissions(ctx interface{}) ([]*domain.Permission, error)
}

// CreateRoleInput / UpdateRoleInput are the lifecycle inputs consumed by the
// wf-callback handlers; kept here as the shared contract.
type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
	IsDefault   bool
}

type UpdateRoleInput struct {
	Name        *string
	Description *string
}

type RoleHandler struct {
	roleSvc  RoleService
	auditSvc *audit.Service
	logger   zerolog.Logger
}

func NewRoleHandler(roleSvc RoleService, auditSvc *audit.Service, logger zerolog.Logger) *RoleHandler {
	return &RoleHandler{
		roleSvc:  roleSvc,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

func (h *RoleHandler) RegisterRoutes(r *gin.RouterGroup) {
	roles := r.Group("/roles")
	{
		roles.GET("", h.handleListRoles)
		roles.GET("/:id", h.handleGetRole)
		roles.GET("/:id/permissions", h.handleGetRolePermissions)
	}

	permissions := r.Group("/permissions")
	{
		permissions.GET("", h.handleListPermissions)
	}
}

func (h *RoleHandler) handleListRoles(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}

	roles, err := h.roleSvc.ListRoles(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgID.String()).Msg("list roles failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *RoleHandler) handleGetRole(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	role, err := h.roleSvc.GetRole(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("get role failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) handleGetRolePermissions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	permissions, err := h.roleSvc.GetPermissions(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("get role permissions failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func (h *RoleHandler) handleListPermissions(c *gin.Context) {
	permissions, err := h.roleSvc.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("list permissions failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}
