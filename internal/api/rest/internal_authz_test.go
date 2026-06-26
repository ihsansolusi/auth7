package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/rs/zerolog"
)

type fakeUserRoles struct{ roles []*domain.UserRole }

func (f fakeUserRoles) GetUserRoles(_ interface{}, _ uuid.UUID) ([]*domain.UserRole, error) {
	return f.roles, nil
}

type fakeRolePerms struct{ byRole map[uuid.UUID][]*domain.Permission }

func (f fakeRolePerms) GetPermissions(_ interface{}, roleID uuid.UUID) ([]*domain.Permission, error) {
	return f.byRole[roleID], nil
}

func buildPDPRouter(userRoles []*domain.UserRole, perms map[uuid.UUID][]*domain.Permission) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// No time-gate (tw=nil) → role-based decision only; time-gate is covered by
	// the authz package's own tests.
	checker := newTimeGatedChecker(nil, nil)
	h := newAuthzPDPHandler(fakeUserRoles{roles: userRoles}, fakeRolePerms{byRole: perms}, checker, zerolog.Nop())
	g := r.Group("/internal/v1")
	h.registerRoutes(g)
	return r
}

func postCheck(t *testing.T, r *gin.Engine, body map[string]string) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/internal/v1/authz/check", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func TestPDPCheck_RoleBased(t *testing.T) {
	org := uuid.New()
	branch := uuid.New()
	user := uuid.New()
	roleID := uuid.New()

	userRoles := []*domain.UserRole{
		{RoleID: roleID, OrgID: org, BranchID: nil}, // org-wide, active (RevokedAt nil)
	}
	perms := map[uuid.UUID][]*domain.Permission{
		roleID: {{Code: "report:view"}},
	}
	r := buildPDPRouter(userRoles, perms)

	base := map[string]string{"user_id": user.String(), "org_id": org.String(), "branch_id": branch.String()}

	t.Run("granted permission → allowed", func(t *testing.T) {
		body := map[string]string{}
		for k, v := range base {
			body[k] = v
		}
		body["permission"] = "report:view"
		code, resp := postCheck(t, r, body)
		if code != http.StatusOK || resp["allowed"] != true {
			t.Fatalf("expected 200 allowed=true, got %d %v", code, resp)
		}
	})

	t.Run("ungranted permission → denied", func(t *testing.T) {
		body := map[string]string{}
		for k, v := range base {
			body[k] = v
		}
		body["permission"] = "transaction:create"
		code, resp := postCheck(t, r, body)
		if code != http.StatusOK || resp["allowed"] != false {
			t.Fatalf("expected 200 allowed=false, got %d %v", code, resp)
		}
	})

	t.Run("invalid user_id → 400", func(t *testing.T) {
		code, _ := postCheck(t, r, map[string]string{"user_id": "nope", "org_id": org.String(), "branch_id": branch.String(), "permission": "x"})
		if code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", code)
		}
	})
}

func TestPDPCheck_BranchScopedRoleFiltered(t *testing.T) {
	org := uuid.New()
	branch := uuid.New()
	otherBranch := uuid.New()
	user := uuid.New()
	roleID := uuid.New()

	// role assignment scoped to a DIFFERENT branch → must not grant for `branch`.
	userRoles := []*domain.UserRole{
		{RoleID: roleID, OrgID: org, BranchID: &otherBranch},
	}
	perms := map[uuid.UUID][]*domain.Permission{roleID: {{Code: "report:view"}}}
	r := buildPDPRouter(userRoles, perms)

	code, resp := postCheck(t, r, map[string]string{
		"user_id": user.String(), "org_id": org.String(), "branch_id": branch.String(), "permission": "report:view",
	})
	if code != http.StatusOK || resp["allowed"] != false {
		t.Fatalf("expected denied (role scoped to other branch), got %d %v", code, resp)
	}
}
