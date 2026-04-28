package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type OAuth2ClientService interface {
	ListClients(ctx interface{}, orgID uuid.UUID) ([]*domain.Client, error)
	GetClient(ctx interface{}, id uuid.UUID) (*domain.Client, error)
	CreateClient(ctx interface{}, orgID uuid.UUID, input CreateClientInput) (*domain.Client, error)
	UpdateClient(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input UpdateClientInput) (*domain.Client, error)
	DeleteClient(ctx interface{}, id uuid.UUID) error
}

type CreateClientInput struct {
	ClientID                string
	Name                    string
	Description             string
	ClientType              domain.ClientType
	TokenEndpointAuthMethod domain.TokenEndpointAuthMethod
	AllowedScopes           []string
	AllowedRedirectURIs     []string
	AllowedOrigins          []string
	TokenExpiration         int
	RefreshTokenExpiration  int
	AllowMultipleTokens     bool
	SkipConsentScreen       bool
}

type UpdateClientInput struct {
	Name                    *string
	Description             *string
	AllowedScopes           *[]string
	AllowedRedirectURIs     *[]string
	AllowedOrigins          *[]string
	TokenExpiration         *int
	RefreshTokenExpiration  *int
	AllowMultipleTokens     *bool
	SkipConsentScreen       *bool
	IsActive                *bool
}

type OAuth2ClientHandler struct {
	clientSvc OAuth2ClientService
	auditSvc  *audit.Service
	logger    zerolog.Logger
}

func NewOAuth2ClientHandler(clientSvc OAuth2ClientService, auditSvc *audit.Service, logger zerolog.Logger) *OAuth2ClientHandler {
	return &OAuth2ClientHandler{
		clientSvc: clientSvc,
		auditSvc:  auditSvc,
		logger:    logger,
	}
}

func (h *OAuth2ClientHandler) RegisterRoutes(r *gin.RouterGroup) {
	clients := r.Group("/oauth2/clients")
	{
		clients.GET("", h.handleListClients)
		clients.POST("", h.handleCreateClient)
		clients.GET("/:id", h.handleGetClient)
		clients.PUT("/:id", h.handleUpdateClient)
		clients.DELETE("/:id", h.handleDeleteClient)
	}
}

func (h *OAuth2ClientHandler) handleListClients(c *gin.Context) {
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

	clients, err := h.clientSvc.ListClients(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("list clients failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func (h *OAuth2ClientHandler) handleCreateClient(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	var input CreateClientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	client, err := h.clientSvc.CreateClient(c.Request.Context(), orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgStr).Msg("create client failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "create_client", "oauth2_client", client.ID.String(), nil, clientToJSON(client))

	c.JSON(http.StatusCreated, client)
}

func (h *OAuth2ClientHandler) handleGetClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	client, err := h.clientSvc.GetClient(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("client", idStr).Msg("get client failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, client)
}

func (h *OAuth2ClientHandler) handleUpdateClient(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	oldClient, _ := h.clientSvc.GetClient(c.Request.Context(), id)

	var input UpdateClientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	client, err := h.clientSvc.UpdateClient(c.Request.Context(), id, orgID, input)
	if err != nil {
		h.logger.Error().Err(err).Str("client", idStr).Msg("update client failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "update_client", "oauth2_client", idStr, clientToJSON(oldClient), clientToJSON(client))

	c.JSON(http.StatusOK, client)
}

func (h *OAuth2ClientHandler) handleDeleteClient(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	oldClient, _ := h.clientSvc.GetClient(c.Request.Context(), id)

	if err := h.clientSvc.DeleteClient(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("client", idStr).Msg("delete client failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "delete_client", "oauth2_client", idStr, clientToJSON(oldClient), nil)

	c.JSON(http.StatusOK, gin.H{"deactivated": true})
}

func (h *OAuth2ClientHandler) logAction(orgID uuid.UUID, c *gin.Context, action, resourceType, resourceID string, oldVal, newVal domain.JSON) {
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

func clientToJSON(c *domain.Client) domain.JSON {
	if c == nil {
		return nil
	}
	return domain.JSON{
		"id":                       c.ID.String(),
		"client_id":                c.ID.String(),
		"name":                     c.Name,
		"client_type":              string(c.ClientType),
		"token_endpoint_auth_method": string(c.TokenEndpointAuthMethod),
		"is_active":                c.IsActive,
		"created_at":               c.CreatedAt.Format(time.RFC3339),
	}
}
