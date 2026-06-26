package wfcallback

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/mailer"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/rs/zerolog"
)

// wfUserToJSON renders a user for audit snapshots.
func wfUserToJSON(u *domain.User) domain.JSON {
	if u == nil {
		return nil
	}
	return domain.JSON{
		"id":         u.ID.String(),
		"username":   u.Username,
		"email":      u.Email,
		"full_name":  u.FullName,
		"status":     string(u.Status),
		"created_at": u.CreatedAt.Format(time.RFC3339),
	}
}

// UserWfHandler serves the workflow7 service-task callbacks for the user
// lifecycle and user-role / user-branch assignments. Assignments use
// REPLACE-the-whole-set semantics: wf-set-roles and wf-set-branches receive the
// full desired set and reconcile against the current active rows. Roles are
// org-wide only (branch_id NULL).
type UserWfHandler struct {
	userSvc     UserService
	userRoleSvc UserRoleService
	branchSvc   BranchService
	store       *postgres.Store
	auditSvc    *audit.Service
	mailer      mailer.Mailer
	logger      zerolog.Logger
}

func NewUserWfHandler(
	userSvc UserService,
	userRoleSvc UserRoleService,
	branchSvc BranchService,
	store *postgres.Store,
	auditSvc *audit.Service,
	m mailer.Mailer,
	logger zerolog.Logger,
) *UserWfHandler {
	return &UserWfHandler{
		userSvc:     userSvc,
		userRoleSvc: userRoleSvc,
		branchSvc:   branchSvc,
		store:       store,
		auditSvc:    auditSvc,
		mailer:      m,
		logger:      logger,
	}
}

func (h *UserWfHandler) RegisterRoutes(g *gin.RouterGroup) {
	users := g.Group("/users")
	{
		users.POST("/wf-create", h.handleWfCreate)
		users.PUT("/:id/wf-update", h.handleWfUpdate)
		users.POST("/:id/wf-delete", h.handleWfDelete)
		users.POST("/:id/wf-lock", h.handleWfLock)
		users.POST("/:id/wf-unlock", h.handleWfUnlock)
		users.POST("/:id/wf-set-roles", h.handleWfSetRoles)
		users.POST("/:id/wf-set-branches", h.handleWfSetBranches)
	}
}

func (h *UserWfHandler) audit(data map[string]any, wfInstanceID string, orgID, actorID uuid.UUID, actorEmail, action, resourceType, resourceID string, oldV, newV domain.JSON) {
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:         orgID,
		ActorID:       actorID,
		ActorEmail:    actorEmail,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		OldValue:      oldV,
		NewValue:      newV,
		IPAddress:     dataStr(data, "ip_address"),
		UserAgent:     dataStr(data, "user_agent"),
		BranchID:      dataStr(data, "branch_id"),
		BranchCode:    dataStr(data, "branch_code"),
		SessionID:     dataStr(data, "session_id"),
		CorrelationID: wfInstanceID,
	})
}

// idLabels loads an id->label map (label = code/branch_code, fallback name) for a
// reference table, so audit before/after snapshots show human-readable codes.
// The query must SELECT (id, code, name) filtered by org_id = $1.
func (h *UserWfHandler) idLabels(ctx context.Context, query string, orgID uuid.UUID) map[string]string {
	m := map[string]string{}
	rows, err := h.store.Pool().Query(ctx, query, orgID)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code, name string
		if scanErr := rows.Scan(&id, &code, &name); scanErr != nil {
			continue
		}
		label := code
		if label == "" {
			label = name
		}
		if label == "" {
			label = id.String()
		}
		m[id.String()] = label
	}
	return m
}

// ── user lifecycle ──────────────────────────────────────────────────────────

// handleWfCreate creates the user and, atomically within the same callback,
// applies the initial org-wide roles (data.roles = [{role_id}]) and the primary
// branch (data.primary_branch_id). A failure in any step fails the workflow step.
func (h *UserWfHandler) handleWfCreate(c *gin.Context) {
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}

	// Password handling: either the admin-provided password, or an
	// auto-generated one. `require_password_change` forces a reset at first login.
	plainPassword := dataStr(env.Data, "password")
	autoGenerate := dataBool(env.Data, "auto_generate_password")
	requireChange := dataBool(env.Data, "require_password_change")
	if autoGenerate {
		plainPassword = password.Generate()
	}

	input := adminpkg.CreateUserInput{
		Username:              dataStr(env.Data, "username"),
		Email:                 dataStr(env.Data, "email"),
		FullName:              dataStr(env.Data, "full_name"),
		PreferredLocale:       dataStr(env.Data, "preferred_locale"),
		Password:              plainPassword,
		RequirePasswordChange: requireChange,
		CreatedBy:             actorID,
	}
	user, err := h.userSvc.CreateUser(c.Request.Context(), orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf create user failed")
		return
	}

	// Initial org-wide role assignments.
	for _, rm := range dataMaps(env.Data, "roles") {
		rid, perr := uuid.Parse(dataStr(rm, "role_id"))
		if perr != nil {
			continue
		}
		if _, aerr := h.userRoleSvc.AssignRole(c.Request.Context(), user.ID, rid, orgID, nil, actorID); aerr != nil {
			wfFail(c, h.logger, aerr, "wf create: assign role failed")
			return
		}
	}

	// Primary branch assignment.
	if pb := dataStr(env.Data, "primary_branch_id"); pb != "" {
		bid, perr := uuid.Parse(pb)
		if perr == nil {
			if _, berr := h.branchSvc.AssignUserToBranch(c.Request.Context(), user.ID, bid, orgID,
				adminpkg.UserBranchParams{BranchID: bid, IsPrimary: true, AssignedBy: actorID}); berr != nil {
				wfFail(c, h.logger, berr, "wf create: assign primary branch failed")
				return
			}
		}
	}

	// Email the temporary password to the user (best-effort, async). Only for
	// the auto-generate path — a manually-set password is communicated by the admin.
	if autoGenerate && user.Email != "" && h.mailer != nil {
		email, uname, pw := user.Email, user.Username, plainPassword
		go func() {
			html, rerr := mailer.RenderNewAccountEmail("Akun Auth7 Anda", uname, pw)
			if rerr != nil {
				return
			}
			_ = h.mailer.Send(context.Background(), email, "Akun Auth7 Anda — Password Sementara", html)
		}()
	}

	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "create_user", "user", user.ID.String(), nil, wfUserToJSON(user))

	resp := gin.H{"id": user.ID.String(), "success": true}
	// Surface the generated password ONCE so the admin can relay it. Captured by
	// the workflow on_complete_vars and read back by the BFF for a one-time display.
	if autoGenerate {
		resp["generated_password"] = plainPassword
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserWfHandler) handleWfUpdate(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)

	input := adminpkg.UpdateUserInput{
		Username:  dataStrPtr(env.Data, "username"),
		Email:     dataStrPtr(env.Data, "email"),
		FullName:  dataStrPtr(env.Data, "full_name"),
		UpdatedBy: &actorID,
	}
	if s := dataStrPtr(env.Data, "status"); s != nil {
		st := domain.UserStatus(*s)
		input.Status = &st
	}

	user, err := h.userSvc.UpdateUser(c.Request.Context(), id, orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf update user failed")
		return
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "update_user", "user", id.String(), wfUserToJSON(oldUser), wfUserToJSON(user))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

func (h *UserWfHandler) handleWfDelete(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err := h.userSvc.DeleteUser(c.Request.Context(), id, orgID); err != nil {
		wfFail(c, h.logger, err, "wf delete user failed")
		return
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "delete_user", "user", id.String(), wfUserToJSON(oldUser), nil)
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

func (h *UserWfHandler) handleWfLock(c *gin.Context) {
	h.statusChange(c, "lock_user", func(c *gin.Context, id, orgID uuid.UUID) error {
		return h.userSvc.LockUser(c.Request.Context(), id, orgID)
	})
}

func (h *UserWfHandler) handleWfUnlock(c *gin.Context) {
	h.statusChange(c, "unlock_user", func(c *gin.Context, id, orgID uuid.UUID) error {
		return h.userSvc.UnlockUser(c.Request.Context(), id, orgID)
	})
}

func (h *UserWfHandler) statusChange(c *gin.Context, action string, fn func(*gin.Context, uuid.UUID, uuid.UUID) error) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err := fn(c, id, orgID); err != nil {
		wfFail(c, h.logger, err, "wf "+action+" failed")
		return
	}
	newUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, action, "user", id.String(), wfUserToJSON(oldUser), wfUserToJSON(newUser))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

// ── role assignment (replace whole set, org-wide only) ──────────────────────

func (h *UserWfHandler) handleWfSetRoles(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}

	desired := map[uuid.UUID]bool{}
	for _, rm := range dataMaps(env.Data, "roles") {
		if rid, perr := uuid.Parse(dataStr(rm, "role_id")); perr == nil {
			desired[rid] = true
		}
	}

	current, err := h.userRoleSvc.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		wfFail(c, h.logger, err, "wf set roles: load current failed")
		return
	}
	currentSet := map[uuid.UUID]bool{}
	for _, ur := range current {
		// Reconcile org-wide assignments only (branch_id NULL, active).
		if ur.RevokedAt == nil && ur.BranchID == nil {
			currentSet[ur.RoleID] = true
		}
	}

	for rid := range currentSet {
		if !desired[rid] {
			if rerr := h.userRoleSvc.RevokeRole(c.Request.Context(), userID, rid, orgID, actorID); rerr != nil {
				wfFail(c, h.logger, rerr, "wf set roles: revoke failed")
				return
			}
		}
	}
	for rid := range desired {
		if !currentSet[rid] {
			if _, aerr := h.userRoleSvc.AssignRole(c.Request.Context(), userID, rid, orgID, nil, actorID); aerr != nil {
				wfFail(c, h.logger, aerr, "wf set roles: assign failed")
				return
			}
		}
	}

	roleLabels := h.idLabels(c.Request.Context(), `SELECT id, code, name FROM roles WHERE org_id=$1`, orgID)
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "set_roles", "user_role", userID.String(),
		domain.JSON{"roles": mapToLabels(uuidSetKeys(currentSet), roleLabels)},
		domain.JSON{"roles": mapToLabels(uuidSetKeys(desired), roleLabels)})
	c.JSON(http.StatusOK, gin.H{"id": userID.String(), "success": true})
}

// ── branch assignment (replace whole set; exactly one primary) ──────────────

func (h *UserWfHandler) handleWfSetBranches(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}

	primaryStr := dataStr(env.Data, "primary_branch_id")
	primaryID, err := uuid.Parse(primaryStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "primary_branch_id required", "success": false})
		return
	}
	// desired branch_id -> isPrimary. Primary first; secondaries default non-primary.
	desired := map[uuid.UUID]bool{primaryID: true}
	for _, bm := range dataMaps(env.Data, "branches") {
		if bid, perr := uuid.Parse(dataStr(bm, "branch_id")); perr == nil {
			if _, exists := desired[bid]; !exists {
				desired[bid] = false
			}
		}
	}

	ctx := c.Request.Context()
	tx, err := h.store.Pool().Begin(ctx)
	if err != nil {
		wfFail(c, h.logger, err, "wf set branches: begin tx failed")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) Clear every active primary flag so a new primary can be set without
	//    colliding with the partial unique (user_id, is_primary) WHERE is_primary.
	if _, err = tx.Exec(ctx,
		`UPDATE user_branch_assignments SET is_primary = false WHERE user_id = $1 AND revoked_at IS NULL`,
		userID); err != nil {
		wfFail(c, h.logger, err, "wf set branches: clear primary failed")
		return
	}

	// 2) Revoke active assignments no longer desired.
	rows, err := tx.Query(ctx,
		`SELECT branch_id FROM user_branch_assignments WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		wfFail(c, h.logger, err, "wf set branches: load current failed")
		return
	}
	var activeIDs []uuid.UUID
	for rows.Next() {
		var bid uuid.UUID
		if scanErr := rows.Scan(&bid); scanErr != nil {
			rows.Close()
			wfFail(c, h.logger, scanErr, "wf set branches: scan failed")
			return
		}
		activeIDs = append(activeIDs, bid)
	}
	rows.Close()
	if rows.Err() != nil {
		wfFail(c, h.logger, rows.Err(), "wf set branches: rows error")
		return
	}
	actorStr := actorID.String()
	for _, bid := range activeIDs {
		if _, exists := desired[bid]; !exists {
			if _, err = tx.Exec(ctx,
				`UPDATE user_branch_assignments SET revoked_at = NOW(), revoked_by = $2 WHERE user_id = $1 AND branch_id = $3 AND revoked_at IS NULL`,
				userID, actorStr, bid); err != nil {
				wfFail(c, h.logger, err, "wf set branches: revoke failed")
				return
			}
		}
	}

	// 3) Upsert every desired branch as non-primary (reactivates revoked rows;
	//    Create's plain ON CONFLICT DO NOTHING cannot reactivate, so use DO UPDATE).
	for bid := range desired {
		if _, err = tx.Exec(ctx,
			`INSERT INTO user_branch_assignments (id, org_id, user_id, branch_id, is_primary, assigned_by, assigned_at)
			 VALUES ($1, $2, $3, $4, false, $5, NOW())
			 ON CONFLICT (user_id, branch_id) DO UPDATE
			 SET is_primary = false, revoked_at = NULL, revoked_by = NULL, assigned_by = EXCLUDED.assigned_by, assigned_at = NOW()`,
			uuid.New(), orgID, userID, bid, actorStr); err != nil {
			wfFail(c, h.logger, err, "wf set branches: upsert failed")
			return
		}
	}

	// 4) Set the single primary last.
	if _, err = tx.Exec(ctx,
		`UPDATE user_branch_assignments SET is_primary = true WHERE user_id = $1 AND branch_id = $2 AND revoked_at IS NULL`,
		userID, primaryID); err != nil {
		wfFail(c, h.logger, err, "wf set branches: set primary failed")
		return
	}

	if err = tx.Commit(ctx); err != nil {
		wfFail(c, h.logger, err, "wf set branches: commit failed")
		return
	}

	branchLabels := h.idLabels(c.Request.Context(), `SELECT id, branch_code, name FROM branches WHERE org_id=$1`, orgID)
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "set_branches", "user_branch", userID.String(),
		domain.JSON{"branches": mapToLabels(uuidSliceStr(activeIDs), branchLabels)},
		domain.JSON{
			"primary_branch": mapToLabels([]string{primaryStr}, branchLabels)[0],
			"branches":       mapToLabels(uuidSetKeys(desired), branchLabels),
		})
	c.JSON(http.StatusOK, gin.H{"id": userID.String(), "success": true})
}
