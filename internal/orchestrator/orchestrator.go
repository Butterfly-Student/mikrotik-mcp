package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"mikrotik-mcp/internal/ai/bridge"
	"mikrotik-mcp/internal/ai/zai"
)

// ZAIClient adalah interface untuk AI chat client.
type ZAIClient interface {
	Chat(ctx context.Context, req zai.ChatRequest) (*zai.ChatResponse, error)
}

// MCPBridge adalah interface untuk MCP tool bridge.
type MCPBridge interface {
	ToZAITools() []zai.Tool
	Execute(ctx context.Context, call zai.FunctionCall, opts bridge.ExecuteOptions) string
	ToolCount() int
	ToolNames() []string
}

// SessionManager adalah interface untuk manajemen history percakapan.
type SessionManager interface {
	GetHistory(ctx context.Context, phone string) ([]zai.Message, error)
	AppendMessages(ctx context.Context, phone string, msgs ...zai.Message) error
	ResetSession(ctx context.Context, phone string) error
}

type Config struct {
	ZAI          ZAIClient
	Bridge       MCPBridge
	Session      SessionManager
	SystemPrompt string
	Model        string
	MaxTokens    int
	Temperature  float64
	MaxLoops     int
	ThinkingMode string // "enabled" | "disabled" | "" (default server)
}

type Orchestrator struct {
	Config
	logger *zap.Logger
}

func New(cfg Config, logger *zap.Logger) *Orchestrator {
	if cfg.MaxLoops == 0 {
		cfg.MaxLoops = 5
	}
	return &Orchestrator{Config: cfg, logger: logger}
}

// Process adalah entry point — dipanggil dari WA handler per pesan masuk
func (o *Orchestrator) Process(ctx context.Context, phone, accessLevel, userText string) (string, error) {

	// ── Tangani command khusus sebelum masuk ke GLM ───────────────────────────
	if resp, handled := o.handleSpecialCommand(ctx, phone, accessLevel, userText); handled {
		return resp, nil
	}

	// ── 1. Ambil history percakapan dari SQLite ───────────────────────────────
	history, err := o.Session.GetHistory(ctx, phone)
	if err != nil {
		o.logger.Warn("get history failed, proceeding with empty", zap.Error(err))
		history = []zai.Message{}
	}

	// ── 2. Susun messages: [system, ...history, user_baru] ───────────────────
	messages := []zai.Message{
		{Role: "system", Content: o.SystemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, zai.Message{Role: "user", Content: userText})

	// ── 3. Ambil tools dari bridge (sudah di-cache saat startup) ─────────────
	tools := o.Bridge.ToZAITools()

	// newMessages = semua pesan baru di sesi ini, akan disimpan ke SQLite
	newMessages := []zai.Message{{Role: "user", Content: userText}}

	// ── 4. Function Call Loop ─────────────────────────────────────────────────
	for loop := 0; loop < o.MaxLoops; loop++ {
		o.logger.Debug("GLM request",
			zap.String("phone", phone),
			zap.Int("loop", loop),
			zap.Int("total_messages", len(messages)),
		)

		// Kirim ke Z.AI GLM
		chatReq := zai.ChatRequest{
			Model:       o.Model,
			Messages:    messages,
			Tools:       tools,
			ToolChoice:  "auto",
			MaxTokens:   o.MaxTokens,
			Temperature: o.Temperature,
		}
		if o.ThinkingMode != "" {
			chatReq.Thinking = &zai.Thinking{Type: o.ThinkingMode}
		}
		resp, err := o.ZAI.Chat(ctx, chatReq)
		if err != nil {
			return "", fmt.Errorf("Z.AI error: %w", err)
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message
		assistantMsg.Role = "assistant"

		// ── CASE A: GLM selesai — ada jawaban teks ────────────────────────────
		if choice.FinishReason == "stop" || len(assistantMsg.ToolCalls) == 0 {
			o.logger.Info("GLM done",
				zap.String("phone", phone),
				zap.Int("loops_used", loop+1),
			)
			newMessages = append(newMessages, assistantMsg)
			_ = o.Session.AppendMessages(ctx, phone, newMessages...)
			return assistantMsg.Content, nil
		}

		// ── CASE B: GLM minta eksekusi tool(s) ───────────────────────────────
		o.logger.Info("GLM requested tools",
			zap.String("phone", phone),
			zap.Int("tool_count", len(assistantMsg.ToolCalls)),
		)

		messages = append(messages, assistantMsg)
		newMessages = append(newMessages, assistantMsg)

		for _, tc := range assistantMsg.ToolCalls {
			o.logger.Info("executing tool",
				zap.String("name", tc.Function.Name),
				zap.String("call_id", tc.ID),
			)

			result := o.Bridge.Execute(ctx, tc.Function, bridge.ExecuteOptions{
				Phone:       phone,
				AccessLevel: accessLevel,
			})

			toolMsg := zai.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			}
			messages = append(messages, toolMsg)
			newMessages = append(newMessages, toolMsg)
		}
	}

	o.logger.Warn("max function call loops reached",
		zap.String("phone", phone),
		zap.Int("max_loops", o.MaxLoops),
	)
	_ = o.Session.AppendMessages(ctx, phone, newMessages...)
	return "Maaf, permintaan ini terlalu kompleks untuk satu proses. Coba pecah menjadi beberapa pertanyaan terpisah.", nil
}

func (o *Orchestrator) handleSpecialCommand(ctx context.Context, phone, accessLevel, text string) (string, bool) {
	cmd := strings.ToLower(strings.TrimSpace(text))
	switch cmd {
	case "/reset":
		_ = o.Session.ResetSession(ctx, phone)
		return "Riwayat percakapan berhasil dihapus.", true

	case "/status":
		return fmt.Sprintf(
			"*Status Sistem*\n\n- AI Model: `%s`\n- MCP Tools: %d tools\n- Akses Anda: `%s`",
			o.Model, o.Bridge.ToolCount(), accessLevel,
		), true

	case "/tools":
		names := o.Bridge.ToolNames()
		if len(names) == 0 {
			return "Tidak ada tools yang tersedia saat ini.", true
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("*%d MCP Tools Tersedia:*\n\n", len(names)))
		for _, n := range names {
			sb.WriteString("- `" + n + "`\n")
		}
		return sb.String(), true

	case "/whoami":
		return fmt.Sprintf("Nomor: `%s`\nAkses: `%s`", phone, accessLevel), true

	case "/help":
		return buildHelpMessage(), true
	}
	return "", false
}

func buildHelpMessage() string {
	return `*MikroBot — Asisten MikroTik*

Kirim perintah dalam bahasa natural. Contoh:

*Lihat / Query*
- Tampilkan semua IP pool
- Cek traffic interface ether1
- Lihat user hotspot aktif
- Tampilkan firewall rules
- Info CPU, RAM, uptime router

*Konfigurasi* _(akses full)_
- Tambah IP pool nama: pool-baru ranges: 10.0.1.1-10.0.1.100
- Block IP 192.168.1.50
- Buat user hotspot: budi / pass: 1234
- Limit bandwidth IP 192.168.1.100 jadi 2Mbps down 1Mbps up

*Monitoring*
- Monitor traffic ether1 selama 10 detik
- Tampilkan log firewall terbaru

*Perintah Bot*
` + "`/reset`" + `  — Hapus riwayat chat
` + "`/status`" + ` — Status sistem & AI
` + "`/tools`" + `  — Lihat semua kemampuan
` + "`/whoami`" + ` — Info akses Anda`
}
