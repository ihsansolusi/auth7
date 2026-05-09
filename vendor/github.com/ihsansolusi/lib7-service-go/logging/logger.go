// Package logging provides structured logging utilities for core7 services.
package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Options configures the logger.
type Options struct {
	Level    zerolog.Level
	TimeZone string // default: "Asia/Jakarta"
	Pretty   bool   // true = human-readable console output, false = JSON
}

// NewLogger creates a new application logger.
func NewLogger(opts Options) zerolog.Logger {
	return newLogger(opts)
}

// NewAuditLogger creates a dedicated logger for the audit stream.
// The resulting logger should be wrapped in AuditLogger.
func NewAuditLogger(opts Options) zerolog.Logger {
	l := newLogger(opts)
	return l.With().Str("stream", "audit").Logger()
}

func newLogger(opts Options) zerolog.Logger {
	if opts.TimeZone == "" {
		opts.TimeZone = "Asia/Jakarta"
	}

	loc, err := time.LoadLocation(opts.TimeZone)
	if err != nil {
		loc = time.UTC
	}

	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(loc)
	}

	var w io.Writer = os.Stdout
	if opts.Pretty {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	return zerolog.New(w).Level(opts.Level).With().Timestamp().Logger()
}
