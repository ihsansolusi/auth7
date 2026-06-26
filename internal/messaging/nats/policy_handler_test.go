package nats

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ihsansolusi/auth7/internal/service/opacache"
)

// TestHandleParamUpdated_PrefixInvalidatesScopedKeys verifies that a
// policy7.params.updated event for "operational_hours" clears both the bare and
// the scope-suffixed cache keys (the fetch-through consumers key by
// "opa:<org>:operational_hours:<scope>").
func TestHandleParamUpdated_PrefixInvalidatesScopedKeys(t *testing.T) {
	cache := opacache.NewCache(time.Minute, zerolog.Nop())
	cache.Set("opa:org-1:operational_hours", "bare")
	cache.Set("opa:org-1:operational_hours:branch-9", "scoped")
	cache.Set("opa:org-1:transaction_limit", "untouched")
	cache.Set("opa:org-2:operational_hours:branch-1", "other-org")

	h := NewPolicyUpdateHandler(cache, zerolog.Nop())

	payload, err := json.Marshal(PolicyParamUpdatedEvent{
		OrgID:         "org-1",
		ParameterName: "operational_hours",
		UpdatedAt:     time.Unix(0, 0),
	})
	require.NoError(t, err)

	require.NoError(t, h.HandleParamUpdated(payload))

	_, ok := cache.Get("opa:org-1:operational_hours")
	require.False(t, ok, "bare key should be invalidated")
	_, ok = cache.Get("opa:org-1:operational_hours:branch-9")
	require.False(t, ok, "scoped key should be invalidated")

	_, ok = cache.Get("opa:org-1:transaction_limit")
	require.True(t, ok, "other params must be untouched")
	_, ok = cache.Get("opa:org-2:operational_hours:branch-1")
	require.True(t, ok, "other orgs must be untouched")
}

func TestHandleParamDeleted_InvalidatesOrgScope(t *testing.T) {
	cache := opacache.NewCache(time.Minute, zerolog.Nop())
	cache.Set("opa:org-1:operational_hours:branch-9", "scoped")
	cache.Set("opa:org-1:transaction_limit", "x")
	cache.Set("opa:org-2:operational_hours", "other-org")

	h := NewPolicyUpdateHandler(cache, zerolog.Nop())

	payload, err := json.Marshal(PolicyParamDeletedEvent{
		OrgID:         "org-1",
		ParameterName: "operational_hours",
		DeletedAt:     time.Unix(0, 0),
	})
	require.NoError(t, err)

	require.NoError(t, h.HandleParamDeleted(payload))

	require.Equal(t, 1, cache.Len(), "all org-1 params cleared, org-2 retained")
	_, ok := cache.Get("opa:org-2:operational_hours")
	require.True(t, ok)
}
