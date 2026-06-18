package audit

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/rs/zerolog"
)

// auditIngestSubject is the JetStream subject audit7 consumes durably for
// forwarded MODIFICATION events (covered by the AUDIT7_EVENTS stream).
const auditIngestSubject = "audit7.ingest.auth7"

// AuditPublisher publishes a pre-marshalled audit event durably to JetStream
// (implemented by nats.EventPublisher.PublishAudit). Decoupled via interface so
// the audit package does not import the messaging package.
type AuditPublisher interface {
	PublishAudit(subject string, data []byte, msgID string) error
}

// Audit7Forwarder forwards admin/workflow audit entries to the central audit7
// service (OJK system of record) as MODIFICATION events, in addition to auth7's
// own audit_logs table. Delivery is durable: it publishes to JetStream with a
// persist-ack, so events survive audit7 downtime. A nil forwarder is a no-op.
type Audit7Forwarder struct {
	pub    AuditPublisher
	logger zerolog.Logger
}

// NewAudit7Forwarder returns a forwarder, or nil when no publisher is available
// (audit7 forwarding disabled).
func NewAudit7Forwarder(pub AuditPublisher, logger zerolog.Logger) *Audit7Forwarder {
	if pub == nil {
		return nil
	}
	return &Audit7Forwarder{pub: pub, logger: logger}
}

// forward publishes one audit log to audit7's JetStream ingest subject.
// in carries context (branch/session/correlation) not persisted in audit_logs.
// The publish is synchronous (persist-ack) but wrapped in a goroutine so it
// never blocks the request; failures are logged (the local audit_logs row
// remains the authoritative fallback).
func (f *Audit7Forwarder) forward(log *domain.AuditLog, in LogInput) {
	if f == nil {
		return
	}

	actorID := log.ActorID.String()
	display := log.ActorEmail
	if display == "" {
		display = actorID
	}

	eventID := uuid.NewString()
	event := map[string]any{
		"event_id":       eventID,
		"occurred_at":    log.CreatedAt.UTC(),
		"org_id":         log.OrgID.String(),
		"actor":          map[string]any{"type": "user", "id": actorID, "display": display},
		"event_category": "MODIFICATION",
		"action":         log.Action,
		"resource":       map[string]any{"type": log.ResourceType, "id": log.ResourceID},
		"result":         "SUCCESS",
		"severity":       "INFO",
		"channel":        "BFF",
		"source_app":     "auth7",
		"module":         "access-management",
	}
	if log.IPAddress != "" {
		event["ip_address"] = log.IPAddress
	}
	if log.UserAgent != "" {
		event["user_agent"] = log.UserAgent
	}
	if in.BranchID != "" {
		event["branch_id"] = in.BranchID
	}
	if in.BranchCode != "" {
		event["branch_code"] = in.BranchCode
	}
	if in.SessionID != "" {
		event["session_id"] = in.SessionID
	}
	if in.CorrelationID != "" {
		event["correlation_id"] = in.CorrelationID
	}
	if len(log.OldValue) > 0 {
		event["before_snapshot"] = log.OldValue
	}
	if len(log.NewValue) > 0 {
		event["after_snapshot"] = log.NewValue
	}

	body, err := json.Marshal(event)
	if err != nil {
		f.logger.Warn().Err(err).Msg("audit7 forward: marshal failed")
		return
	}

	go func() {
		if err := f.pub.PublishAudit(auditIngestSubject, body, eventID); err != nil {
			f.logger.Warn().Err(err).Str("action", log.Action).Msg("audit7 forward: jetstream publish failed (local audit_logs retains record)")
		}
	}()
}
