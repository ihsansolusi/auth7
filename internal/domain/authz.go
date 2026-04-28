package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsDefault   bool       `json:"is_default"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (r *Role) Validate() error {
	if r.OrgID == uuid.Nil {
		return fmt.Errorf("org_id is required")
	}
	if len(r.Name) < 2 || len(r.Name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	return nil
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ResourceType string   `json:"resource_type"`
	CreatedAt   time.Time `json:"created_at"`
}

func (p *Permission) Validate() error {
	if len(p.Code) < 2 || len(p.Code) > 100 {
		return fmt.Errorf("code must be between 2 and 100 characters")
	}
	return nil
}

type UserRole struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	BranchID   *uuid.UUID `json:"branch_id,omitempty"`
	OrgID      uuid.UUID  `json:"org_id"`
	GrantedBy  uuid.UUID  `json:"granted_by"`
	GrantedAt  time.Time  `json:"granted_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	RevokedBy  *uuid.UUID `json:"revoked_by,omitempty"`
}

func (ur *UserRole) IsActive() bool {
	return ur.RevokedAt == nil
}

type ABACPolicy struct {
	ID          uuid.UUID              `json:"id"`
	OrgID       uuid.UUID              `json:"org_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Effect      string                 `json:"effect"`
	Conditions  map[string]interface{} `json:"conditions"`
	Fields      []string               `json:"fields"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

const (
	ABACEffectAllow = "allow"
	ABACEffectDeny  = "deny"
)

func (p *ABACPolicy) Validate() error {
	if p.OrgID == uuid.Nil {
		return fmt.Errorf("org_id is required")
	}
	if len(p.Name) < 2 || len(p.Name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	if p.Effect != ABACEffectAllow && p.Effect != ABACEffectDeny {
		return fmt.Errorf("effect must be 'allow' or 'deny'")
	}
	return nil
}

type FieldMask struct {
	Field     string `json:"field"`
	MaskValue string `json:"mask_value"`
	Reason    string `json:"reason"`
}

type BranchScope string

const (
	BranchScopeOwn      BranchScope = "own"
	BranchScopeAssigned BranchScope = "assigned"
	BranchScopeAll      BranchScope = "all"
)

type AuthContext struct {
	UserID       uuid.UUID
	OrgID        uuid.UUID
	BranchID     uuid.UUID
	Roles        []string
	Permissions  []string
	BranchScope  BranchScope
	FieldMasks   []FieldMask
	Attributes   map[string]string
}

type AuthorizationResult struct {
	Allowed    bool
	Reason     string
	FieldMasks []FieldMask
}