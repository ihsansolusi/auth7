// Package policy7client is a thin HTTP client for the policy7 service, used by
// auth7 to consume policy7-owned parameters (currently operational_hours) as
// ABAC context.
//
// Why not import github.com/ihsansolusi/policy7/pkg/client directly? That SDK
// package only needs net/http + google/uuid, but its *module* go.mod requires
// lib7-service-go v0.12.2 (auth7 is pinned to v0.5.0). Adding the module would
// MVS-force a 7-minor-version bump of lib7 across all of auth7 — a destabilizing
// change unrelated to this feature. So we replicate the SDK's exact wire
// contract here instead: same endpoint shape (GET
// /v1/params/{category}/{name}/effective), same headers (X-Service-ID,
// X-API-Key, X-Org-ID), same {"data": ...} envelope. Behaviour is identical;
// the dependency boundary stays clean.
package policy7client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// categoryOperationalHours is the policy7 parameter category that holds branch /
// module operating windows.
const categoryOperationalHours = "operational_hours"

// Fetcher fetches policy7 parameters over HTTP. It implements
// authz.OperationalHoursFetcher.
type Fetcher struct {
	baseURL    string
	serviceID  string
	apiKey     string
	paramName  string
	httpClient *http.Client
}

// New builds a Fetcher from the policy7 base URL, M2M credentials, and the
// operational_hours parameter name to resolve.
func New(baseURL, serviceID, apiKey, paramName string) *Fetcher {
	return &Fetcher{
		baseURL:    baseURL,
		serviceID:  serviceID,
		apiKey:     apiKey,
		paramName:  paramName,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchOperationalHours resolves the effective operational_hours parameter for
// the given scope and returns its raw JSON value.
//
// policy7's effective-parameter endpoint resolves user->role->branch->global by
// org context server-side; the current contract only carries org_id (via
// X-Org-ID), so roleID and branchID are accepted for forward-compatibility and
// to document caller intent, but are not yet transmitted. The opacache key still
// scopes by branch so the cache stays correct once policy7 gains role/branch
// resolution on this endpoint.
func (f *Fetcher) FetchOperationalHours(ctx context.Context, orgID, roleID, branchID string) (json.RawMessage, error) {
	const op = "policy7client.Fetcher.FetchOperationalHours"

	path := fmt.Sprintf("%s/v1/params/%s/%s/effective", f.baseURL, categoryOperationalHours, f.paramName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", op, err)
	}
	req.Header.Set("X-Service-ID", f.serviceID)
	req.Header.Set("X-API-Key", f.apiKey)
	req.Header.Set("X-Org-ID", orgID)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", op, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: unexpected status code: %d", op, resp.StatusCode)
	}

	// policy7 wraps responses in {"data": {...parameter...}}; the parameter's
	// operating-window payload lives in the "value" field.
	var env struct {
		Data struct {
			Value json.RawMessage `json:"value"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("%s: decode envelope: %w", op, err)
	}
	if len(env.Data.Value) == 0 {
		return nil, fmt.Errorf("%s: empty parameter value", op)
	}
	return env.Data.Value, nil
}
