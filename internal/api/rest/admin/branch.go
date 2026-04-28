package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type BranchHandler struct {
	branchSvc interface {
		CreateBranchType(ctx interface{}, orgID uuid.UUID, params BranchTypeParams) (*domain.BranchType, error)
		GetBranchType(ctx interface{}, id, orgID uuid.UUID) (*domain.BranchType, error)
		ListBranchTypes(ctx interface{}, orgID uuid.UUID) ([]*domain.BranchType, error)
		UpdateBranchType(ctx interface{}, id, orgID uuid.UUID, params BranchTypeParams) (*domain.BranchType, error)
		DeleteBranchType(ctx interface{}, id, orgID uuid.UUID) error
		CreateBranch(ctx interface{}, orgID uuid.UUID, params BranchParams) (*domain.Branch, error)
		GetBranch(ctx interface{}, id, orgID uuid.UUID) (*domain.Branch, error)
		ListBranches(ctx interface{}, orgID uuid.UUID) ([]*domain.Branch, error)
		UpdateBranch(ctx interface{}, id, orgID uuid.UUID, params BranchParams) (*domain.Branch, error)
		DeleteBranch(ctx interface{}, id, orgID uuid.UUID) error
		AssignUserToBranch(ctx interface{}, userID, branchID, orgID uuid.UUID, params UserBranchParams) (*domain.UserBranchAssignment, error)
		GetUserBranches(ctx interface{}, userID uuid.UUID) ([]*domain.UserBranchAssignment, error)
		RevokeUserBranch(ctx interface{}, assignmentID, orgID, revokedBy uuid.UUID) error
	}
	auditSvc *audit.Service
	logger   zerolog.Logger
}

type BranchTypeParams struct {
	Code            string
	Label           string
	ShortCode       string
	Level           int
	IsOperational   bool
	CanHaveChildren bool
	SortOrder       int
}

type BranchParams struct {
	BranchTypeID uuid.UUID
	ParentID     *uuid.UUID
	Code         string
	Name         string
	Address      string
	Phone        string
}

type UserBranchParams struct {
	BranchID   uuid.UUID
	Role       string
	IsPrimary  bool
	AssignedBy uuid.UUID
}

func NewBranchHandler(branchSvc interface {
	CreateBranchType(ctx interface{}, orgID uuid.UUID, params BranchTypeParams) (*domain.BranchType, error)
	GetBranchType(ctx interface{}, id, orgID uuid.UUID) (*domain.BranchType, error)
	ListBranchTypes(ctx interface{}, orgID uuid.UUID) ([]*domain.BranchType, error)
	UpdateBranchType(ctx interface{}, id, orgID uuid.UUID, params BranchTypeParams) (*domain.BranchType, error)
	DeleteBranchType(ctx interface{}, id, orgID uuid.UUID) error
	CreateBranch(ctx interface{}, orgID uuid.UUID, params BranchParams) (*domain.Branch, error)
	GetBranch(ctx interface{}, id, orgID uuid.UUID) (*domain.Branch, error)
	ListBranches(ctx interface{}, orgID uuid.UUID) ([]*domain.Branch, error)
	UpdateBranch(ctx interface{}, id, orgID uuid.UUID, params BranchParams) (*domain.Branch, error)
	DeleteBranch(ctx interface{}, id, orgID uuid.UUID) error
	AssignUserToBranch(ctx interface{}, userID, branchID, orgID uuid.UUID, params UserBranchParams) (*domain.UserBranchAssignment, error)
	GetUserBranches(ctx interface{}, userID uuid.UUID) ([]*domain.UserBranchAssignment, error)
	RevokeUserBranch(ctx interface{}, assignmentID, orgID, revokedBy uuid.UUID) error
}, auditSvc *audit.Service, logger zerolog.Logger) *BranchHandler {
	return &BranchHandler{
		branchSvc: branchSvc,
		auditSvc:  auditSvc,
		logger:    logger,
	}
}

func (h *BranchHandler) RegisterBranchTypeRoutes(r *gin.RouterGroup) {
	branchTypes := r.Group("/branch-types")
	{
		branchTypes.GET("", h.handleListBranchTypes)
		branchTypes.POST("", h.handleCreateBranchType)
		branchTypes.GET("/:id", h.handleGetBranchType)
		branchTypes.PUT("/:id", h.handleUpdateBranchType)
		branchTypes.DELETE("/:id", h.handleDeleteBranchType)
	}
}

func (h *BranchHandler) RegisterBranchRoutes(r *gin.RouterGroup) {
	branches := r.Group("/branches")
	{
		branches.GET("", h.handleListBranches)
		branches.POST("", h.handleCreateBranch)
		branches.GET("/:id", h.handleGetBranch)
		branches.PUT("/:id", h.handleUpdateBranch)
		branches.DELETE("/:id", h.handleDeleteBranch)
	}

	r.GET("/users/:id/branches", h.handleGetUserBranches)
	r.POST("/users/:id/branches", h.handleAssignUserToBranch)
	r.DELETE("/users/:user_id/branches/:assignment_id", h.handleRevokeUserBranch)
}

func (h *BranchHandler) handleListBranchTypes(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	branchTypes, err := h.branchSvc.ListBranchTypes(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list branch types failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, gin.H{"branch_types": branchTypes})
}

func (h *BranchHandler) handleCreateBranchType(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	var params BranchTypeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}

	bt, err := h.branchSvc.CreateBranchType(c.Request.Context(), orgID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("create branch type failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "create_branch_type", "branch_type", bt.ID.String(), nil, branchTypeToJSON(bt))

	c.JSON(201, bt)
}

func (h *BranchHandler) handleGetBranchType(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	bt, err := h.branchSvc.GetBranchType(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get branch type failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, bt)
}

func (h *BranchHandler) handleUpdateBranchType(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	oldBT, _ := h.branchSvc.GetBranchType(c.Request.Context(), id, orgID)

	var params BranchTypeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}

	bt, err := h.branchSvc.UpdateBranchType(c.Request.Context(), id, orgID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("update branch type failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "update_branch_type", "branch_type", id.String(), branchTypeToJSON(oldBT), branchTypeToJSON(bt))

	c.JSON(200, bt)
}

func (h *BranchHandler) handleDeleteBranchType(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	oldBT, _ := h.branchSvc.GetBranchType(c.Request.Context(), id, orgID)

	if err := h.branchSvc.DeleteBranchType(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Msg("delete branch type failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "delete_branch_type", "branch_type", id.String(), branchTypeToJSON(oldBT), nil)

	c.JSON(200, gin.H{"deleted": true})
}

func (h *BranchHandler) handleListBranches(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	branches, err := h.branchSvc.ListBranches(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list branches failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, gin.H{"branches": branches})
}

func (h *BranchHandler) handleCreateBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)

	var params BranchParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}

	branch, err := h.branchSvc.CreateBranch(c.Request.Context(), orgID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("create branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "create_branch", "branch", branch.ID.String(), nil, branchToJSON(branch))

	c.JSON(201, branch)
}

func (h *BranchHandler) handleGetBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	branch, err := h.branchSvc.GetBranch(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, branch)
}

func (h *BranchHandler) handleUpdateBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	oldBranch, _ := h.branchSvc.GetBranch(c.Request.Context(), id, orgID)

	var params BranchParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}

	branch, err := h.branchSvc.UpdateBranch(c.Request.Context(), id, orgID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("update branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "update_branch", "branch", id.String(), branchToJSON(oldBranch), branchToJSON(branch))

	c.JSON(200, branch)
}

func (h *BranchHandler) handleDeleteBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	id, _ := uuid.Parse(c.Param("id"))

	oldBranch, _ := h.branchSvc.GetBranch(c.Request.Context(), id, orgID)

	if err := h.branchSvc.DeleteBranch(c.Request.Context(), id, orgID); err != nil {
		h.logger.Error().Err(err).Msg("delete branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "delete_branch", "branch", id.String(), branchToJSON(oldBranch), nil)

	c.JSON(200, gin.H{"deleted": true})
}

func (h *BranchHandler) handleGetUserBranches(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))

	assignments, err := h.branchSvc.GetUserBranches(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("get user branches failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, gin.H{"assignments": assignments})
}

func (h *BranchHandler) handleAssignUserToBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	userID, _ := uuid.Parse(c.Param("id"))

	var params UserBranchParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}

	if params.BranchID == uuid.Nil {
		c.JSON(400, gin.H{"error": "branch_id required"})
		return
	}

	assignment, err := h.branchSvc.AssignUserToBranch(c.Request.Context(), userID, params.BranchID, orgID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("assign user to branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	h.logAction(orgID, c, "assign_user_branch", "user_branch", assignment.ID.String(), nil, userBranchToJSON(assignment))

	c.JSON(201, assignment)
}

func (h *BranchHandler) handleRevokeUserBranch(c *gin.Context) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		c.JSON(400, gin.H{"error": "org_id required"})
		return
	}
	orgID, _ := uuid.Parse(orgStr)
	assignmentID, _ := uuid.Parse(c.Param("assignment_id"))

	if err := h.branchSvc.RevokeUserBranch(c.Request.Context(), assignmentID, orgID, uuid.Nil); err != nil {
		h.logger.Error().Err(err).Msg("revoke user branch failed")
		c.JSON(500, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(200, gin.H{"revoked": true})
}

func (h *BranchHandler) logAction(orgID uuid.UUID, c *gin.Context, action, resourceType, resourceID string, oldVal, newVal domain.JSON) {
	actorID, actorEmail := getActorFromContext(c)
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValue:     oldVal,
		NewValue:     newVal,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})
}

func branchTypeToJSON(bt *domain.BranchType) domain.JSON {
	if bt == nil {
		return nil
	}
	return domain.JSON{
		"id":       bt.ID.String(),
		"code":     bt.Code,
		"label":    bt.Label,
		"level":    bt.Level,
	}
}

func branchToJSON(b *domain.Branch) domain.JSON {
	if b == nil {
		return nil
	}
	return domain.JSON{
		"id":   b.ID.String(),
		"code": b.Code,
		"name": b.Name,
	}
}

func userBranchToJSON(uba *domain.UserBranchAssignment) domain.JSON {
	if uba == nil {
		return nil
	}
	return domain.JSON{
		"id":        uba.ID.String(),
		"user_id":   uba.UserID.String(),
		"branch_id": uba.BranchID.String(),
		"role":      uba.Role,
	}
}
