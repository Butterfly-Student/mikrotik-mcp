# MikroTik MCP

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![MCP Protocol](https://img.shields.io/badge/MCP-Protocol-blue)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**MikroTik MCP** adalah server [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) yang menghubungkan MikroTik RouterOS dengan AI seperti Claude, GPT, dan model AI lainnya. Memungkinkan pengelolaan router MikroTik menggunakan natural language melalui protokol standar MCP.

---

## 🎯 Fitur Utama

| Modul | Operasi | Deskripsi |
|-------|---------|-----------|
| **IP Pool** | CRUD | Kelola IP pool untuk DHCP, Hotspot, dan PPPoE |
| **Firewall** | CRUD + Toggle | Atur filter rules, NAT, dan mangle |
| **Interface** | Read + Monitor | Lihat status interface dan monitoring traffic realtime |
| **Hotspot** | CRUD + Active Users | Kelola user hotspot dan sesi aktif |
| **Queue** | CRUD | Simple queue dan queue tree untuk bandwidth management |
| **System** | Read + Reboot | Monitor resource, logs, dan kontrol sistem |

### Keunggulan

- 🔌 **Dual Transport**: Mendukung stdio (default) dan SSE transport
- 🔒 **Keamanan**: Mode read-only dan konfirmasi untuk operasi destruktif
- 📊 **Realtime**: Monitoring traffic dan log secara realtime
- 🏗️ **Clean Architecture**: Mudah di-maintain dan di-extend
- 🧪 **Well Tested**: Unit test dan integration test coverage

---

## 📋 Prasyarat

- **Go** 1.25 atau lebih baru
- **MikroTik RouterOS** dengan API enabled (port 8728/8729)
- **AI Client** yang mendukung MCP (Claude Desktop, Claude Code, dll.)

---

## 🚀 Instalasi

### Dari Source

```bash
# Clone repository
git clone https://github.com/yourusername/mikrotik-mcp.git
cd mikrotik-mcp

# Build binary
go build -o mikrotik-mcp ./cmd/server

# Atau langsung run
go run ./cmd/server
```

### Pre-built Binary

Download binary yang sudah di-build dari [Releases](https://github.com/yourusername/mikrotik-mcp/releases).

---

## ⚙️ Konfigurasi

Buat file `config.yaml` di direktori yang sama dengan binary:

```yaml
mikrotik:
  host: "192.168.88.1"      # IP router MikroTik
  port: 8728                 # 8728 untuk plain, 8729 untuk TLS
  username: "admin"          # Username RouterOS
  password: "${MIKROTIK_PASSWORD}"  # Gunakan env var untuk keamanan
  use_tls: false             # true untuk koneksi TLS
  reconnect_interval: 5s     # Interval reconnect saat putus
  timeout: 10s               # Timeout operasi API

mcp:
  transport: "stdio"         # stdio | sse
  port: 8080                 # Hanya untuk transport SSE
  read_only: false           # true = hanya operasi read/list

log:
  level: "info"              # debug | info | warn | error
  format: "json"             # json | console
```

### Environment Variables

```bash
# Password MikroTik (direkomendasikan)
export MIKROTIK_PASSWORD=your_secure_password

# Path config alternatif
export CONFIG_PATH=/path/to/config.yaml
```

---

## 🤖 Integrasi dengan AI Client

### Claude Desktop (stdio - rekomendasi)

Edit file konfigurasi Claude Desktop:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mikrotik": {
      "command": "/path/to/mikrotik-mcp",
      "args": [],
      "env": {
        "MIKROTIK_PASSWORD": "your_password"
      }
    }
  }
}
```

### Claude Desktop (SSE)

```json
{
  "mcpServers": {
    "mikrotik": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

### Kimi Code CLI

```bash
# Tambahkan MCP server
kimi mcp add mikrotik --command "/path/to/mikrotik-mcp"

# Atau dengan env var
kimi mcp add mikrotik --command "/path/to/mikrotik-mcp" \
  --env MIKROTIK_PASSWORD=your_password
```

---

## 📖 Daftar Tools MCP

### IP Pool Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `list_ip_pools` | Daftar semua IP pool | - |
| `add_ip_pool` | Tambah IP pool baru | `name`, `ranges`, `next_pool` (opt), `comment` (opt) |
| `update_ip_pool` | Update IP pool | `id`, `ranges` (opt), `comment` (opt) |
| `delete_ip_pool` | Hapus IP pool | `id`, `confirm` |

### Firewall Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `list_firewall_rules` | Daftar firewall filter rules | - |
| `add_firewall_rule` | Tambah rule baru | `chain`, `action`, `src_address`, `dst_address`, `protocol`, `dst_port`, `comment` |
| `delete_firewall_rule` | Hapus rule | `id`, `confirm` |
| `toggle_firewall_rule` | Enable/disable rule | `id`, `disabled` |

### Interface Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `list_interfaces` | Daftar semua interface | - |
| `watch_traffic` | Monitor traffic realtime | `interface`, `seconds` |

### Hotspot Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `list_hotspot_users` | Daftar user hotspot | - |
| `list_hotspot_active` | Daftar sesi aktif | - |
| `add_hotspot_user` | Tambah user baru | `name`, `password`, `profile`, `mac_address`, `comment` |
| `delete_hotspot_user` | Hapus user | `id`, `confirm` |

### Queue Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `list_queues` | Daftar simple queues | - |
| `add_queue` | Tambah queue baru | `name`, `target`, `max_limit`, `limit_at`, `comment` |
| `delete_queue` | Hapus queue | `id`, `confirm` |

### System Tools

| Tool | Deskripsi | Parameter |
|------|-----------|-----------|
| `get_resource` | Info CPU, RAM, uptime | - |
| `get_logs` | Ambil log router | `topics` (opt), `limit` (opt) |
| `reboot_router` | Reboot router | `confirm` |

---

## 💬 Contoh Penggunaan

Setelah terintegrasi dengan AI client, Anda bisa berinteraksi dengan MikroTik menggunakan natural language:

> **User**: "List semua IP pool di router"
> 
> **AI**: "Berikut daftar IP pool yang ada di router:
> - `default-dhcp`: 192.168.88.10-192.168.88.254
> - `hotspot-pool`: 10.5.50.2-10.5.50.254"

> **User**: "Tambahkan firewall rule untuk block IP 192.168.1.100"
> 
> **AI**: "Saya akan menambahkan firewall rule untuk memblokir IP 192.168.1.100..."

> **User**: "Monitor traffic interface ether1 selama 10 detik"
> 
> **AI**: "Memulai monitoring traffic pada ether1..."

---

## 🏗️ Arsitektur

Project ini mengikuti **Clean Architecture** dengan pemisahan concerns yang tegas:

```
┌─────────────────────────────────────────────┐
│           cmd/ & tools/  (Delivery)          │
│  ┌───────────────────────────────────────┐  │
│  │     internal/usecase/  (Use Case)     │  │
│  │  ┌─────────────────────────────────┐  │  │
│  │  │   domain/  (Entity, DTO, Repo)  │  │  │
│  │  └─────────────────────────────────┘  │  │
│  └───────────────────────────────────────┘  │
│    internal/mikrotik/  (Adapter/Infra)       │
└─────────────────────────────────────────────┘
```

### Alur Data

```
AI Client (Claude / GPT)
         │
         │  MCP Protocol (JSON-RPC / stdio / SSE)
         ▼
MCP Server  (tools/)
         │
         │  Memanggil use case via interface
         ▼
Use Case Layer  (internal/usecase/)
         │
         │  Orkestrasi logika bisnis
         ▼
Repository Interface  (domain/repository/)
         │
         │  Abstraksi komunikasi ke router
         ▼
MikroTik Adapter  (internal/mikrotik/)
         │
         │  go-routeros v3
         ▼
MikroTik RouterOS
```

---

## 📁 Struktur Project

```
mikrotik-mcp/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point, wiring dependencies
├── domain/                          # Layer domain - zero external dependency
│   ├── entity/                      # Entity bisnis
│   │   ├── router.go
│   │   ├── interface.go
│   │   ├── ip_pool.go
│   │   ├── firewall.go
│   │   ├── hotspot.go
│   │   ├── queue.go
│   │   └── system.go
│   ├── dto/                         # Data Transfer Objects
│   │   ├── interface_dto.go
│   │   ├── ip_pool_dto.go
│   │   ├── firewall_dto.go
│   │   ├── hotspot_dto.go
│   │   ├── queue_dto.go
│   │   └── system_dto.go
│   └── repository/                  # Interface repository
│       ├── interface_repo.go
│       ├── ip_pool_repo.go
│       ├── firewall_repo.go
│       ├── hotspot_repo.go
│       ├── queue_repo.go
│       └── system_repo.go
├── internal/                        # Kode internal
│   ├── mikrotik/                    # Adapter - implementasi repository
│   │   ├── client.go                # Koneksi & reconnect
│   │   ├── listener.go              # Realtime listen/subscribe
│   │   ├── ip_pool.go
│   │   ├── firewall.go
│   │   ├── interface.go
│   │   ├── hotspot.go
│   │   ├── queue.go
│   │   └── system.go
│   ├── usecase/                     # Logika bisnis
│   │   ├── ip_pool_usecase.go
│   │   ├── firewall_usecase.go
│   │   ├── interface_usecase.go
│   │   ├── hotspot_usecase.go
│   │   ├── queue_usecase.go
│   │   └── system_usecase.go
│   └── config/
│       └── config.go                # Viper config loader
├── tools/                           # MCP Tools definitions
│   ├── registry.go                  # Register semua tools
│   ├── ip_pool_tools.go
│   ├── firewall_tools.go
│   ├── interface_tools.go
│   ├── hotspot_tools.go
│   ├── queue_tools.go
│   └── system_tools.go
├── pkg/                             # Reusable utilities
│   └── logger/                      # Zap wrapper
├── config.yaml                      # Contoh konfigurasi
├── MIKROTIK_API_REFRENCES.MD        # Referensi lengkap API RouterOS
├── PLAN.md                          # Dokumentasi arsitektur
└── README.md                        # Dokumentasi ini
```

---

## 🧪 Testing

### Unit Test

```bash
go test ./... -v
```

### Integration Test (memerlukan router MikroTik)

```bash
# Set konfigurasi test router
export MIKROTIK_TEST_HOST=192.168.88.1
export MIKROTIK_TEST_USER=admin
export MIKROTIK_TEST_PASS=password

go test ./internal/mikrotik/... -v -tags=integration
```

### E2E Test

```bash
go test ./tools/... -v -tags=e2e
```

---

## 🔒 Keamanan

| Aspek | Implementasi |
|-------|--------------|
| **Read-only Mode** | Flag `read_only: true` di config — semua operasi write diblok |
| **Konfirmasi Destruktif** | Tool `reboot_router`, `delete_*` memerlukan `confirm=true` |
| **TLS Support** | Gunakan port 8729 dengan `use_tls: true` |
| **Env Vars untuk Secrets** | Password dari environment variable |
| **Context Timeout** | Semua operasi I/O menggunakan `context.WithTimeout` |

### Best Practices

1. **Gunakan mode read-only** untuk monitoring saja
2. **Gunakan TLS** di production environment
3. **Gunakan user dengan privilege minimal** di RouterOS
4. **Simpan password di environment variable**, bukan di file config
5. **Enable audit logging** (jika tersedia)

---

## 🐛 Troubleshooting

### Connection Refused

```
failed to connect to mikrotik: dial tcp 192.168.88.1:8728: connectex: No connection could be made
```

**Solusi:**
- Pastikan API RouterOS di-enable: `/ip service enable api`
- Cek firewall: `/ip firewall filter print`
- Verifikasi IP dan port

### Authentication Failed

```
failed to connect to mikrotik: cannot log in
```

**Solusi:**
- Periksa username dan password
- Pastikan user memiliki permission yang cukup
- Cek apakah ada IP service access restriction

### TLS Connection Error

**Solusi:**
- Pastikan certificate sudah di-generate di RouterOS
- Gunakan port 8729
- Cek `use_tls: true` di config

---

## 📝 Referensi API

Lihat [MIKROTIK_API_REFRENCES.MD](MIKROTIK_API_REFRENCES.MD) untuk dokumentasi lengkap commands, properties, dan data untuk integrasi via go-routeros v3.

---

## 🤝 Kontribusi

Kontribusi sangat diterima! Silakan buat issue atau pull request.

### Development Workflow

1. Fork repository
2. Buat branch fitur: `git checkout -b feature/nama-fitur`
3. Commit perubahan: `git commit -am 'Add fitur baru'`
4. Push ke branch: `git push origin feature/nama-fitur`
5. Buat Pull Request

---

## 📄 Lisensi

MIT License - lihat [LICENSE](LICENSE) untuk detail lengkap.

---

## 🙏 Kredit

- [go-routeros](https://github.com/go-routeros/routeros) - Client RouterOS API untuk Go
- [mcp-go](https://github.com/mark3labs/mcp-go) - SDK MCP untuk Go
- [MikroTik](https://mikrotik.com/) - RouterOS

---

## 📞 Dukungan

- 📧 Email: your.email@example.com
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/mikrotik-mcp/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/yourusername/mikrotik-mcp/discussions)

---

<p align="center">Made with ❤️ for MikroTik enthusiasts</p>

  # 1. Copy dan isi .env
  cp .env.example .env  # isi MIKROTIK_PASSWORD dan ZAI_API_KEY

  # 2. Jalankan gowa (proses terpisah)
  ./gowa --port 3000 --webhook http://localhost:8090/webhook/message

  # 3. Jalankan MCP Server (mode SSE wajib)
  go run ./cmd/server   # config.yaml mcp.transport: "sse"

  # 4. Jalankan Bot Service
  go run ./cmd/bot