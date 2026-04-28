package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/authz"
)

type AuthMiddleware struct {
	checker *authz.PermissionChecker
}

func NewAuthMiddleware(checker *authz.PermissionChecker) *AuthMiddleware {
	return &AuthMiddleware{
		checker: checker,
	}
}

func (m *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := extractAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		result, err := m.checker.CheckPermission(c.Request.Context(), authCtx, permission)
		if err != nil || !result.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": result.Reason})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequirePermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := extractAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		for _, perm := range permissions {
			result, err := m.checker.CheckPermission(c.Request.Context(), authCtx, perm)
			if err != nil || !result.Allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": result.Reason})
				return
			}
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := extractAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		allowed := false
		for _, perm := range permissions {
			result, err := m.checker.CheckPermission(c.Request.Context(), authCtx, perm)
			if err == nil && result.Allowed {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no matching permission"})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireBranchAccess(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := extractAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		branchIDStr := c.GetHeader("X-Branch-ID")
		if branchIDStr == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "branch_id required"})
			return
		}

		branchID, err := uuid.Parse(branchIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
			return
		}

		result, err := m.checker.CheckBranchScope(c.Request.Context(), authCtx, branchID)
		if err != nil || !result.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": result.Reason})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireFourLayer(params authz.AuthParams) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := extractAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		fourLayer := authz.NewFourLayerAuth(m.checker)
		result, err := fourLayer.Authorize(c.Request.Context(), authCtx, params)
		if err != nil || !result.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": result.Reason})
			return
		}

		c.Next()
	}
}

func extractAuthContext(c *gin.Context) *domain.AuthContext {
	userIDStr, _ := c.Get("user_id")
	orgIDStr, _ := c.Get("org_id")
	branchIDStr, _ := c.Get("branch_id")
	permissionsRaw, _ := c.Get("permissions")
	branchScopeStr, _ := c.Get("branch_scope")

	var userID, orgID, branchID uuid.UUID
	if s, ok := userIDStr.(string); ok {
		userID, _ = uuid.Parse(s)
	}
	if s, ok := orgIDStr.(string); ok {
		orgID, _ = uuid.Parse(s)
	}
	if s, ok := branchIDStr.(string); ok {
		branchID, _ = uuid.Parse(s)
	}

	var permissions []string
	if perms, ok := permissionsRaw.([]string); ok {
		permissions = perms
	}

	branchScope := domain.BranchScopeAssigned
	if scope, ok := branchScopeStr.(string); ok {
		switch scope {
		case "own":
			branchScope = domain.BranchScopeOwn
		case "assigned":
			branchScope = domain.BranchScopeAssigned
		case "all":
			branchScope = domain.BranchScopeAll
		}
	}

	return &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		Permissions: permissions,
		BranchScope: branchScope,
	}
}

func (m *AuthMiddleware) BuildAuthContext(ctx context.Context, userID, orgID, branchID uuid.UUID) *domain.AuthContext {
	return &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		BranchScope: domain.BranchScopeAssigned,
	}
}

func ParseTokenFromHeader(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}