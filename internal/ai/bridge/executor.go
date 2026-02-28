package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/zai"
	"mikrotik-mcp/internal/mcpclient"
)

type ExecuteOptions struct {
	Phone       string // nomor WA — untuk audit log
	AccessLevel string // "full" | "readonly"
}

// Execute mengeksekusi satu GLM FunctionCall sebagai MCP tool call.
// Selalu return string — error disampaikan sebagai teks agar GLM bisa jelaskan ke user.
func (b *MCPBridge) Execute(ctx context.Context, call zai.FunctionCall, opts ExecuteOptions) string {

	// ── 1. Access Control ─────────────────────────────────────────────────────
	if opts.AccessLevel == "readonly" && isWriteTool(call.Name) {
		b.logger.Warn("access denied",
			zap.String("phone", opts.Phone),
			zap.String("tool", call.Name),
		)
		return fmt.Sprintf(
			`{"error":"access_denied","message":"Tool '%s' memerlukan akses full. Akses Anda: readonly."}`,
			call.Name,
		)
	}

	// ── 2. Parse arguments JSON string dari GLM ───────────────────────────────
	var args map[string]interface{}
	if call.Arguments != "" && call.Arguments != "{}" {
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			b.logger.Error("parse tool args failed",
				zap.String("tool", call.Name),
				zap.String("raw_args", call.Arguments),
				zap.Error(err),
			)
			return fmt.Sprintf(`{"error":"invalid_args","message":"Gagal parse arguments: %s"}`, err.Error())
		}
	}

	b.logger.Info("executing tool",
		zap.String("phone", opts.Phone),
		zap.String("tool", call.Name),
		zap.Any("args", args),
	)

	// ── 3. Audit log sebelum eksekusi ─────────────────────────────────────────
	if b.auditLogger != nil {
		b.auditLogger.Before(ctx, opts.Phone, call.Name, args)
	}

	// ── 4. Eksekusi ke MCP server (dengan timeout sendiri) ────────────────────
	toolCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := b.mcpClient.CallTool(toolCtx, call.Name, args)

	// ── 5. Audit log setelah eksekusi ─────────────────────────────────────────
	if b.auditLogger != nil {
		b.auditLogger.After(ctx, opts.Phone, call.Name, err)
	}

	// ── 6. Handle error dari MCP ──────────────────────────────────────────────
	if err != nil {
		b.logger.Error("tool execution failed",
			zap.String("tool", call.Name),
			zap.Error(err),
		)
		return fmt.Sprintf(`{"error":"execution_failed","message":"Gagal menjalankan %s: %s"}`,
			call.Name, err.Error())
	}

	// ── 7. Serialize hasil ke string untuk dikirim ke GLM ─────────────────────
	return extractText(result)
}

// extractText mengambil teks dari CallResult, menggabungkan jika ada multiple blocks
func extractText(result *mcpclient.CallResult) string {
	if result == nil || len(result.Content) == 0 {
		return `{"result":"ok"}`
	}
	var parts []string
	for _, b := range result.Content {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	if len(parts) == 0 {
		return `{"result":"ok"}`
	}
	if len(parts) == 1 {
		return parts[0]
	}
	combined, _ := json.Marshal(parts)
	return string(combined)
}
