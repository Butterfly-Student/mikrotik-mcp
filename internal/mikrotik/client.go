package mikrotik

import (
	"context"
	"fmt"
	"sync"
	"time"

	routeros "github.com/go-routeros/routeros/v3"
	"go.uber.org/zap"
)


type Config struct {
	Host              string
	Port              int
	Username          string
	Password          string
	UseTLS            bool
	ReconnectInterval time.Duration
	Timeout           time.Duration
}

type Client struct {
	conn   *routeros.Client
	config Config
	mu     sync.RWMutex
	logger *zap.Logger
}

func NewClient(cfg Config, logger *zap.Logger) *Client {
	return &Client{config: cfg, logger: logger}
}

func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	var conn *routeros.Client
	var err error

	if c.config.UseTLS {
		conn, err = routeros.DialTLSContext(ctx, addr, c.config.Username, c.config.Password, nil)
	} else {
		conn, err = routeros.DialContext(ctx, addr, c.config.Username, c.config.Password)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to mikrotik at %s: %w", addr, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.logger.Info("connected to mikrotik", zap.String("host", c.config.Host))
	return nil
}

func (c *Client) Reconnect(ctx context.Context) error {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.Connect(ctx); err != nil {
				c.logger.Warn("reconnect failed, retrying",
					zap.Duration("after", backoff),
					zap.Error(err),
				)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			c.logger.Info("reconnected to mikrotik")
			return nil
		}
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) Run(sentence ...string) (*routeros.Reply, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to mikrotik")
	}
	return c.conn.Run(sentence...)
}

func (c *Client) RunArgs(args []string) (*routeros.Reply, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to mikrotik")
	}
	return c.conn.RunArgs(args)
}

func (c *Client) ListenArgs(args []string) (*routeros.ListenReply, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to mikrotik")
	}
	return c.conn.ListenArgs(args)
}
