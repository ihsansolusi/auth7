package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// oauth2ClientWfHandler serves the workflow7 service-task callbacks for the
// OAuth2 client lifecycle (CRUD; no sub-collections), mirroring the user/role
// wf-* pattern. Reuses the package-level helpers (wfEnvelope, dataStr, dataBool,
// dataStrPtr, paramID, wfFail, bindWfEnvelope).
type oauth2ClientWfHandler struct {
	clientSvc *adminOAuth2ClientSvc
	auditSvc  *audit.Service
	logger    zerolog.Logger
}

func newOAuth2ClientWfHandler(clientSvc *adminOAuth2ClientSvc, auditSvc *audit.Service, logger zerolog.Logger) *oauth2ClientWfHandler {
	return &oauth2ClientWfHandler{clientSvc: clientSvc, auditSvc: auditSvc, logger: logger}
}

func (h *oauth2ClientWfHandler) registerRoutes(g *gin.RouterGroup) {
	clients := g.Group("/oauth2/clients")
	{
		clients.POST("/wf-create", h.handleWfCreate)
		clients.PUT("/:id/wf-update", h.handleWfUpdate)
		clients.POST("/:id/wf-delete", h.handleWfDelete)
	}
}

// dataInt reads a numeric field (JSON numbers decode to float64).
func dataInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// dataStrSlice reads an array-of-strings field (e.g. allowed_scopes).
func dataStrSlice(m map[string]any, key string) []string {
	out := []string{}
	if m == nil {
		return out
	}
	arr, ok := m[key].([]any)
	if !ok {
		return out
	}
	for _, it := range arr {
		if s, ok := it.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (h *oauth2ClientWfHandler) audit(orgID, actorID uuid.UUID, actorEmail, action, resourceID string, oldV, newV domain.JSON) {
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: "oauth2_client",
		ResourceID:   resourceID,
		OldValue:     oldV,
		NewValue:     newV,
	})
}

func wfClientToJSON(c *domain.Client) domain.JSON {
	if c == nil {
		return nil
	}
	return domain.JSON{
		"id":         c.ID.String(),
		"client_id":  c.ClientID,
		"name":       c.Name,
		"client_type": string(c.ClientType),
		"is_active":  c.IsActive,
	}
}

func (h *oauth2ClientWfHandler) handleWfCreate(c *gin.Context) {
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	input := adminpkg.CreateClientInput{
		Name:                    dataStr(env.Data, "name"),
		Description:             dataStr(env.Data, "description"),
		ClientType:              domain.ClientType(dataStr(env.Data, "client_type")),
		TokenEndpointAuthMethod: domain.TokenEndpointAuthMethod(dataStr(env.Data, "token_endpoint_auth_method")),
		AllowedScopes:           dataStrSlice(env.Data, "allowed_scopes"),
		AllowedRedirectURIs:     dataStrSlice(env.Data, "allowed_redirect_uris"),
		AllowedOrigins:          dataStrSlice(env.Data, "allowed_origins"),
		TokenExpiration:         dataInt(env.Data, "token_expiration"),
		RefreshTokenExpiration:  dataInt(env.Data, "refresh_token_expiration"),
		AllowMultipleTokens:     dataBool(env.Data, "allow_multiple_tokens"),
		SkipConsentScreen:       dataBool(env.Data, "skip_consent_screen"),
		ClientSecret:            dataStr(env.Data, "client_secret"),
	}
	client, err := h.clientSvc.CreateClient(c.Request.Context(), orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf create oauth2 client failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "create_client", client.ID.String(), nil, wfClientToJSON(client))
	c.JSON(http.StatusOK, gin.H{"id": client.ID.String(), "success": true})
}

func (h *oauth2ClientWfHandler) handleWfUpdate(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldClient, _ := h.clientSvc.GetClient(c.Request.Context(), id)

	// Full-snapshot update (the form sends all editable fields).
	name := dataStr(env.Data, "name")
	desc := dataStr(env.Data, "description")
	scopes := dataStrSlice(env.Data, "allowed_scopes")
	uris := dataStrSlice(env.Data, "allowed_redirect_uris")
	origins := dataStrSlice(env.Data, "allowed_origins")
	tokenExp := dataInt(env.Data, "token_expiration")
	refreshExp := dataInt(env.Data, "refresh_token_expiration")
	allowMulti := dataBool(env.Data, "allow_multiple_tokens")
	skipConsent := dataBool(env.Data, "skip_consent_screen")
	isActive := dataBool(env.Data, "is_active")
	input := adminpkg.UpdateClientInput{
		Name:                   &name,
		Description:            &desc,
		AllowedScopes:          &scopes,
		AllowedRedirectURIs:    &uris,
		AllowedOrigins:         &origins,
		TokenExpiration:        &tokenExp,
		RefreshTokenExpiration: &refreshExp,
		AllowMultipleTokens:    &allowMulti,
		SkipConsentScreen:      &skipConsent,
		IsActive:               &isActive,
	}
	client, err := h.clientSvc.UpdateClient(c.Request.Context(), id, orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf update oauth2 client failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "update_client", id.String(), wfClientToJSON(oldClient), wfClientToJSON(client))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

func (h *oauth2ClientWfHandler) handleWfDelete(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	_, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldClient, _ := h.clientSvc.GetClient(c.Request.Context(), id)
	if err := h.clientSvc.DeleteClient(c.Request.Context(), id); err != nil {
		wfFail(c, h.logger, err, "wf delete oauth2 client failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "delete_client", id.String(), wfClientToJSON(oldClient), nil)
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}
