package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool    *pgxpool.Pool
	replica *pgxpool.Pool

	UserRepository              *UserRepository
	CredentialRepository       *CredentialRepository
	VerificationTokenRepository *VerificationTokenRepository
	SessionRepository          *SessionRepository
	MFAConfigRepository        *MFAConfigRepository
	EmailOTPCodeRepository     *EmailOTPCodeRepository
	RoleRepository            *RoleRepository
	PermissionRepository      *PermissionRepository
	RolePermissionRepository *RolePermissionRepository
	UserRoleRepository        *UserRoleRepository
	AuditLogRepository        *AuditLogRepository
}

func New(pool *pgxpool.Pool, replica *pgxpool.Pool) *Store {
	s := &Store{pool: pool, replica: replica}
	s.UserRepository = &UserRepository{pool: pool}
	s.CredentialRepository = &CredentialRepository{pool: pool}
	s.VerificationTokenRepository = &VerificationTokenRepository{pool: pool}
	s.SessionRepository = &SessionRepository{pool: pool}
	s.MFAConfigRepository = &MFAConfigRepository{pool: pool}
	s.EmailOTPCodeRepository = &EmailOTPCodeRepository{pool: pool}
	s.RoleRepository = &RoleRepository{pool: pool}
	s.PermissionRepository = &PermissionRepository{pool: pool}
	s.RolePermissionRepository = &RolePermissionRepository{pool: pool}
	s.UserRoleRepository = &UserRoleRepository{pool: pool}
	s.AuditLogRepository = &AuditLogRepository{pool: pool}
	return s
}

func (s *Store) Users() store.UserStore                { return s.UserRepository }
func (s *Store) Credentials() store.CredentialStore  { return s.CredentialRepository }
func (s *Store) Sessions() store.SessionStore         { return s.SessionRepository }
func (s *Store) MFAConfigs() store.MFAConfigStore     { return s.MFAConfigRepository }
func (s *Store) EmailOTPCodes() store.EmailOTPCodeStore { return s.EmailOTPCodeRepository }
func (s *Store) Roles() store.RoleStore               { return s.RoleRepository }
func (s *Store) Permissions() store.PermissionStore  { return s.PermissionRepository }
func (s *Store) RolePermissions() store.RolePermissionStore { return s.RolePermissionRepository }
func (s *Store) UserRoles() store.UserRoleStore       { return s.UserRoleRepository }
func (s *Store) AuditLogs() store.AuditLogStore      { return s.AuditLogRepository }

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

type SessionRepository struct {
	pool *pgxpool.Pool
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	const op = "postgres.SessionRepository.Create"
	q := `
		INSERT INTO sessions (id, user_id, org_id, client_id, ip_address, user_agent, device_info, scopes, created_at, last_used_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, q,
		session.ID, session.UserID, session.OrgID, session.ClientID,
		session.IPAddress, session.UserAgent, session.DeviceInfo, session.Scopes,
		session.CreatedAt, session.LastUsedAt, session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	const op = "postgres.SessionRepository.GetByID"
	q := `
		SELECT id, user_id, org_id, client_id, ip_address, user_agent, device_info, scopes, created_at, last_used_at, expires_at, revoked_at, revoked_by, revoke_reason
		FROM sessions WHERE id = $1
	`
	var session domain.Session
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&session.ID, &session.UserID, &session.OrgID, &session.ClientID,
		&session.IPAddress, &session.UserAgent, &session.DeviceInfo, &session.Scopes,
		&session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt,
		&session.RevokedAt, &session.RevokedBy, &session.RevokeReason,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &session, nil
}

func (r *SessionRepository) Update(ctx context.Context, session *domain.Session) error {
	const op = "postgres.SessionRepository.Update"
	q := `
		UPDATE sessions SET last_used_at = $2, device_info = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, session.ID, session.LastUsedAt, session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID, reason string) error {
	const op = "postgres.SessionRepository.Revoke"
	q := `UPDATE sessions SET revoked_at = NOW(), revoked_by = $2, revoke_reason = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, revokedBy, reason)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *SessionRepository) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	const op = "postgres.SessionRepository.RevokeAll"
	q := `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type MFAConfigRepository struct {
	pool *pgxpool.Pool
}

func (r *MFAConfigRepository) Create(ctx context.Context, cfg *domain.MFAConfig) error {
	const op = "postgres.MFAConfigRepository.Create"
	q := `
		INSERT INTO mfa_configs (id, user_id, totp_secret_encrypted, totp_secret_iv, is_totp_enabled, is_email_otp_enabled, is_backup_codes_enabled, backup_codes_hash, mfa_enabled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, q,
		cfg.ID, cfg.UserID, cfg.TOTPSecretEncrypted, cfg.TOTPSecretIV,
		cfg.IsTOTPEnabled, cfg.IsEmailOTPEnabled, cfg.IsBackupCodesEnabled,
		cfg.BackupCodesHash, cfg.MFAEnabledAt, cfg.CreatedAt, cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *MFAConfigRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error) {
	const op = "postgres.MFAConfigRepository.GetByUserID"
	q := `
		SELECT id, user_id, totp_secret_encrypted, totp_secret_iv, is_totp_enabled, is_email_otp_enabled, is_backup_codes_enabled, backup_codes_hash, mfa_enabled_at, created_at, updated_at
		FROM mfa_configs WHERE user_id = $1
	`
	var cfg domain.MFAConfig
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&cfg.ID, &cfg.UserID, &cfg.TOTPSecretEncrypted, &cfg.TOTPSecretIV,
		&cfg.IsTOTPEnabled, &cfg.IsEmailOTPEnabled, &cfg.IsBackupCodesEnabled,
		&cfg.BackupCodesHash, &cfg.MFAEnabledAt, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &cfg, nil
}

func (r *MFAConfigRepository) Update(ctx context.Context, cfg *domain.MFAConfig) error {
	const op = "postgres.MFAConfigRepository.Update"
	q := `
		UPDATE mfa_configs SET
			totp_secret_encrypted = $2, totp_secret_iv = $3,
			is_totp_enabled = $4, is_email_otp_enabled = $5,
			is_backup_codes_enabled = $6, backup_codes_hash = $7,
			mfa_enabled_at = $8, updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.pool.Exec(ctx, q,
		cfg.UserID, cfg.TOTPSecretEncrypted, cfg.TOTPSecretIV,
		cfg.IsTOTPEnabled, cfg.IsEmailOTPEnabled, cfg.IsBackupCodesEnabled,
		cfg.BackupCodesHash, cfg.MFAEnabledAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *MFAConfigRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	const op = "postgres.MFAConfigRepository.Delete"
	q := `DELETE FROM mfa_configs WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type EmailOTPCodeRepository struct {
	pool *pgxpool.Pool
}

func (r *EmailOTPCodeRepository) Create(ctx context.Context, code *domain.EmailOTPCode) error {
	const op = "postgres.EmailOTPCodeRepository.Create"
	q := `
		INSERT INTO email_otp_codes (id, user_id, code, purpose, expires_at, used_at, attempts, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q,
		code.ID, code.UserID, code.Code, code.Purpose,
		code.ExpiresAt, code.UsedAt, code.Attempts, code.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *EmailOTPCodeRepository) GetByUserIDAndPurpose(ctx context.Context, userID uuid.UUID, purpose string) (*domain.EmailOTPCode, error) {
	const op = "postgres.EmailOTPCodeRepository.GetByUserIDAndPurpose"
	q := `
		SELECT id, user_id, code, purpose, expires_at, used_at, attempts, created_at
		FROM email_otp_codes
		WHERE user_id = $1 AND purpose = $2 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`
	var code domain.EmailOTPCode
	err := r.pool.QueryRow(ctx, q, userID, purpose).Scan(
		&code.ID, &code.UserID, &code.Code, &code.Purpose,
		&code.ExpiresAt, &code.UsedAt, &code.Attempts, &code.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &code, nil
}

func (r *EmailOTPCodeRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailOTPCode, error) {
	const op = "postgres.EmailOTPCodeRepository.GetActiveByUserID"
	q := `
		SELECT id, user_id, code, purpose, expires_at, used_at, attempts, created_at
		FROM email_otp_codes
		WHERE user_id = $1 AND expires_at > NOW() AND used_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	var code domain.EmailOTPCode
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&code.ID, &code.UserID, &code.Code, &code.Purpose,
		&code.ExpiresAt, &code.UsedAt, &code.Attempts, &code.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &code, nil
}

func (r *EmailOTPCodeRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.EmailOTPCodeRepository.MarkUsed"
	q := `UPDATE email_otp_codes SET used_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *EmailOTPCodeRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.EmailOTPCodeRepository.IncrementAttempts"
	q := `UPDATE email_otp_codes SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *EmailOTPCodeRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	const op = "postgres.EmailOTPCodeRepository.DeleteByUserID"
	q := `DELETE FROM email_otp_codes WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type RoleRepository struct {
	pool *pgxpool.Pool
}

func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	const op = "postgres.RoleRepository.Create"
	q := `
		INSERT INTO roles (id, org_id, code, name, description, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q,
		role.ID, role.OrgID, role.Code, role.Name, role.Description,
		role.IsDefault, role.CreatedAt, role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *RoleRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Role, error) {
	const op = "postgres.RoleRepository.GetByID"
	q := `
		SELECT id, org_id, code, name, description, is_default, created_at, updated_at
		FROM roles WHERE id = $1 AND org_id = $2
	`
	var role domain.Role
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&role.ID, &role.OrgID, &role.Code, &role.Name, &role.Description,
		&role.IsDefault, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &role, nil
}

func (r *RoleRepository) GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.Role, error) {
	const op = "postgres.RoleRepository.GetByCode"
	q := `
		SELECT id, org_id, code, name, description, is_default, created_at, updated_at
		FROM roles WHERE org_id = $1 AND code = $2
	`
	var role domain.Role
	err := r.pool.QueryRow(ctx, q, orgID, code).Scan(
		&role.ID, &role.OrgID, &role.Code, &role.Name, &role.Description,
		&role.IsDefault, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &role, nil
}

func (r *RoleRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Role, error) {
	const op = "postgres.RoleRepository.ListByOrg"
	q := `
		SELECT id, org_id, code, name, description, is_default, created_at, updated_at
		FROM roles WHERE org_id = $1 ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(
			&role.ID, &role.OrgID, &role.Code, &role.Name, &role.Description,
			&role.IsDefault, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		roles = append(roles, &role)
	}
	return roles, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	const op = "postgres.RoleRepository.Update"
	q := `
		UPDATE roles SET name = $3, description = $4, updated_at = $5
		WHERE id = $1 AND org_id = $2
	`
	_, err := r.pool.Exec(ctx, q,
		role.ID, role.OrgID, role.Name, role.Description, role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *RoleRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	const op = "postgres.RoleRepository.Delete"
	q := `DELETE FROM roles WHERE id = $1 AND org_id = $2`
	_, err := r.pool.Exec(ctx, q, id, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type PermissionRepository struct {
	pool *pgxpool.Pool
}

func (r *PermissionRepository) Create(ctx context.Context, perm *domain.Permission) error {
	const op = "postgres.PermissionRepository.Create"
	q := `
		INSERT INTO permissions (id, code, name, description, resource_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q,
		perm.ID, perm.Code, perm.Name, perm.Description, perm.ResourceType, perm.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	const op = "postgres.PermissionRepository.GetByID"
	q := `
		SELECT id, code, name, description, resource_type, created_at
		FROM permissions WHERE id = $1
	`
	var perm domain.Permission
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&perm.ID, &perm.Code, &perm.Name, &perm.Description,
		&perm.ResourceType, &perm.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &perm, nil
}

func (r *PermissionRepository) GetByCode(ctx context.Context, code string) (*domain.Permission, error) {
	const op = "postgres.PermissionRepository.GetByCode"
	q := `
		SELECT id, code, name, description, resource_type, created_at
		FROM permissions WHERE code = $1
	`
	var perm domain.Permission
	err := r.pool.QueryRow(ctx, q, code).Scan(
		&perm.ID, &perm.Code, &perm.Name, &perm.Description,
		&perm.ResourceType, &perm.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &perm, nil
}

func (r *PermissionRepository) List(ctx context.Context) ([]*domain.Permission, error) {
	const op = "postgres.PermissionRepository.List"
	q := `
		SELECT id, code, name, description, resource_type, created_at
		FROM permissions ORDER BY resource_type, name
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		var perm domain.Permission
		if err := rows.Scan(
			&perm.ID, &perm.Code, &perm.Name, &perm.Description,
			&perm.ResourceType, &perm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		perms = append(perms, &perm)
	}
	return perms, nil
}

func (r *PermissionRepository) ListByResourceType(ctx context.Context, resourceType string) ([]*domain.Permission, error) {
	const op = "postgres.PermissionRepository.ListByResourceType"
	q := `
		SELECT id, code, name, description, resource_type, created_at
		FROM permissions WHERE resource_type = $1 ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, resourceType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		var perm domain.Permission
		if err := rows.Scan(
			&perm.ID, &perm.Code, &perm.Name, &perm.Description,
			&perm.ResourceType, &perm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		perms = append(perms, &perm)
	}
	return perms, nil
}

type RolePermissionRepository struct {
	pool *pgxpool.Pool
}

func (r *RolePermissionRepository) Assign(ctx context.Context, roleID, permissionID uuid.UUID) error {
	const op = "postgres.RolePermissionRepository.Assign"
	q := `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, q, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *RolePermissionRepository) Revoke(ctx context.Context, roleID, permissionID uuid.UUID) error {
	const op = "postgres.RolePermissionRepository.Revoke"
	q := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.pool.Exec(ctx, q, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *RolePermissionRepository) GetByRole(ctx context.Context, roleID uuid.UUID) ([]*domain.Permission, error) {
	const op = "postgres.RolePermissionRepository.GetByRole"
	q := `
		SELECT p.id, p.code, p.name, p.description, p.resource_type, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.resource_type, p.name
	`
	rows, err := r.pool.Query(ctx, q, roleID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		var perm domain.Permission
		if err := rows.Scan(
			&perm.ID, &perm.Code, &perm.Name, &perm.Description,
			&perm.ResourceType, &perm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		perms = append(perms, &perm)
	}
	return perms, nil
}

func (r *RolePermissionRepository) DeleteByRole(ctx context.Context, roleID uuid.UUID) error {
	const op = "postgres.RolePermissionRepository.DeleteByRole"
	q := `DELETE FROM role_permissions WHERE role_id = $1`
	_, err := r.pool.Exec(ctx, q, roleID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type UserRoleRepository struct {
	pool *pgxpool.Pool
}

func (r *UserRoleRepository) Create(ctx context.Context, ur *domain.UserRole) error {
	const op = "postgres.UserRoleRepository.Create"
	q := `
		INSERT INTO user_roles (id, user_id, role_id, org_id, branch_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, q,
		ur.ID, ur.UserID, ur.RoleID, ur.OrgID, ur.BranchID, ur.GrantedBy, ur.GrantedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserRole, error) {
	const op = "postgres.UserRoleRepository.GetByID"
	q := `
		SELECT id, user_id, role_id, org_id, branch_id, granted_by, granted_at, revoked_at, revoked_by
		FROM user_roles WHERE id = $1
	`
	var ur domain.UserRole
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&ur.ID, &ur.UserID, &ur.RoleID, &ur.OrgID, &ur.BranchID,
		&ur.GrantedBy, &ur.GrantedAt, &ur.RevokedAt, &ur.RevokedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &ur, nil
}

func (r *UserRoleRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	const op = "postgres.UserRoleRepository.GetByUser"
	q := `
		SELECT id, user_id, role_id, org_id, branch_id, granted_by, granted_at, revoked_at, revoked_by
		FROM user_roles WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY granted_at DESC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var roles []*domain.UserRole
	for rows.Next() {
		var ur domain.UserRole
		if err := rows.Scan(
			&ur.ID, &ur.UserID, &ur.RoleID, &ur.OrgID, &ur.BranchID,
			&ur.GrantedBy, &ur.GrantedAt, &ur.RevokedAt, &ur.RevokedBy,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		roles = append(roles, &ur)
	}
	return roles, nil
}

func (r *UserRoleRepository) GetByBranch(ctx context.Context, branchID uuid.UUID) ([]*domain.UserRole, error) {
	const op = "postgres.UserRoleRepository.GetByBranch"
	q := `
		SELECT id, user_id, role_id, org_id, branch_id, granted_by, granted_at, revoked_at, revoked_by
		FROM user_roles WHERE branch_id = $1 AND revoked_at IS NULL
		ORDER BY granted_at DESC
	`
	rows, err := r.pool.Query(ctx, q, branchID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var roles []*domain.UserRole
	for rows.Next() {
		var ur domain.UserRole
		if err := rows.Scan(
			&ur.ID, &ur.UserID, &ur.RoleID, &ur.OrgID, &ur.BranchID,
			&ur.GrantedBy, &ur.GrantedAt, &ur.RevokedAt, &ur.RevokedBy,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		roles = append(roles, &ur)
	}
	return roles, nil
}

func (r *UserRoleRepository) Revoke(ctx context.Context, id, orgID, revokedBy uuid.UUID) error {
	const op = "postgres.UserRoleRepository.Revoke"
	q := `
		UPDATE user_roles SET revoked_at = NOW(), revoked_by = $3
		WHERE id = $1 AND org_id = $2 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, q, id, orgID, revokedBy)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserRoleRepository) RevokeByUserAndRole(ctx context.Context, userID, roleID, orgID, revokedBy uuid.UUID) error {
	const op = "postgres.UserRoleRepository.RevokeByUserAndRole"
	q := `
		UPDATE user_roles SET revoked_at = NOW(), revoked_by = $4
		WHERE user_id = $1 AND role_id = $2 AND org_id = $3 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, q, userID, roleID, orgID, revokedBy)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type AuditLogRepository struct {
	pool *pgxpool.Pool
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	const op = "postgres.AuditLogRepository.Create"
	q := `
		INSERT INTO audit_logs (
			id, org_id, actor_id, actor_email, action, resource_type,
			resource_id, old_value, new_value, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, q,
		log.ID, log.OrgID, log.ActorID, log.ActorEmail, log.Action,
		log.ResourceType, log.ResourceID, log.OldValue, log.NewValue,
		log.IPAddress, log.UserAgent, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error) {
	const op = "postgres.AuditLogRepository.List"

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]interface{}, 0)
	argIndex := 1

	whereClause := ""
	if filter.OrgID != nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIndex)
		args = append(args, *filter.OrgID)
		argIndex++
	}
	if filter.ActorID != nil {
		whereClause += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, *filter.ActorID)
		argIndex++
	}
	if filter.Action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filter.Action)
		argIndex++
	}
	if filter.ResourceType != "" {
		whereClause += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, filter.ResourceType)
		argIndex++
	}
	if filter.ResourceID != "" {
		whereClause += fmt.Sprintf(" AND resource_id = $%d", argIndex)
		args = append(args, filter.ResourceID)
		argIndex++
	}
	if filter.FromDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.FromDate)
		argIndex++
	}
	if filter.ToDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.ToDate)
		argIndex++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE 1=1%s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: count: %w", op, err)
	}

	q := fmt.Sprintf(`
		SELECT id, org_id, actor_id, actor_email, action, resource_type,
			resource_id, old_value, new_value, ip_address, user_agent, created_at
		FROM audit_logs WHERE 1=1%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		var log domain.AuditLog
		if err := rows.Scan(
			&log.ID, &log.OrgID, &log.ActorID, &log.ActorEmail,
			&log.Action, &log.ResourceType, &log.ResourceID,
			&log.OldValue, &log.NewValue, &log.IPAddress,
			&log.UserAgent, &log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("%s: scan: %w", op, err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}
