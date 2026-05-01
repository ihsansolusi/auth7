package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type Subscriber struct {
	client        *Client
	logger        zerolog.Logger
	subscriptions []*nats.Subscription
}

func NewSubscriber(client *Client, logger zerolog.Logger) *Subscriber {
	return &Subscriber{
		client: client,
		logger: logger,
	}
}

func (s *Subscriber) Subscribe(subject string, handler func(data []byte) error) error {
	const op = "messaging.Subscribe"

	if s.client == nil || s.client.conn == nil {
		return nil
	}

	sub, err := s.client.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			s.logger.Error().
				Str("op", op).
				Str("subject", subject).
				Err(err).
				Msg("failed to handle NATS message")
		}
	})

	if err != nil {
		s.logger.Warn().
			Str("op", op).
			Str("subject", subject).
			Err(err).
			Msg("failed to subscribe, will retry on reconnect")
		return nil
	}

	s.subscriptions = append(s.subscriptions, sub)

	s.logger.Info().
		Str("op", op).
		Str("subject", subject).
		Msg("subscribed to NATS subject")

	return nil
}

func (s *Subscriber) UnsubscribeAll() error {
	for _, sub := range s.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			s.logger.Warn().Err(err).Msg("failed to unsubscribe")
		}
	}
	s.subscriptions = nil
	return nil
}

func (s *Subscriber) Close() {
	s.UnsubscribeAll()
}
