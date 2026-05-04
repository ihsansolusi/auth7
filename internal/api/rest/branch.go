package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
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
		// Look up branch code and name from branches table
		var code, name string
		err := store.Pool().QueryRow(c.Request.Context(),
			"SELECT code, name FROM branches WHERE id = $1", a.BranchID).Scan(&code, &name)
		if err == nil {
			branchInfo["code"] = code
			branchInfo["name"] = name
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
		c.JSON(http.StatusForbidden, gin.H{"error": "user not assigned to this branch"})
		return
	}

	jwtSvc, ok := s.deps.JWTSvc.(interface {
		IssueAccessToken(sessionID string, userID, orgID uuid.UUID, claims jwt.Claims) (string, *jwt.AccessToken, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt service unavailable"})
		return
	}

	orgID, _ := uuid.Parse(claims.OrgID)

	newClaims := jwt.Claims{
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
		BranchID: branchID.String(),
	}

	newSessionID := uuid.New().String()
	newAccessToken, _, err := jwtSvc.IssueAccessToken(newSessionID, userID, orgID, newClaims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
		"expires_in":   900,
		"branch_id":    branchID.String(),
		"switched_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

func trimBearer(auth string) string {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
		return ""
	}
	return auth[len(prefix):]
}
