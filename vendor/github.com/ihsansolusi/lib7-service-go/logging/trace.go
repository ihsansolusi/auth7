package logging

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// WithTrace returns a logger enriched with trace_id and span_id from the
// OTEL span in ctx. If the span is not valid, the original logger is returned
// unchanged.
func WithTrace(ctx context.Context, logger zerolog.Logger) zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return logger
	}
	return logger.With().
		Str("trace_id", span.SpanContext().TraceID().String()).
		Str("span_id", span.SpanContext().SpanID().String()).
		Logger()
}
