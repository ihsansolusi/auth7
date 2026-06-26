// Package branchsync polls the core7-service-enterprise source-contract endpoint
// for branches and upserts them into auth7.branches.  Runs as a background
// goroutine started in cmd/server/start.go.
//
// Why a poller instead of NATS/webhooks: the source-of-truth contract
// (Plan 13 W2) is HTTP-only at the moment, and the data volume is small
// (a few hundred branches max per tenant).  Poll interval defaults to 5
// minutes; configurable via env.
package branchsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Config holds the runtime parameters loaded from env vars.
type Config struct {
	// SourceURL is the full URL of the enterprise source-contract endpoint,
	// e.g. http://core7-service-enterprise:8090/v1/source-contracts/branches
	SourceURL string

	// ClientID and ClientSecret are the OAuth2 client_credentials for the
	// branchsync M2M client registered in auth7. Both must be set to enable
	// the poller.
	ClientID     string
	ClientSecret string

	// TokenEndpoint is the full URL of auth7's token endpoint,
	// e.g. http://localhost:4445/oauth2/token
	TokenEndpoint string

	// OrgID is the tenant marker. Sent as X-Actor-Org-Id; also written to
	// auth7.branches.org_id so cross-service joins work.  Defaults to the
	// demo tenant 00000000-0000-0000-0000-000000000001 if empty.
	OrgID uuid.UUID

	// Interval between polls. Default 5 minutes.
	Interval time.Duration

	// PerPage controls how many branches we ask for per page. 200 covers
	// every reasonable BJBS-scale tenant.
	PerPage int

	// HTTPTimeout per request.
	HTTPTimeout time.Duration
}

// DefaultConfig returns a Config populated with sensible defaults.
// Override via Config{} literal in main.go before passing to NewPoller.
func DefaultConfig() Config {
	return Config{
		Interval:    5 * time.Minute,
		PerPage:     200,
		HTTPTimeout: 30 * time.Second,
		OrgID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

// Poller pulls branch projections from enterprise and upserts into auth7.
type Poller struct {
	cfg     Config
	pool    *pgxpool.Pool
	client  *http.Client
	logger  zerolog.Logger
	enabled bool
}

// NewPoller constructs a Poller. If cfg.SourceURL is empty, Run returns
// immediately (poller disabled) — handy for unit tests / local dev.
func NewPoller(cfg Config, pool *pgxpool.Pool, logger zerolog.Logger) *Poller {
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.PerPage == 0 {
		cfg.PerPage = 200
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	return &Poller{
		cfg:     cfg,
		pool:    pool,
		client:  &http.Client{Timeout: cfg.HTTPTimeout},
		logger:  logger.With().Str("component", "branchsync").Logger(),
		enabled: cfg.SourceURL != "" && cfg.ClientID != "",
	}
}

// Run blocks until ctx is cancelled, polling at cfg.Interval. Returns nil
// on graceful shutdown.  Errors during a poll are logged but do not stop
// the loop — the next tick may succeed.
func (p *Poller) Run(ctx context.Context) error {
	const op = "branchsync.Poller.Run"
	if !p.enabled {
		p.logger.Info().Str("op", op).Msg("disabled — SourceURL or ClientID empty, skipping")
		return nil
	}
	p.logger.Info().Str("op", op).
		Str("source_url", p.cfg.SourceURL).
		Dur("interval", p.cfg.Interval).
		Msg("starting branch sync poller")

	// First run immediately so a fresh service comes up with branches synced.
	if err := p.tick(ctx); err != nil {
		p.logger.Error().Err(err).Str("op", op).Msg("initial sync failed")
	}

	t := time.NewTicker(p.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Str("op", op).Msg("stopped")
			return nil
		case <-t.C:
			if err := p.tick(ctx); err != nil {
				p.logger.Error().Err(err).Str("op", op).Msg("sync failed")
			}
		}
	}
}

// ─── one polling cycle ───────────────────────────────────────────────────────

// branchProjectionItem is the minimal subset of the enterprise branch DTO
// that auth7 needs for access decisions and JWT branch_code claims.
type branchProjectionItem struct {
	BranchID   string `json:"branch_id"`
	BranchCode string `json:"branch_code"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at"`
}

type contractEnvelope struct {
	Data []branchProjectionItem `json:"data"`
	Meta struct {
		Total      int `json:"total"`
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
}

// tick performs one full fetch + upsert cycle, paginating until all pages
// are read.  Returns the first error encountered or nil if successful.
//
// auth7.branches is a projection, NOT master data — the enterprise domain
// owns the lifecycle.  So a full, error-free pass also tombstones (sets
// is_active=false) any branch that no longer appears in the source.  The
// deactivation runs ONLY after every page fetched OK: if any page errors we
// return early and leave existing rows untouched, so a transient enterprise
// outage can never wipe the projection.
func (p *Poller) tick(ctx context.Context) error {
	const op = "branchsync.Poller.tick"

	page := 1
	totalApplied := 0
	// seen accumulates every branch ID returned across all pages of this pass.
	seen := make(map[uuid.UUID]struct{})

	for {
		env, err := p.fetchPage(ctx, page)
		if err != nil {
			return fmt.Errorf("%s page=%d: %w", op, page, err)
		}
		if len(env.Data) == 0 {
			break
		}
		applied, err := p.upsertBatch(ctx, env.Data, seen)
		if err != nil {
			return fmt.Errorf("%s upsert page=%d: %w", op, page, err)
		}
		totalApplied += applied
		if env.Meta.TotalPages <= page {
			break
		}
		page++
	}

	// Full pass succeeded (no early return above) — reconcile deletions.
	deactivated, err := p.deactivateAbsent(ctx, seen)
	if err != nil {
		return fmt.Errorf("%s deactivate: %w", op, err)
	}

	p.logger.Info().Str("op", op).
		Int("applied", totalApplied).
		Int("seen", len(seen)).
		Int("deactivated", deactivated).
		Msg("branch sync complete")
	return nil
}

func (p *Poller) fetchPage(ctx context.Context, page int) (*contractEnvelope, error) {
	const op = "branchsync.Poller.fetchPage"

	url := fmt.Sprintf("%s?page=%d&per_page=%d", p.cfg.SourceURL, page, p.cfg.PerPage)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s build req: %w", op, err)
	}
	token, err := p.getM2MToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s get m2m token: %w", op, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Actor-Org-Id", p.cfg.OrgID.String())
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s http: %w", op, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s upstream %d: %s", op, resp.StatusCode, snippet(body))
	}

	var env contractEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%s decode: %w", op, err)
	}
	return &env, nil
}

// upsertBatch performs the UPSERT for the items in a single page using a tx.
// Only the 5 projection columns are stored — branch hierarchy and type live
// in the enterprise domain.  Every successfully-parsed branch ID (regardless
// of its active status) is recorded in seen so the caller can tombstone the
// branches that the source no longer reports.
func (p *Poller) upsertBatch(ctx context.Context, items []branchProjectionItem, seen map[uuid.UUID]struct{}) (int, error) {
	const op = "branchsync.Poller.upsertBatch"

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s begin: %w", op, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck — rollback after commit is a no-op

	applied := 0
	for _, item := range items {
		id, err := uuid.Parse(item.BranchID)
		if err != nil {
			p.logger.Warn().Str("branch_id", item.BranchID).Err(err).Msg("skip — invalid uuid")
			continue
		}
		seen[id] = struct{}{}
		// Honor is_active=false from the source — a branch reported as
		// non-active is deactivated here, not merely skipped.
		isActive := strings.EqualFold(item.Status, "active")

		_, err = tx.Exec(ctx, `
			INSERT INTO branches (id, org_id, branch_code, is_active, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (id) DO UPDATE SET
				org_id      = EXCLUDED.org_id,
				branch_code = EXCLUDED.branch_code,
				is_active   = EXCLUDED.is_active,
				updated_at  = EXCLUDED.updated_at
			WHERE branches.updated_at < EXCLUDED.updated_at
		`, id, p.cfg.OrgID, item.BranchCode, isActive)
		if err != nil {
			return applied, fmt.Errorf("%s upsert id=%s: %w", op, id, err)
		}
		applied++
	}

	if err := tx.Commit(ctx); err != nil {
		return applied, fmt.Errorf("%s commit: %w", op, err)
	}
	return applied, nil
}

// deactivateAbsent tombstones every still-active branch for this tenant that
// the source did not report in the completed pass.  MUST be called only after
// a fully successful fetch (see tick) — running it on a partial pass would
// deactivate branches that merely live on an unfetched page.
//
// Guard: an empty seen set is treated as a no-op.  A successful pass that
// returns zero branches almost always means a misconfigured source or an
// upstream that briefly served an empty list — not "every branch was deleted".
// Refusing to deactivate on empty avoids wiping the whole projection.
func (p *Poller) deactivateAbsent(ctx context.Context, seen map[uuid.UUID]struct{}) (int, error) {
	const op = "branchsync.Poller.deactivateAbsent"

	if len(seen) == 0 {
		p.logger.Warn().Str("op", op).
			Msg("source returned zero branches — skipping deactivation to avoid wiping the projection")
		return 0, nil
	}

	// Pass IDs as text[] and cast to uuid[] — robust under pgx's default type
	// map without a registered uuid array codec.
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id.String())
	}

	tag, err := p.pool.Exec(ctx, `
		UPDATE branches
		SET is_active = false, updated_at = NOW()
		WHERE org_id = $1 AND is_active = true AND id <> ALL($2::uuid[])
	`, p.cfg.OrgID, ids)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	n := int(tag.RowsAffected())
	if n > 0 {
		p.logger.Info().Str("op", op).Int("deactivated", n).
			Msg("tombstoned branches absent from source")
	}
	return n, nil
}

// getM2MToken fetches a short-lived bearer token from auth7 using the
// client_credentials grant. Called once per fetchPage — tokens are not
// cached because the poller interval is long relative to token TTL.
func (p *Poller) getM2MToken(ctx context.Context) (string, error) {
	const op = "branchsync.Poller.getM2MToken"

	body := strings.NewReader(
		"grant_type=client_credentials" +
			"&client_id=" + p.cfg.ClientID +
			"&client_secret=" + p.cfg.ClientSecret,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenEndpoint, body)
	if err != nil {
		return "", fmt.Errorf("%s build req: %w", op, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoding")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s http: %w", op, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s token endpoint %d: %s", op, resp.StatusCode, snippet(raw))
	}

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("%s parse response: %w", op, err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("%s empty access_token in response", op)
	}
	return tok.AccessToken, nil
}

func snippet(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}
