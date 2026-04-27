package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool    *pgxpool.Pool
	replica *pgxpool.Pool

	UserRepository              *UserRepository
	CredentialRepository       *CredentialRepository
	VerificationTokenRepository *VerificationTokenRepository
}

func New(pool *pgxpool.Pool, replica *pgxpool.Pool) *Store {
	s := &Store{pool: pool, replica: replica}
	s.UserRepository = &UserRepository{pool: pool}
	s.CredentialRepository = &CredentialRepository{pool: pool}
	s.VerificationTokenRepository = &VerificationTokenRepository{pool: pool}
	return s
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const op = "postgres.UserRepository.Create"
	q := `
		INSERT INTO users (
			id, org_id, username, email, full_name, status,
			email_verified, mfa_enabled, mfa_method, mfa_reset_required,
			require_password_change, failed_login_attempts, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.pool.Exec(ctx, q,
		user.ID, user.OrgID, user.Username, user.Email, user.FullName, user.Status,
		user.EmailVerified, user.MFAEnabled, user.MFAMethod, user.MFAResetRequired,
		user.RequirePasswordChange, user.FailedLoginAttempts, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const op = "postgres.UserRepository.GetByID"
	q := `
		SELECT id, org_id, username, email, full_name, status,
			email_verified, mfa_enabled, mfa_method, mfa_reset_required,
			require_password_change, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, password_changed_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&user.ID, &user.OrgID, &user.Username, &user.Email, &user.FullName, &user.Status,
		&user.EmailVerified, &user.MFAEnabled, &user.MFAMethod, &user.MFAResetRequired,
		&user.RequirePasswordChange, &user.FailedLoginAttempts, &user.LockedUntil,
		&user.LastLoginAt, &user.LastLoginIP, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, orgID uuid.UUID, username string) (*domain.User, error) {
	const op = "postgres.UserRepository.GetByUsername"
	q := `
		SELECT id, org_id, username, email, full_name, status,
			email_verified, mfa_enabled, mfa_method, mfa_reset_required,
			require_password_change, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, password_changed_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM users WHERE org_id = $1 AND username = $2 AND deleted_at IS NULL
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, q, orgID, username).Scan(
		&user.ID, &user.OrgID, &user.Username, &user.Email, &user.FullName, &user.Status,
		&user.EmailVerified, &user.MFAEnabled, &user.MFAMethod, &user.MFAResetRequired,
		&user.RequirePasswordChange, &user.FailedLoginAttempts, &user.LockedUntil,
		&user.LastLoginAt, &user.LastLoginIP, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error) {
	const op = "postgres.UserRepository.GetByEmail"
	q := `
		SELECT id, org_id, username, email, full_name, status,
			email_verified, mfa_enabled, mfa_method, mfa_reset_required,
			require_password_change, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, password_changed_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM users WHERE org_id = $1 AND email = $2 AND deleted_at IS NULL
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, q, orgID, email).Scan(
		&user.ID, &user.OrgID, &user.Username, &user.Email, &user.FullName, &user.Status,
		&user.EmailVerified, &user.MFAEnabled, &user.MFAMethod, &user.MFAResetRequired,
		&user.RequirePasswordChange, &user.FailedLoginAttempts, &user.LockedUntil,
		&user.LastLoginAt, &user.LastLoginIP, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const op = "postgres.UserRepository.Update"
	q := `
		UPDATE users SET
			username = $2, email = $3, full_name = $4, status = $5,
			email_verified = $6, mfa_enabled = $7, mfa_method = $8, mfa_reset_required = $9,
			require_password_change = $10, failed_login_attempts = $11, locked_until = $12,
			last_login_at = $13, last_login_ip = $14, password_changed_at = $15,
			updated_at = $16, updated_by = $17
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, q,
		user.ID, user.Username, user.Email, user.FullName, user.Status,
		user.EmailVerified, user.MFAEnabled, user.MFAMethod, user.MFAResetRequired,
		user.RequirePasswordChange, user.FailedLoginAttempts, user.LockedUntil,
		user.LastLoginAt, user.LastLoginIP, user.PasswordChangedAt,
		user.UpdatedAt, user.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.UserRepository.Delete"
	q := `UPDATE users SET deleted_at = NOW(), status = 'deleted' WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.User, int, error) {
	const op = "postgres.UserRepository.ListByOrg"
	countQ := `SELECT COUNT(*) FROM users WHERE org_id = $1 AND deleted_at IS NULL`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: count: %w", op, err)
	}

	q := `
		SELECT id, org_id, username, email, full_name, status,
			email_verified, mfa_enabled, mfa_method, mfa_reset_required,
			require_password_change, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, password_changed_at,
			created_at, updated_at, deleted_at, created_by, updated_by
		FROM users WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, q, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.ID, &user.OrgID, &user.Username, &user.Email, &user.FullName, &user.Status,
			&user.EmailVerified, &user.MFAEnabled, &user.MFAMethod, &user.MFAResetRequired,
			&user.RequirePasswordChange, &user.FailedLoginAttempts, &user.LockedUntil,
			&user.LastLoginAt, &user.LastLoginIP, &user.PasswordChangedAt,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.CreatedBy, &user.UpdatedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("%s: scan: %w", op, err)
		}
		users = append(users, &user)
	}
	return users, total, nil
}

type CredentialRepository struct {
	pool *pgxpool.Pool
}

func (r *CredentialRepository) Create(ctx context.Context, cred *domain.UserCredential) error {
	const op = "postgres.CredentialRepository.Create"
	q := `
		INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, version, is_current, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q,
		cred.ID, cred.UserID, cred.CredentialType, cred.SecretHash,
		cred.Version, cred.IsCurrent, cred.CreatedAt, cred.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *CredentialRepository) GetCurrentByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserCredential, error) {
	const op = "postgres.CredentialRepository.GetCurrentByUserID"
	q := `
		SELECT id, user_id, credential_type, secret_hash, version, is_current, created_at, expires_at
		FROM user_credentials
		WHERE user_id = $1 AND is_current = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	var cred domain.UserCredential
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&cred.ID, &cred.UserID, &cred.CredentialType, &cred.SecretHash,
		&cred.Version, &cred.IsCurrent, &cred.CreatedAt, &cred.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &cred, nil
}

func (r *CredentialRepository) GetHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UserCredential, error) {
	const op = "postgres.CredentialRepository.GetHistory"
	q := `
		SELECT id, user_id, credential_type, secret_hash, version, is_current, created_at, expires_at
		FROM user_credentials
		WHERE user_id = $1 AND is_current = false
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var creds []*domain.UserCredential
	for rows.Next() {
		var cred domain.UserCredential
		if err := rows.Scan(
			&cred.ID, &cred.UserID, &cred.CredentialType, &cred.SecretHash,
			&cred.Version, &cred.IsCurrent, &cred.CreatedAt, &cred.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		creds = append(creds, &cred)
	}
	return creds, nil
}

func (r *CredentialRepository) Update(ctx context.Context, cred *domain.UserCredential) error {
	const op = "postgres.CredentialRepository.Update"
	q := `
		UPDATE user_credentials SET
			is_current = $2, expires_at = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, cred.ID, cred.IsCurrent, cred.ExpiresAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *CredentialRepository) RetireOldCredentials(ctx context.Context, userID uuid.UUID, keepCount int) error {
	const op = "postgres.CredentialRepository.RetireOldCredentials"
	q := `
		UPDATE user_credentials SET expires_at = NOW()
		WHERE user_id = $1 AND is_current = false AND expires_at IS NULL
		AND id NOT IN (
			SELECT id FROM user_credentials
			WHERE user_id = $1 AND is_current = false
			ORDER BY created_at DESC
			LIMIT $2
		)
	`
	_, err := r.pool.Exec(ctx, q, userID, keepCount)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type VerificationTokenRepository struct {
	pool *pgxpool.Pool
}

func (r *VerificationTokenRepository) Create(ctx context.Context, token *domain.VerificationToken) error {
	const op = "postgres.VerificationTokenRepository.Create"
	q := `
		INSERT INTO verification_tokens (id, user_id, token, token_type, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		token.ID, token.UserID, token.Token, token.TokenType, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *VerificationTokenRepository) GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error) {
	const op = "postgres.VerificationTokenRepository.GetByToken"
	q := `
		SELECT id, user_id, token, token_type, expires_at, used_at, created_at
		FROM verification_tokens
		WHERE token = $1
	`
	var vt domain.VerificationToken
	err := r.pool.QueryRow(ctx, q, token).Scan(
		&vt.ID, &vt.UserID, &vt.Token, &vt.TokenType, &vt.ExpiresAt, &vt.UsedAt, &vt.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &vt, nil
}

func (r *VerificationTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.VerificationTokenRepository.MarkUsed"
	q := `UPDATE verification_tokens SET used_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *VerificationTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const op = "postgres.VerificationTokenRepository.DeleteByUserID"
	q := `DELETE FROM verification_tokens WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
