package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type RoleService interface {
	ListRoles(ctx interface{}, orgID uuid.UUID) ([]*domain.Role, error)
	GetRole(ctx interface{}, id, orgID uuid.UUID) (*domain.Role, error)
	CreateRole(ctx interface{}, orgID uuid.UUID, input CreateRoleInput) (*domain.Role, error)
	UpdateRole(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input UpdateRoleInput) (*domain.Role, error)
	DeleteRole(ctx interface{}, id, orgID uuid.UUID) error
	AssignPermissions(ctx interface{}, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error)
	ListPermissions(ctx interface{}) ([]*domain.Permission, error)
}

type CreateRoleInput struct {
	Code        string
	Name       string
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
		roles.POST("", h.handleCreateRole)
		roles.GET("/:id", h.handleGetRole)
		roles.PUT("/:id", h.handleUpdateRole)
		roles.DELETE("/:id", h.handleDeleteRole)
		roles.GET("/:id/permissions", h.handleGetRolePermissions)
		roles.POST("/:id/permissions", h.handleAssignPermissions)
	}

	permissions := r.Group("/permissions")
	{
		permissions.GET("", h.handleListPermissions)
	}
}

func (h *RoleHandler) handleListRoles(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, err := uuid.Parse(orgStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	roles, err := h.roleSvc.ListRoles(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("list roles failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *RoleHandler) handleCreateRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	var input CreateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	role, err := h.roleSvc.CreateRole(c.Request.Context(), orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("create role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "create_role", "role", role.ID.String(), nil, roleToJSON(role))

	c.JSON(http.StatusCreated, role)
}

func (h *RoleHandler) handleGetRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	role, err := h.roleSvc.GetRole(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("get role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) handleUpdateRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	oldRole, _ := h.roleSvc.GetRole(c.Request.Context(), id, orgID)

	var input UpdateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	role, err := h.roleSvc.UpdateRole(c.Request.Context(), id, orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("update role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "update_role", "role", idStr, roleToJSON(oldRole), roleToJSON(role))

	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) handleDeleteRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	oldRole, _ := h.roleSvc.GetRole(c.Request.Context(), id, orgID)

	if err := h.roleSvc.DeleteRole(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("delete role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "delete_role", "role", idStr, roleToJSON(oldRole), nil)

	c.JSON(http.StatusOK, gin.H{"deleted": true})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func (h *RoleHandler) handleAssignPermissions(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var input struct {
		PermissionIDs []string `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	permIDs := make([]uuid.UUID, len(input.PermissionIDs))
	for i, pStr := range input.PermissionIDs {
		if pid, err := uuid.Parse(pStr); err == nil {
			permIDs[i] = pid
		}
	}

	if err := h.roleSvc.AssignPermissions(c.Request.Context(), id, permIDs); err != nil {
		h.logger.Error().Err(err).Str("role", idStr).Msg("assign permissions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "assign_permissions", "role", idStr, nil, domain.JSON{"permissions": input.PermissionIDs})

	c.JSON(http.StatusOK, gin.H{"assigned": true})
}

func (h *RoleHandler) handleListPermissions(c *gin.Context) {
	permissions, err := h.roleSvc.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("list permissions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func (h *RoleHandler) logAction(orgID uuid.UUID, c *gin.Context, action, resourceType, resourceID string, oldVal, newVal domain.JSON) {
	actorID, actorEmail := getActorFromContext(c)
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValue:     oldVal,
		NewValue:     newVal,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})
}

func roleToJSON(r *domain.Role) domain.JSON {
	if r == nil {
		return nil
	}
	return domain.JSON{
		"id":          r.ID.String(),
		"code":        r.Code,
		"name":        r.Name,
		"description": r.Description,
		"is_default":  r.IsDefault,
	}
}
