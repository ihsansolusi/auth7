package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// OAuth2ClientService is the read surface for the admin HTTP API. Mutations flow
// through workflow7 → the M2M /internal/v1 wf-callbacks; the concrete adapter
// still implements create/update/delete for those callbacks.
type OAuth2ClientService interface {
	ListClients(ctx interface{}, orgID uuid.UUID) ([]*domain.Client, error)
	GetClient(ctx interface{}, id uuid.UUID) (*domain.Client, error)
}

// CreateClientInput / UpdateClientInput are the lifecycle inputs consumed by the
// wf-callback handlers; kept here as the shared contract.
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
	ClientSecret            string
}

type UpdateClientInput struct {
	Name                   *string
	Description            *string
	AllowedScopes          *[]string
	AllowedRedirectURIs    *[]string
	AllowedOrigins         *[]string
	TokenExpiration        *int
	RefreshTokenExpiration *int
	AllowMultipleTokens    *bool
	SkipConsentScreen      *bool
	IsActive               *bool
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
		clients.GET("/:id", h.handleGetClient)
	}
}

func (h *OAuth2ClientHandler) handleListClients(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}

	clients, err := h.clientSvc.ListClients(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org", orgID.String()).Msg("list clients failed")
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
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
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, client)
}
