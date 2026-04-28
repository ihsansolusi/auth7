package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type UserRoleService interface {
	AssignRole(ctx interface{}, userID, roleID, orgID uuid.UUID, branchID *uuid.UUID, grantedBy uuid.UUID) (*domain.UserRole, error)
	RevokeRole(ctx interface{}, userID, roleID, orgID, revokedBy uuid.UUID) error
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
	r.POST("/users/:id/roles", h.handleAssignRole)
	r.DELETE("/users/:user_id/roles/:role_id", h.handleRevokeRole)
	r.GET("/users/:id/roles", h.handleGetUserRoles)
	r.GET("/branches/:id/roles", h.handleGetBranchRoles)
}

func (h *UserRoleHandler) handleAssignRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var input struct {
		RoleID    string     `json:"role_id"`
		BranchID  string     `json:"branch_id,omitempty"`
		GrantedBy string     `json:"granted_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	roleID, _ := uuid.Parse(input.RoleID)
	var branchID *uuid.UUID
	if input.BranchID != "" {
		bid, _ := uuid.Parse(input.BranchID)
		branchID = &bid
	}
	grantedBy, _ := uuid.Parse(input.GrantedBy)

	assignment, err := h.userRoleSvc.AssignRole(c.Request.Context(), userID, roleID, orgID, branchID, grantedBy)
	if err != nil {
		h.logger.Error().Err(err).Msg("assign role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "assign_role", "user_role", assignment.ID.String(), nil, userRoleToJSON(assignment))

	c.JSON(http.StatusCreated, assignment)
}

func (h *UserRoleHandler) handleRevokeRole(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	userID, _ := uuid.Parse(c.Param("user_id"))
	roleID, _ := uuid.Parse(c.Param("role_id"))

	if err := h.userRoleSvc.RevokeRole(c.Request.Context(), userID, roleID, orgID, uuid.Nil); err != nil {
		h.logger.Error().Err(err).Msg("revoke role failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "revoke_role", "user_role", roleID.String(), nil, nil)

	c.JSON(http.StatusOK, gin.H{"revoked": true})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *UserRoleHandler) logAction(orgID uuid.UUID, c *gin.Context, action, resourceType, resourceID string, oldVal, newVal domain.JSON) {
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

func userRoleToJSON(ur *domain.UserRole) domain.JSON {
	if ur == nil {
		return nil
	}
	return domain.JSON{
		"id":        ur.ID.String(),
		"user_id":   ur.UserID.String(),
		"role_id":   ur.RoleID.String(),
		"org_id":    ur.OrgID.String(),
	}
}
