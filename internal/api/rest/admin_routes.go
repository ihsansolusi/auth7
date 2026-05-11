package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/api/middleware"
	adminpkg "github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	jwtpkg "github.com/ihsansolusi/auth7/internal/service/jwt"
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
	adminpkg.NewAuditHandler(auditSvc, s.deps.Logger).RegisterRoutes(adminV1)

	adminV1.GET("/dashboard/stats", s.handleAdminStats(store))
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
