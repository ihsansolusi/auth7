package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const opLogAudit = "audit.Service.Log"

type Store interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error)
}

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) Create(ctx context.Context, log *domain.AuditLog) error {
	const op = "audit.PGStore.Create"
	q := `
		INSERT INTO audit_logs (
			id, org_id, actor_id, actor_email, action, resource_type,
			resource_id, old_value, new_value, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.pool.Exec(ctx, q,
		log.ID, log.OrgID, log.ActorID, log.ActorEmail, log.Action,
		log.ResourceType, log.ResourceID, log.OldValue, log.NewValue,
		log.IPAddress, log.UserAgent, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *PGStore) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error) {
	const op = "audit.PGStore.List"

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]interface{}, 0)
	argIndex := 1

	whereClause := ""
	if filter.OrgID != nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIndex)
		args = append(args, *filter.OrgID)
		argIndex++
	}
	if filter.ActorID != nil {
		whereClause += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, *filter.ActorID)
		argIndex++
	}
	if filter.Action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filter.Action)
		argIndex++
	}
	if filter.ResourceType != "" {
		whereClause += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, filter.ResourceType)
		argIndex++
	}
	if filter.ResourceID != "" {
		whereClause += fmt.Sprintf(" AND resource_id = $%d", argIndex)
		args = append(args, filter.ResourceID)
		argIndex++
	}
	if filter.FromDate != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.FromDate)
		argIndex++
	}
	if filter.ToDate != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.ToDate)
		argIndex++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE 1=1%s", whereClause)
	var total int
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: count: %w", op, err)
	}

	q := fmt.Sprintf(`
		SELECT id, org_id, actor_id, actor_email, action, resource_type,
			resource_id, old_value, new_value, ip_address::text, user_agent, created_at
		FROM audit_logs WHERE 1=1%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		var log domain.AuditLog
		if err := rows.Scan(
			&log.ID, &log.OrgID, &log.ActorID, &log.ActorEmail,
			&log.Action, &log.ResourceType, &log.ResourceID,
			&log.OldValue, &log.NewValue, &log.IPAddress,
			&log.UserAgent, &log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("%s: scan: %w", op, err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

type Service struct {
	store     Store
	forwarder *Audit7Forwarder
	logger    zerolog.Logger
}

func NewService(store Store) *Service {
	return &Service{store: store, logger: zerolog.Nop()}
}

// SetForwarder enables forwarding of recorded audit logs to the central audit7
// service. Passing nil (or never calling it) keeps forwarding disabled. The
// forwarder's logger is reused for surfacing async audit failures.
func (s *Service) SetForwarder(f *Audit7Forwarder) {
	s.forwarder = f
	if f != nil {
		s.logger = f.logger
	}
}

type LogInput struct {
	OrgID         uuid.UUID
	ActorID       uuid.UUID
	ActorEmail    string
	Action        string
	ResourceType  string
	ResourceID    string
	OldValue      domain.JSON
	NewValue      domain.JSON
	IPAddress     string
	UserAgent     string
	BranchID      string
	BranchCode    string
	SessionID     string
	CorrelationID string
}

func (s *Service) Log(ctx context.Context, input LogInput) error {
	if s.store == nil {
		return nil
	}

	// audit_logs.old_value / new_value are JSONB NOT NULL DEFAULT '{}'. Operations
	// without a prior/after state (create, delete, set_roles, set_branches) pass
	// nil — coerce to an empty object so the INSERT doesn't violate NOT NULL.
	oldValue := input.OldValue
	if oldValue == nil {
		oldValue = domain.JSON{}
	}
	newValue := input.NewValue
	if newValue == nil {
		newValue = domain.JSON{}
	}

	log := &domain.AuditLog{
		ID:           uuid.New(),
		OrgID:        input.OrgID,
		ActorID:      input.ActorID,
		ActorEmail:   input.ActorEmail,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		OldValue:     oldValue,
		NewValue:     newValue,
		IPAddress:    input.IPAddress,
		UserAgent:    input.UserAgent,
		CreatedAt:    time.Now(),
	}

	if err := s.store.Create(ctx, log); err != nil {
		return fmt.Errorf("%s: %w", opLogAudit, err)
	}

	// Mirror to the central audit7 store (best-effort, fire-and-forget).
	// input carries context (branch/session/correlation) not stored locally.
	s.forwarder.forward(log, input)
	return nil
}

func (s *Service) LogAsync(input LogInput) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Log(ctx, input); err != nil {
			// Audit is compliance-critical — never drop a failure silently.
			s.logger.Error().Err(err).Str("action", input.Action).
				Str("resource_id", input.ResourceID).Msg("async audit log failed")
		}
	}()
}

func (s *Service) Query(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int, error) {
	if s.store == nil {
		return []*domain.AuditLog{}, 0, nil
	}
	return s.store.List(ctx, filter)
}
