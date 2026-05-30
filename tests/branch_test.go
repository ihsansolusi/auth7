package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBranchTypeValidation(t *testing.T) {
	bt := &domain.BranchType{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Code:  "KC",
		Label: "Kantor Cabang",
		Level: 1,
	}

	assert.True(t, bt.ValidateCode())
	assert.Equal(t, "KC", bt.Code)
	assert.Equal(t, 1, bt.Level)
}

func TestBranchIsActive(t *testing.T) {
	activeBranch := &domain.Branch{
		ID:     uuid.New(),
		Active: true,
	}
	assert.True(t, activeBranch.IsActive())

	inactiveBranch := &domain.Branch{
		ID:     uuid.New(),
		Active: false,
	}
	assert.False(t, inactiveBranch.IsActive())
}

func TestBranchCodeValidation(t *testing.T) {
	validBranch := &domain.Branch{BranchCode: "KC-BDG-001"}
	assert.True(t, validBranch.ValidateCode())

	invalidBranch := &domain.Branch{BranchCode: "A"}
	assert.False(t, invalidBranch.ValidateCode())
}

func TestUserBranchAssignmentIsActive(t *testing.T) {
	activeAssignment := &domain.UserBranchAssignment{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		BranchID:  uuid.New(),
		RevokedAt: nil,
	}
	assert.True(t, activeAssignment.IsActive())

	revokedAssignment := &domain.UserBranchAssignment{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		BranchID:  uuid.New(),
		RevokedAt: timePtr(),
	}
	assert.False(t, revokedAssignment.IsActive())
}

func TestUserBranchRoles(t *testing.T) {
	assert.Equal(t, "teller", domain.UserBranchRoleTeller)
	assert.Equal(t, "supervisor", domain.UserBranchRoleSupervisor)
	assert.Equal(t, "manager", domain.UserBranchRoleManager)
	assert.Equal(t, "admin", domain.UserBranchRoleAdmin)
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

func timePtr() *time.Time {
	t := time.Now()
	return &t
}
