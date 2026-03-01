package session

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/zai"
)

const (
	MaxHistoryMessages = 20
	SessionTTL         = 2 * time.Hour
)

type Manager struct {
	store      *Store
	sessionTTL time.Duration
	maxHistory int
	logger     *zap.Logger
}

func NewManager(store *Store, sessionTTL time.Duration, maxHistory int, logger *zap.Logger) *Manager {
	if sessionTTL == 0 {
		sessionTTL = SessionTTL
	}
	if maxHistory == 0 {
		maxHistory = MaxHistoryMessages
	}
	return &Manager{
		store:      store,
		sessionTTL: sessionTTL,
		maxHistory: maxHistory,
		logger:     logger,
	}
}

// GetHistory mengambil history yang masih dalam TTL, siap dimasukkan ke ChatRequest
func (m *Manager) GetHistory(ctx context.Context, phone string) ([]zai.Message, error) {
	rows, err := m.store.GetRecentMessages(ctx, phone, m.maxHistory)
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-m.sessionTTL)
	var result []zai.Message
	for _, row := range rows {
		if row.CreatedAt.Before(cutoff) {
			continue
		}
		msg := zai.Message{
			Role:       row.Role,
			Content:    row.Content,
			ToolCallID: row.ToolCallID,
			Name:       row.Name,
		}
		if row.ToolCallsJSON != "" {
			_ = json.Unmarshal([]byte(row.ToolCallsJSON), &msg.ToolCalls)
		}
		result = append(result, msg)
	}
	return result, nil
}

// AppendMessages menyimpan messages baru ke SQLite
func (m *Manager) AppendMessages(ctx context.Context, phone string, msgs ...zai.Message) error {
	for _, msg := range msgs {
		tcJSON := ""
		if len(msg.ToolCalls) > 0 {
			b, _ := json.Marshal(msg.ToolCalls)
			tcJSON = string(b)
		}
		if err := m.store.SaveMessage(ctx, phone,
			msg.Role, msg.Content, tcJSON, msg.ToolCallID, msg.Name); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ResetSession(ctx context.Context, phone string) error {
	return m.store.DeleteMessages(ctx, phone)
}
