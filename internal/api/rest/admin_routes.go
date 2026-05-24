package rest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/api/middleware"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	jwtpkg "github.com/ihsansolusi/auth7/internal/service/jwt"
	sessionpkg "github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

// adminClaims wraps jwtpkg.Claims to expose simple getter methods expected by AdminAuth middleware.
type adminClaims struct{ *jwtpkg.Claims }

func (c *adminClaims) GetSubject() string { return c.Claims.RegisteredClaims.Subject }
func (c *adminClaims) GetOrgID() string   { return c.Claims.OrgID }
func (c *adminClaims) GetRoles() []string { return c.Claims.Roles }
func (c *adminClaims) GetEmail() string   { return c.Claims.Email }

// RegisterAdminV1Routes wires admin handlers under /admin/v1 with JWT + role enforcement.
func (s *Server) RegisterAdminV1Routes(r *gin.Engine) {
	store, ok := s.deps.Store.(*postgres.Store)
	if !ok {
		s.deps.Logger.Warn().Msg("admin routes: store type assertion failed, skipping")
		return
	}
	jwtSvc, ok := s.deps.JWTSvc.(*jwtpkg.Service)
	if !ok {
		s.deps.Logger.Warn().Msg("admin routes: jwtSvc type assertion failed, skipping")
		return
	}

	auditSvc := audit.NewService(audit.NewPGStore(store.Pool()))

	bearerMW := func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			c.Abort()
			return
		}
		rawClaims, err := jwtSvc.VerifyAccessToken(auth[7:])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			c.Abort()
			return
		}
		c.Set("claims", &adminClaims{rawClaims})
		c.Next()
	}

	adminV1 := r.Group("/admin/v1")
	adminV1.Use(bearerMW)
	adminV1.Use(middleware.AdminAuth(middleware.DefaultAdminAuthConfig(), auditSvc, s.deps.Logger))

	adminpkg.NewUserHandler(newAdminUserSvc(store), auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	adminpkg.NewRoleHandler(newAdminRoleSvc(store), auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	adminpkg.NewFacadeHandler(store, auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	// UserRoleHandler excluded: its /users/:user_id/* path conflicts with UserHandler's /users/:id/*
	// Register only non-conflicting branch roles route from UserRoleHandler:
	userRoleSvcInst := newAdminUserRoleSvc(store)
	adminV1.GET("/branches/:id/roles", func(c *gin.Context) {
		branchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
			return
		}
		roles, err := userRoleSvcInst.GetBranchRoles(c.Request.Context(), branchID)
		if err != nil {
			s.deps.Logger.Error().Err(err).Msg("get branch roles failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"roles": roles})
	})
	adminpkg.NewAuditHandler(auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	adminpkg.NewOAuth2ClientHandler(newAdminOAuth2ClientSvc(store), auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	adminpkg.NewBranchHandler(newAdminBranchSvc(store), auditSvc, s.deps.Logger).RegisterBranchTypeRoutes(adminV1)

	if sessionSvc, ok := s.deps.SessionSvc.(*sessionpkg.Service); ok {
		adminpkg.NewSessionHandler(sessionSvc, auditSvc, s.deps.Logger).RegisterRoutes(adminV1)
	}

	adminV1.GET("/dashboard/stats", s.handleAdminStats(store))

	// Branch default roles — configured defaults per branch (separate from per-user assignments).
	adminV1.GET("/branches/:id/default-roles", s.handleGetBranchDefaultRoles(store))
	adminV1.PUT("/branches/:id/default-roles", s.handlePutBranchDefaultRoles(store))
}

func (s *Server) handleGetBranchDefaultRoles(store *postgres.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		branchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
			return
		}

		var branchName string
		_ = store.Pool().QueryRow(ctx, `SELECT name FROM branches WHERE id = $1`, branchID).Scan(&branchName)

		const q = `
			SELECT r.id, r.name, COALESCE(bdr.is_default, false) AS is_default
			FROM roles r
			LEFT JOIN branch_default_roles bdr
			  ON bdr.role_id = r.id AND bdr.branch_id = $1
			WHERE r.org_id = (SELECT org_id FROM branches WHERE id = $1)
			ORDER BY r.name`
		rows, err := store.Pool().Query(ctx, q, branchID)
		if err != nil {
			s.deps.Logger.Error().Err(err).Msg("get branch default roles failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		defer rows.Close()
		type defaultRole struct {
			RoleID    uuid.UUID `json:"role_id"`
			RoleName  string    `json:"role_name"`
			IsDefault bool      `json:"is_default"`
		}
		out := []defaultRole{}
		for rows.Next() {
			var dr defaultRole
			if err := rows.Scan(&dr.RoleID, &dr.RoleName, &dr.IsDefault); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
				return
			}
			out = append(out, dr)
		}

		c.JSON(http.StatusOK, gin.H{
			"branch_id":     branchID.String(),
			"branch_name":   branchName,
			"default_roles": out,
		})
	}
}

func (s *Server) handlePutBranchDefaultRoles(store *postgres.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		branchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
			return
		}

		var body struct {
			DefaultRoles []struct {
				RoleID    string `json:"role_id"`
				IsDefault bool   `json:"is_default"`
			} `json:"default_roles"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		tx, err := store.Pool().Begin(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if _, err := tx.Exec(ctx, `DELETE FROM branch_default_roles WHERE branch_id = $1`, branchID); err != nil {
			s.deps.Logger.Error().Err(err).Msg("clear branch default roles failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		for _, dr := range body.DefaultRoles {
			if !dr.IsDefault {
				continue
			}
			roleID, perr := uuid.Parse(dr.RoleID)
			if perr != nil {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO branch_default_roles (branch_id, role_id, is_default) VALUES ($1, $2, true)`,
				branchID, roleID,
			); err != nil {
				s.deps.Logger.Error().Err(err).Msg("insert branch default role failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
				return
			}
		}
		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"saved": true})
	}
}

// ── Stats ─────────────────────────────────────────────────────────────────────

func (s *Server) handleAdminStats(store *postgres.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		orgIDStr := c.Query("org_id")
		if orgIDStr == "" {
			orgIDStr = "00000000-0000-0000-0000-000000000001"
		}
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}

		users, totalUsers, _ := store.UserRepository.ListByOrg(ctx, orgID, 1000, 0)
		activeUsers, lockedUsers := 0, 0
		for _, u := range users {
			switch u.Status {
			case domain.UserStatusActive:
				activeUsers++
			case domain.UserStatusLocked:
				lockedUsers++
			}
		}

		roles, _ := store.RoleRepository.ListByOrg(ctx, orgID)

		_, auditTotal, _ := store.AuditLogRepository.List(ctx, domain.AuditLogFilter{Limit: 1})

		c.JSON(http.StatusOK, gin.H{
			"totalUsers":       totalUsers,
			"activeUsers":      activeUsers,
			"lockedUsers":      lockedUsers,
			"totalRoles":       len(roles),
			"totalBranches":    0,
			"totalClients":     0,
			"recentAuditCount": auditTotal,
		})
	}
}

// ── adminUserSvc ──────────────────────────────────────────────────────────────

type adminUserSvc struct{ store *postgres.Store }

func newAdminUserSvc(s *postgres.Store) *adminUserSvc { return &adminUserSvc{store: s} }

func (s *adminUserSvc) ListUsers(ctx interface{}, orgID uuid.UUID, limit, offset int, status string) ([]*domain.User, int, error) {
	users, total, err := s.store.UserRepository.ListByOrg(ctx.(context.Context), orgID, limit, offset)
	if err != nil || status == "" {
		return users, total, err
	}
	filtered := make([]*domain.User, 0, len(users))
	for _, u := range users {
		if string(u.Status) == status {
			filtered = append(filtered, u)
		}
	}
	return filtered, len(filtered), nil
}

func (s *adminUserSvc) GetUser(ctx interface{}, id, _ uuid.UUID) (*domain.User, error) {
	return s.store.UserRepository.GetByID(ctx.(context.Context), id)
}

func (s *adminUserSvc) CreateUser(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateUserInput) (*domain.User, error) {
	now := time.Now()
	user := &domain.User{
		ID:        uuid.Must(uuid.NewV7()),
		OrgID:     orgID,
		Username:  input.Username,
		Email:     input.Email,
		FullName:  input.FullName,
		Status:    domain.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return user, s.store.UserRepository.Create(ctx.(context.Context), user)
}

func (s *adminUserSvc) UpdateUser(ctx interface{}, id, _ uuid.UUID, input adminpkg.UpdateUserInput) (*domain.User, error) {
	user, err := s.store.UserRepository.GetByID(ctx.(context.Context), id)
	if err != nil {
		return nil, err
	}
	if input.Username != nil {
		user.Username = *input.Username
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.FullName != nil {
		user.FullName = *input.FullName
	}
	if input.Status != nil {
		user.Status = *input.Status
	}
	user.UpdatedAt = time.Now()
	return user, s.store.UserRepository.Update(ctx.(context.Context), user)
}

func (s *adminUserSvc) DeleteUser(ctx interface{}, id, _ uuid.UUID) error {
	return s.store.UserRepository.Delete(ctx.(context.Context), id)
}

func (s *adminUserSvc) LockUser(ctx interface{}, id, _ uuid.UUID) error {
	return s.updateUserStatus(ctx.(context.Context), id, domain.UserStatusLocked)
}

func (s *adminUserSvc) UnlockUser(ctx interface{}, id, _ uuid.UUID) error {
	return s.updateUserStatus(ctx.(context.Context), id, domain.UserStatusActive)
}

func (s *adminUserSvc) SuspendUser(ctx interface{}, id, _ uuid.UUID) error {
	return s.updateUserStatus(ctx.(context.Context), id, domain.UserStatusSuspended)
}

func (s *adminUserSvc) updateUserStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	user, err := s.store.UserRepository.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.Status = status
	user.UpdatedAt = time.Now()
	return s.store.UserRepository.Update(ctx, user)
}

// ── adminRoleSvc ──────────────────────────────────────────────────────────────

type adminRoleSvc struct{ store *postgres.Store }

func newAdminRoleSvc(s *postgres.Store) *adminRoleSvc { return &adminRoleSvc{store: s} }

func (s *adminRoleSvc) ListRoles(ctx interface{}, orgID uuid.UUID) ([]*domain.Role, error) {
	return s.store.RoleRepository.ListByOrg(ctx.(context.Context), orgID)
}

func (s *adminRoleSvc) GetRole(ctx interface{}, id, orgID uuid.UUID) (*domain.Role, error) {
	return s.store.RoleRepository.GetByID(ctx.(context.Context), id, orgID)
}

func (s *adminRoleSvc) CreateRole(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateRoleInput) (*domain.Role, error) {
	now := time.Now()
	role := &domain.Role{
		ID:          uuid.Must(uuid.NewV7()),
		OrgID:       orgID,
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		IsDefault:   input.IsDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return role, s.store.RoleRepository.Create(ctx.(context.Context), role)
}

func (s *adminRoleSvc) UpdateRole(ctx interface{}, id, orgID uuid.UUID, input adminpkg.UpdateRoleInput) (*domain.Role, error) {
	role, err := s.store.RoleRepository.GetByID(ctx.(context.Context), id, orgID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		role.Name = *input.Name
	}
	if input.Description != nil {
		role.Description = *input.Description
	}
	role.UpdatedAt = time.Now()
	return role, s.store.RoleRepository.Update(ctx.(context.Context), role)
}

func (s *adminRoleSvc) DeleteRole(ctx interface{}, id, orgID uuid.UUID) error {
	return s.store.RoleRepository.Delete(ctx.(context.Context), id, orgID)
}

func (s *adminRoleSvc) AssignPermissions(ctx interface{}, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	c := ctx.(context.Context)
	if err := s.store.RolePermissionRepository.DeleteByRole(c, roleID); err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		if err := s.store.RolePermissionRepository.Assign(c, roleID, pid); err != nil {
			return err
		}
	}
	return nil
}

func (s *adminRoleSvc) GetPermissions(ctx interface{}, roleID uuid.UUID) ([]*domain.Permission, error) {
	return s.store.RolePermissionRepository.GetByRole(ctx.(context.Context), roleID)
}

func (s *adminRoleSvc) ListPermissions(ctx interface{}) ([]*domain.Permission, error) {
	return s.store.PermissionRepository.List(ctx.(context.Context))
}

// ── adminUserRoleSvc ──────────────────────────────────────────────────────────

type adminUserRoleSvc struct{ store *postgres.Store }

func newAdminUserRoleSvc(s *postgres.Store) *adminUserRoleSvc { return &adminUserRoleSvc{store: s} }

func (s *adminUserRoleSvc) AssignRole(ctx interface{}, userID, roleID, orgID uuid.UUID, branchID *uuid.UUID, grantedBy uuid.UUID) (*domain.UserRole, error) {
	now := time.Now()
	ur := &domain.UserRole{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		RoleID:    roleID,
		OrgID:     orgID,
		BranchID:  branchID,
		GrantedBy: grantedBy,
		GrantedAt: now,
	}
	return ur, s.store.UserRoleRepository.Create(ctx.(context.Context), ur)
}

func (s *adminUserRoleSvc) RevokeRole(ctx interface{}, userID, roleID, orgID, revokedBy uuid.UUID) error {
	return s.store.UserRoleRepository.RevokeByUserAndRole(ctx.(context.Context), userID, roleID, orgID, revokedBy)
}

func (s *adminUserRoleSvc) GetUserRoles(ctx interface{}, userID uuid.UUID) ([]*domain.UserRole, error) {
	return s.store.UserRoleRepository.GetByUser(ctx.(context.Context), userID)
}

func (s *adminUserRoleSvc) GetBranchRoles(ctx interface{}, branchID uuid.UUID) ([]*domain.UserRole, error) {
	return s.store.UserRoleRepository.GetByBranch(ctx.(context.Context), branchID)
}

// ── adminOAuth2ClientSvc ──────────────────────────────────────────────────────

type adminOAuth2ClientSvc struct{ store *postgres.Store }

func newAdminOAuth2ClientSvc(s *postgres.Store) *adminOAuth2ClientSvc {
	return &adminOAuth2ClientSvc{store: s}
}

func (s *adminOAuth2ClientSvc) ListClients(ctx interface{}, orgID uuid.UUID) ([]*domain.Client, error) {
	c := ctx.(context.Context)
	const q = `
		SELECT id, client_id, org_id, name, COALESCE(description, ''),
		       client_type, token_endpoint_auth_method,
		       COALESCE(allowed_scopes, '{}'), COALESCE(allowed_redirect_uris, '{}'), COALESCE(allowed_origins, '{}'),
		       token_expiration, refresh_token_expiration,
		       allow_multiple_tokens, skip_consent_screen, is_active,
		       created_at, updated_at
		FROM oauth2_clients
		WHERE org_id = $1
		ORDER BY created_at DESC`
	rows, err := s.store.Pool().Query(c, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Client
	for rows.Next() {
		cl := &domain.Client{}
		var cType, authMethod, clientIDStr string
		if err := rows.Scan(
			&cl.ID, &clientIDStr, &cl.OrgID, &cl.Name, &cl.Description,
			&cType, &authMethod,
			&cl.AllowedScopes, &cl.AllowedRedirectURIs, &cl.AllowedOrigins,
			&cl.TokenExpiration, &cl.RefreshTokenExpiration,
			&cl.AllowMultipleTokens, &cl.SkipConsentScreen, &cl.IsActive,
			&cl.CreatedAt, &cl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cl.ClientID = clientIDStr
		cl.ClientType = domain.ClientType(cType)
		cl.TokenEndpointAuthMethod = domain.TokenEndpointAuthMethod(authMethod)
		result = append(result, cl)
	}
	return result, rows.Err()
}

func (s *adminOAuth2ClientSvc) GetClient(ctx interface{}, id uuid.UUID) (*domain.Client, error) {
	c := ctx.(context.Context)
	const q = `
		SELECT id, client_id, org_id, name, COALESCE(description, ''),
		       client_type, token_endpoint_auth_method,
		       COALESCE(allowed_scopes, '{}'), COALESCE(allowed_redirect_uris, '{}'), COALESCE(allowed_origins, '{}'),
		       COALESCE(client_secret_hash, ''),
		       token_expiration, refresh_token_expiration,
		       allow_multiple_tokens, skip_consent_screen, is_active,
		       created_at, updated_at,
		       COALESCE(app_url, ''), COALESCE(icon_name, ''), COALESCE(icon_color, '')
		FROM oauth2_clients
		WHERE id = $1`
	cl := &domain.Client{}
	var cType, authMethod, clientIDStr string
	err := s.store.Pool().QueryRow(c, q, id).Scan(
		&cl.ID, &clientIDStr, &cl.OrgID, &cl.Name, &cl.Description,
		&cType, &authMethod,
		&cl.AllowedScopes, &cl.AllowedRedirectURIs, &cl.AllowedOrigins,
		&cl.ClientSecretHash,
		&cl.TokenExpiration, &cl.RefreshTokenExpiration,
		&cl.AllowMultipleTokens, &cl.SkipConsentScreen, &cl.IsActive,
		&cl.CreatedAt, &cl.UpdatedAt,
		&cl.AppURL, &cl.IconName, &cl.IconColor,
	)
	if err != nil {
		return nil, err
	}
	cl.ClientID = clientIDStr
	cl.ClientType = domain.ClientType(cType)
	cl.TokenEndpointAuthMethod = domain.TokenEndpointAuthMethod(authMethod)
	return cl, nil
}

func adminHashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(h[:])
}

func (s *adminOAuth2ClientSvc) CreateClient(ctx interface{}, orgID uuid.UUID, input adminpkg.CreateClientInput) (*domain.Client, error) {
	c := ctx.(context.Context)
	now := time.Now()
	id := uuid.Must(uuid.NewV7())
	cl := &domain.Client{
		ID:                      id,
		ClientID:                id.String(),
		OrgID:                   orgID,
		Name:                    input.Name,
		Description:             input.Description,
		ClientType:              input.ClientType,
		TokenEndpointAuthMethod: input.TokenEndpointAuthMethod,
		AllowedScopes:           input.AllowedScopes,
		AllowedRedirectURIs:     input.AllowedRedirectURIs,
		AllowedOrigins:          input.AllowedOrigins,
		TokenExpiration:         input.TokenExpiration,
		RefreshTokenExpiration:  input.RefreshTokenExpiration,
		AllowMultipleTokens:     input.AllowMultipleTokens,
		SkipConsentScreen:       input.SkipConsentScreen,
		IsActive:                true,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if input.ClientSecret != "" {
		cl.ClientSecretHash = adminHashSecret(input.ClientSecret)
	}
	return cl, s.store.OAuth2ClientRepository.CreateClient(c, cl)
}

func (s *adminOAuth2ClientSvc) UpdateClient(ctx interface{}, id uuid.UUID, _ uuid.UUID, input adminpkg.UpdateClientInput) (*domain.Client, error) {
	c := ctx.(context.Context)
	cl, err := s.GetClient(c, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		cl.Name = *input.Name
	}
	if input.Description != nil {
		cl.Description = *input.Description
	}
	if input.AllowedScopes != nil {
		cl.AllowedScopes = *input.AllowedScopes
	}
	if input.AllowedRedirectURIs != nil {
		cl.AllowedRedirectURIs = *input.AllowedRedirectURIs
	}
	if input.AllowedOrigins != nil {
		cl.AllowedOrigins = *input.AllowedOrigins
	}
	if input.TokenExpiration != nil {
		cl.TokenExpiration = *input.TokenExpiration
	}
	if input.RefreshTokenExpiration != nil {
		cl.RefreshTokenExpiration = *input.RefreshTokenExpiration
	}
	if input.AllowMultipleTokens != nil {
		cl.AllowMultipleTokens = *input.AllowMultipleTokens
	}
	if input.SkipConsentScreen != nil {
		cl.SkipConsentScreen = *input.SkipConsentScreen
	}
	if input.IsActive != nil {
		cl.IsActive = *input.IsActive
	}
	cl.UpdatedAt = time.Now()
	const q = `
		UPDATE oauth2_clients SET
			name = $1, description = $2, client_type = $3,
			token_endpoint_auth_method = $4,
			allowed_scopes = $5, allowed_redirect_uris = $6, allowed_origins = $7,
			token_expiration = $8, refresh_token_expiration = $9,
			allow_multiple_tokens = $10, skip_consent_screen = $11,
			is_active = $12, updated_at = $13
		WHERE id = $14`
	_, err = s.store.Pool().Exec(c, q,
		cl.Name, cl.Description, string(cl.ClientType),
		string(cl.TokenEndpointAuthMethod),
		cl.AllowedScopes, cl.AllowedRedirectURIs, cl.AllowedOrigins,
		cl.TokenExpiration, cl.RefreshTokenExpiration,
		cl.AllowMultipleTokens, cl.SkipConsentScreen,
		cl.IsActive, cl.UpdatedAt, id,
	)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func (s *adminOAuth2ClientSvc) DeleteClient(ctx interface{}, id uuid.UUID) error {
	_, err := s.store.Pool().Exec(ctx.(context.Context), `DELETE FROM oauth2_clients WHERE id = $1`, id)
	return err
}

// ── adminBranchSvc ────────────────────────────────────────────────────────────
// Branch-types CRUD via BranchTypeRepository. Branch/UserBranch methods are
// stubs (RegisterBranchTypeRoutes only invokes branch-type methods).

type adminBranchSvc struct{ store *postgres.Store }

func newAdminBranchSvc(s *postgres.Store) *adminBranchSvc { return &adminBranchSvc{store: s} }

func (s *adminBranchSvc) CreateBranchType(ctx interface{}, orgID uuid.UUID, params adminpkg.BranchTypeParams) (*domain.BranchType, error) {
	c := ctx.(context.Context)
	if existing, _ := s.store.BranchTypeRepository.GetByCode(c, orgID, params.Code); existing != nil {
		return nil, fmt.Errorf("branch type code already exists")
	}
	bt := &domain.BranchType{
		ID:              uuid.New(),
		OrgID:           orgID,
		Code:            params.Code,
		Label:           params.Label,
		ShortCode:       params.ShortCode,
		Level:           params.Level,
		IsOperational:   params.IsOperational,
		CanHaveChildren: params.CanHaveChildren,
		SortOrder:       params.SortOrder,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	return bt, s.store.BranchTypeRepository.Create(c, bt)
}

func (s *adminBranchSvc) GetBranchType(ctx interface{}, id, orgID uuid.UUID) (*domain.BranchType, error) {
	return s.store.BranchTypeRepository.GetByID(ctx.(context.Context), id, orgID)
}

func (s *adminBranchSvc) ListBranchTypes(ctx interface{}, orgID uuid.UUID) ([]*domain.BranchType, error) {
	return s.store.BranchTypeRepository.ListByOrg(ctx.(context.Context), orgID)
}

func (s *adminBranchSvc) UpdateBranchType(ctx interface{}, id, orgID uuid.UUID, params adminpkg.BranchTypeParams) (*domain.BranchType, error) {
	c := ctx.(context.Context)
	bt, err := s.store.BranchTypeRepository.GetByID(c, id, orgID)
	if err != nil {
		return nil, err
	}
	bt.Code = params.Code
	bt.Label = params.Label
	bt.ShortCode = params.ShortCode
	bt.Level = params.Level
	bt.IsOperational = params.IsOperational
	bt.CanHaveChildren = params.CanHaveChildren
	bt.SortOrder = params.SortOrder
	bt.UpdatedAt = time.Now()
	return bt, s.store.BranchTypeRepository.Update(c, bt)
}

func (s *adminBranchSvc) DeleteBranchType(ctx interface{}, id, orgID uuid.UUID) error {
	return s.store.BranchTypeRepository.Delete(ctx.(context.Context), id, orgID)
}

// Branch and UserBranch methods are unsupported (only branch-type routes registered).

func (s *adminBranchSvc) CreateBranch(ctx interface{}, orgID uuid.UUID, params adminpkg.BranchParams) (*domain.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) GetBranch(ctx interface{}, id, orgID uuid.UUID) (*domain.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) ListBranches(ctx interface{}, orgID uuid.UUID) ([]*domain.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) UpdateBranch(ctx interface{}, id, orgID uuid.UUID, params adminpkg.BranchParams) (*domain.Branch, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) DeleteBranch(ctx interface{}, id, orgID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) AssignUserToBranch(ctx interface{}, userID, branchID, orgID uuid.UUID, params adminpkg.UserBranchParams) (*domain.UserBranchAssignment, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) GetUserBranches(ctx interface{}, userID uuid.UUID) ([]*domain.UserBranchAssignment, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *adminBranchSvc) RevokeUserBranch(ctx interface{}, assignmentID, orgID, revokedBy uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
