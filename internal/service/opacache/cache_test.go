package opacache

import (
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestGetOrFetch_MissThenHit(t *testing.T) {
	c := NewCache(time.Minute, zerolog.Nop())

	calls := 0
	fetch := func() (interface{}, error) {
		calls++
		return "value", nil
	}

	// Miss: fetch is invoked and the result cached.
	v, err := c.GetOrFetch("opa:org:operational_hours:global", fetch)
	require.NoError(t, err)
	require.Equal(t, "value", v)
	require.Equal(t, 1, calls)

	// Hit: fetch is NOT invoked again.
	v, err = c.GetOrFetch("opa:org:operational_hours:global", fetch)
	require.NoError(t, err)
	require.Equal(t, "value", v)
	require.Equal(t, 1, calls, "second call should hit the cache")
}

func TestGetOrFetch_RefetchesAfterInvalidation(t *testing.T) {
	c := NewCache(time.Minute, zerolog.Nop())

	calls := 0
	fetch := func() (interface{}, error) {
		calls++
		return calls, nil
	}

	_, err := c.GetOrFetch("key", fetch)
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	// Simulate a policy7.params.updated NATS event invalidating the entry.
	c.Invalidate("key")

	v, err := c.GetOrFetch("key", fetch)
	require.NoError(t, err)
	require.Equal(t, 2, calls, "invalidation should force a re-fetch")
	require.Equal(t, 2, v)
}

func TestGetOrFetch_PrefixInvalidationClearsScopedKeys(t *testing.T) {
	c := NewCache(time.Minute, zerolog.Nop())

	calls := 0
	fetch := func() (interface{}, error) { calls++; return "v", nil }

	// Scoped key as produced by the time-window evaluator.
	key := "opa:org-1:operational_hours:branch-9"
	_, _ = c.GetOrFetch(key, fetch)
	require.Equal(t, 1, calls)

	// HandleParamUpdated invalidates by prefix "opa:<org>:<param>".
	c.InvalidateByPrefix("opa:org-1:operational_hours")

	_, _ = c.GetOrFetch(key, fetch)
	require.Equal(t, 2, calls, "prefix invalidation must clear scope-suffixed keys")
}

func TestGetOrFetch_ErrorNotCached(t *testing.T) {
	c := NewCache(time.Minute, zerolog.Nop())

	calls := 0
	_, err := c.GetOrFetch("k", func() (interface{}, error) {
		calls++
		return nil, errors.New("policy7 unreachable")
	})
	require.Error(t, err)
	require.Equal(t, 0, c.Len(), "failed fetch must not be cached")

	// Next call retries (fetch invoked again) and can succeed.
	v, err := c.GetOrFetch("k", func() (interface{}, error) {
		calls++
		return "ok", nil
	})
	require.NoError(t, err)
	require.Equal(t, "ok", v)
	require.Equal(t, 2, calls)
}
