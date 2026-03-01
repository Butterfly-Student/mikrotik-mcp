package session

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"

	"mikrotik-mcp/internal/ai/zai"
)

// openManagerDB reuses the same schema helper as store_test.go
func openManagerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		phone TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS messages (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		phone        TEXT NOT NULL,
		role         TEXT NOT NULL,
		content      TEXT NOT NULL DEFAULT '',
		tool_calls   TEXT,
		tool_call_id TEXT,
		name         TEXT,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestManager(db *sql.DB, ttl time.Duration, maxHistory int) *Manager {
	return NewManager(NewStore(db), ttl, maxHistory, zap.NewNop())
}

// ── AppendMessages ────────────────────────────────────────────────────────────

func TestAppendMessages_StoresMultiple(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	msgs := []zai.Message{
		{Role: "user", Content: "tampilkan pool"},
		{Role: "assistant", Content: "Ada 2 pool."},
	}
	err := mgr.AppendMessages(ctx, "628001", msgs...)
	require.NoError(t, err)

	history, err := mgr.GetHistory(ctx, "628001")
	require.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "user", history[0].Role)
	assert.Equal(t, "assistant", history[1].Role)
}

func TestAppendMessages_WithToolCalls(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	assistantMsg := zai.Message{
		Role: "assistant",
		ToolCalls: []zai.ToolCall{{
			ID:   "call_abc",
			Type: "function",
			Function: zai.FunctionCall{
				Name:      "list_ip_pools",
				Arguments: "{}",
			},
		}},
	}
	toolMsg := zai.Message{
		Role:       "tool",
		Content:    `{"pools":[]}`,
		ToolCallID: "call_abc",
		Name:       "list_ip_pools",
	}

	err := mgr.AppendMessages(ctx, "628001", assistantMsg, toolMsg)
	require.NoError(t, err)

	history, err := mgr.GetHistory(ctx, "628001")
	require.NoError(t, err)
	require.Len(t, history, 2)

	assert.Equal(t, "assistant", history[0].Role)
	require.Len(t, history[0].ToolCalls, 1)
	assert.Equal(t, "list_ip_pools", history[0].ToolCalls[0].Function.Name)

	assert.Equal(t, "tool", history[1].Role)
	assert.Equal(t, "call_abc", history[1].ToolCallID)
	assert.Equal(t, "list_ip_pools", history[1].Name)
}

// ── GetHistory ────────────────────────────────────────────────────────────────

func TestGetHistory_MaxHistoryRespected(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 5)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = mgr.AppendMessages(ctx, "628001", zai.Message{Role: "user", Content: "msg"})
	}

	history, err := mgr.GetHistory(ctx, "628001")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(history), 5)
}

func TestGetHistory_EmptyWhenNoMessages(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	history, err := mgr.GetHistory(ctx, "628999")
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestGetHistory_IsolatedByPhone(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	_ = mgr.AppendMessages(ctx, "628001", zai.Message{Role: "user", Content: "from 001"})
	_ = mgr.AppendMessages(ctx, "628002", zai.Message{Role: "user", Content: "from 002"})

	h1, _ := mgr.GetHistory(ctx, "628001")
	h2, _ := mgr.GetHistory(ctx, "628002")

	require.Len(t, h1, 1)
	require.Len(t, h2, 1)
	assert.Equal(t, "from 001", h1[0].Content)
	assert.Equal(t, "from 002", h2[0].Content)
}

// ── ResetSession ──────────────────────────────────────────────────────────────

func TestResetSession_ClearsHistory(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	_ = mgr.AppendMessages(ctx, "628001",
		zai.Message{Role: "user", Content: "hello"},
		zai.Message{Role: "assistant", Content: "hi"},
	)

	err := mgr.ResetSession(ctx, "628001")
	require.NoError(t, err)

	history, err := mgr.GetHistory(ctx, "628001")
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestResetSession_DoesNotAffectOtherPhones(t *testing.T) {
	db := openManagerDB(t)
	mgr := newTestManager(db, 2*time.Hour, 20)
	ctx := context.Background()

	_ = mgr.AppendMessages(ctx, "628001", zai.Message{Role: "user", Content: "hello"})
	_ = mgr.AppendMessages(ctx, "628002", zai.Message{Role: "user", Content: "stays"})

	_ = mgr.ResetSession(ctx, "628001")

	h2, _ := mgr.GetHistory(ctx, "628002")
	assert.Len(t, h2, 1)
}

// ── DefaultValues ─────────────────────────────────────────────────────────────

func TestNewManager_DefaultsApplied(t *testing.T) {
	db := openManagerDB(t)
	mgr := NewManager(NewStore(db), 0, 0, zap.NewNop())

	assert.Equal(t, SessionTTL, mgr.sessionTTL)
	assert.Equal(t, MaxHistoryMessages, mgr.maxHistory)
}
