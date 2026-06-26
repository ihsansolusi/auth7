package admin

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5"
)

func TestRespondError_StatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"not_found", domain.ErrNotFound, http.StatusNotFound, "not_found"},
		{"no_rows", pgx.ErrNoRows, http.StatusNotFound, "not_found"},
		{"wrapped_not_found", fmt.Errorf("admin.GetUser: %w", domain.ErrNotFound), http.StatusNotFound, "not_found"},
		{"already_exists", domain.ErrAlreadyExists, http.StatusConflict, "already_exists"},
		{"permission_denied", domain.ErrPermissionDenied, http.StatusForbidden, "forbidden"},
		{"generic", errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			respondError(c, tc.err)

			if w.Code != tc.wantCode {
				t.Fatalf("expected status %d, got %d", tc.wantCode, w.Code)
			}
			if want := `"error":"` + tc.wantBody + `"`; !contains(w.Body.String(), want) {
				t.Errorf("expected body to contain %s, got %s", want, w.Body.String())
			}
		})
	}
}

type fakeOrgClaims struct{ org string }

func (f fakeOrgClaims) GetOrgID() string { return f.org }

func TestRequireOrgID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const claimOrg = "11111111-1111-1111-1111-111111111111"
	const otherOrg = "22222222-2222-2222-2222-222222222222"

	newCtx := func(query, claimOrgVal string, hasClaim bool) (*gin.Context, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+query, nil)
		if hasClaim {
			c.Set("claims", fakeOrgClaims{org: claimOrgVal})
		}
		return c, w
	}

	t.Run("claim is authoritative (query ignored)", func(t *testing.T) {
		c, _ := newCtx("?org_id="+otherOrg, claimOrg, true)
		got, ok := requireOrgID(c)
		if !ok || got.String() != claimOrg {
			t.Fatalf("expected %s ok=true, got %s ok=%v", claimOrg, got, ok)
		}
	})

	t.Run("empty claim falls back to query (super_admin)", func(t *testing.T) {
		c, _ := newCtx("?org_id="+otherOrg, "", true)
		got, ok := requireOrgID(c)
		if !ok || got.String() != otherOrg {
			t.Fatalf("expected %s ok=true, got %s ok=%v", otherOrg, got, ok)
		}
	})

	t.Run("no claim + no query → 400", func(t *testing.T) {
		c, w := newCtx("", "", false)
		if _, ok := requireOrgID(c); ok || w.Code != http.StatusBadRequest {
			t.Fatalf("expected ok=false + 400, got ok=%v code=%d", ok, w.Code)
		}
	})

	t.Run("invalid org_id → 400", func(t *testing.T) {
		c, w := newCtx("?org_id=not-a-uuid", "", true)
		if _, ok := requireOrgID(c); ok || w.Code != http.StatusBadRequest {
			t.Fatalf("expected ok=false + 400, got ok=%v code=%d", ok, w.Code)
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
