package infrastructure

import (
	"context"

	"github.com/ihsansolusi/auth7/internal/mailer"
	"github.com/ihsansolusi/auth7/pkg/config"
	"github.com/rs/zerolog"
)

func NewMailer(ctx context.Context, cfg config.SMTPConfig, logger zerolog.Logger) (mailer.Mailer, error) {
	const op = "infrastructure.NewMailer"

	if !cfg.IsConfigured() {
		logger.Warn().Str("op", op).Msg("SMTP not configured, using noop mailer")
		return mailer.NewNoopMailer(), nil
	}

	m := mailer.NewSMTPMailer(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.From, cfg.StartTLS)

	logger.Info().
		Str("op", op).
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("from", cfg.From).
		Msg("SMTP mailer initialized")

	return m, nil
}
