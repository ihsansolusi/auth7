package authz

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/opacache"
)

// mockFetcher is a stub OperationalHoursFetcher that returns canned JSON and
// counts how many times it was invoked (to assert cache fetch-through).
type mockFetcher struct {
	raw   json.RawMessage
	err   error
	calls int
}

func (m *mockFetcher) FetchOperationalHours(_ context.Context, _, _, _ string) (json.RawMessage, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.raw, nil
}

const aggregateHours = `{
	"timezone": "WIB",
	"weekday":  {"open": "08:00", "close": "16:00"},
	"saturday": {"open": "08:00", "close": "12:00"},
	"sunday":   null
}`

// jakarta returns a time at the given wall-clock in Asia/Jakarta (WIB).
func jakarta(t *testing.T, year int, month time.Month, day, hour, min int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	require.NoError(t, err)
	return time.Date(year, month, day, hour, min, 0, 0, loc)
}

func TestOperationalHours_IsOpenAt(t *testing.T) {
	oh, err := parseOperationalHours(json.RawMessage(aggregateHours))
	require.NoError(t, err)

	// 2026-06-22 is a Monday.
	tests := []struct {
		name     string
		when     time.Time
		wantOpen bool
	}{
		{"weekday inside", jakarta(t, 2026, 6, 22, 10, 0), true},
		{"weekday at open boundary", jakarta(t, 2026, 6, 22, 8, 0), true},
		{"weekday before open", jakarta(t, 2026, 6, 22, 7, 59), false},
		{"weekday at close boundary (exclusive)", jakarta(t, 2026, 6, 22, 16, 0), false},
		{"weekday after close", jakarta(t, 2026, 6, 22, 18, 30), false},
		{"saturday inside", jakarta(t, 2026, 6, 27, 9, 0), true},
		{"saturday after short close", jakarta(t, 2026, 6, 27, 13, 0), false},
		{"sunday always closed", jakarta(t, 2026, 6, 28, 10, 0), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			open, reason := oh.IsOpenAt(tc.when)
			require.Equal(t, tc.wantOpen, open, reason)
		})
	}
}

func TestOperationalHours_Timezone(t *testing.T) {
	oh, err := parseOperationalHours(json.RawMessage(aggregateHours))
	require.NoError(t, err)

	// 03:00 UTC on Monday == 10:00 WIB (inside 08:00-16:00).
	utc := time.Date(2026, 6, 22, 3, 0, 0, 0, time.UTC)
	loc, _ := resolveLocation("WIB", "Asia/Jakarta")
	open, reason := oh.IsOpenAt(utc.In(loc))
	require.True(t, open, reason)

	// 10:00 UTC on Monday == 17:00 WIB (outside).
	utc = time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	open, reason = oh.IsOpenAt(utc.In(loc))
	require.False(t, open, reason)
}

func TestOperationalHours_FlatFallback(t *testing.T) {
	flat := `{"timezone":"WIB","is_open":true,"open_time":"09:00","close_time":"14:00"}`
	oh, err := parseOperationalHours(json.RawMessage(flat))
	require.NoError(t, err)

	open, _ := oh.IsOpenAt(jakarta(t, 2026, 6, 22, 10, 0))
	require.True(t, open)
	open, _ = oh.IsOpenAt(jakarta(t, 2026, 6, 22, 15, 0))
	require.False(t, open)

	// is_open=false => closed regardless of window.
	closed := `{"is_open":false,"open_time":"09:00","close_time":"14:00"}`
	oh, err = parseOperationalHours(json.RawMessage(closed))
	require.NoError(t, err)
	open, _ = oh.IsOpenAt(jakarta(t, 2026, 6, 22, 10, 0))
	require.False(t, open)
}

func TestOperationalHours_OvernightWindow(t *testing.T) {
	overnight := `{"timezone":"WIB","weekday":{"open":"22:00","close":"02:00"}}`
	oh, err := parseOperationalHours(json.RawMessage(overnight))
	require.NoError(t, err)

	open, _ := oh.IsOpenAt(jakarta(t, 2026, 6, 22, 23, 30))
	require.True(t, open)
	open, _ = oh.IsOpenAt(jakarta(t, 2026, 6, 22, 1, 0))
	require.True(t, open)
	open, _ = oh.IsOpenAt(jakarta(t, 2026, 6, 22, 12, 0))
	require.False(t, open)
}

// newTestEvaluator builds an evaluator backed by a real opacache and a fixed
// clock at the given instant.
func newTestEvaluator(fetcher OperationalHoursFetcher, now time.Time, failOpen bool) (*TimeWindowEvaluator, *opacache.Cache) {
	cache := opacache.NewCache(time.Minute, zerolog.Nop())
	e := NewTimeWindowEvaluator(cache, fetcher, "Asia/Jakarta", failOpen, zerolog.Nop())
	e.now = func() time.Time { return now }
	return e, cache
}

func testAuthCtx() *domain.AuthContext {
	return &domain.AuthContext{
		UserID:   uuid.New(),
		OrgID:    uuid.New(),
		BranchID: uuid.New(),
		Roles:    []string{"teller"},
	}
}

func TestTimeWindowEvaluator_AllowAndDeny(t *testing.T) {
	authCtx := testAuthCtx()

	// Inside hours -> allowed.
	f := &mockFetcher{raw: json.RawMessage(aggregateHours)}
	e, _ := newTestEvaluator(f, jakarta(t, 2026, 6, 22, 10, 0), true)
	res, err := e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.True(t, res.Allowed, res.Reason)

	// Outside hours -> denied.
	f = &mockFetcher{raw: json.RawMessage(aggregateHours)}
	e, _ = newTestEvaluator(f, jakarta(t, 2026, 6, 22, 19, 0), true)
	res, err = e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.False(t, res.Allowed, res.Reason)
}

func TestTimeWindowEvaluator_FetchThroughAndInvalidation(t *testing.T) {
	authCtx := testAuthCtx()
	f := &mockFetcher{raw: json.RawMessage(aggregateHours)}
	e, cache := newTestEvaluator(f, jakarta(t, 2026, 6, 22, 10, 0), true)

	_, err := e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.Equal(t, 1, f.calls)

	// Second evaluation hits the cache; fetcher not called again.
	_, err = e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.Equal(t, 1, f.calls, "operational_hours should be served from cache")

	// A policy7.params.updated event invalidates by prefix; next eval re-fetches.
	cache.InvalidateByPrefix("opa:" + authCtx.OrgID.String() + ":operational_hours")
	_, err = e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.Equal(t, 2, f.calls, "invalidation must trigger a fresh policy7 fetch")
}

func TestTimeWindowEvaluator_FailOpenAndFailClosed(t *testing.T) {
	authCtx := testAuthCtx()
	now := jakarta(t, 2026, 6, 22, 10, 0)

	// Fail-open: fetch error -> allowed.
	f := &mockFetcher{err: errors.New("policy7 down")}
	e, _ := newTestEvaluator(f, now, true)
	res, err := e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.True(t, res.Allowed, "fail-open should allow on fetch error")

	// Fail-closed: fetch error -> denied.
	f = &mockFetcher{err: errors.New("policy7 down")}
	e, _ = newTestEvaluator(f, now, false)
	res, err = e.Evaluate(context.Background(), authCtx)
	require.NoError(t, err)
	require.False(t, res.Allowed, "fail-closed should deny on fetch error")
}

// TestCheckDataAccess_TimeGate verifies the time-gate is wired into the authz
// decision path: a time-gated permission is denied outside hours and allowed
// within, while a non-gated permission is unaffected.
func TestCheckDataAccess_TimeGate(t *testing.T) {
	gated := "transaction:create"
	authCtx := &domain.AuthContext{
		UserID:      uuid.New(),
		OrgID:       uuid.New(),
		BranchID:    uuid.New(),
		Permissions: []string{gated, "report:view"},
		BranchScope: domain.BranchScopeAll,
	}

	// Outside hours -> deny.
	f := &mockFetcher{raw: json.RawMessage(aggregateHours)}
	twDeny, _ := newTestEvaluator(f, jakarta(t, 2026, 6, 22, 20, 0), false)
	checker := NewPermissionChecker(nil, nil, nil).WithTimeGate(twDeny, []string{gated})

	res, err := checker.CheckDataAccess(context.Background(), authCtx, gated, "transaction", uuid.New())
	require.NoError(t, err)
	require.False(t, res.Allowed, "time-gated action must be denied outside hours")

	// Inside hours -> allow.
	f = &mockFetcher{raw: json.RawMessage(aggregateHours)}
	twAllow, _ := newTestEvaluator(f, jakarta(t, 2026, 6, 22, 10, 0), false)
	checker = NewPermissionChecker(nil, nil, nil).WithTimeGate(twAllow, []string{gated})

	res, err = checker.CheckDataAccess(context.Background(), authCtx, gated, "transaction", uuid.New())
	require.NoError(t, err)
	require.True(t, res.Allowed, "time-gated action must be allowed within hours")

	// Non-gated permission: time-gate is skipped (fetcher never consulted).
	f = &mockFetcher{raw: json.RawMessage(aggregateHours)}
	twSkip, _ := newTestEvaluator(f, jakarta(t, 2026, 6, 22, 20, 0), false)
	checker = NewPermissionChecker(nil, nil, nil).WithTimeGate(twSkip, []string{gated})

	res, err = checker.CheckDataAccess(context.Background(), authCtx, "report:view", "report", uuid.New())
	require.NoError(t, err)
	require.True(t, res.Allowed, "non-gated permission must not be time-gated")
	require.Equal(t, 0, f.calls, "non-gated permission must not fetch operational_hours")
}
