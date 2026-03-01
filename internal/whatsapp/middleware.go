package whatsapp

import (
	"sync"
	"time"

	"mikrotik-mcp/internal/config"
)

type AuthUser struct {
	Phone  string
	Name   string
	Access string // "full" | "readonly"
}

type rateLimiter struct {
	tokens   int
	max      int
	interval time.Duration
	lastReset time.Time
	mu       sync.Mutex
}

func newRateLimiter(max int, interval time.Duration) *rateLimiter {
	return &rateLimiter{
		tokens:    max,
		max:       max,
		interval:  interval,
		lastReset: time.Now(),
	}
}

func (r *rateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Sub(r.lastReset) >= r.interval {
		r.tokens = r.max
		r.lastReset = now
	}
	if r.tokens <= 0 {
		return false
	}
	r.tokens--
	return true
}

type Middleware struct {
	users    map[string]AuthUser
	limiters map[string]*rateLimiter
	mu       sync.RWMutex
}

func NewMiddleware(users []config.AuthUser) *Middleware {
	m := &Middleware{
		users:    make(map[string]AuthUser),
		limiters: make(map[string]*rateLimiter),
	}
	for _, u := range users {
		m.users[u.Phone] = AuthUser{
			Phone:  u.Phone,
			Name:   u.Name,
			Access: u.Access,
		}
	}
	return m
}

func (m *Middleware) IsAuthorized(phone string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.users[phone]
	return ok
}

func (m *Middleware) GetAccessLevel(phone string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.users[phone]; ok {
		return u.Access
	}
	return "readonly"
}

// Allow cek rate limit: 10 request per menit per nomor
func (m *Middleware) Allow(phone string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.limiters[phone]; !ok {
		m.limiters[phone] = newRateLimiter(10, time.Minute)
	}
	return m.limiters[phone].allow()
}
