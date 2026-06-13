package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BranchRepository struct {
	pool *pgxpool.Pool
}

func (r *BranchRepository) Create(ctx context.Context, b *domain.Branch) error {
	const op = "postgres.BranchRepository.Create"
	_, err := r.pool.Exec(ctx, `
		INSERT INTO branches (id, org_id, branch_code, name, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		b.ID, b.OrgID, b.BranchCode, b.Name, b.Active, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchRepository) Update(ctx context.Context, b *domain.Branch) error {
	const op = "postgres.BranchRepository.Update"
	_, err := r.pool.Exec(ctx, `
		UPDATE branches SET name = $1, is_active = $2, updated_at = $3
		WHERE id = $4 AND org_id = $5`,
		b.Name, b.Active, b.UpdatedAt, b.ID, b.OrgID,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	const op = "postgres.BranchRepository.Delete"
	_, err := r.pool.Exec(ctx, `DELETE FROM branches WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BranchRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Branch, error) {
	const op = "postgres.BranchRepository.GetByID"
	var b domain.Branch
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, branch_code, name, is_active, updated_at
		FROM branches WHERE id = $1 AND org_id = $2`,
		id, orgID,
	).Scan(&b.ID, &b.OrgID, &b.BranchCode, &b.Name, &b.Active, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &b, nil
}

func (r *BranchRepository) GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.Branch, error) {
	const op = "postgres.BranchRepository.GetByCode"
	var b domain.Branch
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, branch_code, name, is_active, updated_at
		FROM branches WHERE org_id = $1 AND branch_code = $2`,
		orgID, code,
	).Scan(&b.ID, &b.OrgID, &b.BranchCode, &b.Name, &b.Active, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &b, nil
}

func (r *BranchRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Branch, error) {
	const op = "postgres.BranchRepository.ListByOrg"
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, branch_code, name, is_active, updated_at
		FROM branches WHERE org_id = $1
		ORDER BY branch_code`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*domain.Branch
	for rows.Next() {
		var b domain.Branch
		if err := rows.Scan(&b.ID, &b.OrgID, &b.BranchCode, &b.Name, &b.Active, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

// ── UserBranchAssignmentRepository write methods ─────────────────────────────

func (r *UserBranchAssignmentRepository) Create(ctx context.Context, uba *domain.UserBranchAssignment) error {
	const op = "postgres.UserBranchAssignmentRepository.Create"
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_branch_assignments
		    (id, org_id, user_id, branch_id, is_primary, assigned_by, assigned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, branch_id) DO NOTHING`,
		uba.ID, uba.OrgID, uba.UserID, uba.BranchID, uba.IsPrimary,
		uba.AssignedBy.String(), uba.AssignedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserBranchAssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "postgres.UserBranchAssignmentRepository.Delete"
	_, err := r.pool.Exec(ctx, `DELETE FROM user_branch_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *UserBranchAssignmentRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserBranchAssignment, error) {
	const op = "postgres.UserBranchAssignmentRepository.GetByUser"
	rows, err := r.pool.Query(ctx, `
		SELECT uba.id, uba.user_id, uba.branch_id, uba.is_primary,
		       COALESCE(b.branch_code, ''), COALESCE(b.name, ''),
		       uba.org_id, uba.assigned_by, uba.assigned_at,
		       uba.revoked_at, uba.revoked_by
		FROM user_branch_assignments uba
		JOIN branches b ON b.id = uba.branch_id
		WHERE uba.user_id = $1
		ORDER BY uba.is_primary DESC, uba.assigned_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*domain.UserBranchAssignment
	for rows.Next() {
		a, err := scanUserBranchAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *UserBranchAssignmentRepository) GetByBranch(ctx context.Context, branchID uuid.UUID) ([]*domain.UserBranchAssignment, error) {
	const op = "postgres.UserBranchAssignmentRepository.GetByBranch"
	rows, err := r.pool.Query(ctx, `
		SELECT uba.id, uba.user_id, uba.branch_id, uba.is_primary,
		       COALESCE(b.branch_code, ''), COALESCE(b.name, ''),
		       uba.org_id, uba.assigned_by, uba.assigned_at,
		       uba.revoked_at, uba.revoked_by
		FROM user_branch_assignments uba
		JOIN branches b ON b.id = uba.branch_id
		WHERE uba.branch_id = $1
		ORDER BY uba.assigned_at DESC`,
		branchID,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []*domain.UserBranchAssignment
	for rows.Next() {
		a, err := scanUserBranchAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *UserBranchAssignmentRepository) GetPrimary(ctx context.Context, userID uuid.UUID) (*domain.UserBranchAssignment, error) {
	return r.GetPrimaryByUserID(ctx, userID)
}

func (r *UserBranchAssignmentRepository) Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	const op = "postgres.UserBranchAssignmentRepository.Revoke"
	var revokedByStr *string
	if revokedBy != uuid.Nil {
		s := revokedBy.String()
		revokedByStr = &s
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE user_branch_assignments
		SET revoked_at = NOW(), revoked_by = $2
		WHERE id = $1 AND revoked_at IS NULL`,
		id, revokedByStr,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// scanUserBranchAssignment scans a full row including org_id, assigned_by, timestamps.
func scanUserBranchAssignment(rows interface {
	Scan(dest ...any) error
}) (*domain.UserBranchAssignment, error) {
	var a domain.UserBranchAssignment
	var assignedByStr string
	var revokedByStr *string
	err := rows.Scan(
		&a.ID, &a.UserID, &a.BranchID, &a.IsPrimary,
		&a.BranchCode, &a.BranchName,
		&a.OrgID, &assignedByStr, &a.AssignedAt,
		&a.RevokedAt, &revokedByStr,
	)
	if err != nil {
		return nil, err
	}
	a.AssignedBy, _ = uuid.Parse(assignedByStr)
	if revokedByStr != nil {
		id, _ := uuid.Parse(*revokedByStr)
		a.RevokedBy = &id
	}
	return &a, nil
}

func NewBranchRepository(pool *pgxpool.Pool) *BranchRepository {
	return &BranchRepository{pool: pool}
}
