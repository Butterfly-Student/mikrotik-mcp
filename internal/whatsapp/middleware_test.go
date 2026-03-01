package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mikrotik-mcp/internal/config"
)

func testMiddleware() *Middleware {
	return NewMiddleware([]config.AuthUser{
		{Phone: "6281000000001", Name: "Admin", Access: "full"},
		{Phone: "6281000000002", Name: "Staff", Access: "readonly"},
	})
}

// ── IsAuthorized ──────────────────────────────────────────────────────────────

func TestIsAuthorized_RegisteredUser(t *testing.T) {
	m := testMiddleware()
	assert.True(t, m.IsAuthorized("6281000000001"))
	assert.True(t, m.IsAuthorized("6281000000002"))
}

func TestIsAuthorized_UnregisteredUser(t *testing.T) {
	m := testMiddleware()
	assert.False(t, m.IsAuthorized("6289999999999"))
}

func TestIsAuthorized_Empty(t *testing.T) {
	m := testMiddleware()
	assert.False(t, m.IsAuthorized(""))
}

// ── GetAccessLevel ────────────────────────────────────────────────────────────

func TestGetAccessLevel_FullUser(t *testing.T) {
	m := testMiddleware()
	assert.Equal(t, "full", m.GetAccessLevel("6281000000001"))
}

func TestGetAccessLevel_ReadonlyUser(t *testing.T) {
	m := testMiddleware()
	assert.Equal(t, "readonly", m.GetAccessLevel("6281000000002"))
}

func TestGetAccessLevel_UnknownFallsToReadonly(t *testing.T) {
	m := testMiddleware()
	assert.Equal(t, "readonly", m.GetAccessLevel("6289999999999"))
}

// ── Allow (rate limiter) ──────────────────────────────────────────────────────

func TestAllow_InitiallyAllows(t *testing.T) {
	m := testMiddleware()
	// 10 requests diizinkan dalam 1 menit
	for i := 0; i < 10; i++ {
		assert.True(t, m.Allow("6281000000001"), "request %d should be allowed", i+1)
	}
}

func TestAllow_BlocksAfterLimit(t *testing.T) {
	m := testMiddleware()
	phone := "6281000000099"
	// Habiskan semua token
	for i := 0; i < 10; i++ {
		m.Allow(phone)
	}
	// Request ke-11 harus ditolak
	assert.False(t, m.Allow(phone))
}

func TestAllow_DifferentPhonesSeparateLimits(t *testing.T) {
	m := testMiddleware()
	phoneA := "6281111111111"
	phoneB := "6282222222222"

	// Habiskan limit phoneA
	for i := 0; i < 10; i++ {
		m.Allow(phoneA)
	}
	assert.False(t, m.Allow(phoneA))

	// phoneB tidak terpengaruh
	assert.True(t, m.Allow(phoneB))
}

func TestAllow_NewPhoneGetsOwnLimiter(t *testing.T) {
	m := testMiddleware()
	// Phone yang belum pernah dilihat harus langsung mendapat limiter baru
	assert.True(t, m.Allow("6283333333333"))
}
