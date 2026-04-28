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

type AuditService interface {
	Query(filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error)
}

type AuditHandler struct {
	auditSvc *audit.Service
	logger   zerolog.Logger
}

func NewAuditHandler(auditSvc *audit.Service, logger zerolog.Logger) *AuditHandler {
	return &AuditHandler{
		auditSvc: auditSvc,
		logger:   logger,
	}
}

func (h *AuditHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/audit-logs", h.handleQueryAuditLogs)
}

func (h *AuditHandler) handleQueryAuditLogs(c *gin.Context) {
	filter := domain.AuditLogFilter{}

	if orgStr := c.Query("org_id"); orgStr != "" {
		orgID, err := uuid.Parse(orgStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		filter.OrgID = &orgID
	}

	if actorStr := c.Query("actor_id"); actorStr != "" {
		actorID, err := uuid.Parse(actorStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid actor_id"})
			return
		}
		filter.ActorID = &actorID
	}

	filter.Action = c.Query("action")
	filter.ResourceType = c.Query("resource_type")
	filter.ResourceID = c.Query("resource_id")

	if fromStr := c.Query("from_date"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.FromDate = &t
		}
	}

	if toStr := c.Query("to_date"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.ToDate = &t
		}
	}

	filter.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	filter.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := h.auditSvc.Query(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("query audit logs failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": logs,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})
}
