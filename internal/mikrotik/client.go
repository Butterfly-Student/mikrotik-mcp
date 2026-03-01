package mikrotik

import (
	"context"
	"fmt"
	"sync"
	"time"

	routeros "github.com/go-routeros/routeros/v3"
	"go.uber.org/zap"
)

const (
	DefaultQueueSize    = 100
	reconnectBaseDelay  = time.Second
	reconnectMaxDelay   = 30 * time.Second
)

// Config holds connection parameters for a single MikroTik router.
type Config struct {
	Host              string
	Port              int
	Username          string
	Password          string
	UseTLS            bool
	ReconnectInterval time.Duration
	Timeout           time.Duration // per-command timeout (default 10s)
	PoolSize          int           // unused field kept for config compatibility
}

// Client wraps a single async RouterOS connection.
//
// Calling Async() on the underlying *routeros.Client enables the library's
// built-in tag multiplexing: a single TCP connection handles many concurrent
// Run / Listen calls without extra goroutines or locking on our side.
//
// All exported methods are safe for concurrent use.
type Client struct {
	conn        *routeros.Client
	config      Config
	asyncCtx    context.Context    // lives for the lifetime of Client
	asyncCancel context.CancelFunc // cancelled by Close()
	mu          sync.RWMutex
	closed      bool
	logger      *zap.Logger
}

// NewClient creates a Client. Call Connect before using it.
func NewClient(cfg Config, logger *zap.Logger) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:      cfg,
		asyncCtx:    ctx,
		asyncCancel: cancel,
		logger:      logger,
	}
}

// Connect dials the router and switches the connection to async mode.
// In async mode the library multiplexes concurrent requests over one TCP
// connection using internal tags — no connection pool required.
func (c *Client) Connect(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return fmt.Errorf("connect mikrotik %s: %w", c.config.Host, err)
	}

	// AsyncContext starts the internal tag-based read loop. The returned channel
	// receives a single error when the async loop terminates (connection lost).
	errCh := conn.AsyncContext(c.asyncCtx)

	// Set the default channel buffer for Listen operations.
	conn.Queue = DefaultQueueSize

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.watchAsync(errCh)

	c.logger.Info("connected to mikrotik (async)",
		zap.String("host", c.config.Host),
		zap.Bool("is_async", conn.IsAsync()),
	)
	return nil
}

// Close cancels the async context and closes the underlying connection.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	c.asyncCancel()
	if conn != nil {
		conn.Close() //nolint:errcheck
	}
}

// IsAsync reports whether the underlying connection is in async mode.
func (c *Client) IsAsync() bool {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	return conn != nil && conn.IsAsync()
}

// dial opens a single RouterOS connection (dial + login).
func (c *Client) dial(ctx context.Context) (*routeros.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	if c.config.UseTLS {
		return routeros.DialTLSContext(ctx, addr, c.config.Username, c.config.Password, nil)
	}
	return routeros.DialContext(ctx, addr, c.config.Username, c.config.Password)
}

// watchAsync waits for the async loop to terminate. On unexpected failure it
// triggers automatic reconnection.
func (c *Client) watchAsync(errCh <-chan error) {
	err := <-errCh

	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return // expected shutdown, do nothing
	}

	c.logger.Warn("async connection lost, reconnecting",
		zap.String("host", c.config.Host),
		zap.Error(err),
	)
	c.mu.Lock()
	c.conn = nil
	c.mu.Unlock()

	go c.reconnect()
}

// reconnect dials a new connection with exponential backoff and re-enables async mode.
func (c *Client) reconnect() {
	backoff := reconnectBaseDelay
	for {
		c.mu.RLock()
		closed := c.closed
		c.mu.RUnlock()
		if closed {
			return
		}

		dialCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := c.dial(dialCtx)
		cancel()

		if err == nil {
			conn.Queue = DefaultQueueSize
			errCh := conn.AsyncContext(c.asyncCtx)

			c.mu.Lock()
			if !c.closed {
				c.conn = conn
				c.mu.Unlock()
				go c.watchAsync(errCh)
				c.logger.Info("reconnected to mikrotik", zap.String("host", c.config.Host))
				return
			}
			c.mu.Unlock()
			conn.Close() //nolint:errcheck
			return
		}

		c.logger.Warn("reconnect failed, retrying",
			zap.String("host", c.config.Host),
			zap.Duration("after", backoff),
			zap.Error(err),
		)
		time.Sleep(backoff)
		if backoff < reconnectMaxDelay {
			backoff *= 2
		}
	}
}

// ─── Command execution ────────────────────────────────────────────────────────

// conn returns the current connection or an error if disconnected.
func (c *Client) getConn() (*routeros.Client, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("not connected to mikrotik (%s)", c.config.Host)
	}
	return conn, nil
}

// RunContext executes a RouterOS command with the given context.
// In async mode the library tags the request internally, so many goroutines
// can call RunContext concurrently on the same Client without blocking each other.
func (c *Client) RunContext(ctx context.Context, sentence ...string) (*routeros.Reply, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}
	return conn.RunContext(ctx, sentence...)
}

// Run executes a RouterOS command using the configured per-command timeout.
func (c *Client) Run(sentence ...string) (*routeros.Reply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	return c.RunContext(ctx, sentence...)
}

// RunArgs is a slice-based variant of Run.
func (c *Client) RunArgs(args []string) (*routeros.Reply, error) {
	return c.Run(args...)
}

// RunArgsContext is a slice-based variant of RunContext.
func (c *Client) RunArgsContext(ctx context.Context, args []string) (*routeros.Reply, error) {
	return c.RunContext(ctx, args...)
}

// RunMany executes multiple RouterOS commands concurrently.
// Because the connection is in async mode, all commands fly over the same TCP
// connection simultaneously — no extra connections or locking needed.
// Results are returned in the same order as the input commands.
func (c *Client) RunMany(ctx context.Context, commands [][]string) ([]*routeros.Reply, []error) {
	conn, err := c.getConn()
	if err != nil {
		errs := make([]error, len(commands))
		for i := range errs {
			errs[i] = err
		}
		return make([]*routeros.Reply, len(commands)), errs
	}

	type result struct {
		idx   int
		reply *routeros.Reply
		err   error
	}

	ch := make(chan result, len(commands))
	for i, cmd := range commands {
		go func(idx int, sentence []string) {
			reply, err := conn.RunContext(ctx, sentence...)
			ch <- result{idx: idx, reply: reply, err: err}
		}(i, cmd)
	}

	replies := make([]*routeros.Reply, len(commands))
	errs := make([]error, len(commands))
	for range commands {
		r := <-ch
		replies[r.idx] = r.reply
		errs[r.idx] = r.err
	}
	return replies, errs
}

// ─── Streaming ────────────────────────────────────────────────────────────────

// ListenArgs starts a streaming RouterOS command (e.g. /interface/monitor-traffic).
// The returned *routeros.ListenReply streams sentences via Chan().
// Call Cancel() or CancelContext() when done.
//
// Because the connection is async, Listen and Run calls co-exist on the same
// TCP connection without interfering.
func (c *Client) ListenArgs(args []string) (*routeros.ListenReply, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}
	return conn.ListenArgsContext(c.asyncCtx, args)
}

// ListenArgsContext is the context-aware variant of ListenArgs.
func (c *Client) ListenArgsContext(ctx context.Context, args []string) (*routeros.ListenReply, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}
	return conn.ListenArgsContext(ctx, args)
}

// ListenArgsQueue starts a streaming command with a custom receive-channel buffer size.
func (c *Client) ListenArgsQueue(args []string, queueSize int) (*routeros.ListenReply, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}
	return conn.ListenArgsQueueContext(c.asyncCtx, args, queueSize)
}
