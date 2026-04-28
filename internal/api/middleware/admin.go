package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

const (
	opAdminAuth = "middleware.AdminAuth"
)

type AdminAuthConfig struct {
	RateLimit      int
	BurstLimit     int
	AllowedRoles   map[string]bool
	RequireOrgMatch bool
}

func DefaultAdminAuthConfig() AdminAuthConfig {
	return AdminAuthConfig{
		RateLimit:      10,
		BurstLimit:     20,
		AllowedRoles:   map[string]bool{"admin": true, "super_admin": true},
		RequireOrgMatch: true,
	}
}

type AdminRateLimiter struct {
	tokens    map[string][]time.Time
	mu        sync.Mutex
	rateLimit int
	burst     int
	window    time.Duration
}

func NewAdminRateLimiter(rateLimit, burst int) *AdminRateLimiter {
	return &AdminRateLimiter{
		tokens:    make(map[string][]time.Time),
		rateLimit: rateLimit,
		burst:     burst,
		window:    time.Second,
	}
}

func (r *AdminRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	tokens := r.tokens[key]
	valid := make([]time.Time, 0)
	for _, t := range tokens {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.burst {
		r.tokens[key] = valid
		return false
	}

	if len(valid) >= r.rateLimit {
		r.tokens[key] = valid
		return false
	}

	valid = append(valid, now)
	r.tokens[key] = valid
	return true
}

func AdminAuth(cfg AdminAuthConfig, auditLogger *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	limiter := NewAdminRateLimiter(cfg.RateLimit, cfg.BurstLimit)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			logger.Warn().
				Str("ip", ip).
				Str("path", c.Request.URL.Path).
				Msg("admin rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" || len(auth) < 8 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			c.Abort()
			return
		}

		claims, ok := c.Get("claims")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			c.Abort()
			return
		}

		tokenClaims, ok := claims.(interface {
			GetSubject() string
			GetOrgID() string
			GetRoles() []string
		})
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range tokenClaims.GetRoles() {
			if cfg.AllowedRoles[role] {
				hasRole = true
				break
			}
		}

		if !hasRole {
			logger.Warn().
				Str("subject", tokenClaims.GetSubject()).
				Strs("roles", tokenClaims.GetRoles()).
				Str("path", c.Request.URL.Path).
				Msg("admin access denied - insufficient role")
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			c.Abort()
			return
		}

		if cfg.RequireOrgMatch {
			orgID := c.Query("org_id")
			tokenOrgID := tokenClaims.GetOrgID()
			if orgID != "" && tokenOrgID != "" && orgID != tokenOrgID {
				logger.Warn().
					Str("subject", tokenClaims.GetSubject()).
					Str("token_org", tokenOrgID).
					Str("request_org", orgID).
					Msg("admin org mismatch")
				c.JSON(http.StatusForbidden, gin.H{"error": "org mismatch"})
				c.Abort()
				return
			}
		}

		c.Set("admin", true)
		c.Next()
	}
}

func GetActorFromContext(c *gin.Context) (uuid.UUID, string) {
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

func LogAdminAction(auditLogger *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if !c.IsAborted() {
			actorID, actorEmail := GetActorFromContext(c)

			orgStr := c.Query("org_id")
			var orgID uuid.UUID
			if orgStr != "" {
				orgID, _ = uuid.Parse(orgStr)
			}

			action := c.Request.Method + " " + c.Request.URL.Path
			resourceType := extractResourceType(c.Request.URL.Path)

			auditLogger.Log(c.Request.Context(), audit.LogInput{
				OrgID:        orgID,
				ActorID:      actorID,
				ActorEmail:   actorEmail,
				Action:       action,
				ResourceType: resourceType,
				ResourceID:   c.Param("id"),
				IPAddress:    c.ClientIP(),
				UserAgent:    c.Request.UserAgent(),
			})
		}
	}
}

func extractResourceType(path string) string {
	if len(path) < 2 {
		return ""
	}
	parts := make([]string, 0)
	start := 0
	for i, ch := range path {
		if ch == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}

	if len(parts) >= 2 && parts[0] == "admin" {
		return parts[1]
	}
	return ""
}
