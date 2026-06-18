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

// roleWfHandler serves the workflow7 service-task callbacks for the role
// lifecycle + permission assignment, mirroring the user wf-* pattern. Reuses the
// package-level helpers from internal_user_wf.go (wfEnvelope, dataStr, dataMaps,
// dataStrPtr, dataBool, paramID, wfFail).
type roleWfHandler struct {
	roleSvc  *adminRoleSvc
	auditSvc *audit.Service
	logger   zerolog.Logger
}

func newRoleWfHandler(roleSvc *adminRoleSvc, auditSvc *audit.Service, logger zerolog.Logger) *roleWfHandler {
	return &roleWfHandler{roleSvc: roleSvc, auditSvc: auditSvc, logger: logger}
}

func (h *roleWfHandler) registerRoutes(g *gin.RouterGroup) {
	roles := g.Group("/roles")
	{
		roles.POST("/wf-create", h.handleWfCreate)
		roles.PUT("/:id/wf-update", h.handleWfUpdate)
		roles.POST("/:id/wf-delete", h.handleWfDelete)
		roles.POST("/:id/wf-set-permissions", h.handleWfSetPermissions)
	}
}

// bindWfEnvelope parses the workflow envelope and resolves org_id + actor from
// data (free-function variant usable by any wf handler in this package).
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

func (h *roleWfHandler) audit(data map[string]any, wfInstanceID string, orgID, actorID uuid.UUID, actorEmail, action, resourceType, resourceID string, oldV, newV domain.JSON) {
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

// permLabel returns a human-readable permission label: code, else name, else id.
func permLabel(p *domain.Permission) string {
	if p == nil {
		return ""
	}
	if p.Code != "" {
		return p.Code
	}
	if p.Name != "" {
		return p.Name
	}
	return p.ID.String()
}

func roleToJSON(r *domain.Role) domain.JSON {
	if r == nil {
		return nil
	}
	return domain.JSON{
		"id":          r.ID.String(),
		"code":        r.Code,
		"name":        r.Name,
		"description": r.Description,
		"is_default":  r.IsDefault,
		"created_at":  r.CreatedAt.Format(time.RFC3339),
	}
}

// permissionIDsFromData reads data.permissions = [{ permission_id }].
func permissionIDsFromData(data map[string]any) []uuid.UUID {
	out := []uuid.UUID{}
	for _, pm := range dataMaps(data, "permissions") {
		if pid, err := uuid.Parse(dataStr(pm, "permission_id")); err == nil {
			out = append(out, pid)
		}
	}
	return out
}

func (h *roleWfHandler) handleWfCreate(c *gin.Context) {
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	input := adminpkg.CreateRoleInput{
		Code:        dataStr(env.Data, "code"),
		Name:        dataStr(env.Data, "name"),
		Description: dataStr(env.Data, "description"),
		IsDefault:   dataBool(env.Data, "is_default"),
	}
	role, err := h.roleSvc.CreateRole(c.Request.Context(), orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf create role failed")
		return
	}
	// Initial permission assignment (optional).
	if perms := permissionIDsFromData(env.Data); len(perms) > 0 {
		if err := h.roleSvc.AssignPermissions(c.Request.Context(), role.ID, perms); err != nil {
			wfFail(c, h.logger, err, "wf create role: assign permissions failed")
			return
		}
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "create_role", "role", role.ID.String(), nil, roleToJSON(role))
	c.JSON(http.StatusOK, gin.H{"id": role.ID.String(), "success": true})
}

func (h *roleWfHandler) handleWfUpdate(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldRole, _ := h.roleSvc.GetRole(c.Request.Context(), id, orgID)
	input := adminpkg.UpdateRoleInput{
		Name:        dataStrPtr(env.Data, "name"),
		Description: dataStrPtr(env.Data, "description"),
	}
	role, err := h.roleSvc.UpdateRole(c.Request.Context(), id, orgID, input)
	if err != nil {
		wfFail(c, h.logger, err, "wf update role failed")
		return
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "update_role", "role", id.String(), roleToJSON(oldRole), roleToJSON(role))
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

func (h *roleWfHandler) handleWfDelete(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	oldRole, _ := h.roleSvc.GetRole(c.Request.Context(), id, orgID)
	if err := h.roleSvc.DeleteRole(c.Request.Context(), id, orgID); err != nil {
		wfFail(c, h.logger, err, "wf delete role failed")
		return
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "delete_role", "role", id.String(), roleToJSON(oldRole), nil)
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}

// handleWfSetPermissions replaces the role's full permission set
// (AssignPermissions already does delete-all + re-insert).
func (h *roleWfHandler) handleWfSetPermissions(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		return
	}
	env, orgID, actorID, actorEmail, ok := bindWfEnvelope(c)
	if !ok {
		return
	}
	// Capture current permissions (before) as codes for the audit snapshot.
	beforePerms := []string{}
	if cur, gerr := h.roleSvc.GetPermissions(c.Request.Context(), id); gerr == nil {
		for _, p := range cur {
			beforePerms = append(beforePerms, permLabel(p))
		}
	}

	perms := permissionIDsFromData(env.Data)
	if err := h.roleSvc.AssignPermissions(c.Request.Context(), id, perms); err != nil {
		wfFail(c, h.logger, err, "wf set role permissions failed")
		return
	}

	// Resolve desired permission ids -> codes via the full permission catalog.
	permLabels := map[string]string{}
	if all, lerr := h.roleSvc.ListPermissions(c.Request.Context()); lerr == nil {
		for _, p := range all {
			permLabels[p.ID.String()] = permLabel(p)
		}
	}
	afterPerms := make([]string, 0, len(perms))
	for _, p := range perms {
		if l, ok := permLabels[p.String()]; ok {
			afterPerms = append(afterPerms, l)
		} else {
			afterPerms = append(afterPerms, p.String())
		}
	}
	h.audit(env.Data, env.WfInstanceID, orgID, actorID, actorEmail, "set_permissions", "role_permission", id.String(),
		domain.JSON{"permissions": beforePerms},
		domain.JSON{"permissions": afterPerms})
	c.JSON(http.StatusOK, gin.H{"id": id.String(), "success": true})
}
