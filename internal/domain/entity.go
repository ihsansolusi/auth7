package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusCreated              UserStatus = "created"
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusActive              UserStatus = "active"
	UserStatusInactive            UserStatus = "inactive"
	UserStatusLocked             UserStatus = "locked"
	UserStatusSuspended           UserStatus = "suspended"
	UserStatusDeleted            UserStatus = "deleted"
)

type MFAMethod string

const (
	MFAMethodNone      MFAMethod = ""
	MFAMethodTOTP      MFAMethod = "totp"
	MFAMethodEmailOTP  MFAMethod = "email_otp"
)

type User struct {
	ID                     uuid.UUID  `json:"id"`
	OrgID                  uuid.UUID  `json:"org_id"`
	Username               string     `json:"username"`
	Email                  string     `json:"email"`
	FullName               string     `json:"full_name"`
	Status                 UserStatus `json:"status"`
	EmailVerified          bool       `json:"email_verified"`
	MFAEnabled             bool       `json:"mfa_enabled"`
	MFAMethod             MFAMethod  `json:"mfa_method"`
	MFAResetRequired       bool       `json:"mfa_reset_required"`
	RequirePasswordChange  bool       `json:"require_password_change"`
	FailedLoginAttempts    int        `json:"failed_login_attempts"`
	LockedUntil            *time.Time `json:"locked_until,omitempty"`
	LastLoginAt            *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP            string     `json:"last_login_ip,omitempty"`
	PasswordChangedAt      *time.Time `json:"password_changed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
	CreatedBy              *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy              *uuid.UUID `json:"updated_by,omitempty"`
}

func (u *User) Validate() error {
	var errs []string

	if u.OrgID == uuid.Nil {
		errs = append(errs, "org_id is required")
	}

	if len(u.Username) < 3 || len(u.Username) > 100 {
		errs = append(errs, "username must be between 3 and 100 characters")
	}

	if !isValidEmail(NormalizeEmail(u.Email)) {
		errs = append(errs, "invalid email format")
	}

	if len(u.FullName) < 1 || len(u.FullName) > 255 {
		errs = append(errs, "full_name must be between 1 and 255 characters")
	}

	switch u.Status {
	case UserStatusCreated, UserStatusPendingVerification,
		UserStatusActive, UserStatusInactive, UserStatusLocked,
		UserStatusSuspended, UserStatusDeleted:
	default:
		errs = append(errs, fmt.Sprintf("invalid user status: %s", u.Status))
	}

	if len(errs) > 0 {
		return fmt.Errorf("user validation failed: %s", errs)
	}
	return nil
}

func (u *User) CanLogin() bool {
	return u.Status == UserStatusActive && !u.IsLocked()
}

func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

type UserCredential struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	CredentialType string     `json:"credential_type"`
	SecretHash     string     `json:"secret_hash"`
	Version        int        `json:"version"`
	IsCurrent      bool       `json:"is_current"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

const CredentialTypePassword = "password"
const CredentialTypeAPIKey = "api_key"

type UserAttribute struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Session struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	OrgID        uuid.UUID  `json:"org_id"`
	ClientID     string     `json:"client_id,omitempty"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	DeviceInfo   JSON       `json:"device_info,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   time.Time  `json:"last_used_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    *uuid.UUID `json:"revoked_by,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

type JSON map[string]interface{}

type VerificationToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

const TokenTypeEmailVerification = "email_verification"
const TokenTypePasswordRecovery = "password_recovery"

func (t *VerificationToken) IsValid() bool {
	if t.UsedAt != nil {
		return false
	}
	return time.Now().Before(t.ExpiresAt)
}

type PasswordPolicy struct {
	MinLength           int  `json:"min_length"`
	RequireUppercase    bool `json:"require_uppercase"`
	RequireLowercase    bool `json:"require_lowercase"`
	RequireNumber       bool `json:"require_number"`
	RequireSymbol       bool `json:"require_symbol"`
	MaxAgeDays          int  `json:"max_age_days"`
	HistoryCount        int  `json:"history_count"`
	PasswordCannotContainUsername bool `json:"password_cannot_contain_username"`
	PasswordCannotContainEmail     bool `json:"password_cannot_contain_email"`
}

func (p *PasswordPolicy) Validate(password, username, email string) error {
	var errs []string

	if len(password) < p.MinLength {
		errs = append(errs, fmt.Sprintf("password must be at least %d characters", p.MinLength))
	}

	if p.RequireUppercase && !containsUppercase(password) {
		errs = append(errs, "password must contain at least one uppercase letter")
	}

	if p.RequireLowercase && !containsLowercase(password) {
		errs = append(errs, "password must contain at least one lowercase letter")
	}

	if p.RequireNumber && !containsNumber(password) {
		errs = append(errs, "password must contain at least one number")
	}

	if p.RequireSymbol && !containsSymbol(password) {
		errs = append(errs, "password must contain at least one symbol")
	}

	if p.PasswordCannotContainUsername && containsString(password, username) {
		errs = append(errs, "password cannot contain username")
	}

	if p.PasswordCannotContainEmail && containsString(password, email) {
		errs = append(errs, "password cannot contain email")
	}

	if len(errs) > 0 {
		return fmt.Errorf("password validation failed: %s", errs)
	}
	return nil
}

func containsUppercase(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func containsLowercase(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func containsNumber(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func containsSymbol(s string) bool {
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			if c != ' ' {
				return true
			}
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:                       8,
	RequireUppercase:               true,
	RequireLowercase:               true,
	RequireNumber:                  true,
	RequireSymbol:                  false,
	MaxAgeDays:                     90,
	HistoryCount:                   5,
	PasswordCannotContainUsername:  true,
	PasswordCannotContainEmail:     true,
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type Organization struct {
	ID        uuid.UUID  `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Domain    string     `json:"domain"`
	Status    string     `json:"status"`
	Settings  JSON       `json:"settings"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type BranchType struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	Code            string    `json:"code"`
	Label           string    `json:"label"`
	ShortCode       string    `json:"short_code"`
	Level           int       `json:"level"`
	IsOperational   bool      `json:"is_operational"`
	CanHaveChildren bool      `json:"can_have_children"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Branch struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	BranchTypeID uuid.UUID  `json:"branch_type_id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	Address      string     `json:"address"`
	Phone        string     `json:"phone"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type BranchHierarchy struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	ChildID   uuid.UUID  `json:"child_id"`
	Path      string     `json:"path"`
	Depth     int        `json:"depth"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserBranchAssignment struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	BranchID   uuid.UUID `json:"branch_id"`
	IsPrimary  bool      `json:"is_primary"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
