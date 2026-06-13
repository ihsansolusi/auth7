package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/messaging/nats"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

func (s *Server) RegisterBranchRoutes(r *gin.Engine) {
	admin := r.Group("/admin/branch-types")
	{
		admin.GET("", s.handleListBranchTypes)
		admin.POST("", s.handleCreateBranchType)
		admin.GET("/:id", s.handleGetBranchType)
		admin.PUT("/:id", s.handleUpdateBranchType)
		admin.DELETE("/:id", s.handleDeleteBranchType)
	}

	adminBranches := r.Group("/admin/branches")
	{
		adminBranches.GET("", s.handleListBranches)
		adminBranches.POST("", s.handleCreateBranch)
		adminBranches.GET("/:id", s.handleGetBranch)
		adminBranches.PUT("/:id", s.handleUpdateBranch)
		adminBranches.DELETE("/:id", s.handleDeleteBranch)
	}

	r.GET("/auth/branches", s.handleListUserBranches)
	r.POST("/auth/switch-branch", s.handleSwitchBranch)

	// W17 cross-service lookup: pejabat picker on bos7-financing reads
	// from this. User-JWT (delegated via BFF token exchange); org-scoped
	// inside the handler from claims.OrgID.
	r.POST("/v1/users/lookup/query", s.handleUserLookup)
}

func (s *Server) handleListBranchTypes(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}

	_ = orgID

	c.JSON(http.StatusOK, gin.H{
		"branch_types": []interface{}{},
	})
}

func (s *Server) handleCreateBranchType(c *gin.Context) {
	var req struct {
		Code           string `json:"code"`
		Label          string `json:"label"`
		ShortCode      string `json:"short_code"`
		Level          int    `json:"level"`
		IsOperational  bool   `json:"is_operational"`
		CanHaveChildren bool   `json:"can_have_children"`
		SortOrder      int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          uuid.New().String(),
		"code":        req.Code,
		"label":       req.Label,
		"short_code":  req.ShortCode,
		"level":       req.Level,
		"created_at":  "2026-04-27T10:00:00Z",
	})
}

func (s *Server) handleGetBranchType(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"code":  "KC",
		"label": "Kantor Cabang",
		"level": 1,
	})
}

func (s *Server) handleUpdateBranchType(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"code":  "KC",
		"label": "Kantor Cabang",
		"level": 1,
	})
}

func (s *Server) handleDeleteBranchType(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Server) handleListBranches(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}

	_ = orgID

	c.JSON(http.StatusOK, gin.H{
		"branches": []interface{}{},
	})
}

func (s *Server) handleCreateBranch(c *gin.Context) {
	var req struct {
		BranchTypeID string `json:"branch_type_id"`
		ParentID     string `json:"parent_id,omitempty"`
		Code         string `json:"code"`
		Name         string `json:"name"`
		Address      string `json:"address"`
		Phone        string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            uuid.New().String(),
		"branch_type_id": req.BranchTypeID,
		"code":          req.Code,
		"name":          req.Name,
		"status":        "active",
		"created_at":    "2026-04-27T10:00:00Z",
	})
}

func (s *Server) handleGetBranch(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"code":  "KC-BDG-001",
		"name":  "Kantor Cabang Bandung",
		"status": "active",
	})
}

func (s *Server) handleUpdateBranch(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"code":  "KC-BDG-001",
		"name":  "Kantor Cabang Bandung",
		"status": "active",
	})
}

func (s *Server) handleDeleteBranch(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Server) handleListUserBranches(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	tokenStr := trimBearer(auth)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	sessionSvc, ok := s.deps.SessionSvc.(interface {
		VerifyAccessToken(ctx context.Context, token string) (*jwt.Claims, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session service unavailable"})
		return
	}

	claims, err := sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	store, ok := s.deps.Store.(*postgres.Store)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"branches": []interface{}{}})
		return
	}

	assignments, err := store.UserBranchAssignmentRepository.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"branches": []interface{}{}})
		return
	}

	branches := make([]gin.H, 0, len(assignments))
	for _, a := range assignments {
		branchInfo := gin.H{
			"id":          a.BranchID.String(),
			"org_id":      a.OrgID.String(),
			"role":        a.Role,
			"is_primary":  a.IsPrimary,
			"assigned_at": a.AssignedAt,
		}
		// Look up branch_code + name from branches projection table.
		var branchCode, branchName string
		if err := store.Pool().QueryRow(c.Request.Context(),
			"SELECT COALESCE(branch_code, ''), COALESCE(name, '') FROM branches WHERE id = $1",
			a.BranchID).Scan(&branchCode, &branchName); err == nil {
			branchInfo["branch_code"] = branchCode
			branchInfo["branch_name"] = branchName
		}
		branches = append(branches, branchInfo)
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

func (s *Server) handleSwitchBranch(c *gin.Context) {
	var req struct {
		BranchID string `json:"branch_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	tokenStr := trimBearer(auth)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	sessionSvc, ok := s.deps.SessionSvc.(interface {
		VerifyAccessToken(ctx context.Context, token string) (*jwt.Claims, error)
		RevokeSession(ctx context.Context, sessionID string) error
		CreateSession(ctx context.Context, userID, orgID uuid.UUID, ipAddress, userAgent string, claims jwt.Claims) (*session.LoginResult, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session service unavailable"})
		return
	}

	claims, err := sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	store, ok := s.deps.Store.(*postgres.Store)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store unavailable"})
		return
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id"})
		return
	}

	assignment, err := store.UserBranchAssignmentRepository.GetByUserAndBranch(c.Request.Context(), userID, branchID)
	if err != nil || assignment == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not assigned to this branch or branch inactive"})
		return
	}

	orgID, _ := uuid.Parse(claims.OrgID)

	roles, _ := store.UserRoleRepository.GetRoleCodesByUser(c.Request.Context(), userID)

	if err := sessionSvc.RevokeSession(c.Request.Context(), claims.SessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke previous session"})
		return
	}

	newClaims := jwt.Claims{
		Username:   claims.Username,
		Email:      claims.Email,
		Roles:      roles,
		BranchID:   branchID.String(),
		BranchCode: assignment.BranchCode,
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := sessionSvc.CreateSession(c.Request.Context(), userID, orgID, ipAddress, userAgent, newClaims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create new session"})
		return
	}

	switchedAt := time.Now().UTC()

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    result.ExpiresIn,
		"session_id":    result.SessionID,
		"branch_id":     branchID.String(),
		"branch_code":   assignment.BranchCode,
		"switched_at":   switchedAt.Format(time.RFC3339),
	})

	if s.deps.EventPub != nil {
		_ = s.deps.EventPub.PublishBranchSwitched(c.Request.Context(), nats.BranchSwitchedEvent{
			UserID:        userID.String(),
			Username:      claims.Username,
			OrgID:         orgID.String(),
			OldSessionID:  claims.SessionID,
			NewSessionID:  result.SessionID,
			OldBranchID:   claims.BranchID,
			OldBranchCode: claims.BranchCode,
			NewBranchID:   branchID.String(),
			NewBranchCode: assignment.BranchCode,
			IPAddress:     ipAddress,
			UserAgent:     userAgent,
			SwitchedAt:    switchedAt,
		})
	}
}

func trimBearer(auth string) string {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
		return ""
	}
	return auth[len(prefix):]
}
