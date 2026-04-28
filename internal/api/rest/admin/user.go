package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type UserService interface {
	ListUsers(ctx interface{}, orgID uuid.UUID, limit, offset int, status string) ([]*domain.User, int, error)
	GetUser(ctx interface{}, id, orgID uuid.UUID) (*domain.User, error)
	CreateUser(ctx interface{}, orgID uuid.UUID, input CreateUserInput) (*domain.User, error)
	UpdateUser(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input UpdateUserInput) (*domain.User, error)
	DeleteUser(ctx interface{}, id, orgID uuid.UUID) error
	LockUser(ctx interface{}, id, orgID uuid.UUID) error
	UnlockUser(ctx interface{}, id, orgID uuid.UUID) error
	SuspendUser(ctx interface{}, id, orgID uuid.UUID) error
}

type CreateUserInput struct {
	Username  string
	Email     string
	FullName  string
	Password  string
	CreatedBy uuid.UUID
}

type UpdateUserInput struct {
	Username  *string
	Email     *string
	FullName  *string
	Status    *domain.UserStatus
	UpdatedBy *uuid.UUID
}

type UserHandler struct {
	userSvc    UserService
	auditSvc   *audit.Service
	logger     zerolog.Logger
}

func NewUserHandler(userSvc UserService, auditSvc *audit.Service, logger zerolog.Logger) *UserHandler {
	return &UserHandler{
		userSvc:  userSvc,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.handleListUsers)
		users.POST("", h.handleCreateUser)
		users.GET("/:id", h.handleGetUser)
		users.PUT("/:id", h.handleUpdateUser)
		users.DELETE("/:id", h.handleDeleteUser)
		users.POST("/:id/lock", h.handleLockUser)
		users.POST("/:id/unlock", h.handleUnlockUser)
		users.POST("/:id/suspend", h.handleSuspendUser)
	}
}

func (h *UserHandler) handleListUsers(c *gin.Context) {
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

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")

	users, total, err := h.userSvc.ListUsers(c.Request.Context(), orgID, limit, offset, status)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("list users failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

func (h *UserHandler) handleCreateUser(c *gin.Context) {
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

	var input CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	user, err := h.userSvc.CreateUser(c.Request.Context(), orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("create user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "create_user", "user", user.ID.String(), nil, userToJSON(user))

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) handleGetUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("get user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) handleUpdateUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	user, err := h.userSvc.UpdateUser(c.Request.Context(), id, orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("update user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "update_user", "user", idStr, userToJSON(oldUser), userToJSON(user))

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) handleDeleteUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	if err := h.userSvc.DeleteUser(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("delete user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "delete_user", "user", idStr, userToJSON(oldUser), nil)

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *UserHandler) handleLockUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	if err := h.userSvc.LockUser(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("lock user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "lock_user", "user", idStr, userToJSON(oldUser), nil)

	c.JSON(http.StatusOK, gin.H{"locked": true})
}

func (h *UserHandler) handleUnlockUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	if err := h.userSvc.UnlockUser(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("unlock user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "unlock_user", "user", idStr, userToJSON(oldUser), nil)

	c.JSON(http.StatusOK, gin.H{"unlocked": true})
}

func (h *UserHandler) handleSuspendUser(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	if err := h.userSvc.SuspendUser(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("suspend user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "suspend_user", "user", idStr, userToJSON(oldUser), nil)

	c.JSON(http.StatusOK, gin.H{"suspended": true})
}

func (h *UserHandler) logAction(orgID uuid.UUID, c *gin.Context, action, resourceType, resourceID string, oldVal, newVal domain.JSON) {
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

func getActorFromContext(c *gin.Context) (uuid.UUID, string) {
	claims, ok := c.Get("claims")
	if !ok {
		return uuid.Nil, ""
	}
	tokenClaims, ok := claims.(interface {
		GetSubject() string
		GetEmail() string
	})
	if !ok {
		return uuid.Nil, ""
	}
	var actorID uuid.UUID
	if id, err := uuid.Parse(tokenClaims.GetSubject()); err == nil {
		actorID = id
	}
	return actorID, tokenClaims.GetEmail()
}

func userToJSON(u *domain.User) domain.JSON {
	if u == nil {
		return nil
	}
	return domain.JSON{
		"id":          u.ID.String(),
		"username":    u.Username,
		"email":       u.Email,
		"full_name":   u.FullName,
		"status":      string(u.Status),
		"locked_until": "",
		"created_at":  u.CreatedAt.Format(time.RFC3339),
	}
}
