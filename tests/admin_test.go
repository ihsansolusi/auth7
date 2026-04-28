package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/api/middleware"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/rs/zerolog"
)

type mockAuditStore struct {
	logs []*domain.AuditLog
}

func (m *mockAuditStore) Create(ctx context.Context, log *domain.AuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditStore) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error) {
	return m.logs, len(m.logs), nil
}

type mockClaims struct {
	subject string
	email   string
	orgID   string
	roles   []string
}

func (c *mockClaims) GetSubject() string { return c.subject }
func (c *mockClaims) GetEmail() string   { return c.email }
func (c *mockClaims) GetOrgID() string  { return c.orgID }
func (c *mockClaims) GetRoles() []string { return c.roles }

func setupTestRouter(auditLogger *audit.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.AdminAuth(
		middleware.DefaultAdminAuthConfig(),
		auditLogger,
		zerolog.Nop(),
	))
	adminGroup.Use(middleware.LogAdminAction(auditLogger, zerolog.Nop()))

	adminGroup.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"users": []interface{}{}})
	})

	return r
}

func TestAdminAuthMiddleware_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	auditLogger := audit.NewService(nil)
	cfg := middleware.DefaultAdminAuthConfig()

	r.GET("/admin/test", middleware.AdminAuth(cfg, auditLogger, zerolog.Nop()), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAdminAuthMiddleware_ValidAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	auditLogger := audit.NewService(nil)
	cfg := middleware.DefaultAdminAuthConfig()

	r.Use(func(c *gin.Context) {
		c.Set("claims", &mockClaims{
			subject: uuid.New().String(),
			email:   "admin@test.com",
			orgID:   uuid.New().String(),
			roles:   []string{"admin"},
		})
		c.Next()
	})

	r.GET("/admin/test", middleware.AdminAuth(cfg, auditLogger, zerolog.Nop()), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAdminAuthMiddleware_NonAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	auditLogger := audit.NewService(nil)
	cfg := middleware.DefaultAdminAuthConfig()

	r.Use(func(c *gin.Context) {
		c.Set("claims", &mockClaims{
			subject: uuid.New().String(),
			email:   "user@test.com",
			orgID:   uuid.New().String(),
			roles:   []string{"user"},
		})
		c.Next()
	})

	r.GET("/admin/test", middleware.AdminAuth(cfg, auditLogger, zerolog.Nop()), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestAuditLogService_Log(t *testing.T) {
	store := &mockAuditStore{}
	svc := audit.NewService(store)

	input := audit.LogInput{
		OrgID:        uuid.New(),
		ActorID:      uuid.New(),
		ActorEmail:   "admin@test.com",
		Action:       "create_user",
		ResourceType: "user",
		ResourceID:   uuid.New().String(),
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
	}

	err := svc.Log(context.Background(), input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(store.logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(store.logs))
	}

	if store.logs[0].Action != "create_user" {
		t.Errorf("expected action 'create_user', got '%s'", store.logs[0].Action)
	}
}

func TestAuditLogService_LogAsync(t *testing.T) {
	store := &mockAuditStore{}
	svc := audit.NewService(store)

	input := audit.LogInput{
		OrgID:        uuid.New(),
		ActorID:      uuid.New(),
		ActorEmail:   "admin@test.com",
		Action:       "update_user",
		ResourceType: "user",
	}

	svc.LogAsync(input)

	time.Sleep(100 * time.Millisecond)

	if len(store.logs) != 1 {
		t.Errorf("expected 1 log after async, got %d", len(store.logs))
	}
}

func TestAuditLogService_Query(t *testing.T) {
	store := &mockAuditStore{}
	svc := audit.NewService(store)

	orgID := uuid.New()
	log1 := &domain.AuditLog{
		ID:           uuid.New(),
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Action:       "create_user",
		ResourceType: "user",
		CreatedAt:    time.Now(),
	}
	log2 := &domain.AuditLog{
		ID:           uuid.New(),
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Action:       "delete_user",
		ResourceType: "user",
		CreatedAt:    time.Now(),
	}
	store.logs = []*domain.AuditLog{log1, log2}

	filter := domain.AuditLogFilter{
		OrgID: &orgID,
		Limit: 10,
	}

	logs, total, err := svc.Query(context.Background(), filter)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAuditLogFilter_DefaultLimit(t *testing.T) {
	store := &mockAuditStore{}
	svc := audit.NewService(store)

	store.logs = make([]*domain.AuditLog, 120)
	for i := range store.logs {
		store.logs[i] = &domain.AuditLog{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}
	}

	filter := domain.AuditLogFilter{}
	logs, _, err := svc.Query(context.Background(), filter)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(logs) != 120 {
		t.Errorf("expected 120 logs from mock store, got %d", len(logs))
	}
}

func TestRoleValidation(t *testing.T) {
	role := &domain.Role{
		ID:        uuid.New(),
		OrgID:    uuid.New(),
		Name:     "admin",
		IsDefault: false,
	}

	err := role.Validate()
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestRoleValidation_EmptyName(t *testing.T) {
	role := &domain.Role{
		ID:        uuid.New(),
		OrgID:    uuid.New(),
		Name:     "",
	}

	err := role.Validate()
	if err == nil {
		t.Error("expected validation error for empty name")
	}
}

func TestPermissionValidation(t *testing.T) {
	perm := &domain.Permission{
		ID:           uuid.New(),
		Code:        "users:read",
		Name:        "Read Users",
		ResourceType: "users",
	}

	err := perm.Validate()
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestUserRole_IsActive(t *testing.T) {
	ur := &domain.UserRole{
		ID:         uuid.New(),
		RevokedAt:  nil,
	}

	if !ur.IsActive() {
		t.Error("expected user role to be active when RevokedAt is nil")
	}

	now := time.Now()
	ur.RevokedAt = &now

	if ur.IsActive() {
		t.Error("expected user role to be inactive when RevokedAt is set")
	}
}

func TestAuditLog_JSON(t *testing.T) {
	log := &domain.AuditLog{
		ID:           uuid.New(),
		OrgID:        uuid.New(),
		ActorID:      uuid.New(),
		ActorEmail:   "admin@test.com",
		Action:       "create_user",
		ResourceType: "user",
		ResourceID:   "123",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Errorf("unexpected error marshaling: %v", err)
	}

	if !strings.Contains(string(data), "admin@test.com") {
		t.Error("expected JSON to contain actor email")
	}

	if !strings.Contains(string(data), "create_user") {
		t.Error("expected JSON to contain action")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := middleware.NewAdminRateLimiter(5, 10)

	for i := 0; i < 10; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("request %d should be allowed (within burst limit)", i+1)
		}
	}

	if limiter.Allow("test-key") {
		t.Error("request 11 should be rate limited (burst exceeded)")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := middleware.NewAdminRateLimiter(2, 5)

	for i := 0; i < 5; i++ {
		if !limiter.Allow("key1") {
			t.Errorf("key1 request %d should be allowed (within burst)", i+1)
		}
	}

	if limiter.Allow("key1") {
		t.Error("key1 request 6 should be rate limited (burst exceeded)")
	}

	if !limiter.Allow("key2") {
		t.Error("key2 request 1 should be allowed (different key)")
	}
}

func TestRateLimiter_Burst(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := middleware.NewAdminRateLimiter(2, 3)

	if !limiter.Allow("key") {
		t.Error("request 1 should be allowed")
	}
	if !limiter.Allow("key") {
		t.Error("request 2 should be allowed")
	}
	if !limiter.Allow("key") {
		t.Error("request 3 (burst) should be allowed")
	}
	if limiter.Allow("key") {
		t.Error("request 4 should be rate limited")
	}
}
