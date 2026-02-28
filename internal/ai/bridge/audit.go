package bridge

import (
	"context"
	"database/sql"
	"encoding/json"

	"go.uber.org/zap"
)

type AuditLogger struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewAuditLogger(db *sql.DB, logger *zap.Logger) *AuditLogger {
	return &AuditLogger{db: db, logger: logger}
}

func (a *AuditLogger) Before(ctx context.Context, phone, tool string, args map[string]interface{}) {
	argsJSON, _ := json.Marshal(args)
	go func() {
		_, err := a.db.ExecContext(ctx,
			`INSERT INTO audit_logs (phone, tool_name, args, status, created_at)
             VALUES (?, ?, ?, 'pending', datetime('now'))`,
			phone, tool, string(argsJSON),
		)
		if err != nil {
			a.logger.Warn("audit log before failed", zap.Error(err))
		}
	}()
}

func (a *AuditLogger) After(ctx context.Context, phone, tool string, execErr error) {
	status, errMsg := "success", ""
	if execErr != nil {
		status = "error"
		errMsg = execErr.Error()
	}
	go func() {
		_, err := a.db.ExecContext(ctx,
			`UPDATE audit_logs SET status=?, error=?, finished_at=datetime('now')
             WHERE id = (
               SELECT id FROM audit_logs
               WHERE phone=? AND tool_name=? AND status='pending'
               ORDER BY created_at DESC LIMIT 1
             )`,
			status, errMsg, phone, tool,
		)
		if err != nil {
			a.logger.Warn("audit log after failed", zap.Error(err))
		}
	}()
}
