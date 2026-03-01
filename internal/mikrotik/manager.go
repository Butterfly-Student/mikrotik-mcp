package mikrotik

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Manager manages connections to multiple MikroTik routers, each identified by a name.
// Routers can be added or removed at runtime.
// All methods are safe for concurrent use.
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewManager creates an empty Manager.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// Register connects to a router and registers it under name.
// Returns an error if name is already taken or the connection fails.
func (m *Manager) Register(ctx context.Context, name string, cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("router %q already registered", name)
	}

	c := NewClient(cfg, m.logger.With(zap.String("router", name)))
	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("register router %q: %w", name, err)
	}

	m.clients[name] = c
	m.logger.Info("router registered",
		zap.String("name", name),
		zap.String("host", cfg.Host),
		zap.Int("pool_size", cfg.PoolSize),
	)
	return nil
}

// Get returns the Client for a registered router, or an error if not found.
func (m *Manager) Get(name string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.clients[name]
	if !ok {
		return nil, fmt.Errorf("router %q not registered", name)
	}
	return c, nil
}

// MustGet returns the Client for name, panicking if not registered.
// Useful in init paths where the router is guaranteed to exist.
func (m *Manager) MustGet(name string) *Client {
	c, err := m.Get(name)
	if err != nil {
		panic(err)
	}
	return c
}

// Unregister closes and removes the router identified by name.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[name]; ok {
		c.Close()
		delete(m.clients, name)
		m.logger.Info("router unregistered", zap.String("name", name))
	}
}

// Names returns the names of all currently registered routers.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// CloseAll disconnects every registered router. The Manager is empty after this call.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, c := range m.clients {
		c.Close()
		delete(m.clients, name)
	}
	m.logger.Info("all routers disconnected")
}
