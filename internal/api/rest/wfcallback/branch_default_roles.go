package wfcallback

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/rs/zerolog"
)

// BranchDefaultRolesWfHandler serves the workflow7 callback for setting a
// branch's default roles (the roles auto-assigned to users joining that branch).
// Replace-the-whole-set; master_id (route :id) is the branch id; data.role_ids
// is the desired set.
type BranchDefaultRolesWfHandler struct {
	store    *postgres.Store
	auditSvc *audit.Service
	logger   zerolog.Logger
}

func NewBranchDefaultRolesWfHandler(store *postgres.Store, auditSvc *audit.Service, logger zerolog.Logger) *BranchDefaultRolesWfHandler {
	return &BranchDefaultRolesWfHandler{store: store, auditSvc: auditSvc, logger: logger}
}

func (h *BranchDefaultRolesWfHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/branches/:id/wf-set-default-roles", h.handleWfSetDefaultRoles)
}

func (h *BranchDefaultRolesWfHandler) handleWfSetDefaultRoles(c *gin.Context) {
	branchID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	roleIDs := dataStrSlice(env.Data, "role_ids")

	ctx := c.Request.Context()
	tx, err := h.store.Pool().Begin(ctx)
	if err != nil {
		wfFail(c, h.logger, err, "wf set branch default roles: begin tx failed")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `DELETE FROM branch_default_roles WHERE branch_id = $1`, branchID); err != nil {
		wfFail(c, h.logger, err, "wf set branch default roles: clear failed")
		return
	}
	for _, ridStr := range roleIDs {
		rid, perr := uuid.Parse(ridStr)
		if perr != nil {
			continue
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO branch_default_roles (branch_id, role_id, is_default) VALUES ($1, $2, true)
			 ON CONFLICT (branch_id, role_id) DO UPDATE SET is_default = true`,
			branchID, rid); err != nil {
			wfFail(c, h.logger, err, "wf set branch default roles: insert failed")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		wfFail(c, h.logger, err, "wf set branch default roles: commit failed")
		return
	}

	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       "set_branch_default_roles",
		ResourceType: "branch_default_roles",
		ResourceID:   branchID.String(),
		NewValue:     domain.JSON{"role_ids": roleIDs},
	})
	c.JSON(http.StatusOK, gin.H{"id": branchID.String(), "success": true})
}
