package bridge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/zai"
	"mikrotik-mcp/internal/mcpclient"
)

type MCPBridge struct {
	mcpClient   mcpclient.MCPCaller
	cachedTools []mcpclient.Tool
	auditLogger *AuditLogger
	mu          sync.RWMutex
	logger      *zap.Logger
}

func New(mcpClient mcpclient.MCPCaller, logger *zap.Logger) *MCPBridge {
	return &MCPBridge{mcpClient: mcpClient, logger: logger}
}

func (b *MCPBridge) SetAuditLogger(a *AuditLogger) { b.auditLogger = a }

// RefreshTools mengambil ulang tools dari MCP server — dipanggil saat startup
func (b *MCPBridge) RefreshTools(ctx context.Context) error {
	tools, err := b.mcpClient.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("refresh tools: %w", err)
	}
	b.mu.Lock()
	b.cachedTools = tools
	b.mu.Unlock()
	b.logger.Info("MCP tools refreshed", zap.Int("count", len(tools)))
	return nil
}

// ToZAITools konversi cached MCP tools ke format Tool Z.AI / GLM
func (b *MCPBridge) ToZAITools() []zai.Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]zai.Tool, 0, len(b.cachedTools))
	for _, t := range b.cachedTools {
		result = append(result, zai.Tool{
			Type: "function",
			Function: zai.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  normalizeSchema(t.InputSchema),
			},
		})
	}
	return result
}

// normalizeSchema memastikan schema selalu valid untuk GLM
func normalizeSchema(s map[string]interface{}) map[string]interface{} {
	if len(s) == 0 {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	if _, ok := s["type"]; !ok {
		s["type"] = "object"
	}
	if _, ok := s["properties"]; !ok {
		s["properties"] = map[string]interface{}{}
	}
	return s
}

func (b *MCPBridge) ToolCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.cachedTools)
}

func (b *MCPBridge) ToolNames() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.cachedTools))
	for _, t := range b.cachedTools {
		names = append(names, t.Name)
	}
	return names
}

// isWriteTool cek apakah tool ini operasi write/modifikasi
func isWriteTool(name string) bool {
	for _, prefix := range []string{
		"add_", "create_", "delete_", "remove_",
		"update_", "set_", "enable_", "disable_",
		"toggle_", "reboot_", "reset_", "move_",
		"kick_",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
