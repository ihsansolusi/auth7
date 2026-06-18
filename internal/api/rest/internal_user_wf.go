package rest

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

// wfUserToJSON mirrors admin.userToJSON (which is unexported) for audit snapshots.
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

// userWfHandler serves the workflow7 service-task callbacks for the user
// lifecycle and user-role / user-branch assignments. These run under
// /internal/v1 (M2M-only) and are invoked by workflow7 once an auto-approval
// flow reaches its PROCESS_TO_CORE step.
//
// Contract (mirrors the enterprise wf-* pattern):
//   request  : { "data": {...}, "master_id": "<uuid|"">", "master_type": "AC_USER", "wf_instance_id": "<uuid>" }
//   response : { "id": "<uuid>", "success": true }   on 2xx  (workflow7 on_complete_vars: $.id, $.success)
//             non-2xx + { "error": ..., "success": false } on failure so workflow7 retries / fails the step.
//
// Unlike the /admin/v1 handlers, org_id and the acting user are NOT taken from
// a user JWT (the caller is workflow7's M2M token). They travel inside `data`,
// injected by the BFF from the initiator's JWT before the workflow was started.
//
// Assignments use REPLACE-the-whole-set semantics (master-detail): wf-set-roles
// and wf-set-branches receive the full desired set and reconcile against the
// current active rows. Roles are org-wide only (branch_id NULL).
type userWfHandler struct {
	userSvc     *adminUserSvc
	userRoleSvc *adminUserRoleSvc
	branchSvc   *adminBranchSvc
	store       *postgres.Store
	auditSvc    *audit.Service
	mailer      mailer.Mailer
	logger      zerolog.Logger
}

func newUserWfHandler(
	userSvc *adminUserSvc,
	userRoleSvc *adminUserRoleSvc,
	branchSvc *adminBranchSvc,
	store *postgres.Store,
	auditSvc *audit.Service,
	m mailer.Mailer,
	logger zerolog.Logger,
) *userWfHandler {
	return &userWfHandler{
		userSvc:     userSvc,
		userRoleSvc: userRoleSvc,
		branchSvc:   branchSvc,
		store:       store,
		auditSvc:    auditSvc,
		mailer:      m,
		logger:      logger,
	}
}

func (h *userWfHandler) registerRoutes(g *gin.RouterGroup) {
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

// ── envelope + helpers ────────────────────────────────────────────────────────

type wfEnvelope struct {
	Data         map[string]any `json:"data"`
	MasterID     string         `json:"master_id"`
	MasterType   string         `json:"master_type"`
	WfInstanceID string         `json:"wf_instance_id"`
}

func dataStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func dataBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func dataStrPtr(m map[string]any, key string) *string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	v, ok := raw.(string)
	if !ok {
		return nil
	}
	return &v
}

// dataMaps extracts an array-of-objects field (e.g. data.roles = [{role_id}]).
func dataMaps(m map[string]any, key string) []map[string]any {
	out := []map[string]any{}
	if m == nil {
		return out
	}
	arr, ok := m[key].([]any)
	if !ok {
		return out
	}
	for _, it := range arr {
		if mm, ok := it.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

// bindEnvelope parses the workflow envelope and resolves org_id + actor from data.
func (h *userWfHandler) bindEnvelope(c *gin.Context) (env wfEnvelope, orgID, actorID uuid.UUID, actorEmail string, ok bool) {
	if err := c.ShouldBindJSON(&env); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "success": false})
		return env, orgID, actorID, actorEmail, false
	}
	orgID, err := uuid.Parse(dataStr(env.Data, "org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required", "success": false})
		return env, orgID, actorID, actorEmail, false
	}
	actorID, _ = uuid.Parse(dataStr(env.Data, "actor_id"))
	actorEmail = dataStr(env.Data, "actor_email")
	return env, orgID, actorID, actorEmail, true
}

func (h *userWfHandler) audit(data map[string]any, wfInstanceID string, orgID, actorID uuid.UUID, actorEmail, action, resourceType, resourceID string, oldV, newV domain.JSON) {
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

// uuidSetKeys / uuidSliceStr render id collections for audit before/after snapshots.
func uuidSetKeys(m map[uuid.UUID]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k.String())
	}
	return out
}

func uuidSliceStr(s []uuid.UUID) []string {
	out := make([]string, 0, len(s))
	for _, u := range s {
		out = append(out, u.String())
	}
	return out
}

// idLabels loads an id->label map (label = code/branch_code, fallback name) for a
// reference table, so audit before/after snapshots show human-readable codes.
// The query must SELECT (id, code, name) filtered by org_id = $1.
func (h *userWfHandler) idLabels(ctx context.Context, query string, orgID uuid.UUID) map[string]string {
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

// mapToLabels maps id strings through a label map, falling back to the id.
func mapToLabels(ids []string, labels map[string]string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if l, ok := labels[id]; ok && l != "" {
			out = append(out, l)
		} else {
			out = append(out, id)
		}
	}
	return out
}

func paramID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id", "success": false})
		return id, false
	}
	return id, true
}

func wfFail(c *gin.Context, logger zerolog.Logger, err error, msg string) {
	logger.Error().Err(err).Msg(msg)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "success": false})
}

// ── user lifecycle ──────────────────────────────────────────────────────────

// handleWfCreate creates the user and, atomically within the same callback,
// applies the initial org-wide roles (data.roles = [{role_id}]) and the primary
// branch (data.primary_branch_id). A failure in any step fails the workflow step.
func (h *userWfHandler) handleWfCreate(c *gin.Context) {
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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

func (h *userWfHandler) handleWfUpdate(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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

func (h *userWfHandler) handleWfDelete(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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

func (h *userWfHandler) handleWfLock(c *gin.Context) {
	h.statusChange(c, "lock_user", func(c *gin.Context, id, orgID uuid.UUID) error {
		return h.userSvc.LockUser(c.Request.Context(), id, orgID)
	})
}

func (h *userWfHandler) handleWfUnlock(c *gin.Context) {
	h.statusChange(c, "unlock_user", func(c *gin.Context, id, orgID uuid.UUID) error {
		return h.userSvc.UnlockUser(c.Request.Context(), id, orgID)
	})
}

func (h *userWfHandler) statusChange(c *gin.Context, action string, fn func(*gin.Context, uuid.UUID, uuid.UUID) error) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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

func (h *userWfHandler) handleWfSetRoles(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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

func (h *userWfHandler) handleWfSetBranches(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
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
