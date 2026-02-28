package bridge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/zai"
	"mikrotik-mcp/internal/mcpclient"
)

// ── Mock MCP Caller ───────────────────────────────────────────────────────────

type mockMCPCaller struct{ mock.Mock }

func (m *mockMCPCaller) ListTools(ctx context.Context) ([]mcpclient.Tool, error) {
	args := m.Called(ctx)
	return args.Get(0).([]mcpclient.Tool), args.Error(1)
}

func (m *mockMCPCaller) CallTool(ctx context.Context, name string, a map[string]interface{}) (*mcpclient.CallResult, error) {
	args := m.Called(ctx, name, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcpclient.CallResult), args.Error(1)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func testBridge(caller *mockMCPCaller) *MCPBridge {
	return New(caller, zap.NewNop())
}

func sampleTools() []mcpclient.Tool {
	return []mcpclient.Tool{
		{
			Name:        "list_ip_pools",
			Description: "List all IP pools",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "add_firewall_rule",
			Description: "Add a firewall rule",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chain":  map[string]interface{}{"type": "string"},
					"action": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"chain", "action"},
			},
		},
	}
}

// ── RefreshTools ──────────────────────────────────────────────────────────────

func TestRefreshTools_Success(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("ListTools", mock.Anything).Return(sampleTools(), nil)

	b := testBridge(caller)
	err := b.RefreshTools(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, b.ToolCount())
	caller.AssertExpectations(t)
}

func TestRefreshTools_Error(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("ListTools", mock.Anything).Return([]mcpclient.Tool{}, assert.AnError)

	b := testBridge(caller)
	err := b.RefreshTools(context.Background())

	assert.Error(t, err)
	assert.Equal(t, 0, b.ToolCount())
}

// ── ToZAITools ────────────────────────────────────────────────────────────────

func TestToZAITools_ConvertsCorrectly(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("ListTools", mock.Anything).Return(sampleTools(), nil)

	b := testBridge(caller)
	_ = b.RefreshTools(context.Background())

	tools := b.ToZAITools()

	assert.Len(t, tools, 2)
	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "list_ip_pools", tools[0].Function.Name)
	assert.Equal(t, "List all IP pools", tools[0].Function.Description)
	assert.Equal(t, "add_firewall_rule", tools[1].Function.Name)
}

func TestToZAITools_EmptyWhenNoTools(t *testing.T) {
	b := testBridge(&mockMCPCaller{})
	tools := b.ToZAITools()
	assert.Empty(t, tools)
}

// ── ToolNames ─────────────────────────────────────────────────────────────────

func TestToolNames(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("ListTools", mock.Anything).Return(sampleTools(), nil)

	b := testBridge(caller)
	_ = b.RefreshTools(context.Background())

	names := b.ToolNames()
	assert.ElementsMatch(t, []string{"list_ip_pools", "add_firewall_rule"}, names)
}

// ── normalizeSchema ───────────────────────────────────────────────────────────

func TestNormalizeSchema_EmptyGetsDefaults(t *testing.T) {
	result := normalizeSchema(map[string]interface{}{})
	assert.Equal(t, "object", result["type"])
	assert.NotNil(t, result["properties"])
}

func TestNormalizeSchema_NilGetsDefaults(t *testing.T) {
	result := normalizeSchema(nil)
	assert.Equal(t, "object", result["type"])
}

func TestNormalizeSchema_PreservesExistingType(t *testing.T) {
	input := map[string]interface{}{"type": "object", "properties": map[string]interface{}{"x": "y"}}
	result := normalizeSchema(input)
	assert.Equal(t, "object", result["type"])
}

func TestNormalizeSchema_AddsPropertiesIfMissing(t *testing.T) {
	input := map[string]interface{}{"type": "object"}
	result := normalizeSchema(input)
	assert.NotNil(t, result["properties"])
}

// ── isWriteTool ───────────────────────────────────────────────────────────────

func TestIsWriteTool_WritePrefixes(t *testing.T) {
	writeCases := []string{
		"add_ip_pool", "create_user", "delete_pool", "remove_rule",
		"update_config", "set_bandwidth", "enable_rule", "disable_rule",
		"toggle_firewall", "reboot_router", "reset_config", "move_queue",
		"kick_user",
	}
	for _, tc := range writeCases {
		assert.True(t, isWriteTool(tc), "%s should be write tool", tc)
	}
}

func TestIsWriteTool_ReadPrefixes(t *testing.T) {
	readCases := []string{
		"list_ip_pools", "get_resource", "watch_traffic",
		"list_firewall_rules", "list_hotspot_users",
	}
	for _, tc := range readCases {
		assert.False(t, isWriteTool(tc), "%s should NOT be write tool", tc)
	}
}

// ── ToZAITools concurrency ────────────────────────────────────────────────────

func TestToZAITools_ConcurrentSafe(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("ListTools", mock.Anything).Return(sampleTools(), nil)

	b := testBridge(caller)
	_ = b.RefreshTools(context.Background())

	// Jalankan concurrent reads
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			tools := b.ToZAITools()
			assert.Len(t, tools, 2)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ── Verify concrete types satisfy interfaces ──────────────────────────────────

func TestMCPBridge_SatisfiesOrchestratorInterface(t *testing.T) {
	// Kompilasi-time check: *MCPBridge harus implement semua method yang dibutuhkan
	b := &MCPBridge{}
	var _ interface {
		ToZAITools() []zai.Tool
		Execute(ctx context.Context, call zai.FunctionCall, opts ExecuteOptions) string
		ToolCount() int
		ToolNames() []string
	} = b
}
