package nats

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/ihsansolusi/auth7/internal/service/opacache"
)

type PolicyUpdateHandler struct {
	cache  *opacache.Cache
	logger zerolog.Logger
}

func NewPolicyUpdateHandler(cache *opacache.Cache, logger zerolog.Logger) *PolicyUpdateHandler {
	return &PolicyUpdateHandler{
		cache:  cache,
		logger: logger,
	}
}

func (h *PolicyUpdateHandler) HandleParamUpdated(data []byte) error {
	const op = "messaging.HandleParamUpdated"

	var event PolicyParamUpdatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("%s: unmarshal event: %w", op, err)
	}

	// Prefix-invalidate so scope-suffixed keys are cleared too: the fetch-through
	// consumers key by "opa:<org>:<param>:<scope>" (e.g.
	// "opa:<org>:operational_hours:<branch>"), so an exact "opa:<org>:<param>"
	// delete would miss them. The prefix covers both the bare and scoped forms.
	prefix := fmt.Sprintf("opa:%s:%s", event.OrgID, event.ParameterName)
	h.cache.InvalidateByPrefix(prefix)

	h.logger.Info().
		Str("op", op).
		Str("org_id", event.OrgID).
		Str("parameter", event.ParameterName).
		Msg("OPA cache invalidated for policy param update")

	return nil
}

func (h *PolicyUpdateHandler) HandleParamDeleted(data []byte) error {
	const op = "messaging.HandleParamDeleted"

	var event PolicyParamDeletedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("%s: unmarshal event: %w", op, err)
	}

	prefix := fmt.Sprintf("opa:%s:", event.OrgID)
	h.cache.InvalidateByPrefix(prefix)

	h.logger.Info().
		Str("op", op).
		Str("org_id", event.OrgID).
		Str("parameter", event.ParameterName).
		Msg("OPA cache invalidated for policy param delete")

	return nil
}
