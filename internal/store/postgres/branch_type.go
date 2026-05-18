package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BranchTypeRepository struct {
	pool *pgxpool.Pool
}

func (r *BranchTypeRepository) Create(ctx context.Context, bt *domain.BranchType) error {
	const op = "postgres.BranchTypeRepository.Create"
	if bt.ID == uuid.Nil {
		bt.ID = uuid.New()
	}
	now := time.Now()
	if bt.CreatedAt.IsZero() {
		bt.CreatedAt = now
	}
	bt.UpdatedAt = now
	q := `
		INSERT INTO branch_types (id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, q,
		bt.ID, bt.OrgID, bt.Code, bt.Label, bt.ShortCode, bt.Level,
		bt.IsOperational, bt.CanHaveChildren, bt.SortOrder, bt.CreatedAt, bt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchTypeRepository) Update(ctx context.Context, bt *domain.BranchType) error {
	const op = "postgres.BranchTypeRepository.Update"
	bt.UpdatedAt = time.Now()
	q := `
		UPDATE branch_types
		SET code = $3, label = $4, short_code = $5, level = $6,
		    is_operational = $7, can_have_children = $8, sort_order = $9, updated_at = $10
		WHERE id = $1 AND org_id = $2
	`
	_, err := r.pool.Exec(ctx, q,
		bt.ID, bt.OrgID, bt.Code, bt.Label, bt.ShortCode, bt.Level,
		bt.IsOperational, bt.CanHaveChildren, bt.SortOrder, bt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchTypeRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	const op = "postgres.BranchTypeRepository.Delete"
	_, err := r.pool.Exec(ctx, `DELETE FROM branch_types WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchTypeRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.BranchType, error) {
	const op = "postgres.BranchTypeRepository.GetByID"
	q := `
		SELECT id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order, created_at, updated_at
		FROM branch_types WHERE id = $1 AND org_id = $2
	`
	var bt domain.BranchType
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&bt.ID, &bt.OrgID, &bt.Code, &bt.Label, &bt.ShortCode, &bt.Level,
		&bt.IsOperational, &bt.CanHaveChildren, &bt.SortOrder, &bt.CreatedAt, &bt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &bt, nil
}

func (r *BranchTypeRepository) GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.BranchType, error) {
	const op = "postgres.BranchTypeRepository.GetByCode"
	q := `
		SELECT id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order, created_at, updated_at
		FROM branch_types WHERE org_id = $1 AND code = $2
	`
	var bt domain.BranchType
	err := r.pool.QueryRow(ctx, q, orgID, code).Scan(
		&bt.ID, &bt.OrgID, &bt.Code, &bt.Label, &bt.ShortCode, &bt.Level,
		&bt.IsOperational, &bt.CanHaveChildren, &bt.SortOrder, &bt.CreatedAt, &bt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &bt, nil
}

func (r *BranchTypeRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.BranchType, error) {
	const op = "postgres.BranchTypeRepository.ListByOrg"
	q := `
		SELECT id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order, created_at, updated_at
		FROM branch_types WHERE org_id = $1 ORDER BY sort_order, level, code
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var list []*domain.BranchType
	for rows.Next() {
		var bt domain.BranchType
		if err := rows.Scan(
			&bt.ID, &bt.OrgID, &bt.Code, &bt.Label, &bt.ShortCode, &bt.Level,
			&bt.IsOperational, &bt.CanHaveChildren, &bt.SortOrder, &bt.CreatedAt, &bt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		list = append(list, &bt)
	}
	return list, nil
}
