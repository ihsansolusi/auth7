package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

// branchReadService is the read surface used by the admin HTTP API. Branch &
// branch-type mutations flow through workflow7 → the M2M /internal/v1
// wf-callbacks; the concrete adapter still implements the full lifecycle there.
type branchReadService interface {
	GetBranchType(ctx interface{}, id, orgID uuid.UUID) (*domain.BranchType, error)
	ListBranchTypes(ctx interface{}, orgID uuid.UUID) ([]*domain.BranchType, error)
	GetBranch(ctx interface{}, id, orgID uuid.UUID) (*domain.Branch, error)
	ListBranches(ctx interface{}, orgID uuid.UUID) ([]*domain.Branch, error)
	GetUserBranches(ctx interface{}, userID uuid.UUID) ([]*domain.UserBranchAssignment, error)
}

type BranchHandler struct {
	branchSvc branchReadService
	auditSvc  *audit.Service
	logger    zerolog.Logger
}

// BranchTypeParams / BranchParams / UserBranchParams are the lifecycle inputs
// consumed by the concrete branch service (and the wf-callback handlers); kept
// here as the shared contract even though the admin API no longer mutates.
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

func NewBranchHandler(branchSvc branchReadService, auditSvc *audit.Service, logger zerolog.Logger) *BranchHandler {
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
		branchTypes.GET("/:id", h.handleGetBranchType)
	}
}

func (h *BranchHandler) RegisterBranchRoutes(r *gin.RouterGroup) {
	branches := r.Group("/branches")
	{
		branches.GET("", h.handleListBranches)
		branches.GET("/:id", h.handleGetBranch)
	}

	r.GET("/users/:id/branches", h.handleGetUserBranches)
}

func (h *BranchHandler) handleListBranchTypes(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}

	branchTypes, err := h.branchSvc.ListBranchTypes(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list branch types failed")
		respondError(c, err)
		return
	}

	c.JSON(200, gin.H{"branch_types": branchTypes})
}

func (h *BranchHandler) handleGetBranchType(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}
	id, _ := uuid.Parse(c.Param("id"))

	bt, err := h.branchSvc.GetBranchType(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get branch type failed")
		respondError(c, err)
		return
	}

	c.JSON(200, bt)
}

func (h *BranchHandler) handleListBranches(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}

	branches, err := h.branchSvc.ListBranches(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list branches failed")
		respondError(c, err)
		return
	}

	c.JSON(200, gin.H{"branches": branches})
}

func (h *BranchHandler) handleGetBranch(c *gin.Context) {
	orgID, ok := requireOrgID(c)
	if !ok {
		return
	}
	id, _ := uuid.Parse(c.Param("id"))

	branch, err := h.branchSvc.GetBranch(c.Request.Context(), id, orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get branch failed")
		respondError(c, err)
		return
	}

	c.JSON(200, branch)
}

func (h *BranchHandler) handleGetUserBranches(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))

	assignments, err := h.branchSvc.GetUserBranches(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("get user branches failed")
		respondError(c, err)
		return
	}

	c.JSON(200, gin.H{"assignments": assignments})
}
