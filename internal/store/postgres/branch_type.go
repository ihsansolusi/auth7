package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BranchTypeRepository is retained for compile compatibility only.
// The branch_types table was removed in the 20260530 fresh migration set;
// branch classification now lives in the enterprise service domain.
// All methods return domain.ErrNotFound without touching the database.
type BranchTypeRepository struct {
	pool *pgxpool.Pool
}

func (r *BranchTypeRepository) Create(_ context.Context, _ *domain.BranchType) error {
	return fmt.Errorf("BranchTypeRepository.Create: %w", domain.ErrNotFound)
}

func (r *BranchTypeRepository) Update(_ context.Context, _ *domain.BranchType) error {
	return fmt.Errorf("BranchTypeRepository.Update: %w", domain.ErrNotFound)
}

func (r *BranchTypeRepository) Delete(_ context.Context, _, _ uuid.UUID) error {
	return fmt.Errorf("BranchTypeRepository.Delete: %w", domain.ErrNotFound)
}

func (r *BranchTypeRepository) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.BranchType, error) {
	return nil, fmt.Errorf("BranchTypeRepository.GetByID: %w", domain.ErrNotFound)
}

func (r *BranchTypeRepository) GetByCode(_ context.Context, _ uuid.UUID, _ string) (*domain.BranchType, error) {
	return nil, fmt.Errorf("BranchTypeRepository.GetByCode: %w", domain.ErrNotFound)
}

func (r *BranchTypeRepository) ListByOrg(_ context.Context, _ uuid.UUID) ([]*domain.BranchType, error) {
	return nil, fmt.Errorf("BranchTypeRepository.ListByOrg: %w", domain.ErrNotFound)
}
