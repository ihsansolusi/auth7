package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// UserService is the read surface used by the admin HTTP API. Mutations are not
// exposed here — they flow through workflow7 → the M2M /internal/v1 wf-callbacks.
// The concrete adapter still implements the full lifecycle for those callbacks.
type UserService interface {
	ListUsers(ctx interface{}, orgID uuid.UUID, limit, offset int, status string) ([]*domain.User, int, error)
	GetUser(ctx interface{}, id, orgID uuid.UUID) (*domain.User, error)
}

// CreateUserInput / UpdateUserInput are the lifecycle inputs consumed by the
// wf-callback handlers (admin.CreateUserInput etc.); kept here as the shared
// contract even though the admin API no longer exposes create/update directly.
type CreateUserInput struct {
	Username              string
	Email                 string
	FullName              string
	PreferredLocale       string
	Password              string
	RequirePasswordChange bool
	CreatedBy             uuid.UUID
}

type UpdateUserInput struct {
	Username  *string
	Email     *string
	FullName  *string
	Status    *domain.UserStatus
	UpdatedBy *uuid.UUID
}

type UserHandler struct {
	userSvc  UserService
	auditSvc *audit.Service
	logger   zerolog.Logger
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
		users.GET("/:id", h.handleGetUser)
	}
}

func (h *UserHandler) handleListUsers(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")

	users, total, err := h.userSvc.ListUsers(c.Request.Context(), orgID, limit, offset, status)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgID.String()).Msg("list users failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *UserHandler) handleGetUser(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("user", idStr).Msg("get user failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// getActorFromContext extracts the acting admin's identity from JWT claims.
// Shared across admin handlers (e.g. session revoke audit).
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
