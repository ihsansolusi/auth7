package policy7client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchOperationalHours_WireContract(t *testing.T) {
	const param = "teller_operating_hours"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exact endpoint + headers policy7's effective-parameter contract expects.
		require.Equal(t, "/v1/params/operational_hours/"+param+"/effective", r.URL.Path)
		require.Equal(t, "auth7", r.Header.Get("X-Service-ID"))
		require.Equal(t, "secret-key", r.Header.Get("X-API-Key"))
		require.Equal(t, "org-123", r.Header.Get("X-Org-ID"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"p1","org_id":"org-123","category":"operational_hours","name":"` + param + `","value":{"timezone":"WIB","weekday":{"open":"08:00","close":"16:00"}},"version":2}}`))
	}))
	defer srv.Close()

	f := New(srv.URL, "auth7", "secret-key", param)

	raw, err := f.FetchOperationalHours(context.Background(), "org-123", "teller", "branch-9")
	require.NoError(t, err)
	require.JSONEq(t, `{"timezone":"WIB","weekday":{"open":"08:00","close":"16:00"}}`, string(raw))
}

func TestFetchOperationalHours_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := New(srv.URL, "auth7", "k", "teller_operating_hours")
	_, err := f.FetchOperationalHours(context.Background(), "org-123", "", "")
	require.Error(t, err)
}
