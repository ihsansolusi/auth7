package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/opacache"
)

// OperationalHoursFetcher resolves the effective operational_hours parameter
// value (raw JSON) for a scope from policy7. Implemented by
// internal/policy7client.Fetcher; an interface here keeps authz testable without
// the HTTP client.
type OperationalHoursFetcher interface {
	FetchOperationalHours(ctx context.Context, orgID, roleID, branchID string) (json.RawMessage, error)
}

// dayWindow is a single open/close interval (HH:MM, 24h) for a day.
type dayWindow struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// OperationalHours is the parsed policy7 operational_hours parameter value.
//
// Two value shapes are supported:
//
//	Aggregate weekly schedule (recommended, e.g. param "teller_operating_hours"):
//	  {"weekday":{"open":"08:00","close":"15:00"},
//	   "saturday":{"open":"08:00","close":"12:00"},
//	   "sunday":null, "timezone":"WIB"}
//
//	Flat single-window fallback (per-day params, applied to every day):
//	  {"is_open":true,"open_time":"08:00","close_time":"15:00","timezone":"WIB"}
//
// A nil/absent window for the current day means "closed" (deny). Timezone may be
// an IANA name (Asia/Jakarta) or an Indonesian abbreviation (WIB/WITA/WIT);
// empty falls back to the evaluator's default.
type OperationalHours struct {
	Timezone string `json:"timezone"`

	// Aggregate weekly shape.
	Weekday  *dayWindow `json:"weekday"`
	Saturday *dayWindow `json:"saturday"`
	Sunday   *dayWindow `json:"sunday"`

	// Flat single-window fallback.
	IsOpen    *bool  `json:"is_open"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// parseOperationalHours unmarshals a raw policy7 parameter value.
func parseOperationalHours(raw json.RawMessage) (*OperationalHours, error) {
	var oh OperationalHours
	if err := json.Unmarshal(raw, &oh); err != nil {
		return nil, fmt.Errorf("parse operational_hours: %w", err)
	}
	return &oh, nil
}

// windowForDay returns the applicable open/close window for the given weekday,
// or false if the schedule defines no open window for that day.
func (oh *OperationalHours) windowForDay(d time.Weekday) (dayWindow, bool) {
	switch d {
	case time.Saturday:
		if oh.Saturday != nil {
			return *oh.Saturday, true
		}
	case time.Sunday:
		if oh.Sunday != nil {
			return *oh.Sunday, true
		}
	default:
		if oh.Weekday != nil {
			return *oh.Weekday, true
		}
	}

	// Flat fallback: applies to every day when the aggregate shape is absent.
	if oh.OpenTime != "" || oh.CloseTime != "" {
		if oh.IsOpen != nil && !*oh.IsOpen {
			return dayWindow{}, false
		}
		return dayWindow{Open: oh.OpenTime, Close: oh.CloseTime}, true
	}

	return dayWindow{}, false
}

// IsOpenAt reports whether t (already located in the correct timezone) falls
// within the operating window for its weekday. The returned string is a
// human-readable reason for audit/logging.
func (oh *OperationalHours) IsOpenAt(t time.Time) (bool, string) {
	w, ok := oh.windowForDay(t.Weekday())
	if !ok {
		return false, fmt.Sprintf("closed on %s", t.Weekday())
	}

	openMin, okOpen := parseHHMM(w.Open)
	closeMin, okClose := parseHHMM(w.Close)
	if !okOpen || !okClose || openMin == closeMin {
		// Unparseable or zero-length window (e.g. "00:00"-"00:00") => closed.
		return false, fmt.Sprintf("closed on %s (window %q-%q)", t.Weekday(), w.Open, w.Close)
	}

	nowMin := t.Hour()*60 + t.Minute()

	// Handle overnight windows (close < open), e.g. 22:00-02:00.
	var within bool
	if closeMin > openMin {
		within = nowMin >= openMin && nowMin < closeMin
	} else {
		within = nowMin >= openMin || nowMin < closeMin
	}

	if within {
		return true, fmt.Sprintf("within operating hours %s-%s on %s", w.Open, w.Close, t.Weekday())
	}
	return false, fmt.Sprintf("outside operating hours %s-%s on %s (now %02d:%02d)", w.Open, w.Close, t.Weekday(), t.Hour(), t.Minute())
}

// parseHHMM parses "HH:MM" into minutes since midnight.
func parseHHMM(s string) (int, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// resolveLocation maps a timezone string to a *time.Location, accepting IANA
// names and Indonesian abbreviations. Empty uses fallback.
func resolveLocation(tz, fallback string) (*time.Location, error) {
	if tz == "" {
		tz = fallback
	}
	switch strings.ToUpper(strings.TrimSpace(tz)) {
	case "WIB":
		tz = "Asia/Jakarta"
	case "WITA":
		tz = "Asia/Makassar"
	case "WIT":
		tz = "Asia/Jayapura"
	}
	return time.LoadLocation(tz)
}

// TimeWindowEvaluator denies time-gated actions when "now" (in the parameter's
// timezone) is outside the operational window. It fetches the operational_hours
// parameter through the opacache (fetch-through on miss; NATS-invalidated on
// policy7 param changes).
type TimeWindowEvaluator struct {
	cache     *opacache.Cache
	fetcher   OperationalHoursFetcher
	defaultTZ string
	failOpen  bool
	logger    zerolog.Logger
	now       func() time.Time // injectable clock for tests
}

// NewTimeWindowEvaluator builds a TimeWindowEvaluator. failOpen=true allows
// access when policy7 is unreachable or the parameter is missing/unparseable
// (availability-first); failOpen=false denies (strict).
func NewTimeWindowEvaluator(cache *opacache.Cache, fetcher OperationalHoursFetcher, defaultTZ string, failOpen bool, logger zerolog.Logger) *TimeWindowEvaluator {
	return &TimeWindowEvaluator{
		cache:     cache,
		fetcher:   fetcher,
		defaultTZ: defaultTZ,
		failOpen:  failOpen,
		logger:    logger,
		now:       time.Now,
	}
}

// cacheKey builds the opacache key for an org/branch scope. Aligned with the
// NATS prefix invalidation key "opa:<org>:operational_hours".
func cacheKey(orgID, branchID string) string {
	scope := branchID
	if scope == "" {
		scope = "global"
	}
	return fmt.Sprintf("opa:%s:operational_hours:%s", orgID, scope)
}

// operationalHours returns the parsed, cached operational_hours for the auth
// context's scope, fetching through policy7 on a cache miss.
func (e *TimeWindowEvaluator) operationalHours(ctx context.Context, authCtx *domain.AuthContext) (*OperationalHours, error) {
	orgID := authCtx.OrgID.String()
	branchID := authCtx.BranchID.String()
	roleID := ""
	if len(authCtx.Roles) > 0 {
		roleID = authCtx.Roles[0]
	}

	key := cacheKey(orgID, branchID)
	v, err := e.cache.GetOrFetch(key, func() (interface{}, error) {
		raw, ferr := e.fetcher.FetchOperationalHours(ctx, orgID, roleID, branchID)
		if ferr != nil {
			return nil, ferr
		}
		return parseOperationalHours(raw)
	})
	if err != nil {
		return nil, err
	}

	oh, ok := v.(*OperationalHours)
	if !ok {
		return nil, fmt.Errorf("unexpected cache value type %T", v)
	}
	return oh, nil
}

// Evaluate returns an allow/deny decision based on whether the current time (in
// the parameter's timezone) is within operational hours.
func (e *TimeWindowEvaluator) Evaluate(ctx context.Context, authCtx *domain.AuthContext) (*domain.AuthorizationResult, error) {
	const op = "authz.TimeWindowEvaluator.Evaluate"

	oh, err := e.operationalHours(ctx, authCtx)
	if err != nil {
		return e.onError(op, "fetch operational_hours", err), nil
	}

	loc, err := resolveLocation(oh.Timezone, e.defaultTZ)
	if err != nil {
		return e.onError(op, "resolve timezone", err), nil
	}

	now := e.now().In(loc)
	open, reason := oh.IsOpenAt(now)
	if !open {
		return &domain.AuthorizationResult{
			Allowed: false,
			Reason:  "time-based access denied: " + reason,
		}, nil
	}

	return &domain.AuthorizationResult{
		Allowed: true,
		Reason:  "time-based access allowed: " + reason,
	}, nil
}

// onError implements the fail-open / fail-closed policy on fetch/parse errors.
func (e *TimeWindowEvaluator) onError(op, stage string, err error) *domain.AuthorizationResult {
	e.logger.Warn().
		Str("op", op).
		Str("stage", stage).
		Err(err).
		Bool("fail_open", e.failOpen).
		Msg("time-based ABAC could not evaluate operational_hours")

	if e.failOpen {
		return &domain.AuthorizationResult{
			Allowed: true,
			Reason:  "time-based access allowed (fail-open: operational_hours unavailable)",
		}
	}
	return &domain.AuthorizationResult{
		Allowed: false,
		Reason:  "time-based access denied (fail-closed: operational_hours unavailable)",
	}
}
