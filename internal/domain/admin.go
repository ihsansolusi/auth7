package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	ActorID      uuid.UUID  `json:"actor_id"`
	ActorEmail   string     `json:"actor_email"`
	Action       string     `json:"action"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id,omitempty"`
	OldValue     JSON       `json:"old_value,omitempty"`
	NewValue     JSON       `json:"new_value,omitempty"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AuditLogFilter struct {
	OrgID        *uuid.UUID
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	BranchID     *uuid.UUID
	FromDate     *time.Time
	ToDate       *time.Time
	Limit        int
	Offset       int
}
