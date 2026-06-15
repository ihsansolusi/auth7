package nats

import "time"

const (
	SubjectTokenRevoked      = "auth7.tokens.revoked"
	SubjectTokenRefreshed    = "auth7.tokens.refreshed"
	SubjectSessionCreated    = "auth7.sessions.created"
	SubjectSessionTerminated = "auth7.sessions.terminated"
	SubjectSessionRevokedAll = "auth7.sessions.revoked_all"
	SubjectBranchSwitched    = "auth7.sessions.branch_switched"
	SubjectSecurityAlert     = "auth7.security.alert"
)

type TokenRevokedEvent struct {
	TokenID   string    `json:"token_id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	RevokedBy string    `json:"revoked_by"`
	Reason    string    `json:"reason"`
	RevokedAt time.Time `json:"revoked_at"`
}

type TokenRefreshedEvent struct {
	TokenID     string    `json:"token_id"`
	OrgID       string    `json:"org_id"`
	UserID      string    `json:"user_id"`
	RefreshedAt time.Time `json:"refreshed_at"`
}

type SessionCreatedEvent struct {
	SessionID string    `json:"session_id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionTerminatedEvent struct {
	SessionID    string    `json:"session_id"`
	OrgID        string    `json:"org_id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Reason       string    `json:"reason"`
	IPAddress    string    `json:"ip_address"`
	TerminatedAt time.Time `json:"terminated_at"`
}

type SessionRevokedAllEvent struct {
	OrgID     string    `json:"org_id"`
	RevokedBy string    `json:"revoked_by"`
	RevokedAt time.Time `json:"revoked_at"`
}

type BranchSwitchedEvent struct {
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	OrgID         string    `json:"org_id"`
	OldSessionID  string    `json:"old_session_id"`
	NewSessionID  string    `json:"new_session_id"`
	OldBranchID   string    `json:"old_branch_id"`
	OldBranchCode string    `json:"old_branch_code"`
	NewBranchID   string    `json:"new_branch_id"`
	NewBranchCode string    `json:"new_branch_code"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	SwitchedAt    time.Time `json:"switched_at"`
}

type SecurityAlertType string

const (
	AlertBruteForce      SecurityAlertType = "brute_force"
	AlertNewDevice       SecurityAlertType = "new_device"
	AlertIPChange        SecurityAlertType = "ip_change"
	AlertSuspiciousLogin SecurityAlertType = "suspicious_login"
	AlertAccountLocked   SecurityAlertType = "account_locked"
)

type SecurityAlertEvent struct {
	Type      SecurityAlertType `json:"type"`
	OrgID     string            `json:"org_id"`
	UserID    string            `json:"user_id"`
	IPAddress string            `json:"ip_address"`
	Details   map[string]any    `json:"details"`
	AlertedAt time.Time         `json:"alerted_at"`
}
