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

	cacheKey := fmt.Sprintf("opa:%s:%s", event.OrgID, event.ParameterName)
	h.cache.Invalidate(cacheKey)

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
