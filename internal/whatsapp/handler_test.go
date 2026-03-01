package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mikrotik-mcp/internal/config"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockProcessor struct{ mock.Mock }

func (m *mockProcessor) Process(ctx context.Context, phone, accessLevel, text string) (string, error) {
	args := m.Called(ctx, phone, accessLevel, text)
	return args.String(0), args.Error(1)
}

type mockSender struct{ mock.Mock }

func (m *mockSender) SendText(ctx context.Context, phone, text string) error {
	return m.Called(ctx, phone, text).Error(0)
}

func (m *mockSender) DelayedStatus(ctx context.Context, phone, msg string, delay time.Duration) func() {
	m.Called(ctx, phone, msg, delay)
	return func() {}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func testHandler(proc *mockProcessor, sender *mockSender) *Handler {
	auth := NewMiddleware([]config.AuthUser{
		{Phone: "6281000000001", Name: "Admin", Access: "full"},
	})
	return NewHandler(proc, sender, auth, "", zap.NewNop())
}

func webhookBody(t *testing.T, event, from, body string, isFromMe bool, groupChat bool) *bytes.Buffer {
	t.Helper()
	chatID := from
	if groupChat {
		chatID = "120363402106XXXXX@g.us"
	}
	payload := GowaWebhookPayload{
		Event:    event,
		DeviceID: "628bot@s.whatsapp.net",
		Payload: MessagePayload{
			ID:       "MSGID123",
			ChatID:   chatID,
			From:     from,
			FromName: "Test User",
			Body:     body,
			IsFromMe: isFromMe,
		},
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	req := httptest.NewRequest(http.MethodPost, "/webhook/message", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	proc.AssertNotCalled(t, "Process")
}

func TestHandleWebhook_NonMessageEvent(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	body := webhookBody(t, "message.ack", "6281000000001@s.whatsapp.net", "hello", false, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertNotCalled(t, "Process")
}

func TestHandleWebhook_IgnoresOwnMessage(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	body := webhookBody(t, "message", "6281000000001@s.whatsapp.net", "hello", true, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertNotCalled(t, "Process")
}

func TestHandleWebhook_IgnoresGroupMessage(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	body := webhookBody(t, "message", "6281000000001@s.whatsapp.net", "hello", false, true)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertNotCalled(t, "Process")
}

func TestHandleWebhook_IgnoresEmptyBody(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	body := webhookBody(t, "message", "6281000000001@s.whatsapp.net", "   ", false, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertNotCalled(t, "Process")
}

func TestHandleWebhook_UnauthorizedUser(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	// 6289999999999 tidak ada dalam whitelist
	sender.On("SendText", mock.Anything, "6289999999999", mock.AnythingOfType("string")).Return(nil)

	body := webhookBody(t, "message", "6289999999999@s.whatsapp.net", "list pools", false, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)
	time.Sleep(50 * time.Millisecond) // beri waktu goroutine selesai

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertNotCalled(t, "Process")
	sender.AssertCalled(t, "SendText", mock.Anything, "6289999999999", mock.AnythingOfType("string"))
}

func TestHandleWebhook_ValidMessage_SendsResponse(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	proc.On("Process", mock.Anything, "6281000000001", "full", "list pools").
		Return("Ada 3 IP pool.", nil)
	sender.On("DelayedStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("SendText", mock.Anything, "6281000000001", "Ada 3 IP pool.").Return(nil)

	body := webhookBody(t, "message", "6281000000001@s.whatsapp.net", "list pools", false, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertExpectations(t)
	sender.AssertExpectations(t)
}

func TestHandleWebhook_ProcessError_SendsErrorMessage(t *testing.T) {
	proc := &mockProcessor{}
	sender := &mockSender{}
	h := testHandler(proc, sender)

	proc.On("Process", mock.Anything, "6281000000001", "full", "fail").
		Return("", assert.AnError)
	sender.On("DelayedStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("SendText", mock.Anything, "6281000000001", mock.MatchedBy(func(s string) bool {
		return len(s) > 0
	})).Return(nil)

	body := webhookBody(t, "message", "6281000000001@s.whatsapp.net", "fail", false, false)
	req := httptest.NewRequest(http.MethodPost, "/webhook/message", body)
	w := httptest.NewRecorder()

	h.HandleWebhook(w, req)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code)
	proc.AssertExpectations(t)
}
