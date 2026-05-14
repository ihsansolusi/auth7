package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ihsansolusi/auth7/internal/api/rest/admin"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/rs/zerolog"
)

// mockSessionSvc is a test double for admin.AdminSessionService.
type mockSessionSvc struct {
	sessions []*session.SessionData
	revoked  []string
	getErr   error
	listErr  error
	revokeErr error
}

func (m *mockSessionSvc) ListAllSessions(_ context.Context) ([]*session.SessionData, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.sessions, nil
}

func (m *mockSessionSvc) GetSession(_ context.Context, id string) (*session.SessionData, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSessionSvc) RevokeSession(_ context.Context, id string) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	m.revoked = append(m.revoked, id)
	return nil
}

func buildSessionRouter(svc admin.AdminSessionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	auditSvc := audit.NewService(nil)
	h := admin.NewSessionHandler(svc, auditSvc, zerolog.Nop())

	r.Use(func(c *gin.Context) {
		c.Set("claims", &mockClaims{
			subject: "00000000-0000-0000-0000-000000000001",
			email:   "admin@test.com",
			orgID:   "00000000-0000-0000-0000-000000000002",
			roles:   []string{"admin"},
		})
		c.Next()
	})

	v1 := r.Group("/admin/v1")
	h.RegisterRoutes(v1)
	return r
}

func makeTestSession(id, userID string) *session.SessionData {
	now := time.Now().Unix()
	return &session.SessionData{
		ID:         id,
		UserID:     userID,
		OrgID:      "00000000-0000-0000-0000-000000000002",
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
		CreatedAt:  now,
		ExpiresAt:  now + 28800,
		LastUsedAt: now,
	}
}

// ── GET /admin/v1/sessions ────────────────────────────────────────────────────

func TestListSessions_Empty(t *testing.T) {
	svc := &mockSessionSvc{}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/admin/v1/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
}

func TestListSessions_DefaultPage(t *testing.T) {
	s1 := makeTestSession("sess-1", "user-1")
	s2 := makeTestSession("sess-2", "user-2")
	svc := &mockSessionSvc{sessions: []*session.SessionData{s1, s2}}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/admin/v1/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total"].(float64) != 2 {
		t.Errorf("expected total=2, got %v", resp["total"])
	}
	sessions := resp["sessions"].([]interface{})
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	first := sessions[0].(map[string]interface{})
	if first["session_id"] != "sess-1" {
		t.Errorf("expected sess-1, got %v", first["session_id"])
	}
	if first["user_id"] != "user-1" {
		t.Errorf("expected user-1, got %v", first["user_id"])
	}
	if first["created_at"] == "" || first["expires_at"] == "" {
		t.Error("expected non-empty created_at and expires_at")
	}
}

func TestListSessions_Pagination(t *testing.T) {
	sessions := make([]*session.SessionData, 5)
	for i := range sessions {
		sessions[i] = makeTestSession("sess-"+string(rune('A'+i)), "user")
	}
	svc := &mockSessionSvc{sessions: sessions}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/admin/v1/sessions?page=2&page_size=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total"].(float64) != 5 {
		t.Errorf("expected total=5, got %v", resp["total"])
	}
	items := resp["sessions"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 sessions on page 2, got %d", len(items))
	}
	if resp["page"].(float64) != 2 {
		t.Errorf("expected page=2, got %v", resp["page"])
	}
}

func TestListSessions_PageBeyondEnd(t *testing.T) {
	s1 := makeTestSession("sess-1", "user-1")
	svc := &mockSessionSvc{sessions: []*session.SessionData{s1}}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/admin/v1/sessions?page=99&page_size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["sessions"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 sessions for page beyond end, got %d", len(items))
	}
}

func TestListSessions_ServiceError(t *testing.T) {
	svc := &mockSessionSvc{listErr: context.DeadlineExceeded}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/admin/v1/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ── DELETE /admin/v1/sessions/:id ────────────────────────────────────────────

func TestRevokeSession_Success(t *testing.T) {
	sess := makeTestSession("sess-abc", "user-xyz")
	svc := &mockSessionSvc{sessions: []*session.SessionData{sess}}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/admin/v1/sessions/sess-abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["revoked"] != true {
		t.Errorf("expected revoked=true, got %v", resp["revoked"])
	}
	if len(svc.revoked) != 1 || svc.revoked[0] != "sess-abc" {
		t.Errorf("expected sess-abc to be revoked, got %v", svc.revoked)
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	svc := &mockSessionSvc{sessions: []*session.SessionData{}}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/admin/v1/sessions/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRevokeSession_GetError(t *testing.T) {
	svc := &mockSessionSvc{getErr: context.DeadlineExceeded}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/admin/v1/sessions/sess-abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRevokeSession_RevokeError(t *testing.T) {
	sess := makeTestSession("sess-abc", "user-xyz")
	svc := &mockSessionSvc{
		sessions:  []*session.SessionData{sess},
		revokeErr: context.DeadlineExceeded,
	}
	r := buildSessionRouter(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/admin/v1/sessions/sess-abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
