package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ihsansolusi/auth7/internal/domain"
)

func (s *Server) handleListApps(c *gin.Context) {
	if s.deps.OAuth2ClientSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "apps service unavailable"})
		return
	}
	apps, err := s.deps.OAuth2ClientSvc.ListApps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list apps"})
		return
	}
	if apps == nil {
		apps = []*domain.AppEntry{}
	}
	c.JSON(http.StatusOK, gin.H{"apps": apps})
}
