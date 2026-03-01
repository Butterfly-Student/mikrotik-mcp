package session

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// openTestDB membuka SQLite in-memory dan menjalankan migrasi
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		phone      TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS messages (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		phone        TEXT    NOT NULL,
		role         TEXT    NOT NULL,
		content      TEXT    NOT NULL DEFAULT '',
		tool_calls   TEXT,
		tool_call_id TEXT,
		name         TEXT,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS audit_logs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		phone       TEXT    NOT NULL,
		tool_name   TEXT    NOT NULL,
		args        TEXT,
		status      TEXT DEFAULT 'pending',
		error       TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// ── SaveMessage ───────────────────────────────────────────────────────────────

func TestSaveMessage_Basic(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	err := s.SaveMessage(ctx, "628001", "user", "hello bot", "", "", "")
	require.NoError(t, err)

	rows, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "user", rows[0].Role)
	assert.Equal(t, "hello bot", rows[0].Content)
}

func TestSaveMessage_WithToolCalls(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	tcJSON := `[{"id":"call_1","type":"function","function":{"name":"list_ip_pools","arguments":"{}"}}]`
	err := s.SaveMessage(ctx, "628001", "assistant", "", tcJSON, "", "")
	require.NoError(t, err)

	rows, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "assistant", rows[0].Role)
	assert.Equal(t, tcJSON, rows[0].ToolCallsJSON)
}

func TestSaveMessage_ToolResult(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	err := s.SaveMessage(ctx, "628001", "tool", `{"pools":[]}`, "", "call_1", "list_ip_pools")
	require.NoError(t, err)

	rows, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "tool", rows[0].Role)
	assert.Equal(t, "call_1", rows[0].ToolCallID)
	assert.Equal(t, "list_ip_pools", rows[0].Name)
}

// ── GetRecentMessages ─────────────────────────────────────────────────────────

func TestGetRecentMessages_IsolatedByPhone(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "msg from 001", "", "", "")
	_ = s.SaveMessage(ctx, "628002", "user", "msg from 002", "", "", "")

	rows001, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	assert.Len(t, rows001, 1)
	assert.Equal(t, "msg from 001", rows001[0].Content)

	rows002, err := s.GetRecentMessages(ctx, "628002", 10)
	require.NoError(t, err)
	assert.Len(t, rows002, 1)
	assert.Equal(t, "msg from 002", rows002[0].Content)
}

func TestGetRecentMessages_LimitRespected(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		_ = s.SaveMessage(ctx, "628001", "user", "msg", "", "", "")
	}

	rows, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	assert.Len(t, rows, 10)
}

func TestGetRecentMessages_PreservesOrder(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "first", "", "", "")
	_ = s.SaveMessage(ctx, "628001", "assistant", "second", "", "", "")
	_ = s.SaveMessage(ctx, "628001", "user", "third", "", "", "")

	rows, err := s.GetRecentMessages(ctx, "628001", 10)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, "first", rows[0].Content)
	assert.Equal(t, "second", rows[1].Content)
	assert.Equal(t, "third", rows[2].Content)
}

func TestGetRecentMessages_EmptyPhone(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "hello", "", "", "")

	rows, err := s.GetRecentMessages(ctx, "628999", 10)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// ── DeleteMessages ────────────────────────────────────────────────────────────

func TestDeleteMessages_ClearsPhone(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "hello", "", "", "")
	_ = s.SaveMessage(ctx, "628001", "assistant", "hi", "", "", "")
	_ = s.SaveMessage(ctx, "628002", "user", "stays", "", "", "")

	err := s.DeleteMessages(ctx, "628001")
	require.NoError(t, err)

	rows001, _ := s.GetRecentMessages(ctx, "628001", 10)
	assert.Empty(t, rows001)

	rows002, _ := s.GetRecentMessages(ctx, "628002", 10)
	assert.Len(t, rows002, 1)
}

func TestDeleteMessages_Idempotent(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	// Hapus phone yang tidak ada — tidak boleh error
	err := s.DeleteMessages(ctx, "628999")
	assert.NoError(t, err)
}

// ── Session upsert ────────────────────────────────────────────────────────────

func TestSaveMessage_CreatesSessionRecord(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "hello", "", "", "")

	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE phone=?", "628001").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSaveMessage_SessionUpsertIdempotent(t *testing.T) {
	db := openTestDB(t)
	s := NewStore(db)
	ctx := context.Background()

	_ = s.SaveMessage(ctx, "628001", "user", "msg1", "", "", "")
	_ = s.SaveMessage(ctx, "628001", "user", "msg2", "", "", "")

	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE phone=?", "628001").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // hanya 1 record session
}
