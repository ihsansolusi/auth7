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
	config Config
	logger zerolog.Logger
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

	return &Client{
		conn:   conn,
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
