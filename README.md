# MikroTik MCP + WhatsApp Bot

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![MCP Protocol](https://img.shields.io/badge/MCP-Protocol-blue)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Sistem dua-komponen untuk mengelola **MikroTik RouterOS** menggunakan **AI melalui WhatsApp**:

1. **`cmd/server`** — MCP Server yang mengekspos 26 tools RouterOS via protokol [Model Context Protocol](https://modelcontextprotocol.io/)
2. **`cmd/bot`** — WhatsApp Bot yang menerima pesan, meneruskan ke AI (LLM dengan function calling), lalu AI memanggil MCP tools untuk mengelola router

```
WhatsApp ──▶ gowa ──▶ cmd/bot ──▶ LLM (Groq/Z.AI/dll) ──▶ cmd/server ──▶ MikroTik RouterOS
```

---

## Fitur

| Modul | Tools | Operasi |
|-------|-------|---------|
| **IP Pool** | 5 tools | List, add, update, delete, check used |
| **Firewall** | 4 tools | List, add, delete, toggle enable/disable |
| **Interface** | 2 tools | List, monitor traffic realtime |
| **Hotspot** | 6 tools | List server/user/active, add, delete, kick |
| **Queue** | 5 tools | List simple/tree, add simple, add tree, delete |
| **System** | 4 tools | Get resource, identity, logs, reboot |

**Total: 26 MCP Tools**

### Fitur Bot
- Percakapan multi-turn dengan riwayat per nomor WA (SQLite)
- Function call loop otomatis (AI bisa panggil beberapa tools sekaligus)
- Kontrol akses per nomor (`full` / `readonly`)
- Rate limiting per user
- Audit log setiap eksekusi tool
- Verifikasi HMAC webhook (`X-Hub-Signature-256`)
- Perintah khusus: `/help`, `/status`, `/tools`, `/whoami`, `/reset`

---

## Tech Stack

| Komponen | Library / Teknologi |
|----------|-------------------|
| Language | Go 1.25.5 |
| RouterOS API | `github.com/go-routeros/routeros/v3 v3.0.1` |
| MCP Protocol | `github.com/mark3labs/mcp-go v0.44.1` |
| HTTP Router | `github.com/go-chi/chi/v5 v5.2.5` |
| Config | `github.com/spf13/viper v1.21.0` |
| Logging | `go.uber.org/zap v1.27.1` |
| Database | `modernc.org/sqlite v1.46.1` (pure Go, no CGO) |
| AI Client | OpenAI-compatible API (Groq, Z.AI GLM, dll) |
| WhatsApp Gateway | [gowa](https://github.com/aldinokemal2104/go-whatsapp-web-multidevice) v8 |
| Env loader | `github.com/joho/godotenv v1.5.1` |
| Testing | `github.com/stretchr/testify v1.11.1` |

---

## Prasyarat

- **Go** 1.25 atau lebih baru
- **MikroTik RouterOS** dengan API enabled (port 8728)
- **gowa** (WhatsApp gateway) — binary atau Docker
- **AI API Key** dari provider OpenAI-compatible:
  - [Groq](https://console.groq.com) (gratis, direkomendasikan)
  - Z.AI GLM (`https://open.z.ai`)
  - OpenAI, dll.

### Enable API di RouterOS

```
/ip service enable api
/ip service set api port=8728
```

---

## Instalasi

```bash
git clone https://github.com/yourusername/mikrotik-mcp.git
cd mikrotik-mcp

# Download dependencies
go mod download

# Build kedua binary
go build -o bin/server ./cmd/server
go build -o bin/bot    ./cmd/bot
```

---

## Konfigurasi

Semua konfigurasi ada di satu file `config.yaml`:

```yaml
mikrotik:
  host: "192.168.88.1"         # IP router MikroTik
  port: 8728                    # 8728 plain, 8729 TLS
  username: "admin"
  password: "your_password"
  use_tls: false
  reconnect_interval: 5s
  timeout: 10s

mcp:
  transport: "sse"              # WAJIB "sse" agar bot bisa konek via HTTP
  port: 8080
  read_only: false              # true = hanya operasi read/list

whatsapp:
  gowa_url: "http://localhost:3000"
  gowa_device_id: "UUID-device-dari-gowa"   # Didapat setelah login di gowa
  gowa_username: "user1"        # Basic auth gowa
  gowa_password: "pass1"
  webhook_port: 8090
  webhook_path: "/webhook/message"
  webhook_secret: ""            # Sama dengan WHATSAPP_WEBHOOK_SECRET di gowa

ai:
  api_key: "gsk_xxxxxxxxxxxx"   # API key provider AI
  base_url: "https://api.groq.com/openai/v1"
  model: "llama-3.3-70b-versatile"
  max_tokens: 1024
  temperature: 0.7
  thinking_mode: ""             # "enabled" | "disabled" | "" (khusus GLM-4.7)
  system_prompt: |
    Kamu adalah asisten jaringan bernama MikroBot yang mengelola MikroTik router.

bot:
  mcp_server_url: "http://localhost:8080"
  max_function_call_loops: 5
  session_ttl: 2h
  max_history_messages: 20
  authorized_users:
    - phone: "628xxxxxxxxxx"    # Nomor WA tanpa + (format internasional)
      name: "Admin"
      access: "full"            # full | readonly
    - phone: "628xxxxxxxxxx"
      name: "Staff"
      access: "readonly"

log:
  level: "info"                 # debug | info | warn | error
  format: "json"                # json | console
```

### AI Provider yang Didukung

| Provider | Base URL | Model Contoh |
|----------|----------|-------------|
| **Groq** (gratis) | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` |
| Z.AI GLM | `https://api.z.ai/api/paas/v4` | `glm-4.7`, `glm-4.7-flash` |
| OpenAI | `https://api.openai.com/v1` | `gpt-4o`, `gpt-4o-mini` |

> Untuk Z.AI GLM-4.7 set `thinking_mode: "enabled"` untuk reasoning yang lebih baik.

---

## Cara Menjalankan

Butuh **3 proses** berjalan bersamaan di terminal terpisah.

### Terminal 1 — Gowa (WhatsApp Gateway)

**Jika Docker (direkomendasikan):**

```bash
docker run -d \
  --name gowa \
  -p 3000:3000 \
  -v $(pwd)/gowa-data:/app/storages \
  -e WHATSAPP_WEBHOOK="http://host.docker.internal:8090/webhook/message" \
  -e WHATSAPP_WEBHOOK_SECRET="your-secret" \
  -e WHATSAPP_WEBHOOK_EVENTS="message" \
  aldinokemal2104/go-whatsapp-web-multidevice:latest
```

> Di Linux ganti `host.docker.internal` dengan IP host atau gunakan `--network=host`.

**Jika binary:**

```bash
# Windows PowerShell
$env:WHATSAPP_WEBHOOK="http://localhost:8090/webhook/message"
$env:WHATSAPP_WEBHOOK_SECRET="your-secret"
$env:WHATSAPP_WEBHOOK_EVENTS="message"
./gowa.exe

# Linux/Mac
WHATSAPP_WEBHOOK=http://localhost:8090/webhook/message \
WHATSAPP_WEBHOOK_SECRET=your-secret \
WHATSAPP_WEBHOOK_EVENTS=message \
./gowa
```

**Login WhatsApp:**
1. Buka `http://localhost:3000` di browser
2. Klik **Add Device** → beri nama
3. Scan QR code dengan WhatsApp di HP (**Linked Devices → Link a Device**)
4. Salin Device ID yang muncul di dashboard → isi di `config.yaml` → `gowa_device_id`

### Terminal 2 — MCP Server

```bash
go run ./cmd/server
# atau: ./bin/server
```

Log yang diharapkan:
```
{"msg":"starting mikrotik-mcp","transport":"sse"}
{"msg":"all tools registered"}
{"msg":"starting SSE transport","addr":":8080"}
```

### Terminal 3 — Bot Service

```bash
go run ./cmd/bot
# atau: ./bin/bot
```

Log yang diharapkan:
```
{"msg":"connected to MCP server","url":"http://localhost:8080"}
{"msg":"MCP tools refreshed","count":26}
{"msg":"bot service started","addr":":8090","tools":26}
```

### Verifikasi

```bash
# Cek bot berjalan
curl http://localhost:8090/health
# → {"status":"ok","tools":26}

# Cek MCP server berjalan
curl http://localhost:8080/sse
# → SSE stream terbuka (tekan Ctrl+C)
```

---

## Penggunaan via WhatsApp

Kirim pesan dari nomor yang terdaftar di `authorized_users` ke nomor WA yang login di gowa.

### Perintah Bot

| Perintah | Fungsi |
|----------|--------|
| `/help` | Panduan lengkap |
| `/status` | Info model AI, jumlah tools, akses kamu |
| `/tools` | Daftar semua MCP tools |
| `/whoami` | Nomor dan level akses kamu |
| `/reset` | Hapus riwayat percakapan |

### Contoh Perintah Natural Language

**Monitoring:**
```
Info CPU, RAM, dan uptime router
Tampilkan log router terbaru
Cek semua interface
Monitor traffic ether1 selama 10 detik
```

**IP Pool:**
```
Tampilkan semua IP pool
Tambah IP pool nama: pool-tamu ranges: 192.168.10.1-192.168.10.50
Hapus IP pool pool-tamu
```

**Firewall:**
```
Tampilkan semua firewall rules
Block IP 192.168.1.50
Disable firewall rule nomor 3
```

**Hotspot:**
```
Lihat user hotspot yang sedang online
Buat user hotspot: budi password: 1234
Kick user hotspot budi
```

**Queue / Bandwidth:**
```
Limit bandwidth IP 192.168.1.100 jadi 2Mbps down 1Mbps up
Tampilkan semua queue
Hapus queue limit-budi
```

**Reboot (memerlukan konfirmasi):**
```
Reboot router
```

Bot mendukung percakapan multi-turn — kamu bisa tanya lanjutan dari jawaban sebelumnya.

---

## Struktur Project

```
mikrotik-mcp/
├── cmd/
│   ├── server/
│   │   └── main.go                   # Entry point MCP server
│   └── bot/
│       ├── main.go                   # Entry point WhatsApp bot
│       └── e2e_test.go               # E2E tests (build tag: e2e)
│
├── domain/                           # Layer domain — zero external dependency
│   ├── entity/                       # Struct bisnis (IPPool, FirewallRule, dll)
│   ├── dto/                          # Data Transfer Objects (request/response)
│   └── repository/                   # Interface repository (kontrak)
│
├── internal/
│   ├── config/
│   │   └── config.go                 # Viper config loader
│   ├── mikrotik/                     # Adapter RouterOS (implementasi repository)
│   │   ├── client.go                 # Koneksi & auto-reconnect
│   │   ├── listener.go               # Realtime listen (traffic, logs)
│   │   ├── ip_pool.go
│   │   ├── firewall.go
│   │   ├── interface.go
│   │   ├── hotspot.go
│   │   ├── queue.go
│   │   └── system.go
│   ├── usecase/                      # Business logic layer
│   │   ├── ip_pool_usecase.go
│   │   ├── firewall_usecase.go
│   │   ├── interface_usecase.go
│   │   ├── hotspot_usecase.go
│   │   ├── queue_usecase.go
│   │   └── system_usecase.go
│   ├── ai/
│   │   ├── zai/                      # OpenAI-compatible HTTP client
│   │   │   ├── client.go
│   │   │   └── types.go              # ChatRequest, ChatResponse, Tool, Thinking
│   │   └── bridge/                   # MCP ↔ AI bridge
│   │       ├── bridge.go             # Cache tools, convert MCP→LLM format
│   │       ├── executor.go           # Eksekusi tool dengan access control
│   │       └── audit.go              # Audit log ke SQLite
│   ├── mcpclient/                    # MCP SSE client
│   │   ├── client.go                 # SSE connection dengan long-lived context
│   │   └── types.go
│   ├── orchestrator/
│   │   └── orchestrator.go           # Function call loop + perintah khusus
│   ├── session/                      # Conversation history per nomor WA
│   │   ├── store.go                  # SQLite store
│   │   └── manager.go                # TTL + max history management
│   └── whatsapp/
│       ├── handler.go                # Webhook receiver + HMAC verification
│       ├── sender.go                 # Kirim pesan via gowa REST API
│       ├── middleware.go             # Auth + rate limiting per nomor
│       └── types.go                  # GowaWebhookPayload, GowaSendRequest
│
├── tools/                            # MCP Tool definitions (delivery layer)
│   ├── registry.go                   # RegisterAll — daftarkan semua tools ke MCP server
│   ├── ip_pool_tools.go
│   ├── firewall_tools.go
│   ├── interface_tools.go
│   ├── hotspot_tools.go
│   ├── queue_tools.go
│   └── system_tools.go
│
├── pkg/
│   ├── logger/                       # Zap logger wrapper
│   ├── format/                       # SplitLongMessage (WA 4000 char limit)
│   └── eventbus/                     # Pub/sub event bus
│
├── migrations/
│   └── 001_sessions.sql              # Schema SQLite (sessions, messages, audit_logs)
│
├── config.yaml                       # Konfigurasi utama
├── go.mod
└── README.md
```

---

## Arsitektur

### Clean Architecture

```
┌──────────────────────────────────────────────────┐
│  Delivery Layer                                   │
│  tools/ (MCP tools)  │  cmd/bot (WhatsApp handler)│
├──────────────────────┼───────────────────────────┤
│  Use Case Layer       │  Orchestrator             │
│  internal/usecase/    │  internal/orchestrator/   │
├──────────────────────┴───────────────────────────┤
│  Domain Layer                                     │
│  domain/entity/  domain/dto/  domain/repository/  │
├──────────────────────────────────────────────────┤
│  Infrastructure / Adapter                         │
│  internal/mikrotik/  │  internal/session/         │
│  internal/ai/        │  internal/mcpclient/       │
└──────────────────────────────────────────────────┘
```

### Alur Request WhatsApp

```
HP (WA)
  │ pesan
  ▼
gowa (port 3000)
  │ POST /webhook/message + X-Hub-Signature-256
  ▼
cmd/bot Handler (port 8090)
  │ verifikasi HMAC → auth check → rate limit
  ▼
Orchestrator
  │ load history dari SQLite
  │ kirim ke LLM (Groq/Z.AI)
  ▼
LLM response dengan tool_calls
  │
  ▼
MCP Bridge → cmd/server (port 8080, SSE)
  │ eksekusi tool
  ▼
MikroTik RouterOS (port 8728)
  │ hasil
  ▼
LLM → format jawaban
  │
  ▼
gowa → kirim pesan balik ke HP
```

---

## Testing

### Unit Test

```bash
go test ./...
```

### Integration Test (memerlukan router MikroTik nyata)

```bash
go test ./internal/mikrotik/... -v -tags=integration
```

### E2E Test (mock semua external dependency)

```bash
go test -tags e2e ./cmd/bot/...
```

E2E test menggunakan:
- `httptest.Server` sebagai mock gowa dan mock Z.AI
- SQLite `:memory:` sebagai database
- `stubBridge` sebagai mock MCP bridge
- Channel-based `waitFor` untuk assertion async

---

## Keamanan

| Aspek | Implementasi |
|-------|-------------|
| **Auth WhatsApp** | Whitelist nomor di `authorized_users` |
| **Access Control** | Level `full` (read+write) dan `readonly` |
| **Rate Limiting** | Per nomor WA, menggunakan token bucket |
| **HMAC Webhook** | `X-Hub-Signature-256` SHA256, optional |
| **Read-only Mode** | `mcp.read_only: true` blokir semua operasi write |
| **Konfirmasi Destruktif** | `reboot_router`, `delete_*` memerlukan `confirm=true` |
| **TLS RouterOS** | Gunakan port 8729 + `use_tls: true` |
| **Audit Log** | Setiap eksekusi tool dicatat di SQLite |

---

## Troubleshooting

### MCP Server: `timeout waiting for SSE response`

**Penyebab:** MCP client menggunakan context dengan timeout pendek untuk koneksi SSE yang harus hidup selamanya.

**Pastikan** `config.yaml` menggunakan `transport: "sse"` dan bot bisa mengakses port 8080:
```bash
curl http://localhost:8080/sse
```

---

### Bot: `gowa status 404`

**Penyebab:** Device ID di config tidak sesuai dengan device yang aktif di gowa.

**Solusi:**
1. Buka `http://localhost:3000` di browser
2. Salin Device ID yang terlihat di dashboard
3. Update `config.yaml` → `gowa_device_id`
4. Restart `cmd/bot`

---

### Gowa: `connection refused` saat kirim webhook ke bot

**Penyebab:** Gowa berjalan di Docker dan menggunakan `localhost` yang mengarah ke dalam container, bukan host.

**Solusi:** Gunakan `host.docker.internal` sebagai URL webhook:
```bash
-e WHATSAPP_WEBHOOK="http://host.docker.internal:8090/webhook/message"
```
> Di Linux: gunakan IP host atau `--network=host`

---

### Bot: `Z.AI error [1113]: Insufficient balance`

**Penyebab:** Saldo API Z.AI habis.

**Solusi A:** Top up di https://open.z.ai

**Solusi B:** Ganti ke Groq (gratis):
```yaml
ai:
  api_key: "gsk_xxxx"
  base_url: "https://api.groq.com/openai/v1"
  model: "llama-3.3-70b-versatile"
  thinking_mode: ""
```

---

### RouterOS: `cannot log in`

**Penyebab:** Username/password salah atau user tidak punya akses API.

**Solusi:**
```
/ip service enable api
/user group set full policy=read,write,api,!winbox
```

---

### SQLite: `near "ORDER": syntax error`

**Penyebab:** SQLite tidak mendukung `ORDER BY` di dalam `UPDATE` (syntax MySQL).

**Status:** Sudah diperbaiki di `internal/ai/bridge/audit.go` — menggunakan subquery `WHERE id = (SELECT id ... ORDER BY ... LIMIT 1)`.

---

### Webhook: `401 Unauthorized`

**Penyebab:** `webhook_secret` di `config.yaml` tidak sama dengan `WHATSAPP_WEBHOOK_SECRET` di gowa.

**Solusi:** Pastikan nilainya identik, atau kosongkan keduanya untuk menonaktifkan verifikasi HMAC.

---

## Database Schema

File: `migrations/001_sessions.sql`

```sql
-- Sesi per nomor WA
CREATE TABLE sessions (phone TEXT PRIMARY KEY, ...);

-- Riwayat percakapan
CREATE TABLE messages (
  id INTEGER PRIMARY KEY,
  phone TEXT,
  role TEXT,          -- "user" | "assistant" | "tool"
  content TEXT,
  tool_calls TEXT,    -- JSON, diisi jika role=assistant
  tool_call_id TEXT,  -- diisi jika role=tool
  name TEXT,          -- nama tool, diisi jika role=tool
  created_at DATETIME
);

-- Audit log eksekusi tool
CREATE TABLE audit_logs (
  id INTEGER PRIMARY KEY,
  phone TEXT,
  tool_name TEXT,
  args TEXT,
  status TEXT,        -- "pending" | "success" | "error"
  error TEXT,
  created_at DATETIME,
  finished_at DATETIME
);
```

---

## Kontribusi

1. Fork repository
2. Buat branch: `git checkout -b feature/nama-fitur`
3. Commit: `git commit -m 'feat: tambah fitur baru'`
4. Push: `git push origin feature/nama-fitur`
5. Buat Pull Request

---

## Lisensi

MIT License — lihat [LICENSE](LICENSE) untuk detail.

---

## Kredit

- [go-routeros](https://github.com/go-routeros/routeros) — RouterOS API client untuk Go
- [mcp-go](https://github.com/mark3labs/mcp-go) — MCP SDK untuk Go
- [gowa](https://github.com/aldinokemal2104/go-whatsapp-web-multidevice) — WhatsApp Web gateway
- [Groq](https://groq.com) — Fast LLM inference API
- [MikroTik](https://mikrotik.com) — RouterOS
