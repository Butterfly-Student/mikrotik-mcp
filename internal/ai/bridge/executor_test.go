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

// ── Helpers ───────────────────────────────────────────────────────────────────

func bridgeWithTools(caller *mockMCPCaller) *MCPBridge {
	caller.On("ListTools", mock.Anything).Return(sampleTools(), nil)
	b := New(caller, zap.NewNop())
	_ = b.RefreshTools(context.Background())
	return b
}

// ── Access Control ────────────────────────────────────────────────────────────

func TestExecute_ReadonlyBlocksWriteTool(t *testing.T) {
	caller := &mockMCPCaller{}
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "add_firewall_rule",
		Arguments: `{"chain":"forward","action":"drop"}`,
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "readonly"})

	assert.Contains(t, result, "access_denied")
	assert.Contains(t, result, "readonly")
	caller.AssertNotCalled(t, "CallTool")
}

func TestExecute_ReadonlyAllowsReadTool(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("CallTool", mock.Anything, "list_ip_pools", mock.Anything).
		Return(&mcpclient.CallResult{
			Content: []mcpclient.ContentBlock{{Type: "text", Text: `{"pools":[]}`}},
		}, nil)
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "list_ip_pools",
		Arguments: "{}",
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "readonly"})

	assert.Equal(t, `{"pools":[]}`, result)
	caller.AssertCalled(t, "CallTool", mock.Anything, "list_ip_pools", mock.Anything)
}

func TestExecute_FullAccessAllowsWriteTool(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("CallTool", mock.Anything, "add_firewall_rule", mock.Anything).
		Return(&mcpclient.CallResult{
			Content: []mcpclient.ContentBlock{{Type: "text", Text: `{"success":true}`}},
		}, nil)
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "add_firewall_rule",
		Arguments: `{"chain":"forward","action":"drop"}`,
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "full"})

	assert.Equal(t, `{"success":true}`, result)
	caller.AssertExpectations(t)
}

// ── Argument Parsing ──────────────────────────────────────────────────────────

func TestExecute_InvalidJSONArgs(t *testing.T) {
	caller := &mockMCPCaller{}
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "list_ip_pools",
		Arguments: `{not valid json}`,
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "full"})

	assert.Contains(t, result, "invalid_args")
	caller.AssertNotCalled(t, "CallTool")
}

func TestExecute_EmptyArgs(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("CallTool", mock.Anything, "list_ip_pools", mock.Anything).
		Return(&mcpclient.CallResult{
			Content: []mcpclient.ContentBlock{{Type: "text", Text: "ok"}},
		}, nil)
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "list_ip_pools",
		Arguments: "",
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "full"})

	assert.Equal(t, "ok", result)
}

func TestExecute_EmptyBracesArgs(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("CallTool", mock.Anything, "list_ip_pools", mock.Anything).
		Return(&mcpclient.CallResult{
			Content: []mcpclient.ContentBlock{{Type: "text", Text: "ok"}},
		}, nil)
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "list_ip_pools",
		Arguments: "{}",
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "full"})

	assert.Equal(t, "ok", result)
}

// ── MCP Error Handling ────────────────────────────────────────────────────────

func TestExecute_MCPCallError(t *testing.T) {
	caller := &mockMCPCaller{}
	caller.On("CallTool", mock.Anything, "list_ip_pools", mock.Anything).
		Return(nil, assert.AnError)
	b := bridgeWithTools(caller)

	result := b.Execute(context.Background(), zai.FunctionCall{
		Name:      "list_ip_pools",
		Arguments: "{}",
	}, ExecuteOptions{Phone: "628xxx", AccessLevel: "full"})

	assert.Contains(t, result, "execution_failed")
}

// ── extractText ───────────────────────────────────────────────────────────────

func TestExtractText_NilResult(t *testing.T) {
	assert.Equal(t, `{"result":"ok"}`, extractText(nil))
}

func TestExtractText_EmptyContent(t *testing.T) {
	assert.Equal(t, `{"result":"ok"}`, extractText(&mcpclient.CallResult{}))
}

func TestExtractText_SingleBlock(t *testing.T) {
	r := &mcpclient.CallResult{
		Content: []mcpclient.ContentBlock{{Type: "text", Text: "hello"}},
	}
	assert.Equal(t, "hello", extractText(r))
}

func TestExtractText_MultipleBlocks(t *testing.T) {
	r := &mcpclient.CallResult{
		Content: []mcpclient.ContentBlock{
			{Type: "text", Text: "block1"},
			{Type: "text", Text: "block2"},
		},
	}
	result := extractText(r)
	assert.Contains(t, result, "block1")
	assert.Contains(t, result, "block2")
}

func TestExtractText_NonTextBlockIgnored(t *testing.T) {
	r := &mcpclient.CallResult{
		Content: []mcpclient.ContentBlock{
			{Type: "image", Text: ""},
			{Type: "text", Text: "visible"},
		},
	}
	assert.Equal(t, "visible", extractText(r))
}
