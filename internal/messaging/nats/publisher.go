package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

type Publisher struct {
	client *Client
	logger zerolog.Logger
}

func NewPublisher(client *Client, logger zerolog.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger,
	}
}

func (p *Publisher) Publish(ctx context.Context, subject string, payload interface{}) error {
	const op = "messaging.Publish"

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, p.client.config.PublishTimeout)
	defer cancel()

	err = p.client.conn.Publish(subject, data)
	if err != nil {
		p.logger.Warn().
			Str("op", op).
			Str("subject", subject).
			Err(err).
			Msg("failed to publish NATS event (non-fatal)")
		return fmt.Errorf("%s: publish to %s: %w", op, subject, err)
	}

	p.logger.Debug().
		Str("op", op).
		Str("subject", subject).
		Msg("NATS event published")

	return nil
}
