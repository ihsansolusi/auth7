package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	natslib "github.com/nats-io/nats.go"
)

type Config struct {
	URL            string
	Name           string
	Username       string
	Password       string
	CredsFile      string
	ReconnectWait  time.Duration
	MaxReconnects  int
	PublishTimeout time.Duration
	PublishRetry   int
}

type Client struct {
	conn   *natslib.Conn
	js     natslib.JetStreamContext
	config Config
	logger zerolog.Logger
}

// PublishStream publishes synchronously to JetStream and waits for the persist
// ack, so the message is durably stored (survives audit7 downtime). msgID, when
// non-empty, sets Nats-Msg-Id for server-side dedup. The subject must be covered
// by a stream (audit7 owns AUDIT7_EVENTS over auth7.* + audit7.ingest.*).
func (c *Client) PublishStream(subject string, data []byte, msgID string) error {
	const op = "messaging.PublishStream"
	if c.js == nil {
		return fmt.Errorf("%s: jetstream not initialized", op)
	}
	var opts []natslib.PubOpt
	if msgID != "" {
		opts = append(opts, natslib.MsgId(msgID))
	}
	if _, err := c.js.Publish(subject, data, opts...); err != nil {
		return fmt.Errorf("%s: publish to %s: %w", op, subject, err)
	}
	return nil
}

func NewClient(ctx context.Context, cfg Config, logger zerolog.Logger) (*Client, error) {
	const op = "messaging.NewClient"

	opts := []natslib.Option{
		natslib.Name(cfg.Name),
		natslib.ReconnectWait(cfg.ReconnectWait),
		natslib.MaxReconnects(cfg.MaxReconnects),
		natslib.DisconnectErrHandler(func(nc *natslib.Conn, err error) {
			logger.Warn().Err(err).Msg("NATS disconnected")
		}),
		natslib.ReconnectHandler(func(nc *natslib.Conn) {
			logger.Info().Msg("NATS reconnected")
		}),
		natslib.ClosedHandler(func(nc *natslib.Conn) {
			logger.Info().Msg("NATS connection closed")
		}),
	}

	if cfg.CredsFile != "" {
		opts = append(opts, natslib.UserCredentials(cfg.CredsFile))
	} else if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts, natslib.UserInfo(cfg.Username, cfg.Password))
	}

	conn, err := natslib.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("%s: connect to NATS: %w", op, err)
	}

	// JetStream context for durable publish (audit ingestion). Non-fatal if it
	// fails — core Publish still works.
	js, jsErr := conn.JetStream()
	if jsErr != nil {
		logger.Warn().Err(jsErr).Msg("JetStream context unavailable; durable publish disabled")
	}

	return &Client{
		conn:   conn,
		js:     js,
		config: cfg,
		logger: logger,
	}, nil
}

func (c *Client) Conn() *natslib.Conn {
	return c.conn
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) IsConnected() bool {
	return c.conn.IsConnected()
}
