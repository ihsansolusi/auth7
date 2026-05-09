package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	oauth2svc "github.com/ihsansolusi/auth7/internal/service/oauth2"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	opOAuth2ClientCreate   = "postgres.OAuth2ClientRepository.CreateClient"
	opOAuth2ClientUpdate   = "postgres.OAuth2ClientRepository.UpdateClient"
	opOAuth2ClientGet      = "postgres.OAuth2ClientRepository.GetClient"
	opOAuth2ClientDelete   = "postgres.OAuth2ClientRepository.DeleteClient"
	opOAuth2ClientListApps = "postgres.OAuth2ClientRepository.ListApps"
	opOAuth2AuthCodeCreate = "postgres.OAuth2AuthCodeRepository.Create"
	opOAuth2AuthCodeGet    = "postgres.OAuth2AuthCodeRepository.GetByCode"
	opOAuth2AuthCodeMark   = "postgres.OAuth2AuthCodeRepository.MarkUsed"
	opOAuth2AuthCodeDelete = "postgres.OAuth2AuthCodeRepository.Delete"
)

// --------------------------------------------------------------------------
// OAuth2ClientRepository — implements oauth2svc.DCRStore
// --------------------------------------------------------------------------

type OAuth2ClientRepository struct {
	pool *pgxpool.Pool
}

func (r *OAuth2ClientRepository) CreateClient(ctx context.Context, c *domain.Client) error {
	const op = opOAuth2ClientCreate
	query := `
		INSERT INTO oauth2_clients (
			id, client_id, org_id, name, description,
			client_type, token_endpoint_auth_method,
			allowed_scopes, allowed_redirect_uris, allowed_origins,
			client_secret_hash, token_expiration, refresh_token_expiration,
			allow_multiple_tokens, skip_consent_screen, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18
		)`
	_, err := r.pool.Exec(ctx, query,
		c.ID,
		c.ID.String(), // client_id = id string
		c.OrgID,
		c.Name,
		c.Description,
		string(c.ClientType),
		string(c.TokenEndpointAuthMethod),
		c.AllowedScopes,
		c.AllowedRedirectURIs,
		c.AllowedOrigins,
		c.ClientSecretHash,
		c.TokenExpiration,
		c.RefreshTokenExpiration,
		c.AllowMultipleTokens,
		c.SkipConsentScreen,
		c.IsActive,
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *OAuth2ClientRepository) UpdateClient(ctx context.Context, c *domain.Client) error {
	const op = opOAuth2ClientUpdate
	query := `
		UPDATE oauth2_clients SET
			name = $1, description = $2, client_type = $3,
			token_endpoint_auth_method = $4,
			allowed_scopes = $5, allowed_redirect_uris = $6, allowed_origins = $7,
			client_secret_hash = $8,
			token_expiration = $9, refresh_token_expiration = $10,
			allow_multiple_tokens = $11, skip_consent_screen = $12,
			is_active = $13, updated_at = $14
		WHERE client_id = $15`
	_, err := r.pool.Exec(ctx, query,
		c.Name, c.Description, string(c.ClientType),
		string(c.TokenEndpointAuthMethod),
		c.AllowedScopes, c.AllowedRedirectURIs, c.AllowedOrigins,
		c.ClientSecretHash,
		c.TokenExpiration, c.RefreshTokenExpiration,
		c.AllowMultipleTokens, c.SkipConsentScreen,
		c.IsActive, c.UpdatedAt,
		c.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *OAuth2ClientRepository) GetClient(ctx context.Context, clientID string) (*domain.Client, error) {
	const op = opOAuth2ClientGet
	query := `
		SELECT id, client_id, org_id, name, description,
		       client_type, token_endpoint_auth_method,
		       allowed_scopes, allowed_redirect_uris, allowed_origins,
		       client_secret_hash, token_expiration, refresh_token_expiration,
		       allow_multiple_tokens, skip_consent_screen, is_active,
		       created_at, updated_at,
		       COALESCE(app_url, '') AS app_url,
		       COALESCE(icon_name, '') AS icon_name,
		       COALESCE(icon_color, '') AS icon_color
		FROM oauth2_clients
		WHERE client_id = $1`

	row := r.pool.QueryRow(ctx, query, clientID)

	c := &domain.Client{}
	var clientIDStr string
	var clientTypeStr, authMethodStr string
	var description, clientSecretHash *string
	err := row.Scan(
		&c.ID, &clientIDStr, &c.OrgID, &c.Name, &description,
		&clientTypeStr, &authMethodStr,
		&c.AllowedScopes, &c.AllowedRedirectURIs, &c.AllowedOrigins,
		&clientSecretHash, &c.TokenExpiration, &c.RefreshTokenExpiration,
		&c.AllowMultipleTokens, &c.SkipConsentScreen, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt,
		&c.AppURL, &c.IconName, &c.IconColor,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	c.ClientID = clientIDStr
	if description != nil {
		c.Description = *description
	}
	if clientSecretHash != nil {
		c.ClientSecretHash = *clientSecretHash
	}
	c.ClientType = domain.ClientType(clientTypeStr)
	c.TokenEndpointAuthMethod = domain.TokenEndpointAuthMethod(authMethodStr)
	return c, nil
}

func (r *OAuth2ClientRepository) ListApps(ctx context.Context) ([]*domain.AppEntry, error) {
	const op = opOAuth2ClientListApps
	query := `
		SELECT client_id, name,
		       COALESCE(description, '') AS description,
		       COALESCE(app_url, '')    AS app_url,
		       COALESCE(icon_name, '')  AS icon_name,
		       COALESCE(icon_color, '') AS icon_color
		FROM oauth2_clients
		WHERE is_active = true
		  AND client_type IN ('web', 'spa')
		  AND app_url IS NOT NULL AND app_url <> ''
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var apps []*domain.AppEntry
	for rows.Next() {
		a := &domain.AppEntry{}
		if err := rows.Scan(&a.ClientID, &a.Name, &a.Description, &a.AppURL, &a.IconName, &a.IconColor); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (r *OAuth2ClientRepository) DeleteClient(ctx context.Context, clientID string) error {
	const op = opOAuth2ClientDelete
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth2_clients WHERE client_id = $1`, clientID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// --------------------------------------------------------------------------
// OAuth2AuthCodeRepository — implements oauth2svc.AuthCodeStore
// --------------------------------------------------------------------------

type OAuth2AuthCodeRepository struct {
	pool *pgxpool.Pool
}

func (r *OAuth2AuthCodeRepository) Create(ctx context.Context, code *oauth2svc.AuthCode) error {
	const op = opOAuth2AuthCodeCreate
	query := `
		INSERT INTO oauth2_authorization_codes (
			code, client_id, redirect_uri, scope,
			user_id, username, email, org_id,
			roles, branch_id,
			code_challenge, code_challenge_method,
			expires_at, code_used, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	roles := code.Roles
	if roles == nil {
		roles = []string{}
	}
	_, err := r.pool.Exec(ctx, query,
		code.Code, code.ClientID, code.RedirectURI, code.Scope,
		code.UserID, code.Username, code.Email, code.OrgID,
		roles, code.BranchID,
		code.CodeChallenge, code.CodeChallengeMethod,
		code.ExpiresAt, code.CodeUsed, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *OAuth2AuthCodeRepository) GetByCode(ctx context.Context, code string) (*oauth2svc.AuthCode, error) {
	const op = opOAuth2AuthCodeGet
	query := `
		SELECT code, client_id, redirect_uri, scope,
		       user_id, username, email, org_id,
		       roles, branch_id,
		       code_challenge, code_challenge_method,
		       expires_at, code_used
		FROM oauth2_authorization_codes
		WHERE code = $1`

	row := r.pool.QueryRow(ctx, query, code)

	ac := &oauth2svc.AuthCode{}
	var userIDStr, orgIDStr string
	var username, email, branchID *string
	var roles []string
	err := row.Scan(
		&ac.Code, &ac.ClientID, &ac.RedirectURI, &ac.Scope,
		&userIDStr, &username, &email, &orgIDStr,
		&roles, &branchID,
		&ac.CodeChallenge, &ac.CodeChallengeMethod,
		&ac.ExpiresAt, &ac.CodeUsed,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	ac.UserID, _ = uuid.Parse(userIDStr)
	ac.OrgID, _ = uuid.Parse(orgIDStr)
	if username != nil {
		ac.Username = *username
	}
	if email != nil {
		ac.Email = *email
	}
	if branchID != nil {
		ac.BranchID = *branchID
	}
	ac.Roles = roles
	return ac, nil
}

func (r *OAuth2AuthCodeRepository) MarkUsed(ctx context.Context, code string) error {
	const op = opOAuth2AuthCodeMark
	_, err := r.pool.Exec(ctx, `UPDATE oauth2_authorization_codes SET code_used = true WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *OAuth2AuthCodeRepository) Delete(ctx context.Context, code string) error {
	const op = opOAuth2AuthCodeDelete
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth2_authorization_codes WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
