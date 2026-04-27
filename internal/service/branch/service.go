package branch

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
)

const (
	opBranchTypeCreate = "branch.Service.CreateBranchType"
	opBranchTypeUpdate = "branch.Service.UpdateBranchType"
	opBranchTypeDelete = "branch.Service.DeleteBranchType"
	opBranchTypeGet    = "branch.Service.GetBranchType"

	opBranchCreate = "branch.Service.CreateBranch"
	opBranchUpdate  = "branch.Service.UpdateBranch"
	opBranchDelete  = "branch.Service.DeleteBranch"
	opBranchGet     = "branch.Service.GetBranch"

	opAssignUser   = "branch.Service.AssignUserToBranch"
	opSwitchBranch = "branch.Service.SwitchBranch"
)

type BranchTypeStore interface {
	Create(ctx context.Context, bt *domain.BranchType) error
	Update(ctx context.Context, bt *domain.BranchType) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.BranchType, error)
	GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.BranchType, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.BranchType, error)
}

type BranchStore interface {
	Create(ctx context.Context, b *domain.Branch) error
	Update(ctx context.Context, b *domain.Branch) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Branch, error)
	GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.Branch, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Branch, error)
	ListByParent(ctx context.Context, parentID uuid.UUID) ([]*domain.Branch, error)
	ListByType(ctx context.Context, branchTypeID uuid.UUID) ([]*domain.Branch, error)
}

type HierarchyStore interface {
	Create(ctx context.Context, h *domain.BranchHierarchy) error
	GetAncestors(ctx context.Context, branchID uuid.UUID) ([]*domain.Branch, error)
	GetDescendants(ctx context.Context, branchID uuid.UUID) ([]*domain.Branch, error)
}

type UserBranchStore interface {
	Create(ctx context.Context, uba *domain.UserBranchAssignment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserBranchAssignment, error)
	GetByBranch(ctx context.Context, branchID uuid.UUID) ([]*domain.UserBranchAssignment, error)
	GetPrimary(ctx context.Context, userID uuid.UUID) (*domain.UserBranchAssignment, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error
}

type Service struct {
	branchTypeStore BranchTypeStore
	branchStore     BranchStore
	hierarchyStore  HierarchyStore
	userBranchStore UserBranchStore
}

func NewService(
	branchTypeStore BranchTypeStore,
	branchStore BranchStore,
	hierarchyStore HierarchyStore,
	userBranchStore UserBranchStore,
) *Service {
	return &Service{
		branchTypeStore: branchTypeStore,
		branchStore:     branchStore,
		hierarchyStore:  hierarchyStore,
		userBranchStore: userBranchStore,
	}
}

func (s *Service) CreateBranchType(ctx context.Context, orgID uuid.UUID, params BranchTypeParams) (*domain.BranchType, error) {
	if !params.Validate() {
		return nil, ErrInvalidBranchType
	}

	existing, _ := s.branchTypeStore.GetByCode(ctx, orgID, params.Code)
	if existing != nil {
		return nil, ErrBranchTypeExists
	}

	bt := &domain.BranchType{
		ID:             uuid.New(),
		OrgID:          orgID,
		Code:           params.Code,
		Label:          params.Label,
		ShortCode:      params.ShortCode,
		Level:          params.Level,
		IsOperational:  params.IsOperational,
		CanHaveChildren: params.CanHaveChildren,
		SortOrder:      params.SortOrder,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.branchTypeStore.Create(ctx, bt); err != nil {
		return nil, fmt.Errorf("%s: %w", opBranchTypeCreate, err)
	}

	return bt, nil
}

func (s *Service) GetBranchType(ctx context.Context, id, orgID uuid.UUID) (*domain.BranchType, error) {
	bt, err := s.branchTypeStore.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opBranchTypeGet, err)
	}
	return bt, nil
}

func (s *Service) CreateBranch(ctx context.Context, orgID uuid.UUID, params BranchParams) (*domain.Branch, error) {
	if !params.Validate() {
		return nil, ErrInvalidBranch
	}

	branchType, err := s.branchTypeStore.GetByID(ctx, params.BranchTypeID, orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid branch type: %w", err)
	}

	if params.ParentID != nil {
		parent, err := s.branchStore.GetByID(ctx, *params.ParentID, orgID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent branch: %w", err)
		}
		if !branchType.CanHaveChildren {
			return nil, ErrCannotHaveChildren
		}
		_ = parent
	}

	existing, _ := s.branchStore.GetByCode(ctx, orgID, params.Code)
	if existing != nil {
		return nil, ErrBranchExists
	}

	branch := &domain.Branch{
		ID:           uuid.New(),
		OrgID:        orgID,
		BranchTypeID: params.BranchTypeID,
		ParentID:     params.ParentID,
		Code:         params.Code,
		Name:         params.Name,
		Status:       string(domain.BranchStatusActive),
		Address:      params.Address,
		Phone:        params.Phone,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.branchStore.Create(ctx, branch); err != nil {
		return nil, fmt.Errorf("%s: %w", opBranchCreate, err)
	}

	return branch, nil
}

func (s *Service) AssignUserToBranch(ctx context.Context, userID, branchID, orgID uuid.UUID, params UserBranchParams) (*domain.UserBranchAssignment, error) {
	branch, err := s.branchStore.GetByID(ctx, branchID, orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid branch: %w", err)
	}
	if !branch.IsActive() {
		return nil, ErrBranchNotActive
	}

	if params.IsPrimary {
		if err := s.clearPrimaryBranch(ctx, userID); err != nil {
			return nil, fmt.Errorf("clear primary: %w", err)
		}
	}

	uba := &domain.UserBranchAssignment{
		ID:         uuid.New(),
		UserID:     userID,
		BranchID:   branchID,
		OrgID:      orgID,
		Role:       params.Role,
		IsPrimary:  params.IsPrimary,
		AssignedBy: params.AssignedBy,
		AssignedAt: time.Now(),
	}

	if err := s.userBranchStore.Create(ctx, uba); err != nil {
		return nil, fmt.Errorf("%s: %w", opAssignUser, err)
	}

	return uba, nil
}

func (s *Service) GetUserBranches(ctx context.Context, userID uuid.UUID) ([]*domain.UserBranchAssignment, error) {
	assignments, err := s.userBranchStore.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	active := make([]*domain.UserBranchAssignment, 0)
	for _, a := range assignments {
		if a.IsActive() {
			active = append(active, a)
		}
	}

	return active, nil
}

func (s *Service) SwitchBranch(ctx context.Context, userID, newBranchID, orgID uuid.UUID) error {
	assignments, err := s.userBranchStore.GetByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", opSwitchBranch, err)
	}

	for _, a := range assignments {
		if a.BranchID == newBranchID && a.IsActive() {
			if err := s.clearPrimaryBranch(ctx, userID); err != nil {
				return fmt.Errorf("clear primary: %w", err)
			}

			a.IsPrimary = true
			return nil
		}
	}

	return ErrUserCannotAccessBranch
}

func (s *Service) clearPrimaryBranch(ctx context.Context, userID uuid.UUID) error {
	assignments, _ := s.userBranchStore.GetByUser(ctx, userID)
	for _, a := range assignments {
		if a.IsPrimary && a.IsActive() {
			a.IsPrimary = false
		}
	}
	return nil
}

type BranchTypeParams struct {
	Code           string
	Label          string
	ShortCode      string
	Level          int
	IsOperational  bool
	CanHaveChildren bool
	SortOrder      int
}

func (p BranchTypeParams) Validate() bool {
	return len(p.Code) >= 2 && len(p.Code) <= 50 && len(p.Label) >= 1
}

type BranchParams struct {
	BranchTypeID uuid.UUID
	ParentID     *uuid.UUID
	Code         string
	Name         string
	Address      string
	Phone        string
}

func (p BranchParams) Validate() bool {
	return len(p.Code) >= 2 && len(p.Code) <= 20 && len(p.Name) >= 1
}

type UserBranchParams struct {
	Role       string
	IsPrimary  bool
	AssignedBy uuid.UUID
}