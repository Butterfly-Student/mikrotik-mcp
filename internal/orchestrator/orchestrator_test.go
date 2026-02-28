package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/bridge"
	"mikrotik-mcp/internal/ai/zai"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockZAI struct{ mock.Mock }

func (m *mockZAI) Chat(ctx context.Context, req zai.ChatRequest) (*zai.ChatResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*zai.ChatResponse), args.Error(1)
}

type mockBridge struct{ mock.Mock }

func (m *mockBridge) ToZAITools() []zai.Tool {
	return m.Called().Get(0).([]zai.Tool)
}

func (m *mockBridge) Execute(ctx context.Context, call zai.FunctionCall, opts bridge.ExecuteOptions) string {
	return m.Called(ctx, call, opts).String(0)
}

func (m *mockBridge) ToolCount() int {
	return m.Called().Int(0)
}

func (m *mockBridge) ToolNames() []string {
	return m.Called().Get(0).([]string)
}

type mockSession struct{ mock.Mock }

func (m *mockSession) GetHistory(ctx context.Context, phone string) ([]zai.Message, error) {
	args := m.Called(ctx, phone)
	return args.Get(0).([]zai.Message), args.Error(1)
}

func (m *mockSession) AppendMessages(ctx context.Context, phone string, msgs ...zai.Message) error {
	return m.Called(ctx, phone, msgs).Error(0)
}

func (m *mockSession) ResetSession(ctx context.Context, phone string) error {
	return m.Called(ctx, phone).Error(0)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func testOrchestrator(zaiMock *mockZAI, bridgeMock *mockBridge, sessionMock *mockSession) *Orchestrator {
	return New(Config{
		ZAI:          zaiMock,
		Bridge:       bridgeMock,
		Session:      sessionMock,
		SystemPrompt: "You are MikroBot.",
		Model:        "glm-4-airx",
		MaxTokens:    512,
		Temperature:  0.7,
		MaxLoops:     3,
	}, zap.NewNop())
}

func stopResponse(content string) *zai.ChatResponse {
	return &zai.ChatResponse{
		Choices: []zai.Choice{{
			FinishReason: "stop",
			Message:      zai.Message{Role: "assistant", Content: content},
		}},
	}
}

func toolCallResponse(toolID, toolName, args string) *zai.ChatResponse {
	return &zai.ChatResponse{
		Choices: []zai.Choice{{
			FinishReason: "tool_calls",
			Message: zai.Message{
				Role: "assistant",
				ToolCalls: []zai.ToolCall{{
					ID:   toolID,
					Type: "function",
					Function: zai.FunctionCall{
						Name:      toolName,
						Arguments: args,
					},
				}},
			},
		}},
	}
}

// ── Special Commands ──────────────────────────────────────────────────────────

func TestProcess_SpecialCommand_Reset(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}
	sessionMock.On("ResetSession", mock.Anything, "628xxx").Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "/reset")

	require.NoError(t, err)
	assert.Contains(t, resp, "dihapus")
	sessionMock.AssertCalled(t, "ResetSession", mock.Anything, "628xxx")
	zaiMock.AssertNotCalled(t, "Chat")
}

func TestProcess_SpecialCommand_Status(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}
	bridgeMock.On("ToolCount").Return(18)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "/status")

	require.NoError(t, err)
	assert.Contains(t, resp, "glm-4-airx")
	assert.Contains(t, resp, "18")
	assert.Contains(t, resp, "full")
}

func TestProcess_SpecialCommand_Tools(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}
	bridgeMock.On("ToolNames").Return([]string{"list_ip_pools", "add_firewall_rule"})

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "/tools")

	require.NoError(t, err)
	assert.Contains(t, resp, "list_ip_pools")
	assert.Contains(t, resp, "add_firewall_rule")
}

func TestProcess_SpecialCommand_ToolsEmpty(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}
	bridgeMock.On("ToolNames").Return([]string{})

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "/tools")

	require.NoError(t, err)
	assert.Contains(t, resp, "Tidak ada tools")
}

func TestProcess_SpecialCommand_Whoami(t *testing.T) {
	orch := testOrchestrator(&mockZAI{}, &mockBridge{}, &mockSession{})
	resp, err := orch.Process(context.Background(), "6281234567890", "readonly", "/whoami")

	require.NoError(t, err)
	assert.Contains(t, resp, "6281234567890")
	assert.Contains(t, resp, "readonly")
}

func TestProcess_SpecialCommand_Help(t *testing.T) {
	orch := testOrchestrator(&mockZAI{}, &mockBridge{}, &mockSession{})
	resp, err := orch.Process(context.Background(), "628xxx", "full", "/help")

	require.NoError(t, err)
	assert.Contains(t, resp, "MikroBot")
	assert.Contains(t, resp, "/reset")
}

func TestProcess_SpecialCommand_CaseInsensitive(t *testing.T) {
	sessionMock := &mockSession{}
	sessionMock.On("ResetSession", mock.Anything, "628xxx").Return(nil)
	orch := testOrchestrator(&mockZAI{}, &mockBridge{}, sessionMock)

	resp, err := orch.Process(context.Background(), "628xxx", "full", "  /RESET  ")

	require.NoError(t, err)
	assert.Contains(t, resp, "dihapus")
}

// ── Function Call Loop ────────────────────────────────────────────────────────

func TestProcess_DirectStop(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	sessionMock.On("GetHistory", mock.Anything, "628xxx").Return([]zai.Message{}, nil)
	bridgeMock.On("ToZAITools").Return([]zai.Tool{})
	zaiMock.On("Chat", mock.Anything, mock.AnythingOfType("zai.ChatRequest")).
		Return(stopResponse("Ada 2 IP pool aktif."), nil)
	sessionMock.On("AppendMessages", mock.Anything, "628xxx", mock.Anything).Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "tampilkan IP pool")

	require.NoError(t, err)
	assert.Equal(t, "Ada 2 IP pool aktif.", resp)
}

func TestProcess_ToolCallThenStop(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	sessionMock.On("GetHistory", mock.Anything, "628xxx").Return([]zai.Message{}, nil)
	bridgeMock.On("ToZAITools").Return([]zai.Tool{{
		Type:     "function",
		Function: zai.Function{Name: "list_ip_pools"},
	}})

	// Loop 1: GLM minta tool
	zaiMock.On("Chat", mock.Anything, mock.MatchedBy(func(r zai.ChatRequest) bool {
		// Pesan pertama: system + user
		return len(r.Messages) == 2
	})).Return(toolCallResponse("call_1", "list_ip_pools", "{}"), nil).Once()

	// Bridge eksekusi tool
	bridgeMock.On("Execute", mock.Anything,
		zai.FunctionCall{Name: "list_ip_pools", Arguments: "{}"},
		bridge.ExecuteOptions{Phone: "628xxx", AccessLevel: "full"},
	).Return(`{"pools":["pool-a","pool-b"]}`)

	// Loop 2: GLM jawab "stop"
	zaiMock.On("Chat", mock.Anything, mock.MatchedBy(func(r zai.ChatRequest) bool {
		// Pesan: system + user + assistant(tool_calls) + tool result
		return len(r.Messages) == 4
	})).Return(stopResponse("Ada 2 pool: pool-a dan pool-b."), nil).Once()

	sessionMock.On("AppendMessages", mock.Anything, "628xxx", mock.Anything).Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "tampilkan IP pool")

	require.NoError(t, err)
	assert.Equal(t, "Ada 2 pool: pool-a dan pool-b.", resp)
	zaiMock.AssertNumberOfCalls(t, "Chat", 2)
	bridgeMock.AssertCalled(t, "Execute", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcess_MaxLoopsReached(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	sessionMock.On("GetHistory", mock.Anything, "628xxx").Return([]zai.Message{}, nil)
	bridgeMock.On("ToZAITools").Return([]zai.Tool{})
	bridgeMock.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(`{"ok":true}`)

	// GLM selalu minta tool — tidak pernah "stop"
	zaiMock.On("Chat", mock.Anything, mock.AnythingOfType("zai.ChatRequest")).
		Return(toolCallResponse("call_x", "list_ip_pools", "{}"), nil)

	sessionMock.On("AppendMessages", mock.Anything, "628xxx", mock.Anything).Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "loop forever")

	require.NoError(t, err)
	assert.Contains(t, resp, "terlalu kompleks")
	// MaxLoops=3, jadi Chat dipanggil 3 kali
	zaiMock.AssertNumberOfCalls(t, "Chat", 3)
}

func TestProcess_ZAIError(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	sessionMock.On("GetHistory", mock.Anything, "628xxx").Return([]zai.Message{}, nil)
	bridgeMock.On("ToZAITools").Return([]zai.Tool{})
	zaiMock.On("Chat", mock.Anything, mock.AnythingOfType("zai.ChatRequest")).
		Return(nil, errors.New("API timeout"))

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	_, err := orch.Process(context.Background(), "628xxx", "full", "hello")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Z.AI error")
}

func TestProcess_HistoryError_ContinuesWithEmpty(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	// GetHistory gagal — tapi process harus tetap jalan
	sessionMock.On("GetHistory", mock.Anything, "628xxx").
		Return([]zai.Message{}, errors.New("db error"))
	bridgeMock.On("ToZAITools").Return([]zai.Tool{})
	zaiMock.On("Chat", mock.Anything, mock.AnythingOfType("zai.ChatRequest")).
		Return(stopResponse("ok"), nil)
	sessionMock.On("AppendMessages", mock.Anything, "628xxx", mock.Anything).Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	resp, err := orch.Process(context.Background(), "628xxx", "full", "hello")

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestProcess_IncludesHistoryInMessages(t *testing.T) {
	zaiMock := &mockZAI{}
	bridgeMock := &mockBridge{}
	sessionMock := &mockSession{}

	history := []zai.Message{
		{Role: "user", Content: "pesan lama"},
		{Role: "assistant", Content: "jawaban lama"},
	}
	sessionMock.On("GetHistory", mock.Anything, "628xxx").Return(history, nil)
	bridgeMock.On("ToZAITools").Return([]zai.Tool{})

	var capturedReq zai.ChatRequest
	zaiMock.On("Chat", mock.Anything, mock.AnythingOfType("zai.ChatRequest")).
		Run(func(args mock.Arguments) {
			capturedReq = args.Get(1).(zai.ChatRequest)
		}).
		Return(stopResponse("ok"), nil)
	sessionMock.On("AppendMessages", mock.Anything, "628xxx", mock.Anything).Return(nil)

	orch := testOrchestrator(zaiMock, bridgeMock, sessionMock)
	_, err := orch.Process(context.Background(), "628xxx", "full", "pesan baru")

	require.NoError(t, err)
	// system + history(2) + user_baru = 4 pesan
	assert.Len(t, capturedReq.Messages, 4)
	assert.Equal(t, "system", capturedReq.Messages[0].Role)
	assert.Equal(t, "user", capturedReq.Messages[1].Role)
	assert.Equal(t, "assistant", capturedReq.Messages[2].Role)
	assert.Equal(t, "user", capturedReq.Messages[3].Role)
	assert.Equal(t, "pesan baru", capturedReq.Messages[3].Content)
}
