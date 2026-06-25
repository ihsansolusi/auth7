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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
