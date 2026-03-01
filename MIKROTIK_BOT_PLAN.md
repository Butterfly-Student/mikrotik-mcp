# MikroTik WA Bot — WhatsApp + Z.AI GLM + Custom MCP Bridge
> Ekstensi dari sistem MikroTik MCP Go yang sudah ada, menggunakan gowa untuk WhatsApp,
> Z.AI GLM-5 sebagai AI model, dan custom MCP Bridge yang dibangun sendiri di Go.

---

## Daftar Isi
1. [Gambaran Arsitektur](#1-gambaran-arsitektur)
2. [Tech Stack Tambahan](#2-tech-stack-tambahan)
3. [Struktur Folder Lengkap](#3-struktur-folder-lengkap)
4. [gowa — Setup & Integrasi](#4-gowa--setup--integrasi)
5. [Z.AI GLM Client](#5-zai-glm-client)
6. [MCP Client](#6-mcp-client)
7. [Custom MCP Bridge](#7-custom-mcp-bridge)
8. [Orchestrator — Function Call Loop](#8-orchestrator--function-call-loop)
9. [Session Manager](#9-session-manager)
10. [WhatsApp Handler & Sender](#10-whatsapp-handler--sender)
11. [Authorization & Access Control](#11-authorization--access-control)
12. [Config Lengkap](#12-config-lengkap)
13. [Entry Point Bot](#13-entry-point-bot)
14. [Alur Lengkap End-to-End](#14-alur-lengkap-end-to-end)
15. [Command Khusus Bot](#15-command-khusus-bot)
16. [Error Handling & Edge Cases](#16-error-handling--edge-cases)
17. [Deployment](#17-deployment)
18. [Ringkasan Komponen](#18-ringkasan-komponen)

---

## 1. Gambaran Arsitektur

### Topologi Sistem Keseluruhan

```
┌──────────────────────────────────────────────────────────────────────┐
│                         USER WHATSAPP                                │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ pesan teks
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│             gowa  (go-whatsapp-web-multidevice)                      │
│             Standalone service — port :3000                          │
│  - Manage WhatsApp Web session (QR scan sekali)                      │
│  - POST webhook ke bot service saat ada pesan masuk                  │
│  - Menerima REST call untuk kirim pesan balik                        │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ POST /webhook/message
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│           WhatsApp Bot Service  (Go) — port :8090                    │
│                                                                      │
│  ┌─────────────┐  ┌──────────────────┐  ┌─────────────────────────┐ │
│  │ WA Handler  │─▶│  Middleware      │  │   Session Manager       │ │
│  │ (webhook)   │  │  auth + ratelimit│  │   (SQLite per nomor WA) │ │
│  └──────┬──────┘  └──────────────────┘  └────────────┬────────────┘ │
│         │                                             │              │
│         └─────────────────┬───────────────────────────┘              │
│                           ▼                                          │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                    Orchestrator                                │  │
│  │                                                                │  │
│  │  1. Susun messages (system + history + user)                   │  │
│  │  2. Kirim ke Z.AI GLM dengan tools dari MCP Bridge             │  │
│  │  3. Cek finish_reason:                                         │  │
│  │     - "stop"       → format & kirim ke WA                      │  │
│  │     - "tool_calls" → eksekusi via MCP Bridge → loop ke step 2  │  │
│  │  4. Simpan percakapan ke SQLite                                 │  │
│  └──────────┬────────────────────────────────────┬────────────────┘  │
│             │                                    │                    │
│             ▼                                    ▼                    │
│  ┌─────────────────────┐          ┌──────────────────────────────┐   │
│  │  Z.AI GLM Client    │          │       MCP Bridge             │   │
│  │  POST /chat/        │          │  - Cache tool definitions    │   │
│  │  completions        │          │  - Convert MCP → GLM format  │   │
│  │  api.z.ai           │          │  - Execute tool calls        │   │
│  └─────────────────────┘          └──────────────┬───────────────┘   │
└──────────────────────────────────────────────────┼───────────────────┘
                                                   │ MCP JSON-RPC
                                                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│           MikroTik MCP Server (Go) — cmd/server                      │
│           (existing — SSE mode, port :8080)                          │
└──────────────────────────────┬───────────────────────────────────────┘
                               │ go-routeros v3
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      MikroTik RouterOS                               │
└──────────────────────────────────────────────────────────────────────┘
```

### Poin Penting Arsitektur

**Z.AI sepenuhnya OpenAI-compatible.** Base URL cukup diganti ke `https://api.z.ai/api/paas/v4` — format request, response, dan function calling identik dengan OpenAI. Tidak ada format khusus yang perlu ditangani.

**Custom MCP Bridge dibangun di Go.** Bridge ini yang menjadi jembatan antara format tool definition MCP dan format function GLM. Dibangun sendiri sehingga bisa menyisipkan logic custom seperti access control per user, audit log, dan penanganan error yang informatif.

**MCP Server perlu berjalan dalam mode SSE/HTTP.** Agar MCP Bridge bisa memanggil MCP server via network (bukan stdio), `cmd/server` harus dijalankan dengan transport SSE di port tertentu.

**Function Calling loop dikelola di Orchestrator.** GLM bisa memanggil beberapa tool secara berantai sebelum menghasilkan jawaban akhir. Orchestrator mengulang request ke GLM sampai `finish_reason` adalah `"stop"`.

---

## 2. Tech Stack Tambahan

> Tambahan dari stack yang sudah ada di `mikrotik-mcp-plan.md`

| Teknologi | Library | Kegunaan |
|-----------|---------|----------|
| gowa | `github.com/aldinokemal/go-whatsapp-web-multidevice` | WhatsApp Web — standalone service |
| Z.AI GLM-5 | HTTP REST API (`api.z.ai`) | AI model, OpenAI-compatible, function calling |
| mcp-go client | `github.com/mark3labs/mcp-go/client` | MCP client untuk connect ke MCP server via SSE |
| SQLite | `github.com/mattn/go-sqlite3` | Session & history percakapan per nomor WA |
| Chi | `github.com/go-chi/chi/v5` | HTTP router untuk webhook endpoint |
| godotenv | `github.com/joho/godotenv` | Load `.env` untuk secrets |

---

## 3. Struktur Folder Lengkap

```
mikrotik-mcp/
│
├── cmd/
│   ├── server/
│   │   └── main.go                   # MCP Server (existing) — jalankan mode SSE
│   └── bot/
│       └── main.go                   # Entry point WhatsApp Bot Service
│
├── domain/                           # (existing — tidak berubah)
│   ├── entity/
│   ├── dto/
│   └── repository/
│
├── internal/
│   ├── mikrotik/                     # (existing — tidak berubah)
│   ├── usecase/                      # (existing — tidak berubah)
│   │
│   ├── config/
│   │   └── config.go                 # Update: tambah section bot, ai, whatsapp
│   │
│   ├── whatsapp/
│   │   ├── handler.go                # Webhook handler — terima pesan dari gowa
│   │   ├── sender.go                 # Kirim pesan via gowa REST API
│   │   ├── middleware.go             # Auth nomor WA + rate limiting
│   │   └── types.go                  # Struct webhook payload dari gowa
│   │
│   ├── ai/
│   │   ├── zai/
│   │   │   ├── client.go             # HTTP client untuk Z.AI api.z.ai
│   │   │   └── types.go              # Request/response structs (OpenAI-compatible)
│   │   │
│   │   └── bridge/
│   │       ├── bridge.go             # MCPBridge: cache tools, convert format, access control
│   │       ├── executor.go           # Eksekusi tool call → MCP client → hasil ke GLM
│   │       └── audit.go              # Audit logger ke SQLite
│   │
│   ├── mcpclient/
│   │   ├── client.go                 # MCP SSE client (connect ke cmd/server via HTTP)
│   │   └── types.go                  # Tool, CallResult structs
│   │
│   ├── session/
│   │   ├── manager.go                # Kelola conversation history per nomor WA
│   │   └── store.go                  # SQLite CRUD untuk messages
│   │
│   └── orchestrator/
│       └── orchestrator.go           # Function call loop + special commands
│
├── tools/                            # (existing — tidak berubah)
│
├── pkg/
│   ├── logger/                       # (existing)
│   ├── eventbus/                     # (existing)
│   └── format/
│       └── message.go                # SplitLongMessage untuk WA
│
├── migrations/
│   └── 001_sessions.sql              # Schema SQLite: sessions, messages, audit_logs
│
├── config.yaml
├── .env                              # JANGAN commit ke git
├── .env.example
├── go.mod
└── go.sum
```

---

## 4. gowa — Setup & Integrasi

gowa berjalan sebagai proses terpisah. Bot service komunikasi dengannya murni via HTTP.

### Jalankan gowa

```bash
git clone https://github.com/aldinokemal/go-whatsapp-web-multidevice.git
cd go-whatsapp-web-multidevice
go build -o gowa ./cmd/...

# Jalankan dengan webhook mengarah ke bot service
./gowa --port 3000 --webhook http://localhost:8090/webhook/message
```

Buka `http://localhost:3000` → scan QR code dengan WhatsApp yang akan jadi bot.

### REST API gowa yang Digunakan Bot Service

```
POST /send/message     Kirim pesan teks ke nomor WA
GET  /user/info        Cek info nomor WA
```

### Format Webhook dari gowa ke Bot Service

```json
{
  "from": "6281234567890@s.whatsapp.net",
  "pushname": "Budi",
  "is_from_me": false,
  "is_group": false,
  "chat_jid": "6281234567890@s.whatsapp.net",
  "message": {
    "id": "ABCD1234",
    "text": "tampilkan semua IP pool",
    "timestamp": 1732000000
  }
}
```

---

## 5. Z.AI GLM Client

Z.AI menggunakan format OpenAI-compatible sepenuhnya. Struct yang digunakan identik dengan OpenAI, hanya beda `base_url` dan `api_key`.

### Types

```go
// internal/ai/zai/types.go
package zai

// ── Request ───────────────────────────────────────────────────────────────────

type ChatRequest struct {
    Model       string    `json:"model"`                  // "glm-5"
    Messages    []Message `json:"messages"`
    Tools       []Tool    `json:"tools,omitempty"`
    ToolChoice  string    `json:"tool_choice,omitempty"`  // "auto" | "none"
    Temperature float64   `json:"temperature,omitempty"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
    Role       string     `json:"role"`                    // "system"|"user"|"assistant"|"tool"
    Content    string     `json:"content"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`    // diisi saat role=assistant + minta tool
    ToolCallID string     `json:"tool_call_id,omitempty"` // diisi saat role=tool
    Name       string     `json:"name,omitempty"`
}

// ── Tool / Function Definition ─────────────────────────────────────────────────

type Tool struct {
    Type     string   `json:"type"` // selalu "function"
    Function Function `json:"function"`
}

type Function struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ── Response ──────────────────────────────────────────────────────────────────

type ChatResponse struct {
    ID      string    `json:"id"`
    Object  string    `json:"object"`
    Model   string    `json:"model"`
    Choices []Choice  `json:"choices"`
    Usage   Usage     `json:"usage"`
    Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason"` // "stop" | "tool_calls" | "length"
}

type ToolCall struct {
    ID       string       `json:"id"`
    Type     string       `json:"type"` // "function"
    Function FunctionCall `json:"function"`
}

type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string dari GLM
}

type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### Client

```go
// internal/ai/zai/client.go
package zai

const DefaultBaseURL = "https://api.z.ai/api/paas/v4"

type Client struct {
    apiKey     string
    baseURL    string
    model      string
    httpClient *http.Client
    logger     *zap.Logger
}

func NewClient(apiKey, model string, logger *zap.Logger) *Client {
    return &Client{
        apiKey:  apiKey,
        baseURL: DefaultBaseURL,
        model:   model,
        httpClient: &http.Client{
            Timeout: 60 * time.Second,
        },
        logger: logger,
    }
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    if req.Model == "" {
        req.Model = c.model
    }

    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(
        ctx, http.MethodPost,
        c.baseURL+"/chat/completions",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept-Language", "en-US,en")

    c.logger.Debug("calling Z.AI",
        zap.String("model", req.Model),
        zap.Int("messages", len(req.Messages)),
        zap.Int("tools", len(req.Tools)),
    )

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("http request to Z.AI: %w", err)
    }
    defer resp.Body.Close()

    var result ChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode Z.AI response: %w", err)
    }

    if result.Error != nil {
        return nil, fmt.Errorf("Z.AI error [%s]: %s", result.Error.Code, result.Error.Message)
    }
    if len(result.Choices) == 0 {
        return nil, fmt.Errorf("Z.AI returned empty choices")
    }

    c.logger.Debug("Z.AI responded",
        zap.String("finish_reason", result.Choices[0].FinishReason),
        zap.Int("tokens", result.Usage.TotalTokens),
    )

    return &result, nil
}
```

---

## 6. MCP Client

MCP Client menghubungi MCP Server (`cmd/server`) yang berjalan dalam mode SSE/HTTP.
Menggunakan library `mcp-go` yang sama dengan yang dipakai MCP server — tinggal pakai bagian client-nya.

```go
// internal/mcpclient/types.go
package mcpclient

type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallResult struct {
    Content []ContentBlock `json:"content"`
    IsError bool           `json:"isError"`
}

type ContentBlock struct {
    Type string `json:"type"` // "text"
    Text string `json:"text,omitempty"`
}
```

```go
// internal/mcpclient/client.go
package mcpclient

import (
    mcpClient "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

type Client struct {
    c      *mcpClient.SSEMCPClient
    logger *zap.Logger
}

func NewClient(serverURL string, logger *zap.Logger) (*Client, error) {
    c, err := mcpClient.NewSSEMCPClient(serverURL + "/sse")
    if err != nil {
        return nil, fmt.Errorf("create MCP SSE client: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := c.Start(ctx); err != nil {
        return nil, fmt.Errorf("start MCP client: %w", err)
    }
    if _, err := c.Initialize(ctx, mcp.InitializeRequest{}); err != nil {
        return nil, fmt.Errorf("MCP initialize handshake: %w", err)
    }

    logger.Info("connected to MCP server", zap.String("url", serverURL))
    return &Client{c: c, logger: logger}, nil
}

// ListTools mengambil semua tool yang tersedia dari MCP server
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
    result, err := c.c.ListTools(ctx, mcp.ListToolsRequest{})
    if err != nil {
        return nil, fmt.Errorf("list MCP tools: %w", err)
    }

    tools := make([]Tool, 0, len(result.Tools))
    for _, t := range result.Tools {
        schema := map[string]interface{}{}
        if t.InputSchema.Type != "" {
            schema["type"] = t.InputSchema.Type
        }
        if t.InputSchema.Properties != nil {
            schema["properties"] = t.InputSchema.Properties
        }
        if len(t.InputSchema.Required) > 0 {
            schema["required"] = t.InputSchema.Required
        }
        tools = append(tools, Tool{
            Name:        t.Name,
            Description: t.Description,
            InputSchema: schema,
        })
    }
    return tools, nil
}

// CallTool memanggil satu tool di MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallResult, error) {
    req := mcp.CallToolRequest{}
    req.Params.Name = name
    req.Params.Arguments = args

    result, err := c.c.CallTool(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("call MCP tool %s: %w", name, err)
    }

    cr := &CallResult{IsError: result.IsError}
    for _, block := range result.Content {
        if block.Type == "text" {
            cr.Content = append(cr.Content, ContentBlock{Type: "text", Text: block.Text})
        }
    }
    return cr, nil
}

func (c *Client) Close() { c.c.Close() }
```

---

## 7. Custom MCP Bridge

Bridge ini adalah jantung dari integrasi. Terdiri dari tiga file:

- `bridge.go` — cache tools, konversi MCP → GLM format, access control helper
- `executor.go` — eksekusi satu tool call dengan access check & audit log
- `audit.go` — catat setiap eksekusi tool ke SQLite

### bridge.go

```go
// internal/ai/bridge/bridge.go
package bridge

type MCPBridge struct {
    mcpClient   *mcpclient.Client
    cachedTools []mcpclient.Tool
    auditLogger *AuditLogger   // opsional, set via SetAuditLogger
    mu          sync.RWMutex
    logger      *zap.Logger
}

func New(mcpClient *mcpclient.Client, logger *zap.Logger) *MCPBridge {
    return &MCPBridge{mcpClient: mcpClient, logger: logger}
}

func (b *MCPBridge) SetAuditLogger(a *AuditLogger) { b.auditLogger = a }

// RefreshTools ambil ulang tools dari MCP server — dipanggil saat startup
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
// Langsung dimasukkan ke ChatRequest.Tools
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

// normalizeSchema pastikan schema selalu valid untuk GLM
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
    } {
        if strings.HasPrefix(name, prefix) {
            return true
        }
    }
    return false
}
```

### executor.go

```go
// internal/ai/bridge/executor.go
package bridge

type ExecuteOptions struct {
    Phone       string // nomor WA — untuk audit log
    AccessLevel string // "full" | "readonly"
}

// Execute mengeksekusi satu GLM FunctionCall sebagai MCP tool call
// Selalu return string — error disampaikan sebagai teks agar GLM bisa jelaskan ke user
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

// extractText ambil teks dari CallResult, gabungkan jika ada multiple blocks
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
```

### audit.go

```go
// internal/ai/bridge/audit.go
package bridge

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
             WHERE phone=? AND tool_name=? AND status='pending'
             ORDER BY created_at DESC LIMIT 1`,
            status, errMsg, phone, tool,
        )
        if err != nil {
            a.logger.Warn("audit log after failed", zap.Error(err))
        }
    }()
}
```

---

## 8. Orchestrator — Function Call Loop

Orchestrator mengorkestrasi seluruh alur dari pesan masuk hingga balasan keluar.
Inilah yang mengelola function call loop: kirim ke GLM → cek apakah ada tool call → eksekusi → kirim lagi ke GLM → ulangi sampai `finish_reason = "stop"`.

```go
// internal/orchestrator/orchestrator.go
package orchestrator

type Config struct {
    ZAI          *zai.Client
    Bridge       *bridge.MCPBridge
    Session      *session.Manager
    SystemPrompt string
    Model        string
    MaxTokens    int
    Temperature  float64
    MaxLoops     int // batas loop function call, default 5
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
        resp, err := o.ZAI.Chat(ctx, zai.ChatRequest{
            Model:       o.Model,
            Messages:    messages,
            Tools:       tools,
            ToolChoice:  "auto",
            MaxTokens:   o.MaxTokens,
            Temperature: o.Temperature,
        })
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

        // Tambahkan assistant message (berisi tool_calls) ke messages
        messages = append(messages, assistantMsg)
        newMessages = append(newMessages, assistantMsg)

        // Eksekusi setiap tool call
        for _, tc := range assistantMsg.ToolCalls {
            o.logger.Info("executing tool",
                zap.String("name", tc.Function.Name),
                zap.String("call_id", tc.ID),
            )

            result := o.Bridge.Execute(ctx, tc.Function, bridge.ExecuteOptions{
                Phone:       phone,
                AccessLevel: accessLevel,
            })

            // Buat tool result message — wajib ada ToolCallID agar GLM paham
            toolMsg := zai.Message{
                Role:       "tool",
                Content:    result,
                ToolCallID: tc.ID,            // menghubungkan hasil ke tool call yang diminta
                Name:       tc.Function.Name,
            }
            messages = append(messages, toolMsg)
            newMessages = append(newMessages, toolMsg)
        }

        // Loop → kirim messages terbaru (dengan hasil tool) ke GLM lagi
        // GLM akan membaca hasil dan memutuskan: jawab atau minta tool lagi
    }

    // Jika sudah maxLoops tapi GLM masih minta tool
    o.logger.Warn("max function call loops reached",
        zap.String("phone", phone),
        zap.Int("max_loops", o.MaxLoops),
    )
    _ = o.Session.AppendMessages(ctx, phone, newMessages...)
    return "Maaf, permintaan ini terlalu kompleks untuk satu proses. Coba pecah menjadi beberapa pertanyaan terpisah.", nil
}

// handleSpecialCommand tangani command bot sebelum dikirim ke GLM
func (o *Orchestrator) handleSpecialCommand(ctx context.Context, phone, accessLevel, text string) (string, bool) {
    cmd := strings.ToLower(strings.TrimSpace(text))
    switch cmd {
    case "/reset":
        _ = o.Session.ResetSession(ctx, phone)
        return "✅ Riwayat percakapan berhasil dihapus.", true

    case "/status":
        return fmt.Sprintf(
            "✅ *Status Sistem*\n\n• AI Model: `%s`\n• MCP Tools: %d tools\n• Akses Anda: `%s`",
            o.Model, o.Bridge.ToolCount(), accessLevel,
        ), true

    case "/tools":
        names := o.Bridge.ToolNames()
        if len(names) == 0 {
            return "Tidak ada tools yang tersedia saat ini.", true
        }
        var sb strings.Builder
        sb.WriteString(fmt.Sprintf("🔧 *%d MCP Tools Tersedia:*\n\n", len(names)))
        for _, n := range names {
            sb.WriteString("• `" + n + "`\n")
        }
        return sb.String(), true

    case "/whoami":
        return fmt.Sprintf("📱 Nomor: `%s`\n🔑 Akses: `%s`", phone, accessLevel), true

    case "/help":
        return buildHelpMessage(), true
    }
    return "", false
}

func buildHelpMessage() string {
    return `🤖 *MikroBot — Asisten MikroTik*

Kirim perintah dalam bahasa natural. Contoh:

📋 *Lihat / Query*
• Tampilkan semua IP pool
• Cek traffic interface ether1
• Lihat user hotspot aktif
• Tampilkan firewall rules
• Info CPU, RAM, uptime router

⚙️ *Konfigurasi* _(akses full)_
• Tambah IP pool nama: pool-baru ranges: 10.0.1.1-10.0.1.100
• Block IP 192.168.1.50
• Buat user hotspot: budi / pass: 1234
• Limit bandwidth IP 192.168.1.100 jadi 2Mbps down 1Mbps up

📊 *Monitoring*
• Monitor traffic ether1 selama 10 detik
• Tampilkan log firewall terbaru

🔧 *Perintah Bot*
` + "`/reset`" + `  — Hapus riwayat chat
` + "`/status`" + ` — Status sistem & AI
` + "`/tools`" + `  — Lihat semua kemampuan
` + "`/whoami`" + ` — Info akses Anda`
}
```

---

## 9. Session Manager

### Schema SQLite

```sql
-- migrations/001_sessions.sql

CREATE TABLE IF NOT EXISTS sessions (
    phone      TEXT PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Menyimpan seluruh conversation history per nomor WA
CREATE TABLE IF NOT EXISTS messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    phone        TEXT    NOT NULL,
    role         TEXT    NOT NULL,   -- "user" | "assistant" | "tool"
    content      TEXT    NOT NULL DEFAULT '',
    tool_calls   TEXT,               -- JSON array ToolCall, diisi jika role=assistant
    tool_call_id TEXT,               -- diisi jika role=tool
    name         TEXT,               -- nama tool, diisi jika role=tool
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Audit log setiap tool yang dieksekusi
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    phone       TEXT    NOT NULL,
    tool_name   TEXT    NOT NULL,
    args        TEXT,
    status      TEXT DEFAULT 'pending',  -- "pending" | "success" | "error"
    error       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_msg_phone_time ON messages(phone, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_phone    ON audit_logs(phone, created_at DESC);
```

```go
// internal/session/manager.go
package session

const (
    MaxHistoryMessages = 20
    SessionTTL         = 2 * time.Hour
)

type Manager struct {
    store  *Store
    logger *zap.Logger
}

// GetHistory ambil history yang masih dalam TTL, siap dimasukkan ke ChatRequest
func (m *Manager) GetHistory(ctx context.Context, phone string) ([]zai.Message, error) {
    rows, err := m.store.GetRecentMessages(ctx, phone, MaxHistoryMessages)
    if err != nil {
        return nil, err
    }
    cutoff := time.Now().Add(-SessionTTL)
    var result []zai.Message
    for _, row := range rows {
        if row.CreatedAt.Before(cutoff) {
            continue
        }
        msg := zai.Message{
            Role:       row.Role,
            Content:    row.Content,
            ToolCallID: row.ToolCallID,
            Name:       row.Name,
        }
        if row.ToolCallsJSON != "" {
            _ = json.Unmarshal([]byte(row.ToolCallsJSON), &msg.ToolCalls)
        }
        result = append(result, msg)
    }
    return result, nil
}

// AppendMessages simpan messages baru ke SQLite
func (m *Manager) AppendMessages(ctx context.Context, phone string, msgs ...zai.Message) error {
    for _, msg := range msgs {
        tcJSON := ""
        if len(msg.ToolCalls) > 0 {
            b, _ := json.Marshal(msg.ToolCalls)
            tcJSON = string(b)
        }
        if err := m.store.SaveMessage(ctx, phone,
            msg.Role, msg.Content, tcJSON, msg.ToolCallID, msg.Name); err != nil {
            return err
        }
    }
    return nil
}

func (m *Manager) ResetSession(ctx context.Context, phone string) error {
    return m.store.DeleteMessages(ctx, phone)
}
```

---

## 10. WhatsApp Handler & Sender

```go
// internal/whatsapp/handler.go
package whatsapp

type Handler struct {
    orch   *orchestrator.Orchestrator
    sender *Sender
    auth   *Middleware
    logger *zap.Logger
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    var p GowaWebhookPayload
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // Abaikan: pesan dari diri sendiri, grup, pesan kosong
    if p.IsFromMe || p.IsGroup || strings.TrimSpace(p.Message.Text) == "" {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Balas 200 segera ke gowa — tidak boleh timeout
    w.WriteHeader(http.StatusOK)

    go h.process(p)
}

func (h *Handler) process(p GowaWebhookPayload) {
    phone := p.ExtractPhone()
    text := strings.TrimSpace(p.Message.Text)

    ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
    defer cancel()

    // Auth check
    if !h.auth.IsAuthorized(phone) {
        _ = h.sender.SendText(ctx, phone, "❌ Nomor Anda tidak terdaftar untuk menggunakan layanan ini.")
        return
    }

    // Rate limit
    if !h.auth.Allow(phone) {
        _ = h.sender.SendText(ctx, phone, "⏳ Terlalu banyak permintaan. Tunggu sebentar.")
        return
    }

    // Kirim "sedang memproses" jika lebih dari 3 detik
    stopStatus := h.sender.DelayedStatus(ctx, phone, "⏳ _Sedang memproses..._", 3*time.Second)

    accessLevel := h.auth.GetAccessLevel(phone)
    response, err := h.orch.Process(ctx, phone, accessLevel, text)
    stopStatus()

    if err != nil {
        h.logger.Error("process failed", zap.String("phone", phone), zap.Error(err))
        _ = h.sender.SendText(ctx, phone, "❌ Terjadi kesalahan. Silakan coba lagi.")
        return
    }

    // Kirim response — multi-chunk jika panjang
    for i, chunk := range format.SplitLongMessage(response) {
        if i > 0 {
            time.Sleep(500 * time.Millisecond)
        }
        if err := h.sender.SendText(ctx, phone, chunk); err != nil {
            h.logger.Error("send failed", zap.Error(err))
            break
        }
    }
}
```

```go
// internal/whatsapp/sender.go
package whatsapp

type Sender struct {
    gowaURL    string
    httpClient *http.Client
    logger     *zap.Logger
}

func (s *Sender) SendText(ctx context.Context, phone, text string) error {
    body, _ := json.Marshal(map[string]interface{}{
        "phone": phone, "message": text,
    })
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        s.gowaURL+"/send/message", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := s.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("gowa send: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 {
        return fmt.Errorf("gowa status %d", resp.StatusCode)
    }
    return nil
}

// DelayedStatus kirim pesan status setelah delay, return func untuk cancel
func (s *Sender) DelayedStatus(ctx context.Context, phone, msg string, delay time.Duration) func() {
    t := time.AfterFunc(delay, func() { _ = s.SendText(ctx, phone, msg) })
    return func() { t.Stop() }
}
```

---

## 11. Authorization & Access Control

```go
// internal/whatsapp/middleware.go
package whatsapp

type AuthUser struct {
    Phone  string
    Name   string
    Access string // "full" | "readonly"
}

type Middleware struct {
    users    map[string]AuthUser
    limiters map[string]*rateLimiter
    mu       sync.RWMutex
}

func NewMiddleware(users []AuthUser) *Middleware {
    m := &Middleware{
        users:    make(map[string]AuthUser),
        limiters: make(map[string]*rateLimiter),
    }
    for _, u := range users {
        m.users[u.Phone] = u
    }
    return m
}

func (m *Middleware) IsAuthorized(phone string) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    _, ok := m.users[phone]
    return ok
}

func (m *Middleware) GetAccessLevel(phone string) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if u, ok := m.users[phone]; ok {
        return u.Access
    }
    return "readonly"
}

// Allow cek rate limit: 10 request per menit per nomor
func (m *Middleware) Allow(phone string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    if _, ok := m.limiters[phone]; !ok {
        m.limiters[phone] = newRateLimiter(10, time.Minute)
    }
    return m.limiters[phone].allow()
}
```

---

## 12. Config Lengkap

```yaml
# config.yaml

mikrotik:
  host: "192.168.88.1"
  port: 8728
  username: "admin"
  password: "${MIKROTIK_PASSWORD}"
  use_tls: false
  reconnect_interval: 5s
  timeout: 10s

mcp:
  transport: "sse"       # WAJIB SSE agar bisa diakses via HTTP
  port: 8080
  read_only: false

whatsapp:
  gowa_url: "http://localhost:3000"
  webhook_port: 8090
  webhook_path: "/webhook/message"

ai:
  api_key: "${ZAI_API_KEY}"
  base_url: "https://api.z.ai/api/paas/v4"
  model: "glm-5"           # atau "glm-4-airx"
  max_tokens: 1024
  temperature: 0.7
  system_prompt: |
    Kamu adalah asisten jaringan bernama MikroBot yang mengelola MikroTik router.
    Kemampuanmu: firewall, IP pool, hotspot, queue bandwidth, monitoring interface, dan sistem.

    Panduan:
    - Gunakan bahasa Indonesia yang jelas dan ringkas
    - Format bandwidth dengan satuan yang mudah dibaca (Mbps, Kbps)
    - Sebelum operasi destruktif (hapus, reboot), minta konfirmasi eksplisit
    - Tampilkan hasil dalam format yang rapi menggunakan bold dan list

bot:
  mcp_server_url: "http://localhost:8080"
  max_function_call_loops: 5
  session_ttl: "2h"
  max_history_messages: 20
  authorized_users:
    - phone: "6281234567890"
      name: "Admin Utama"
      access: "full"
    - phone: "6289876543210"
      name: "Staff NOC"
      access: "readonly"

log:
  level: "info"
  format: "json"
```

```bash
# .env.example
MIKROTIK_PASSWORD=password_router
ZAI_API_KEY=api_key_dari_z.ai
```

---

## 13. Entry Point Bot

```go
// cmd/bot/main.go
package main

func main() {
    _ = godotenv.Load()
    cfg := config.Load()
    log := logger.New(cfg.Log)

    // SQLite
    db, err := sql.Open("sqlite3", "./bot.db?_foreign_keys=on")
    if err != nil {
        log.Fatal("open sqlite", zap.Error(err))
    }
    runMigrations(db)
    defer db.Close()

    // Session Manager
    sessionMgr := session.NewManager(session.NewStore(db), log)

    // MCP Client — connect ke MCP server (SSE mode)
    mcpCli, err := mcpclient.NewClient(cfg.Bot.MCPServerURL, log)
    if err != nil {
        log.Fatal("connect to MCP server", zap.Error(err))
    }
    defer mcpCli.Close()

    // MCP Bridge
    mcpBridge := bridge.New(mcpCli, log)
    mcpBridge.SetAuditLogger(bridge.NewAuditLogger(db, log))
    if err := mcpBridge.RefreshTools(context.Background()); err != nil {
        log.Fatal("refresh MCP tools", zap.Error(err))
    }

    // Z.AI Client
    zaiClient := zai.NewClient(cfg.AI.APIKey, cfg.AI.Model, log)

    // Orchestrator
    orch := orchestrator.New(orchestrator.Config{
        ZAI:          zaiClient,
        Bridge:       mcpBridge,
        Session:      sessionMgr,
        SystemPrompt: cfg.AI.SystemPrompt,
        Model:        cfg.AI.Model,
        MaxTokens:    cfg.AI.MaxTokens,
        Temperature:  cfg.AI.Temperature,
        MaxLoops:     cfg.Bot.MaxFunctionCallLoops,
    }, log)

    // WhatsApp
    sender := whatsapp.NewSender(cfg.WhatsApp.GowaURL, log)
    auth := whatsapp.NewMiddleware(cfg.Bot.AuthorizedUsers)
    handler := whatsapp.NewHandler(orch, sender, auth, log)

    // HTTP Router
    r := chi.NewRouter()
    r.Post(cfg.WhatsApp.WebhookPath, handler.HandleWebhook)
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"status":"ok","tools":` +
            fmt.Sprintf("%d", mcpBridge.ToolCount()) + `}`))
    })

    addr := fmt.Sprintf(":%d", cfg.WhatsApp.WebhookPort)
    log.Info("bot service started", zap.String("addr", addr))
    log.Fatal("server error", zap.Error(http.ListenAndServe(addr, r)))
}

func runMigrations(db *sql.DB) {
    sql, _ := os.ReadFile("migrations/001_sessions.sql")
    if _, err := db.Exec(string(sql)); err != nil {
        panic("migration failed: " + err.Error())
    }
}
```

---

## 14. Alur Lengkap End-to-End

### Contoh: "block IP 10.0.0.50"

```
[User WA] → "block IP 10.0.0.50"

[gowa] POST /webhook/message → bot service

[WA Handler]
  auth ✓ | rate limit ✓ | accessLevel = "full"
  → spawn goroutine → orchestrator.Process()

[Orchestrator — Loop 1]
  messages = [
    {role:"system", content:"Kamu asisten MikroTik..."},
    {role:"user",   content:"block IP 10.0.0.50"}
  ]
  → POST https://api.z.ai/api/paas/v4/chat/completions

[Z.AI GLM-5] → finish_reason:"tool_calls"
  tool_calls: [{
    id: "call_x1",
    function: { name:"add_firewall_rule",
      arguments:'{"chain":"forward","action":"drop","src_address":"10.0.0.50"}' }
  }]

[MCP Bridge — executor.go]
  access check: "add_firewall_rule" = write tool, access="full" ✓
  audit log: before
  → mcpClient.CallTool("add_firewall_rule", {chain,action,src_address})

[MCP Server — cmd/server]
  → /ip/firewall/filter/add =chain=forward =action=drop =src-address=10.0.0.50

[RouterOS] rule dibuat, ID *15

[MCP Bridge] result: '{"success":true,"id":"*15"}'
  audit log: after status=success

[Orchestrator — Loop 2]
  messages sekarang:
  [..., {role:"assistant", tool_calls:[...]},
        {role:"tool", tool_call_id:"call_x1", content:'{"success":true,"id":"*15"}'}]
  → POST ke Z.AI lagi

[Z.AI GLM-5] → finish_reason:"stop"
  content: "✅ IP *10.0.0.50* berhasil diblokir!\n
            Rule dibuat di chain *forward* dengan action *drop*.\n
            ID Rule: `*15`"

[Orchestrator] simpan ke SQLite, return response

[WA Handler] → gowa POST /send/message → [User WA] menerima balasan
```

---

## 15. Command Khusus Bot

| Command | Aksi | Siapa |
|---------|------|-------|
| `/help` | Tampilkan daftar kemampuan | Semua |
| `/reset` | Hapus riwayat percakapan | Semua |
| `/status` | Status sistem, model, jumlah tools | Semua |
| `/whoami` | Lihat nomor & level akses | Semua |
| `/tools` | List semua MCP tools | Semua |

---

## 16. Error Handling & Edge Cases

### Timeout Bertingkat

```go
const (
    ProcessTimeout = 90 * time.Second // total proses satu pesan
    ZAITimeout     = 60 * time.Second // per request ke Z.AI
    MCPToolTimeout = 15 * time.Second // per tool call ke MikroTik
    WASendTimeout  = 10 * time.Second // kirim ke gowa
)
```

### Skenario Error

| Skenario | Penanganan |
|----------|-----------|
| gowa tidak bisa dihubungi | Log error, bot tetap jalan |
| MCP server mati saat startup | `log.Fatal` — bot tidak jalan tanpa MCP |
| MCP server mati saat runtime | `Execute()` return error string, GLM jelaskan ke user |
| Z.AI API timeout / error | Return pesan "AI sedang sibuk, coba lagi" |
| Readonly user akses write tool | Bridge return `access_denied`, GLM teruskan ke user |
| Loop tidak selesai di maxLoops | Return pesan partial, sarankan pecah pertanyaan |
| Pesan WA > 4000 karakter | `SplitLongMessage()` → multi-chunk dengan jeda 500ms |
| History terlalu panjang | Ambil hanya `MaxHistoryMessages` terbaru |
| Nomor tidak di whitelist | Return penolakan, tidak proses lebih lanjut |

---

## 17. Deployment

### Tiga Proses

```
Server
├── gowa         (port 3000) — WA session, scan QR sekali
├── cmd/server   (port 8080) — MikroTik MCP Server, mode SSE
└── cmd/bot      (port 8090) — WhatsApp Bot Service
```

### Docker Compose

```yaml
version: "3.8"
services:
  gowa:
    image: aldinokemal2104/go-whatsapp-web-multidevice:latest
    ports: ["3000:3000"]
    volumes: ["./data/gowa:/app/storages"]
    environment:
      WEBHOOK: "http://bot:8090/webhook/message"
    restart: unless-stopped

  mcp-server:
    build: { context: ., dockerfile: Dockerfile.server }
    ports: ["8080:8080"]
    environment:
      MIKROTIK_PASSWORD: "${MIKROTIK_PASSWORD}"
    restart: unless-stopped

  bot:
    build: { context: ., dockerfile: Dockerfile.bot }
    ports: ["8090:8090"]
    environment:
      ZAI_API_KEY: "${ZAI_API_KEY}"
      MIKROTIK_PASSWORD: "${MIKROTIK_PASSWORD}"
    volumes: ["./data/bot.db:/app/bot.db"]
    depends_on: [gowa, mcp-server]
    restart: unless-stopped
```

### Onboarding

```
1. cp .env.example .env  →  isi ZAI_API_KEY dan MIKROTIK_PASSWORD
2. docker-compose up -d
3. Buka http://server:3000 → scan QR dengan WA yang jadi bot
4. Kirim pesan ke nomor yang di-scan → bot aktif ✅
```

---

## 18. Ringkasan Komponen

| Komponen | Status | Perubahan dari Rencana Sebelumnya |
|----------|--------|----------------------------------|
| `domain/`, `internal/mikrotik/`, `internal/usecase/`, `tools/` | ✅ Existing | Tidak berubah |
| `cmd/server/` | ✅ Existing | Perlu set `transport: sse` di config |
| `internal/ai/zai/` | 🆕 Baru | **Ganti dari ZhipuAI ke Z.AI** — base URL `api.z.ai`, model `glm-4.7` |
| `internal/ai/bridge/` | 🆕 Baru | **Custom MCP Bridge** — cache, convert, access control, audit |
| `internal/mcpclient/` | 🆕 Baru | MCP SSE client pakai `mcp-go` yang sama dengan server |
| `internal/session/` | 🆕 Baru | History per nomor WA di SQLite |
| `internal/whatsapp/` | 🆕 Baru | Webhook handler + sender ke gowa |
| `internal/orchestrator/` | 🆕 Baru | Function call loop + special commands |
| `pkg/format/` | 🆕 Baru | Split pesan panjang untuk WA |
| `cmd/bot/` | 🆕 Baru | Entry point + wiring semua dependency |
| `migrations/` | 🆕 Baru | SQLite schema: sessions, messages, audit_logs |
| `gowa` | 🔗 External | Proses terpisah — tidak berubah |
| `Z.AI GLM-5` | 🌐 API | Ganti dari ZhipuAI — `api.z.ai`, OpenAI-compatible |

---

*Dengan arsitektur ini, siapa pun yang nomornya terdaftar di whitelist bisa mengontrol MikroTik router via WhatsApp menggunakan bahasa natural. AI memahami konteks percakapan, menjalankan chain of operations secara otomatis melalui custom MCP Bridge, dan menjaga keamanan lewat access control per nomor WA.*