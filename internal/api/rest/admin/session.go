package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/rs/zerolog"
)

// AdminSessionService is the subset of session.Service used by SessionHandler.
type AdminSessionService interface {
	ListAllSessions(ctx context.Context) ([]*session.SessionData, error)
	GetSession(ctx context.Context, sessionID string) (*session.SessionData, error)
	RevokeSession(ctx context.Context, sessionID string) error
}

// SessionListItem is the response shape for each active session.
type SessionListItem struct {
	SessionID      string `json:"session_id"`
	UserID         string `json:"user_id"`
	Username       string `json:"username,omitempty"`
	OrgID          string `json:"org_id"`
	IPAddress      string `json:"ip_address"`
	UserAgent      string `json:"user_agent"`
	DeviceInfo     string `json:"device_info"`
	ActiveBranchID string `json:"active_branch_id"`
	MFAVerified    bool   `json:"mfa_verified"`
	CreatedAt      string `json:"created_at"`
	LastUsedAt     string `json:"last_used_at"`
	ExpiresAt      string `json:"expires_at"`
}

type SessionHandler struct {
	sessionSvc AdminSessionService
	auditSvc   *audit.Service
	logger     zerolog.Logger
}

func NewSessionHandler(sessionSvc AdminSessionService, auditSvc *audit.Service, logger zerolog.Logger) *SessionHandler {
	return &SessionHandler{
		sessionSvc: sessionSvc,
		auditSvc:   auditSvc,
		logger:     logger,
	}
}

func (h *SessionHandler) RegisterRoutes(r *gin.RouterGroup) {
	sessions := r.Group("/sessions")
	{
		sessions.GET("", h.handleListSessions)
		sessions.DELETE("/:id", h.handleRevokeSession)
	}
}

func (h *SessionHandler) handleListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	all, err := h.sessionSvc.ListAllSessions(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("list sessions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	total := len(all)
	offset := (page - 1) * pageSize

	var items []SessionListItem
	if offset < total {
		end := offset + pageSize
		if end > total {
			end = total
		}
		for _, s := range all[offset:end] {
			items = append(items, sessionToItem(s))
		}
	}
	if items == nil {
		items = []SessionListItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions":  items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *SessionHandler) handleRevokeSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	sess, err := h.sessionSvc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("get session failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	if sess == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}

	if err := h.sessionSvc.RevokeSession(c.Request.Context(), sessionID); err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("revoke session failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	var orgID uuid.UUID
	if sess.OrgID != "" {
		orgID, _ = uuid.Parse(sess.OrgID)
	}
	actorID, actorEmail := getActorFromContext(c)
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       "revoke_session",
		ResourceType: "session",
		ResourceID:   sessionID,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

func sessionToItem(s *session.SessionData) SessionListItem {
	lastUsed := ""
	if s.LastUsedAt > 0 {
		lastUsed = time.Unix(s.LastUsedAt, 0).UTC().Format(time.RFC3339)
	}
	return SessionListItem{
		SessionID:      s.ID,
		UserID:         s.UserID,
		OrgID:          s.OrgID,
		IPAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
		DeviceInfo:     s.DeviceInfo,
		ActiveBranchID: s.ActiveBranchID,
		MFAVerified:    s.MFAVerified,
		CreatedAt:      time.Unix(s.CreatedAt, 0).UTC().Format(time.RFC3339),
		LastUsedAt:     lastUsed,
		ExpiresAt:      time.Unix(s.ExpiresAt, 0).UTC().Format(time.RFC3339),
	}
}
