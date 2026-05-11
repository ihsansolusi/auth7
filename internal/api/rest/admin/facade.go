package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type FacadeHandler struct {
	store    *postgres.Store
	auditSvc *audit.Service
	logger   zerolog.Logger
}

type facadeErrorDescriptor struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status"`
	Message    string `json:"message"`
}

var facadeErrorCatalog = []facadeErrorDescriptor{
	{Code: "AUTH7_FACADE_INVALID_ORG_ID", HTTPStatus: http.StatusBadRequest, Message: "invalid or missing org_id"},
	{Code: "AUTH7_FACADE_INVALID_USER_ID", HTTPStatus: http.StatusBadRequest, Message: "invalid user_id"},
	{Code: "AUTH7_FACADE_INVALID_REQUEST", HTTPStatus: http.StatusBadRequest, Message: "invalid request payload"},
	{Code: "AUTH7_FACADE_NOT_FOUND", HTTPStatus: http.StatusNotFound, Message: "resource not found"},
	{Code: "AUTH7_FACADE_PERMISSION_DENIED", HTTPStatus: http.StatusForbidden, Message: "permission denied"},
	{Code: "AUTH7_FACADE_INTERNAL_ERROR", HTTPStatus: http.StatusInternalServerError, Message: "internal server error"},
}

func NewFacadeHandler(store *postgres.Store, auditSvc *audit.Service, logger zerolog.Logger) *FacadeHandler {
	return &FacadeHandler{
		store:    store,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

func (h *FacadeHandler) RegisterRoutes(r *gin.RouterGroup) {
	facade := r.Group("/facade")
	{
		facade.GET("/contracts/readiness", h.handleContractReadiness)
		facade.GET("/contracts/branch-projections", h.handleBranchProjectionSnapshot)
		facade.GET("/contracts/employee-references/:user_id", h.handleEmployeeReferenceSnapshot)
		facade.GET("/access/users", h.handleAccessUsers)
		facade.GET("/access/roles", h.handleAccessRoles)
		facade.GET("/access/permissions", h.handleAccessPermissions)
		facade.GET("/error-catalog", h.handleErrorCatalog)
		facade.POST("/audit-hooks/admin-actions", h.handleAdminAuditHook)
	}
}

func (h *FacadeHandler) handleContractReadiness(c *gin.Context) {
	pool := h.store.Pool()
	readyBranchProjection := tableExists(c.Request.Context(), pool, "branches") && tableExists(c.Request.Context(), pool, "branch_hierarchies")
	readyEmployeeRef := tableExists(c.Request.Context(), pool, "user_attributes")
	readyPermission := tableExists(c.Request.Context(), pool, "permissions")

	h.writeSuccess(c, gin.H{
		"branch_projection_consumer_ready":  readyBranchProjection,
		"employee_reference_consumer_ready": readyEmployeeRef,
		"legacy_permission_baseline_ready":  readyPermission,
		"wave":                              "W3",
		"mode":                              "runtime-endpoints+adapter-readiness",
	}, nil)
}

func (h *FacadeHandler) handleBranchProjectionSnapshot(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		h.writeCatalogError(c, facadeErrorCatalog[0], nil)
		return
	}

	const q = `
		SELECT id, org_id, branch_type_id, parent_id, code, name, status, updated_at
		FROM branches
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := h.store.Pool().Query(c.Request.Context(), q, orgID)
	if err != nil {
		h.writeMappedError(c, err)
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			branchID     uuid.UUID
			orgIDVal     uuid.UUID
			branchTypeID uuid.UUID
			parentID     *uuid.UUID
			code         string
			name         string
			status       string
			updatedAt    time.Time
		)
		if err := rows.Scan(&branchID, &orgIDVal, &branchTypeID, &parentID, &code, &name, &status, &updatedAt); err != nil {
			h.writeMappedError(c, err)
			return
		}

		items = append(items, gin.H{
			"branch_id":                    branchID.String(),
			"org_id":                       orgIDVal.String(),
			"branch_code":                  code,
			"branch_name":                  name,
			"branch_status":                status,
			"branch_type_id":               branchTypeID.String(),
			"parent_id":                    stringifyUUIDPtr(parentID),
			"updated_at":                   updatedAt.Format(time.RFC3339),
			"enterprise_mapping_reference": "pending-source-contract",
		})
	}

	h.writeSuccess(c, gin.H{
		"items": items,
	}, gin.H{
		"count": len(items),
	})
}

func (h *FacadeHandler) handleEmployeeReferenceSnapshot(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		h.writeCatalogError(c, facadeErrorCatalog[1], nil)
		return
	}

	const q = `
		SELECT key, value
		FROM user_attributes
		WHERE user_id = $1
		  AND key IN ('employee_id','department_code','position_code','home_enterprise_branch_id','employment_status')
	`
	rows, err := h.store.Pool().Query(c.Request.Context(), q, userID)
	if err != nil {
		h.writeMappedError(c, err)
		return
	}
	defer rows.Close()

	attrs := gin.H{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			h.writeMappedError(c, err)
			return
		}
		attrs[k] = v
	}

	h.writeSuccess(c, gin.H{
		"user_id":     userID.String(),
		"attributes":  attrs,
		"consumer_of": "core7-service-enterprise",
	}, nil)
}

func (h *FacadeHandler) handleAccessUsers(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		h.writeCatalogError(c, facadeErrorCatalog[0], nil)
		return
	}

	users, total, err := h.store.UserRepository.ListByOrg(c.Request.Context(), orgID, 100, 0)
	if err != nil {
		h.writeMappedError(c, err)
		return
	}

	h.writeSuccess(c, gin.H{
		"items": users,
	}, gin.H{
		"total": total,
	})
}

func (h *FacadeHandler) handleAccessRoles(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		h.writeCatalogError(c, facadeErrorCatalog[0], nil)
		return
	}

	roles, err := h.store.RoleRepository.ListByOrg(c.Request.Context(), orgID)
	if err != nil {
		h.writeMappedError(c, err)
		return
	}

	h.writeSuccess(c, gin.H{
		"items": roles,
	}, gin.H{
		"count": len(roles),
	})
}

func (h *FacadeHandler) handleAccessPermissions(c *gin.Context) {
	perms, err := h.store.PermissionRepository.List(c.Request.Context())
	if err != nil {
		h.writeMappedError(c, err)
		return
	}

	h.writeSuccess(c, gin.H{
		"items": perms,
	}, gin.H{
		"count": len(perms),
	})
}

func (h *FacadeHandler) handleErrorCatalog(c *gin.Context) {
	h.writeSuccess(c, gin.H{
		"items": facadeErrorCatalog,
	}, gin.H{
		"version": "w3-v1",
	})
}

func (h *FacadeHandler) handleAdminAuditHook(c *gin.Context) {
	var input struct {
		OrgID         string      `json:"org_id"`
		Action        string      `json:"action"`
		ResourceType  string      `json:"resource_type"`
		ResourceID    string      `json:"resource_id"`
		CorrelationID string      `json:"correlation_id"`
		OldValue      domain.JSON `json:"old_value"`
		NewValue      domain.JSON `json:"new_value"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.writeCatalogError(c, facadeErrorCatalog[2], nil)
		return
	}

	orgID, err := uuid.Parse(input.OrgID)
	if err != nil {
		h.writeCatalogError(c, facadeErrorCatalog[0], nil)
		return
	}

	actorID, actorEmail := getActorFromContext(c)
	action := "facade.admin_action"
	if input.Action != "" {
		action = "facade." + sanitizeAction(input.Action)
	}

	newVal := domain.JSON{
		"correlation_id": input.CorrelationID,
		"source":         "bos7-enterprise",
	}
	for k, v := range input.NewValue {
		newVal[k] = v
	}

	h.auditSvc.LogAsync(audit.LogInput{
		OrgID:        orgID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: defaultStr(input.ResourceType, "facade_admin"),
		ResourceID:   input.ResourceID,
		OldValue:     input.OldValue,
		NewValue:     newVal,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})

	h.writeSuccess(c, gin.H{
		"accepted": true,
		"action":   action,
	}, nil)
}

func (h *FacadeHandler) writeSuccess(c *gin.Context, data gin.H, meta gin.H) {
	body := gin.H{
		"success": true,
		"data":    data,
	}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(http.StatusOK, body)
}

func (h *FacadeHandler) writeCatalogError(c *gin.Context, desc facadeErrorDescriptor, details gin.H) {
	errBody := gin.H{
		"code":        desc.Code,
		"message":     desc.Message,
		"http_status": desc.HTTPStatus,
	}
	if details != nil {
		errBody["details"] = details
	}
	c.JSON(desc.HTTPStatus, gin.H{
		"success": false,
		"error":   errBody,
	})
}

func (h *FacadeHandler) writeMappedError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrPermissionDenied):
		h.writeCatalogError(c, facadeErrorCatalog[4], nil)
		return
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		h.writeCatalogError(c, facadeErrorCatalog[3], nil)
		return
	default:
		h.logger.Error().Err(err).Msg("facade endpoint failed")
		h.writeCatalogError(c, facadeErrorCatalog[5], nil)
	}
}

func tableExists(ctx context.Context, pool *pgxpool.Pool, tableName string) bool {
	var exists bool
	err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+tableName).Scan(&exists)
	return err == nil && exists
}

func parseOrgID(c *gin.Context) (uuid.UUID, bool) {
	orgStr := c.Query("org_id")
	if orgStr == "" {
		orgStr = claimsOrgID(c)
	}
	if orgStr == "" {
		return uuid.Nil, false
	}
	orgID, err := uuid.Parse(orgStr)
	if err != nil {
		return uuid.Nil, false
	}
	return orgID, true
}

func stringifyUUIDPtr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func defaultStr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func sanitizeAction(action string) string {
	a := strings.TrimSpace(strings.ToLower(action))
	if a == "" {
		return "admin_action"
	}
	a = strings.ReplaceAll(a, " ", "_")
	a = strings.ReplaceAll(a, "/", "_")
	a = strings.ReplaceAll(a, ":", "_")
	return a
}
