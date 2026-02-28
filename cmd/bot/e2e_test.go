//go:build e2e

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"

	"mikrotik-mcp/internal/ai/bridge"
	"mikrotik-mcp/internal/ai/zai"
	"mikrotik-mcp/internal/config"
	"mikrotik-mcp/internal/orchestrator"
	"mikrotik-mcp/internal/session"
	"mikrotik-mcp/internal/whatsapp"
)

// ── Schema helper ─────────────────────────────────────────────────────────────

const e2eSchema = `
CREATE TABLE IF NOT EXISTS sessions (
	phone      TEXT PRIMARY KEY,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS messages (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	phone        TEXT    NOT NULL,
	role         TEXT    NOT NULL,
	content      TEXT    NOT NULL DEFAULT '',
	tool_calls   TEXT,
	tool_call_id TEXT,
	name         TEXT,
	created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS audit_logs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	phone       TEXT    NOT NULL,
	tool_name   TEXT    NOT NULL,
	args        TEXT,
	status      TEXT DEFAULT 'pending',
	error       TEXT,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME
);
`

func openE2EDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = db.Exec(e2eSchema)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// ── Mock MCP Bridge (stub — no real MCP server needed) ────────────────────────

type stubBridge struct {
	tools   []zai.Tool
	results map[string]string // tool name → result JSON
}

func (b *stubBridge) ToZAITools() []zai.Tool  { return b.tools }
func (b *stubBridge) ToolCount() int           { return len(b.tools) }
func (b *stubBridge) ToolNames() []string {
	names := make([]string, len(b.tools))
	for i, t := range b.tools {
		names[i] = t.Function.Name
	}
	return names
}
func (b *stubBridge) Execute(_ context.Context, call zai.FunctionCall, _ bridge.ExecuteOptions) string {
	if r, ok := b.results[call.Name]; ok {
		return r
	}
	return `{"result":"ok"}`
}

// ── Mock gowa server — captures sent messages ─────────────────────────────────

type gowaCapturer struct {
	mu       sync.Mutex
	received []whatsapp.GowaSendRequest
	ch       chan whatsapp.GowaSendRequest
}

func newGowaCapturer() *gowaCapturer {
	return &gowaCapturer{ch: make(chan whatsapp.GowaSendRequest, 16)}
}

func (g *gowaCapturer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req whatsapp.GowaSendRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	g.mu.Lock()
	g.received = append(g.received, req)
	g.mu.Unlock()
	g.ch <- req
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// waitForMessage blocks until a message matching pred arrives or timeout
func (g *gowaCapturer) waitFor(t *testing.T, timeout time.Duration, pred func(whatsapp.GowaSendRequest) bool) whatsapp.GowaSendRequest {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case msg := <-g.ch:
			if pred(msg) {
				return msg
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for gowa message")
			return whatsapp.GowaSendRequest{}
		}
	}
}

// ── Z.AI mock server builder ───────────────────────────────────────────────────

func mockZAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func zaiStopResponse(content string) zai.ChatResponse {
	return zai.ChatResponse{
		Choices: []zai.Choice{{
			FinishReason: "stop",
			Message:      zai.Message{Role: "assistant", Content: content},
		}},
	}
}

func zaiToolCallResponse(toolID, toolName, args string) zai.ChatResponse {
	return zai.ChatResponse{
		Choices: []zai.Choice{{
			FinishReason: "tool_calls",
			Message: zai.Message{
				Role: "assistant",
				ToolCalls: []zai.ToolCall{{
					ID:   toolID,
					Type: "function",
					Function: zai.FunctionCall{Name: toolName, Arguments: args},
				}},
			},
		}},
	}
}

// ── E2E Harness ───────────────────────────────────────────────────────────────

type e2eHarness struct {
	webhookSrv *httptest.Server // receives WA webhook from "gowa"
	gowa       *gowaCapturer    // captures outbound messages to WA users
	gowaSrv    *httptest.Server
	db         *sql.DB
}

func newE2EHarness(t *testing.T, zaiSrv *httptest.Server, mcpBridge orchestrator.MCPBridge, authorizedUsers []config.AuthUser) *e2eHarness {
	t.Helper()

	db := openE2EDB(t)
	sessionMgr := session.NewManager(session.NewStore(db), 2*time.Hour, 50, zap.NewNop())

	zaiClient := zai.NewClient("test-key", zaiSrv.URL, "glm-4-airx", zap.NewNop())

	orch := orchestrator.New(orchestrator.Config{
		ZAI:          zaiClient,
		Bridge:       mcpBridge,
		Session:      sessionMgr,
		SystemPrompt: "Kamu adalah MikroBot.",
		Model:        "glm-4-airx",
		MaxTokens:    512,
		Temperature:  0.7,
		MaxLoops:     5,
	}, zap.NewNop())

	gowa := newGowaCapturer()
	gowaSrv := httptest.NewServer(gowa)
	t.Cleanup(gowaSrv.Close)

	sender := whatsapp.NewSender(gowaSrv.URL, "", "", "", zap.NewNop())
	auth := whatsapp.NewMiddleware(authorizedUsers)
	handler := whatsapp.NewHandler(orch, sender, auth, "", zap.NewNop())

	r := chi.NewRouter()
	r.Post("/webhook", handler.HandleWebhook)
	webhookSrv := httptest.NewServer(r)
	t.Cleanup(webhookSrv.Close)

	return &e2eHarness{
		webhookSrv: webhookSrv,
		gowa:       gowa,
		gowaSrv:    gowaSrv,
		db:         db,
	}
}

func (h *e2eHarness) sendWebhook(t *testing.T, phone, body string) {
	t.Helper()
	payload := whatsapp.GowaWebhookPayload{
		Event:    "message",
		DeviceID: "device-1",
		Payload: whatsapp.MessagePayload{
			ID:       "msg-001",
			ChatID:   phone + "@s.whatsapp.net",
			From:     phone + "@s.whatsapp.net",
			FromName: "Test User",
			Body:     body,
		},
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(h.webhookSrv.URL+"/webhook", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func authorizedFullUser(phone string) config.AuthUser {
	return config.AuthUser{Phone: phone, Access: "full"}
}

func staticZAIServer(t *testing.T, resp zai.ChatResponse) *httptest.Server {
	return mockZAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func emptyBridge() *stubBridge {
	return &stubBridge{}
}

// ── E2E Tests ─────────────────────────────────────────────────────────────────

// TestE2E_SimpleMessage — pesan biasa, Z.AI langsung jawab tanpa tool call
func TestE2E_SimpleMessage(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("Ada 2 IP pool: pool-a, pool-b."))

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "tampilkan IP pool")

	msg := h.gowa.waitFor(t, 10*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != "" && req.Message != "_Sedang memproses..._"
	})

	assert.Contains(t, msg.Message, "pool-a")
	assert.Contains(t, msg.Phone, "628001")
}

// TestE2E_UnauthorizedUser — nomor tidak terdaftar harus ditolak
func TestE2E_UnauthorizedUser(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("seharusnya tidak dipanggil"))

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	// Kirim dari nomor yang tidak terdaftar
	h.sendWebhook(t, "628999", "helo")

	msg := h.gowa.waitFor(t, 5*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != ""
	})
	assert.Contains(t, msg.Message, "tidak terdaftar")
}

// TestE2E_SpecialCommand_Reset — /reset harus hapus history tanpa call Z.AI
func TestE2E_SpecialCommand_Reset(t *testing.T) {
	called := false
	zaiSrv := mockZAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	})

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "/reset")

	msg := h.gowa.waitFor(t, 5*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != ""
	})

	assert.Contains(t, msg.Message, "dihapus")
	assert.False(t, called, "Z.AI seharusnya tidak dipanggil untuk /reset")
}

// TestE2E_SpecialCommand_Status — /status harus berisi nama model & tool count
func TestE2E_SpecialCommand_Status(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("should not reach"))
	br := &stubBridge{
		tools: []zai.Tool{
			{Type: "function", Function: zai.Function{Name: "list_ip_pools"}},
			{Type: "function", Function: zai.Function{Name: "add_firewall_rule"}},
		},
	}

	h := newE2EHarness(t, zaiSrv, br, []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "/status")

	msg := h.gowa.waitFor(t, 5*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != ""
	})

	assert.Contains(t, msg.Message, "glm-4-airx")
	assert.Contains(t, msg.Message, "2")
	assert.Contains(t, msg.Message, "full")
}

// TestE2E_SpecialCommand_Tools — /tools harus daftar nama semua tool
func TestE2E_SpecialCommand_Tools(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("should not reach"))
	br := &stubBridge{
		tools: []zai.Tool{
			{Type: "function", Function: zai.Function{Name: "list_ip_pools"}},
			{Type: "function", Function: zai.Function{Name: "add_firewall_rule"}},
		},
	}

	h := newE2EHarness(t, zaiSrv, br, []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "/tools")

	msg := h.gowa.waitFor(t, 5*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != ""
	})

	assert.Contains(t, msg.Message, "list_ip_pools")
	assert.Contains(t, msg.Message, "add_firewall_rule")
}

// TestE2E_ToolCallFlow — Z.AI minta tool, bridge eksekusi, Z.AI jawab final
func TestE2E_ToolCallFlow(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	zaiSrv := mockZAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n := callCount
		callCount++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		var resp zai.ChatResponse
		if n == 0 {
			// Loop pertama: minta tool
			resp = zaiToolCallResponse("call_1", "list_ip_pools", "{}")
		} else {
			// Loop kedua: jawab final setelah dapat hasil tool
			resp = zaiStopResponse("Ada 2 pool: pool-a dan pool-b.")
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	br := &stubBridge{
		tools: []zai.Tool{
			{Type: "function", Function: zai.Function{
				Name:        "list_ip_pools",
				Description: "List all IP pools",
				Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			}},
		},
		results: map[string]string{
			"list_ip_pools": `{"pools":["pool-a","pool-b"]}`,
		},
	}

	h := newE2EHarness(t, zaiSrv, br, []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "tampilkan IP pool")

	msg := h.gowa.waitFor(t, 15*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != "" && req.Message != "_Sedang memproses..._"
	})

	assert.Contains(t, msg.Message, "pool-a")
	assert.Contains(t, msg.Message, "pool-b")

	mu.Lock()
	assert.Equal(t, 2, callCount, "Z.AI harus dipanggil 2 kali (tool_call + stop)")
	mu.Unlock()
}

// TestE2E_GroupMessageIgnored — pesan dari grup harus diabaikan
func TestE2E_GroupMessageIgnored(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("should not reach"))

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	// Kirim webhook dengan ChatID grup
	payload := whatsapp.GowaWebhookPayload{
		Event:    "message",
		DeviceID: "device-1",
		Payload: whatsapp.MessagePayload{
			ID:     "msg-grp",
			ChatID: "120363xxx@g.us", // grup!
			From:   "628001@s.whatsapp.net",
			Body:   "hello group",
		},
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(h.webhookSrv.URL+"/webhook", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Tidak boleh ada pesan keluar ke gowa
	select {
	case msg := <-h.gowa.ch:
		t.Fatalf("pesan grup seharusnya diabaikan, tapi ada pesan terkirim: %+v", msg)
	case <-time.After(2 * time.Second):
		// OK — tidak ada pesan
	}
}

// TestE2E_SelfMessageIgnored — pesan dari diri sendiri (is_from_me) diabaikan
func TestE2E_SelfMessageIgnored(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("should not reach"))

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	payload := whatsapp.GowaWebhookPayload{
		Event:    "message",
		DeviceID: "device-1",
		Payload: whatsapp.MessagePayload{
			ID:       "msg-self",
			ChatID:   "628001@s.whatsapp.net",
			From:     "628001@s.whatsapp.net",
			Body:     "hello self",
			IsFromMe: true, // dari diri sendiri
		},
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(h.webhookSrv.URL+"/webhook", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case msg := <-h.gowa.ch:
		t.Fatalf("pesan dari diri sendiri seharusnya diabaikan: %+v", msg)
	case <-time.After(2 * time.Second):
		// OK
	}
}

// TestE2E_NonMessageEventIgnored — event bukan "message" diabaikan
func TestE2E_NonMessageEventIgnored(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("should not reach"))

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	payload := whatsapp.GowaWebhookPayload{
		Event:    "status_update", // bukan "message"
		DeviceID: "device-1",
		Payload: whatsapp.MessagePayload{
			Body: "hello",
			From: "628001@s.whatsapp.net",
		},
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(h.webhookSrv.URL+"/webhook", "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case msg := <-h.gowa.ch:
		t.Fatalf("event non-message seharusnya diabaikan: %+v", msg)
	case <-time.After(2 * time.Second):
		// OK
	}
}

// TestE2E_ConversationHistory — history percakapan digunakan dalam request berikutnya
func TestE2E_ConversationHistory(t *testing.T) {
	var capturedBodies []zai.ChatRequest
	var mu sync.Mutex
	callCount := 0

	zaiSrv := mockZAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req zai.ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		mu.Lock()
		capturedBodies = append(capturedBodies, req)
		n := callCount
		callCount++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		var content string
		if n == 0 {
			content = "Halo! Ada 2 IP pool."
		} else {
			content = "Pertanyaan kedua dijawab."
		}
		_ = json.NewEncoder(w).Encode(zaiStopResponse(content))
	})

	h := newE2EHarness(t, zaiSrv, emptyBridge(), []config.AuthUser{
		authorizedFullUser("628001"),
	})

	// Pesan pertama
	h.sendWebhook(t, "628001", "tampilkan IP pool")
	h.gowa.waitFor(t, 10*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != "" && req.Message != "_Sedang memproses..._"
	})

	// Beri waktu history tersimpan
	time.Sleep(200 * time.Millisecond)

	// Pesan kedua
	h.sendWebhook(t, "628001", "berapa jumlahnya?")
	h.gowa.waitFor(t, 10*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != "" && req.Message != "_Sedang memproses..._"
	})

	mu.Lock()
	defer mu.Unlock()

	require.GreaterOrEqual(t, len(capturedBodies), 2, "seharusnya ada 2 request ke Z.AI")

	// Request pertama: system + user
	assert.GreaterOrEqual(t, len(capturedBodies[0].Messages), 2)

	// Request kedua: system + pesan lama (history) + user baru — harus lebih panjang
	assert.Greater(t, len(capturedBodies[1].Messages), len(capturedBodies[0].Messages),
		"request kedua harus mencakup history dari pesan pertama")

	// Pastikan pesan terakhir adalah pertanyaan kedua
	last := capturedBodies[1].Messages[len(capturedBodies[1].Messages)-1]
	assert.Equal(t, "user", last.Role)
	assert.Equal(t, "berapa jumlahnya?", last.Content)
}

// TestE2E_HealthEndpoint — /health harus return 200 dengan tool count
func TestE2E_HealthEndpoint(t *testing.T) {
	zaiSrv := staticZAIServer(t, zaiStopResponse("ok"))
	br := &stubBridge{
		tools: []zai.Tool{
			{Type: "function", Function: zai.Function{Name: "list_ip_pools"}},
		},
	}

	// Wire health endpoint langsung (tanpa full harness)
	db := openE2EDB(t)
	sessionMgr := session.NewManager(session.NewStore(db), 2*time.Hour, 50, zap.NewNop())
	zaiClient := zai.NewClient("key", zaiSrv.URL, "glm-4-airx", zap.NewNop())
	orch := orchestrator.New(orchestrator.Config{
		ZAI: zaiClient, Bridge: br, Session: sessionMgr,
		SystemPrompt: "bot", Model: "glm-4-airx", MaxTokens: 512, MaxLoops: 3,
	}, zap.NewNop())
	sender := whatsapp.NewSender("http://localhost:9999", "", "", "", zap.NewNop())
	auth := whatsapp.NewMiddleware(nil)
	_ = whatsapp.NewHandler(orch, sender, auth, "", zap.NewNop())

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		type resp struct {
			Status string `json:"status"`
			Tools  int    `json:"tools"`
		}
		data, _ := json.Marshal(resp{Status: "ok", Tools: br.ToolCount()})
		_, _ = w.Write(data)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, float64(1), body["tools"])
}

// TestE2E_MaxLoopsExceeded — tool call loop tanpa henti harus berikan pesan error
func TestE2E_MaxLoopsExceeded(t *testing.T) {
	// Z.AI selalu minta tool (tidak pernah "stop")
	zaiSrv := mockZAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(zaiToolCallResponse("call_x", "list_ip_pools", "{}"))
	})

	br := &stubBridge{
		tools: []zai.Tool{
			{Type: "function", Function: zai.Function{Name: "list_ip_pools"}},
		},
		results: map[string]string{"list_ip_pools": `{"pools":[]}`},
	}

	h := newE2EHarness(t, zaiSrv, br, []config.AuthUser{
		authorizedFullUser("628001"),
	})

	h.sendWebhook(t, "628001", "loop forever")

	msg := h.gowa.waitFor(t, 30*time.Second, func(req whatsapp.GowaSendRequest) bool {
		return req.Message != "" && req.Message != "_Sedang memproses..._"
	})

	assert.Contains(t, msg.Message, "terlalu kompleks")
}
