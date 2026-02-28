package zai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func mockZAIServer(t *testing.T, resp ChatResponse, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	return NewClient("test-api-key", serverURL, "glm-4-airx", zap.NewNop())
}

func TestChat_SuccessStop(t *testing.T) {
	srv := mockZAIServer(t, ChatResponse{
		ID:    "chatcmpl-abc123",
		Model: "glm-4-airx",
		Choices: []Choice{{
			Index:        0,
			FinishReason: "stop",
			Message:      Message{Role: "assistant", Content: "Ada 2 IP pool."},
		}},
		Usage: Usage{TotalTokens: 42},
	}, http.StatusOK)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "tampilkan IP pool"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	assert.Equal(t, "Ada 2 IP pool.", resp.Choices[0].Message.Content)
	assert.Equal(t, 42, resp.Usage.TotalTokens)
}

func TestChat_ToolCallResponse(t *testing.T) {
	srv := mockZAIServer(t, ChatResponse{
		Choices: []Choice{{
			FinishReason: "tool_calls",
			Message: Message{
				Role: "assistant",
				ToolCalls: []ToolCall{{
					ID:   "call_xyz",
					Type: "function",
					Function: FunctionCall{
						Name:      "list_ip_pools",
						Arguments: "{}",
					},
				}},
			},
		}},
	}, http.StatusOK)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "lihat pool"}},
		Tools: []Tool{{
			Type: "function",
			Function: Function{
				Name:        "list_ip_pools",
				Description: "List all IP pools",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, "tool_calls", resp.Choices[0].FinishReason)
	assert.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "list_ip_pools", resp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestChat_APIError(t *testing.T) {
	srv := mockZAIServer(t, ChatResponse{
		Error: &APIError{Code: "invalid_api_key", Message: "Invalid API key provided"},
	}, http.StatusOK)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key provided")
}

func TestChat_EmptyChoices(t *testing.T) {
	srv := mockZAIServer(t, ChatResponse{Choices: []Choice{}}, http.StatusOK)
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices")
}

func TestChat_UsesDefaultModel(t *testing.T) {
	var capturedBody ChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(ChatResponse{
			Choices: []Choice{{
				FinishReason: "stop",
				Message:      Message{Role: "assistant", Content: "ok"},
			}},
		})
	}))
	defer srv.Close()

	client := NewClient("key", srv.URL, "glm-4-airx", zap.NewNop())
	_, err := client.Chat(context.Background(), ChatRequest{
		// Model tidak diset — harus pakai default dari client
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "glm-4-airx", capturedBody.Model)
}

func TestChat_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.Error(t, err)
}

func TestChat_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server lambat — context akan dibatalkan dulu
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // langsung cancel

	client := newTestClient(t, srv.URL)
	_, err := client.Chat(ctx, ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.Error(t, err)
}
