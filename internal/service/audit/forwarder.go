package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/rs/zerolog"
)

// Audit7Forwarder forwards admin/workflow audit entries to the central audit7
// service (OJK audit store) as MODIFICATION events, in addition to auth7's own
// audit_logs table. It authenticates via the X-Service-Key header (matching
// audit7's SERVICE_KEY). A nil forwarder is a safe no-op.
type Audit7Forwarder struct {
	baseURL    string
	serviceKey string
	http       *http.Client
	logger     zerolog.Logger
}

// NewAudit7Forwarder returns a forwarder, or nil when baseURL is empty or an
// unexpanded "${VAR}" placeholder (audit7 forwarding disabled).
func NewAudit7Forwarder(baseURL, serviceKey string, logger zerolog.Logger) *Audit7Forwarder {
	if baseURL == "" || strings.HasPrefix(baseURL, "${") {
		return nil
	}
	return &Audit7Forwarder{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		http:       &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

// forward sends one audit log to audit7, fire-and-forget. Errors are logged.
// in carries context (branch/session/correlation) not persisted in audit_logs.
func (f *Audit7Forwarder) forward(log *domain.AuditLog, in LogInput) {
	if f == nil {
		return
	}

	actorID := log.ActorID.String()
	display := log.ActorEmail
	if display == "" {
		display = actorID
	}

	event := map[string]any{
		"event_id":       uuid.NewString(),
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL+"/v1/audit/events", bytes.NewReader(body))
		if err != nil {
			f.logger.Warn().Err(err).Msg("audit7 forward: new request failed")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if f.serviceKey != "" {
			req.Header.Set("X-Service-Key", f.serviceKey)
		}

		resp, err := f.http.Do(req)
		if err != nil {
			f.logger.Warn().Err(err).Str("action", log.Action).Msg("audit7 forward: request failed")
			return
		}
		defer resp.Body.Close() //nolint:errcheck

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
		case resp.StatusCode == http.StatusConflict: // duplicate event_id — already ingested
		default:
			f.logger.Warn().Int("status", resp.StatusCode).Str("action", log.Action).Msg("audit7 forward: non-2xx")
		}
	}()
}
