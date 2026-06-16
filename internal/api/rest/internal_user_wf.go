package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
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
type userWfHandler struct {
	userSvc     *adminUserSvc
	userRoleSvc *adminUserRoleSvc
	branchSvc   *adminBranchSvc
	auditSvc    *audit.Service
	logger      zerolog.Logger
}

func newUserWfHandler(
	userSvc *adminUserSvc,
	userRoleSvc *adminUserRoleSvc,
	branchSvc *adminBranchSvc,
	auditSvc *audit.Service,
	logger zerolog.Logger,
) *userWfHandler {
	return &userWfHandler{
		userSvc:     userSvc,
		userRoleSvc: userRoleSvc,
		branchSvc:   branchSvc,
		auditSvc:    auditSvc,
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
		users.POST("/:id/wf-assign-role", h.handleWfAssignRole)
		users.POST("/:id/wf-revoke-role", h.handleWfRevokeRole)
		users.POST("/:id/wf-assign-branch", h.handleWfAssignBranch)
		users.POST("/:id/wf-revoke-branch", h.handleWfRevokeBranch)
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

func (h *userWfHandler) audit(orgID, actorID uuid.UUID, actorEmail, action, resourceType, resourceID string, oldV, newV domain.JSON) {
	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OldValue:     oldV,
		NewValue:     newV,
	})
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

func (h *userWfHandler) handleWfCreate(c *gin.Context) {
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	input := adminpkg.CreateUserInput{
		Username:  dataStr(env.Data, "username"),
		Email:     dataStr(env.Data, "email"),
		FullName:  dataStr(env.Data, "full_name"),
		Password:  dataStr(env.Data, "password"),
		CreatedBy: actorID,
	}
	user, err := h.userSvc.CreateUser(c.Request.Context(), orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf create user failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "create_user", "user", user.ID.String(), nil, wfUserToJSON(user))
	c.JSON(http.StatusOK, gin.H{"id": user.ID.String(), "success": true})
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
	h.audit(orgID, actorID, actorEmail, "update_user", "user", id.String(), wfUserToJSON(oldUser), wfUserToJSON(user))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

func (h *userWfHandler) handleWfDelete(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	_, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err := h.userSvc.DeleteUser(c.Request.Context(), id, orgID); err != nil {
		wfFail(c, h.logger, err, "wf delete user failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "delete_user", "user", id.String(), wfUserToJSON(oldUser), nil)
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
	_, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	oldUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	if err := fn(c, id, orgID); err != nil {
		wfFail(c, h.logger, err, "wf "+action+" failed")
		return
	}
	newUser, _ := h.userSvc.GetUser(c.Request.Context(), id, orgID)
	h.audit(orgID, actorID, actorEmail, action, "user", id.String(), wfUserToJSON(oldUser), wfUserToJSON(newUser))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

// ── role assignment ─────────────────────────────────────────────────────────

func (h *userWfHandler) handleWfAssignRole(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	roleID, err := uuid.Parse(dataStr(env.Data, "role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_id", "success": false})
		return
	}
	var branchID *uuid.UUID
	if b := dataStr(env.Data, "branch_id"); b != "" {
		if bid, perr := uuid.Parse(b); perr == nil {
			branchID = &bid
		}
	}
	ur, err := h.userRoleSvc.AssignRole(c.Request.Context(), userID, roleID, orgID, branchID, actorID)
	if err != nil {
		wfFail(c, h.logger, err, "wf assign role failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "assign_role", "user_role", userID.String(), nil, domain.JSON{
		"role_id":   roleID.String(),
		"branch_id": dataStr(env.Data, "branch_id"),
	})
	c.JSON(http.StatusOK, gin.H{"id": ur.ID.String(), "success": true})
}

func (h *userWfHandler) handleWfRevokeRole(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	roleID, err := uuid.Parse(dataStr(env.Data, "role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_id", "success": false})
		return
	}
	if err := h.userRoleSvc.RevokeRole(c.Request.Context(), userID, roleID, orgID, actorID); err != nil {
		wfFail(c, h.logger, err, "wf revoke role failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "revoke_role", "user_role", userID.String(), domain.JSON{"role_id": roleID.String()}, nil)
	c.JSON(http.StatusOK, gin.H{"id": userID.String(), "success": true})
}

// ── branch assignment ───────────────────────────────────────────────────────

func (h *userWfHandler) handleWfAssignBranch(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	branchID, err := uuid.Parse(dataStr(env.Data, "branch_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch_id", "success": false})
		return
	}
	params := adminpkg.UserBranchParams{
		BranchID:   branchID,
		IsPrimary:  dataBool(env.Data, "is_primary"),
		AssignedBy: actorID,
	}
	uba, err := h.branchSvc.AssignUserToBranch(c.Request.Context(), userID, branchID, orgID, params)
	if err != nil {
		wfFail(c, h.logger, err, "wf assign branch failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "assign_branch", "user_branch", userID.String(), nil, domain.JSON{
		"branch_id":  branchID.String(),
		"is_primary": params.IsPrimary,
	})
	c.JSON(http.StatusOK, gin.H{"id": uba.ID.String(), "success": true})
}

func (h *userWfHandler) handleWfRevokeBranch(c *gin.Context) {
	userID, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := h.bindEnvelope(c)
	if !ok {
		return
	}
	assignmentID, err := uuid.Parse(dataStr(env.Data, "assignment_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment_id", "success": false})
		return
	}
	if err := h.branchSvc.RevokeUserBranch(c.Request.Context(), assignmentID, orgID, actorID); err != nil {
		wfFail(c, h.logger, err, "wf revoke branch failed")
		return
	}
	h.audit(orgID, actorID, actorEmail, "revoke_branch", "user_branch", userID.String(), domain.JSON{"assignment_id": assignmentID.String()}, nil)
	c.JSON(http.StatusOK, gin.H{"id": assignmentID.String(), "success": true})
}
