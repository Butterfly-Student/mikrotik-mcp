package session

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MessageRow struct {
	ID            int64
	Phone         string
	Role          string
	Content       string
	ToolCallsJSON string
	ToolCallID    string
	Name          string
	CreatedAt     time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetRecentMessages(ctx context.Context, phone string, limit int) ([]MessageRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, phone, role, content,
                COALESCE(tool_calls,''), COALESCE(tool_call_id,''), COALESCE(name,''),
                CAST(strftime('%s', created_at) AS INTEGER)
         FROM messages
         WHERE phone = ?
         ORDER BY created_at ASC, id ASC
         LIMIT ?`,
		phone, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var result []MessageRow
	for rows.Next() {
		var r MessageRow
		var unixTs int64
		if err := rows.Scan(&r.ID, &r.Phone, &r.Role, &r.Content,
			&r.ToolCallsJSON, &r.ToolCallID, &r.Name, &unixTs); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		r.CreatedAt = time.Unix(unixTs, 0).UTC()
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) SaveMessage(ctx context.Context, phone, role, content, toolCallsJSON, toolCallID, name string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (phone, role, content, tool_calls, tool_call_id, name, created_at)
         VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), datetime('now'))`,
		phone, role, content, toolCallsJSON, toolCallID, name,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	// Upsert session updated_at
	_, _ = s.db.ExecContext(ctx,
		`INSERT INTO sessions (phone, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))
         ON CONFLICT(phone) DO UPDATE SET updated_at=datetime('now')`,
		phone,
	)
	return nil
}

func (s *Store) DeleteMessages(ctx context.Context, phone string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE phone = ?`, phone)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE phone = ?`, phone)
	return nil
}
