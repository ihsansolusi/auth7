package domain

import (
	"time"

	"github.com/google/uuid"
)

type MFAConfig struct {
	ID                    uuid.UUID  `json:"id"`
	UserID                uuid.UUID  `json:"user_id"`
	TOTPSecretEncrypted   []byte     `json:"-"`
	TOTPSecretIV          []byte     `json:"-"`
	IsTOTPEnabled         bool       `json:"is_totp_enabled"`
	IsEmailOTPEnabled     bool       `json:"is_email_otp_enabled"`
	IsBackupCodesEnabled  bool       `json:"is_backup_codes_enabled"`
	BackupCodesHash       []string   `json:"-"`
	MFAEnabledAt          *time.Time `json:"mfa_enabled_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (m *MFAConfig) HasTOTP() bool {
	return m.IsTOTPEnabled && len(m.TOTPSecretEncrypted) > 0
}

func (m *MFAConfig) HasEmailOTP() bool {
	return m.IsEmailOTPEnabled
}

func (m *MFAConfig) HasBackupCodes() bool {
	return m.IsBackupCodesEnabled && len(m.BackupCodesHash) > 0
}

func (m *MFAConfig) IsFullyEnabled() bool {
	return (m.IsTOTPEnabled && len(m.TOTPSecretEncrypted) > 0) ||
		m.IsEmailOTPEnabled ||
		(m.IsBackupCodesEnabled && len(m.BackupCodesHash) > 0)
}

type EmailOTPCode struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Code      string     `json:"code"`
	Purpose   string     `json:"purpose"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	Attempts  int        `json:"attempts"`
	CreatedAt time.Time  `json:"created_at"`
}

func (e *EmailOTPCode) IsValid() bool {
	if e.UsedAt != nil {
		return false
	}
	return time.Now().Before(e.ExpiresAt)
}

func (e *EmailOTPCode) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

const (
	OTPPurposeLogin       = "mfa_login"
	OTPPurposeEnroll      = "mfa_enroll"
	OTPPurposeChangeEmail = "change_email"
)