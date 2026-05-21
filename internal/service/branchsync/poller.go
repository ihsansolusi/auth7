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
	"github.com/jackc/pgx/v5"
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

type branchProjectionItem struct {
	BranchID             string `json:"branch_id"`
	BranchCode           string `json:"branch_code"`
	BranchName           string `json:"branch_name"`
	BranchType           string `json:"branch_type"`
	ParentBranchID       string `json:"parent_branch_id"`
	AreaID               string `json:"area_id"`
	BranchClassification string `json:"branch_classification"`
	Timezone             string `json:"timezone"`
	OfficeID             string `json:"office_id"`
	Status               string `json:"status"`
	UpdatedAt            string `json:"updated_at"`
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
func (p *Poller) tick(ctx context.Context) error {
	const op = "branchsync.Poller.tick"

	page := 1
	totalApplied := 0

	for {
		env, err := p.fetchPage(ctx, page)
		if err != nil {
			return fmt.Errorf("%s page=%d: %w", op, page, err)
		}
		if len(env.Data) == 0 {
			break
		}
		applied, err := p.upsertBatch(ctx, env.Data)
		if err != nil {
			return fmt.Errorf("%s upsert page=%d: %w", op, page, err)
		}
		totalApplied += applied
		if env.Meta.TotalPages <= page {
			break
		}
		page++
	}

	p.logger.Info().Str("op", op).
		Int("applied", totalApplied).
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
// Returns the number of rows touched.
func (p *Poller) upsertBatch(ctx context.Context, items []branchProjectionItem) (int, error) {
	const op = "branchsync.Poller.upsertBatch"

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s begin: %w", op, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck — rollback after commit is a no-op

	// Look up the default branch_type_id once per batch. Required by
	// auth7.branches schema (NOT NULL FK to branch_types).  Falls back to
	// inserting a "DEFAULT" branch_type if none exists for this org.
	defaultBranchTypeID, err := p.ensureBranchTypeID(ctx, tx)
	if err != nil {
		return 0, err
	}

	applied := 0
	for _, item := range items {
		id, err := uuid.Parse(item.BranchID)
		if err != nil {
			p.logger.Warn().Str("branch_id", item.BranchID).Err(err).Msg("skip — invalid uuid")
			continue
		}
		isActive := strings.EqualFold(item.Status, "active")
		status := "active"
		if !isActive {
			status = "inactive"
		}

		var parentID, areaID interface{}
		if item.ParentBranchID != "" {
			if u, err := uuid.Parse(item.ParentBranchID); err == nil {
				parentID = u
			}
		}
		if item.AreaID != "" {
			if u, err := uuid.Parse(item.AreaID); err == nil {
				areaID = u
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO branches (
				id, org_id, branch_type_id, code, name, status,
				branch_type, parent_branch_id, area_id, branch_classification
			) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), $8, $9, NULLIF($10,''))
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				status = EXCLUDED.status,
				branch_type = EXCLUDED.branch_type,
				parent_branch_id = EXCLUDED.parent_branch_id,
				area_id = EXCLUDED.area_id,
				branch_classification = EXCLUDED.branch_classification,
				updated_at = NOW()
		`,
			id, p.cfg.OrgID, defaultBranchTypeID,
			item.BranchCode, item.BranchName, status,
			item.BranchType, parentID, areaID, item.BranchClassification,
		)
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

// ensureBranchTypeID returns a valid branch_type_id for the configured org,
// creating a placeholder "DEFAULT" row if the table is empty.  Auth7 branches
// requires the FK; in practice the seed scaffold already populates 9 rows.
func (p *Poller) ensureBranchTypeID(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
	// Try to find any existing branch_type for this org.
	var id uuid.UUID
	row := tx.QueryRow(ctx, `SELECT id FROM branch_types WHERE org_id = $1 LIMIT 1`, p.cfg.OrgID)
	if err := row.Scan(&id); err == nil {
		return id, nil
	}
	// None exists — insert a minimal placeholder.
	newID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO branch_types (id, org_id, code, label, short_code, level, is_operational, can_have_children)
		VALUES ($1, $2, 'DEFAULT', 'Default', 'DEF', 0, TRUE, TRUE)
		ON CONFLICT DO NOTHING
	`, newID, p.cfg.OrgID); err != nil {
		return uuid.Nil, fmt.Errorf("ensureBranchTypeID: %w", err)
	}
	return newID, nil
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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
