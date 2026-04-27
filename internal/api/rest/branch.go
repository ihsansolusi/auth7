package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	c.JSON(http.StatusOK, gin.H{
		"branches": []gin.H{
			{"id": uuid.New().String(), "code": "KC-BDG-001", "name": "Kantor Cabang Bandung", "role": "teller", "is_primary": true},
			{"id": uuid.New().String(), "code": "KCP-DGO-001", "name": "KCP Dago", "role": "teller", "is_primary": false},
		},
	})
}

func (s *Server) handleSwitchBranch(c *gin.Context) {
	var req struct {
		BranchID string `json:"branch_id"`
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

	c.JSON(http.StatusOK, gin.H{
		"branch_id":   req.BranchID,
		"switched_at": "2026-04-27T10:00:00Z",
	})
}