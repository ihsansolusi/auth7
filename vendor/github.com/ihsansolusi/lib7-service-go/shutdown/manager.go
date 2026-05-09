package shutdown

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type hook struct {
	name string
	fn   func(ctx context.Context) error
}

// ShutdownManager executes registered shutdown hooks in registration order
// during service shutdown. Each hook runs sequentially with a shared timeout.
// Hook failures are logged but do not abort remaining hooks.
type ShutdownManager struct {
	hooks   []hook
	timeout time.Duration
	logger  zerolog.Logger
}

// New creates a ShutdownManager. timeout is applied to the parent context
// passed to Wait — all hooks share this deadline.
func New(timeout time.Duration, logger zerolog.Logger) *ShutdownManager {
	return &ShutdownManager{
		timeout: timeout,
		logger:  logger,
	}
}

// Register adds a named shutdown hook. Hooks are executed in the order they
// are registered.
func (m *ShutdownManager) Register(name string, fn func(ctx context.Context) error) {
	m.hooks = append(m.hooks, hook{name: name, fn: fn})
}

// Wait runs all registered hooks sequentially, wrapping ctx in a timeout.
// Errors are logged per-hook; Wait itself always returns nil so that callers
// can confidently call os.Exit after it returns.
func (m *ShutdownManager) Wait(ctx context.Context) error {
	const op = "shutdown.ShutdownManager.Wait"
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	logger := m.logger.With().Str("op", op).Logger()

	for _, h := range m.hooks {
		start := time.Now()
		if err := h.fn(ctx); err != nil {
			logger.Error().
				Str("hook", h.name).
				Dur("duration_ms", time.Since(start)).
				Err(err).
				Msg("shutdown hook failed")
		} else {
			logger.Info().
				Str("hook", h.name).
				Dur("duration_ms", time.Since(start)).
				Msg("shutdown hook completed")
		}
	}

	return nil
}
