// Package wfcallback hosts the workflow7 service-task callback handlers mounted
// under /internal/v1 (M2M-only). They are the write path for Access Management:
// bos7-enterprise submits mutations to workflow7, and once an (auto-)approval
// flow reaches its PROCESS_TO_CORE step, workflow7 invokes these wf-* callbacks.
//
// Kept separate from the user-JWT admin read API (package admin) on purpose:
// different caller (workflow7 M2M token), different trust boundary, different
// contract. Handlers accept their dependencies as interfaces so package rest can
// inject the concrete admin service adapters without an import cycle.
package wfcallback

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/rs/zerolog"
)

// ── injected service interfaces (satisfied by the concrete admin adapters) ──────

// UserService is the user lifecycle surface the wf-callbacks drive.
type UserService interface {
	CreateUser(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateUserInput) (*domain.User, error)
	GetUser(ctx interface{}, id, orgID uuid.UUID) (*domain.User, error)
	UpdateUser(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input adminpkg.UpdateUserInput) (*domain.User, error)
	DeleteUser(ctx interface{}, id, orgID uuid.UUID) error
	LockUser(ctx interface{}, id, orgID uuid.UUID) error
	UnlockUser(ctx interface{}, id, orgID uuid.UUID) error
}

// UserRoleService is the user↔role assignment surface.
type UserRoleService interface {
	AssignRole(ctx interface{}, userID, roleID, orgID uuid.UUID, branchID *uuid.UUID, grantedBy uuid.UUID) (*domain.UserRole, error)
	RevokeRole(ctx interface{}, userID, roleID, orgID, revokedBy uuid.UUID) error
	GetUserRoles(ctx interface{}, userID uuid.UUID) ([]*domain.UserRole, error)
}

// BranchService is the user↔branch assignment surface.
type BranchService interface {
	AssignUserToBranch(ctx interface{}, userID, branchID, orgID uuid.UUID, params adminpkg.UserBranchParams) (*domain.UserBranchAssignment, error)
}

// RoleService is the role lifecycle + permission assignment surface.
type RoleService interface {
	CreateRole(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateRoleInput) (*domain.Role, error)
	UpdateRole(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input adminpkg.UpdateRoleInput) (*domain.Role, error)
	DeleteRole(ctx interface{}, id, orgID uuid.UUID) error
	AssignPermissions(ctx interface{}, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	GetRole(ctx interface{}, id, orgID uuid.UUID) (*domain.Role, error)
	GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error)
	ListPermissions(ctx interface{}) ([]*domain.Permission, error)
}

// ClientService is the OAuth2 client lifecycle surface.
type ClientService interface {
	CreateClient(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateClientInput) (*domain.Client, error)
	GetClient(ctx interface{}, id uuid.UUID) (*domain.Client, error)
	UpdateClient(ctx interface{}, id uuid.UUID, orgID uuid.UUID, input adminpkg.UpdateClientInput) (*domain.Client, error)
	DeleteClient(ctx interface{}, id uuid.UUID) error
}

// ── workflow envelope + data helpers ───────────────────────────────────────────

// wfEnvelope is the request body workflow7 posts to every wf-* callback.
//
//	{ "data": {...}, "master_id": "<uuid|"">", "master_type": "AC_USER", "wf_instance_id": "<uuid>" }
//
// A 2xx + { "id", "success": true } completes the workflow step; a non-2xx +
// { "error", "success": false } makes workflow7 retry/fail the step.
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

// dataInt reads a numeric field (JSON numbers decode to float64).
func dataInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
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

// dataStrSlice reads an array-of-strings field (e.g. allowed_scopes, role_ids).
func dataStrSlice(m map[string]any, key string) []string {
	out := []string{}
	if m == nil {
		return out
	}
	arr, ok := m[key].([]any)
	if !ok {
		return out
	}
	for _, it := range arr {
		if s, ok := it.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// bindWfEnvelope parses the workflow envelope and resolves org_id + actor from data.
func bindWfEnvelope(c *gin.Context) (env wfEnvelope, orgID, actorID uuid.UUID, actorEmail string, ok bool) {
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

func paramID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id", "success": false})
		return id, false
	}
	return id, true
}

func wfFail(c *gin.Context, logger zerolog.Logger, err error, msg string) {
	logger.Error().Err(err).Msg(msg)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "success": false})
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
