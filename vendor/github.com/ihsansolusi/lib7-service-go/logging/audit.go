package logging

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// AuditEvent holds all fields that must be logged for an audit entry.
type AuditEvent struct {
	RequestID string
	TraceID   string
	ActorID   string // user_id from token
	ActorType string // "user" | "service" | "system"
	BranchID  string
	Action    string // e.g. "transfer.initiate"
	Resource  string // e.g. "transfer:uuid"
	Result    string // "success" | "failure"
	Reason    string // failure reason, if any
	IP        string
	Timestamp time.Time
}

// AuditLogger wraps a zerolog.Logger dedicated to the audit stream.
type AuditLogger struct {
	logger zerolog.Logger
}

// NewAuditLoggerWrapper wraps an audit zerolog.Logger (created via NewAuditLogger)
// into an AuditLogger that exposes a structured Log method.
func NewAuditLoggerWrapper(logger zerolog.Logger) *AuditLogger {
	return &AuditLogger{logger: logger}
}

// Log writes a structured audit entry. If Timestamp is zero it is set to now.
func (al *AuditLogger) Log(ctx context.Context, event AuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	al.logger.Info().
		Str("request_id", event.RequestID).
		Str("trace_id", event.TraceID).
		Str("actor_id", event.ActorID).
		Str("actor_type", event.ActorType).
		Str("branch_id", event.BranchID).
		Str("action", event.Action).
		Str("resource", event.Resource).
		Str("result", event.Result).
		Str("reason", event.Reason).
		Str("ip", event.IP).
		Time("timestamp", event.Timestamp).
		Msg("audit")
}
