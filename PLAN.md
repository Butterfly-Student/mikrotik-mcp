# MikroTik MCP Go
### Project Architecture & Technical Plan
> Integrasi RouterOS API dengan Model Context Protocol (MCP) & AI — Versi 1.0

---

## Daftar Isi
1. [Gambaran Umum](#1-gambaran-umum)
2. [Tech Stack](#2-tech-stack)
3. [Clean Architecture](#3-clean-architecture)
4. [Struktur Folder](#4-struktur-folder-lengkap)
5. [Domain Layer — Entity & DTO](#5-domain-layer--entity--dto)
6. [Repository Interfaces](#6-domain-repository-interfaces)
7. [Use Case Layer](#7-use-case-layer)
8. [MikroTik Adapter](#8-mikrotik-adapter)
9. [MCP Tools Layer](#9-mcp-tools-layer)
10. [Event Bus & Realtime](#10-event-bus--realtime)
11. [Konfigurasi & Keamanan](#11-konfigurasi--keamanan)
12. [Testing Strategy](#12-testing-strategy)
13. [Rencana Pengembangan](#13-rencana-pengembangan-fase)

---

## 1. Gambaran Umum

MikroTik MCP Go adalah sistem backend berbasis Go yang menghubungkan MikroTik RouterOS dengan AI melalui Model Context Protocol (MCP). Sistem ini memungkinkan AI seperti Claude atau GPT untuk memahami, memantau, dan mengeksekusi operasi jaringan secara natural language.

**Tujuan Utama:**
- Menyediakan interface yang bersih antara RouterOS API dan dunia luar
- Mengekspos kemampuan MikroTik sebagai MCP Tools yang bisa dipanggil AI
- Mendukung realtime monitoring via RouterOS listen/subscribe
- Arsitektur yang scalable untuk multi-router di masa depan

### 1.1 Alur Sistem

```
AI Client (Claude / GPT / Custom)
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

## 2. Tech Stack

### 2.1 Core

| Teknologi | Versi / Library | Kegunaan |
|-----------|----------------|----------|
| Go | 1.25+ | Bahasa utama — concurrent, performa tinggi, single binary |
| go-routeros | v3 `github.com/go-routeros/routeros/v3` | Client RouterOS API dengan TLS & context support |
| MCP Go SDK | `github.com/mark3labs/mcp-go` | Implementasi MCP server di Go |
| SQLite | `github.com/mattn/go-sqlite3` | Audit log & multi-router config (opsional, ringan) |
| Viper | `github.com/spf13/viper` | Konfigurasi & environment variables |
| Zap | `go.uber.org/zap` | Structured logging performa tinggi |
| Testify | `github.com/stretchr/testify` | Unit & integration testing |

### 2.2 Opsional (Tahap Lanjut)

| Teknologi | Kegunaan | Kapan Diperlukan |
|-----------|----------|-----------------|
| InfluxDB / TimescaleDB | Penyimpanan historical time-series traffic stats | Ketika butuh grafik & analitik historical |
| Redis | Caching data yang jarang berubah dari router | Ketika ada banyak query berulang & multi-router |
| Prometheus + Grafana | Metrics observability untuk sistem Go itu sendiri | Production deployment |
| Docker / Podman | Containerisasi untuk kemudahan deployment | Tahap deployment |

---

## 3. Clean Architecture

Proyek ini mengikuti prinsip Clean Architecture dengan pemisahan concerns yang tegas. Dependency mengalir dari luar ke dalam — layer luar boleh bergantung ke layer dalam, tapi tidak sebaliknya.

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

> **Dependency Rule:** `tools/` & `cmd/` → `usecase/` → `domain/`
> Domain tidak tahu siapa pemanggilnya. `internal/mikrotik/` mengimplementasikan `domain/repository/`.

### 3.1 Lapisan Arsitektur

| Layer | Paket | Tanggung Jawab |
|-------|-------|----------------|
| Domain | `domain/` | Entity, DTO, interface Repository. Tidak bergantung ke library apapun |
| Use Case | `internal/usecase/` | Logika bisnis. Hanya tahu domain interfaces, bukan implementasi |
| Repository / Adapter | `internal/mikrotik/` | Implementasi konkret komunikasi ke RouterOS via go-routeros |
| MCP Tools | `tools/` | Expose use case sebagai MCP Tools. Validasi input, format output untuk AI |
| Infrastructure | `internal/config/`, `pkg/logger/` | Konfigurasi, logging, utilities lintas layer |
| Entry Point | `cmd/server/` | Wiring semua dependency (dependency injection manual atau wire) |

---

## 4. Struktur Folder Lengkap

```
mikrotik-mcp/
├── cmd/
│   └── server/
│       └── main.go                  # Wire semua dependency, start MCP server
│
├── domain/                          # LAYER PALING DALAM — zero external dependency
│   ├── entity/
│   │   ├── router.go                # Router, Connection entity
│   │   ├── interface.go             # NetworkInterface, TrafficStat entity
│   │   ├── ip_pool.go               # IPPool entity
│   │   ├── firewall.go              # FirewallRule, FirewallFilter entity
│   │   ├── hotspot.go               # HotspotUser, HotspotServer entity
│   │   ├── queue.go                 # Queue entity
│   │   └── system.go                # SystemResource, Log entity
│   │
│   ├── dto/                         # Data Transfer Objects — input/output use case
│   │   ├── ip_pool_dto.go
│   │   ├── firewall_dto.go
│   │   ├── interface_dto.go
│   │   ├── hotspot_dto.go
│   │   ├── queue_dto.go
│   │   └── system_dto.go
│   │
│   └── repository/                  # Interface repository (kontrak untuk adapter)
│       ├── ip_pool_repo.go
│       ├── firewall_repo.go
│       ├── interface_repo.go
│       ├── hotspot_repo.go
│       ├── queue_repo.go
│       └── system_repo.go
│
├── internal/                        # Kode internal — tidak boleh di-import dari luar modul
│   ├── mikrotik/                    # ADAPTER — implementasi domain/repository/
│   │   ├── client.go                # Manajemen koneksi, reconnect, TLS
│   │   ├── listener.go              # Realtime listen/subscribe goroutine
│   │   ├── ip_pool.go               # Implements IPPoolRepository
│   │   ├── firewall.go              # Implements FirewallRepository
│   │   ├── interface.go             # Implements InterfaceRepository
│   │   ├── hotspot.go               # Implements HotspotRepository
│   │   ├── queue.go                 # Implements QueueRepository
│   │   └── system.go                # Implements SystemRepository
│   │
│   ├── usecase/
│   │   ├── ip_pool_usecase.go
│   │   ├── firewall_usecase.go
│   │   ├── interface_usecase.go
│   │   ├── hotspot_usecase.go
│   │   ├── queue_usecase.go
│   │   └── system_usecase.go
│   │
│   └── config/
│       └── config.go                # Viper config loader
│
├── tools/                           # MCP Tools definitions
│   ├── registry.go                  # Register semua tools ke MCP server
│   ├── ip_pool_tools.go
│   ├── firewall_tools.go
│   ├── interface_tools.go
│   ├── hotspot_tools.go
│   ├── queue_tools.go
│   └── system_tools.go
│
├── pkg/                             # Reusable utilities (boleh dipakai proyek lain)
│   ├── logger/                      # Zap wrapper
│   └── eventbus/                    # Event bus untuk realtime (pub/sub)
│
├── config.yaml
├── go.mod
└── go.sum
```

---

## 5. Domain Layer — Entity & DTO

### 5.1 Entity vs DTO

| | Entity | DTO |
|---|--------|-----|
| Letak | `domain/entity/` | `domain/dto/` |
| Tujuan | Merepresentasikan objek bisnis sesungguhnya | Membawa data masuk/keluar use case |
| Validasi | Invariant bisnis | Validasi input dari user/AI |
| Bergantung ke | Tidak bergantung ke siapapun | Tidak bergantung ke siapapun |
| Contoh | `IPPool{Name, Ranges, NextPool}` | `CreateIPPoolRequest`, `IPPoolResponse` |

### 5.2 Entity

```go
// domain/entity/ip_pool.go
package entity

type IPPool struct {
    ID       string // .id dari MikroTik
    Name     string
    Ranges   string // e.g. "192.168.1.100-192.168.1.200"
    NextPool string // nama pool berikutnya (opsional)
    Comment  string
}

// domain/entity/firewall.go
type FirewallRule struct {
    ID         string
    Chain      string // input, forward, output
    Action     string // accept, drop, reject
    SrcAddress string
    DstAddress string
    Protocol   string
    DstPort    string
    Comment    string
    Disabled   bool
}

// domain/entity/interface.go
type NetworkInterface struct {
    ID        string
    Name      string
    Type      string // ether, wlan, bridge, vlan
    MacAddress string
    MTU       int
    Running   bool
    Disabled  bool
    Comment   string
}

type TrafficStat struct {
    Interface string
    RxBitsPerSecond int64
    TxBitsPerSecond int64
    Timestamp       time.Time
}

// domain/entity/system.go
type SystemResource struct {
    Uptime      string
    Version     string
    CPULoad     int
    FreeMemory  int64
    TotalMemory int64
    FreeDisk    int64
}

type SystemLog struct {
    Time    string
    Topics  string
    Message string
}
```

### 5.3 DTO

```go
// domain/dto/ip_pool_dto.go
package dto

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateIPPoolRequest struct {
    Name     string `json:"name"     validate:"required"`
    Ranges   string `json:"ranges"   validate:"required"`
    NextPool string `json:"next_pool,omitempty"`
    Comment  string `json:"comment,omitempty"`
}

type UpdateIPPoolRequest struct {
    ID      string `json:"id"      validate:"required"`
    Ranges  string `json:"ranges,omitempty"`
    Comment string `json:"comment,omitempty"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type IPPoolResponse struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Ranges   string `json:"ranges"`
    NextPool string `json:"next_pool,omitempty"`
    Comment  string `json:"comment,omitempty"`
}

type ListIPPoolResponse struct {
    Pools []IPPoolResponse `json:"pools"`
    Total int              `json:"total"`
}
```

```go
// domain/dto/firewall_dto.go

type CreateFirewallRuleRequest struct {
    Chain      string `json:"chain"       validate:"required,oneof=input forward output"`
    Action     string `json:"action"      validate:"required,oneof=accept drop reject"`
    SrcAddress string `json:"src_address,omitempty"`
    DstAddress string `json:"dst_address,omitempty"`
    Protocol   string `json:"protocol,omitempty"`
    DstPort    string `json:"dst_port,omitempty"`
    Comment    string `json:"comment,omitempty"`
}

type FirewallRuleResponse struct {
    ID         string `json:"id"`
    Chain      string `json:"chain"`
    Action     string `json:"action"`
    SrcAddress string `json:"src_address,omitempty"`
    DstAddress string `json:"dst_address,omitempty"`
    Protocol   string `json:"protocol,omitempty"`
    DstPort    string `json:"dst_port,omitempty"`
    Comment    string `json:"comment,omitempty"`
    Disabled   bool   `json:"disabled"`
}

type ListFirewallRuleResponse struct {
    Rules []FirewallRuleResponse `json:"rules"`
    Total int                    `json:"total"`
}
```

```go
// domain/dto/interface_dto.go

type InterfaceResponse struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Type        string `json:"type"`
    MacAddress  string `json:"mac_address"`
    MTU         int    `json:"mtu"`
    Running     bool   `json:"running"`
    Disabled    bool   `json:"disabled"`
}

type WatchTrafficRequest struct {
    Interface string `json:"interface" validate:"required"`
    Seconds   int    `json:"seconds"   validate:"required,min=1,max=60"`
}

type TrafficStatResponse struct {
    Interface string `json:"interface"`
    RxBps     int64  `json:"rx_bps"`
    TxBps     int64  `json:"tx_bps"`
    Timestamp string `json:"timestamp"`
}

// domain/dto/system_dto.go

type SystemResourceResponse struct {
    Uptime      string `json:"uptime"`
    Version     string `json:"version"`
    CPULoad     int    `json:"cpu_load_percent"`
    FreeMemory  string `json:"free_memory"`
    TotalMemory string `json:"total_memory"`
}

type GetLogsRequest struct {
    Topics string `json:"topics,omitempty"` // e.g. "firewall", "dhcp"
    Limit  int    `json:"limit,omitempty"`
}
```

---

## 6. Domain Repository Interfaces

Interface ini adalah kontrak yang harus diimplementasikan oleh mikrotik adapter. Use case hanya tahu interface ini.

```go
// domain/repository/ip_pool_repo.go
package repository

import (
    "context"
    "github.com/yourname/mikrotik-mcp/domain/entity"
    "github.com/yourname/mikrotik-mcp/domain/dto"
)

type IPPoolRepository interface {
    GetAll(ctx context.Context) ([]entity.IPPool, error)
    GetByName(ctx context.Context, name string) (*entity.IPPool, error)
    Create(ctx context.Context, req dto.CreateIPPoolRequest) error
    Update(ctx context.Context, req dto.UpdateIPPoolRequest) error
    Delete(ctx context.Context, id string) error
}

// domain/repository/firewall_repo.go
type FirewallRepository interface {
    GetAll(ctx context.Context) ([]entity.FirewallRule, error)
    Create(ctx context.Context, req dto.CreateFirewallRuleRequest) error
    Delete(ctx context.Context, id string) error
    Toggle(ctx context.Context, id string, disabled bool) error
}

// domain/repository/interface_repo.go
type InterfaceRepository interface {
    GetAll(ctx context.Context) ([]entity.NetworkInterface, error)
    StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error
    StopTrafficMonitor(ctx context.Context, iface string) error
}

// domain/repository/hotspot_repo.go
type HotspotRepository interface {
    GetUsers(ctx context.Context) ([]entity.HotspotUser, error)
    AddUser(ctx context.Context, req dto.CreateHotspotUserRequest) error
    DeleteUser(ctx context.Context, id string) error
}

// domain/repository/system_repo.go
type SystemRepository interface {
    GetResource(ctx context.Context) (*entity.SystemResource, error)
    GetLogs(ctx context.Context, req dto.GetLogsRequest) ([]entity.SystemLog, error)
    Reboot(ctx context.Context) error
}
```

---

## 7. Use Case Layer

Use case berisi logika bisnis. Ia hanya bergantung ke domain interfaces.

```go
// internal/usecase/ip_pool_usecase.go
package usecase

type IPPoolUseCase struct {
    repo   repository.IPPoolRepository
    logger *zap.Logger
}

func NewIPPoolUseCase(repo repository.IPPoolRepository, logger *zap.Logger) *IPPoolUseCase {
    return &IPPoolUseCase{repo: repo, logger: logger}
}

func (uc *IPPoolUseCase) ListPools(ctx context.Context) (*dto.ListIPPoolResponse, error) {
    pools, err := uc.repo.GetAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get ip pools: %w", err)
    }

    responses := make([]dto.IPPoolResponse, len(pools))
    for i, p := range pools {
        responses[i] = dto.IPPoolResponse{
            ID:       p.ID,
            Name:     p.Name,
            Ranges:   p.Ranges,
            NextPool: p.NextPool,
            Comment:  p.Comment,
        }
    }
    return &dto.ListIPPoolResponse{Pools: responses, Total: len(responses)}, nil
}

func (uc *IPPoolUseCase) CreatePool(ctx context.Context, req dto.CreateIPPoolRequest) error {
    // validasi bisnis tambahan jika perlu
    if err := uc.repo.Create(ctx, req); err != nil {
        return fmt.Errorf("failed to create ip pool: %w", err)
    }
    uc.logger.Info("ip pool created", zap.String("name", req.Name))
    return nil
}

func (uc *IPPoolUseCase) DeletePool(ctx context.Context, id string) error {
    return uc.repo.Delete(ctx, id)
}
```

---

## 8. MikroTik Adapter

### 8.1 Client — Koneksi & Reconnect

```go
// internal/mikrotik/client.go
package mikrotik

type Config struct {
    Host              string
    Port              int
    Username          string
    Password          string
    UseTLS            bool
    ReconnectInterval time.Duration
}

type Client struct {
    conn   *routeros.Client
    config Config
    mu     sync.RWMutex
    logger *zap.Logger
}

func NewClient(cfg Config, logger *zap.Logger) *Client {
    return &Client{config: cfg, logger: logger}
}

func (c *Client) Connect(ctx context.Context) error {
    addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
    var conn *routeros.Client
    var err error

    if c.config.UseTLS {
        conn, err = routeros.DialTLS(addr, c.config.Username, c.config.Password, nil)
    } else {
        conn, err = routeros.Dial(addr, c.config.Username, c.config.Password)
    }
    if err != nil {
        return fmt.Errorf("failed to connect to mikrotik: %w", err)
    }

    c.mu.Lock()
    c.conn = conn
    c.mu.Unlock()
    return nil
}

// Reconnect dengan exponential backoff
func (c *Client) Reconnect(ctx context.Context) error {
    backoff := time.Second
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := c.Connect(ctx); err != nil {
                c.logger.Warn("reconnect failed, retrying", zap.Duration("after", backoff))
                time.Sleep(backoff)
                if backoff < 30*time.Second {
                    backoff *= 2
                }
                continue
            }
            c.logger.Info("reconnected to mikrotik")
            return nil
        }
    }
}

func (c *Client) Run(sentence ...string) (*routeros.Reply, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.conn.Run(sentence...)
}
```

### 8.2 Listener — Realtime Subscribe

```go
// internal/mikrotik/listener.go

// StartTrafficMonitor meluncurkan goroutine yang stream traffic data dari RouterOS
// dan mengirim setiap update ke channel ch
func (c *Client) StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error {
    c.mu.RLock()
    listen, err := c.conn.ListenArgs(routeros.Args{
        "/interface/monitor-traffic",
        "=interface=" + iface,
    })
    c.mu.RUnlock()
    if err != nil {
        return fmt.Errorf("failed to start traffic monitor: %w", err)
    }

    go func() {
        defer listen.Cancel()
        for {
            select {
            case <-ctx.Done():
                return
            case sentence, ok := <-listen.Chan():
                if !ok {
                    return
                }
                stat := parseTrafficSentence(sentence, iface)
                select {
                case ch <- stat:
                default: // drop jika channel penuh, tidak blocking
                }
            }
        }
    }()
    return nil
}

func parseTrafficSentence(s *proto.Sentence, iface string) entity.TrafficStat {
    rxBps, _ := strconv.ParseInt(s.Map["rx-bits-per-second"], 10, 64)
    txBps, _ := strconv.ParseInt(s.Map["tx-bits-per-second"], 10, 64)
    return entity.TrafficStat{
        Interface:       iface,
        RxBitsPerSecond: rxBps,
        TxBitsPerSecond: txBps,
        Timestamp:       time.Now(),
    }
}
```

### 8.3 Implementasi Repository

```go
// internal/mikrotik/ip_pool.go

type ipPoolRepository struct {
    client *Client
}

func NewIPPoolRepository(client *Client) repository.IPPoolRepository {
    return &ipPoolRepository{client: client}
}

func (r *ipPoolRepository) GetAll(ctx context.Context) ([]entity.IPPool, error) {
    reply, err := r.client.Run("/ip/pool/print")
    if err != nil {
        return nil, err
    }

    pools := make([]entity.IPPool, 0, len(reply.Re))
    for _, sentence := range reply.Re {
        pools = append(pools, entity.IPPool{
            ID:       sentence.Map[".id"],
            Name:     sentence.Map["name"],
            Ranges:   sentence.Map["ranges"],
            NextPool: sentence.Map["next-pool"],
            Comment:  sentence.Map["comment"],
        })
    }
    return pools, nil
}

func (r *ipPoolRepository) Create(ctx context.Context, req dto.CreateIPPoolRequest) error {
    args := []string{"/ip/pool/add", "=name=" + req.Name, "=ranges=" + req.Ranges}
    if req.NextPool != "" {
        args = append(args, "=next-pool="+req.NextPool)
    }
    if req.Comment != "" {
        args = append(args, "=comment="+req.Comment)
    }
    _, err := r.client.Run(args...)
    return err
}

func (r *ipPoolRepository) Delete(ctx context.Context, id string) error {
    _, err := r.client.Run("/ip/pool/remove", "=.id="+id)
    return err
}
```

---

## 9. MCP Tools Layer

### 9.1 Daftar Semua Tools

| Modul | Tool Name | Deskripsi |
|-------|-----------|-----------|
| IP Pool | `list_ip_pools` | Daftar semua IP pool |
| IP Pool | `add_ip_pool` | Tambah IP pool baru |
| IP Pool | `update_ip_pool` | Update ranges/comment pool |
| IP Pool | `delete_ip_pool` | Hapus IP pool |
| Firewall | `list_firewall_rules` | List semua firewall filter rules |
| Firewall | `add_firewall_rule` | Tambah rule baru (drop/accept) |
| Firewall | `delete_firewall_rule` | Hapus rule berdasarkan ID |
| Firewall | `toggle_firewall_rule` | Enable/disable rule |
| Interface | `list_interfaces` | Daftar semua interface |
| Interface | `watch_traffic` | Monitor traffic realtime selama N detik |
| Hotspot | `list_hotspot_users` | Daftar user hotspot |
| Hotspot | `add_hotspot_user` | Tambah user hotspot baru |
| Hotspot | `delete_hotspot_user` | Hapus user hotspot |
| Queue | `list_queues` | Daftar simple/tree queue |
| Queue | `add_queue` | Tambah queue baru dengan limit bandwidth |
| System | `get_resource` | Info CPU, RAM, uptime router |
| System | `get_logs` | Ambil log router (bisa filter topic) |
| System | `reboot_router` | Reboot router (memerlukan `confirm=true`) |

### 9.2 Tool Definition

```go
// tools/ip_pool_tools.go
package tools

func RegisterIPPoolTools(s *server.MCPServer, uc *usecase.IPPoolUseCase) {

    // list_ip_pools
    s.AddTool(
        mcp.NewTool("list_ip_pools",
            mcp.WithDescription("Menampilkan semua IP pool yang ada di MikroTik"),
        ),
        func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            result, err := uc.ListPools(ctx)
            if err != nil {
                return mcp.NewToolResultError(err.Error()), nil
            }
            return mcp.NewToolResultJSON(result), nil
        },
    )

    // add_ip_pool
    s.AddTool(
        mcp.NewTool("add_ip_pool",
            mcp.WithDescription("Menambahkan IP pool baru ke MikroTik"),
            mcp.WithString("name",
                mcp.Required(),
                mcp.Description("Nama IP pool"),
            ),
            mcp.WithString("ranges",
                mcp.Required(),
                mcp.Description("Range IP, contoh: 192.168.1.100-192.168.1.200"),
            ),
            mcp.WithString("comment",
                mcp.Description("Keterangan opsional"),
            ),
        ),
        func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            name, _ := req.Params.Arguments["name"].(string)
            ranges, _ := req.Params.Arguments["ranges"].(string)
            comment, _ := req.Params.Arguments["comment"].(string)

            err := uc.CreatePool(ctx, dto.CreateIPPoolRequest{
                Name:    name,
                Ranges:  ranges,
                Comment: comment,
            })
            if err != nil {
                return mcp.NewToolResultError(err.Error()), nil
            }
            return mcp.NewToolResultText("IP pool '" + name + "' berhasil dibuat"), nil
        },
    )
}
```

```go
// tools/system_tools.go — contoh tool destruktif dengan konfirmasi

s.AddTool(
    mcp.NewTool("reboot_router",
        mcp.WithDescription("Mereboot MikroTik router. WAJIB sertakan confirm=true"),
        mcp.WithBoolean("confirm",
            mcp.Required(),
            mcp.Description("Harus true untuk mengkonfirmasi reboot"),
        ),
    ),
    func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        confirm, _ := req.Params.Arguments["confirm"].(bool)
        if !confirm {
            return mcp.NewToolResultError("Reboot dibatalkan. Sertakan confirm=true untuk melanjutkan"), nil
        }
        if err := uc.Reboot(ctx); err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        return mcp.NewToolResultText("Router sedang reboot..."), nil
    },
)
```

### 9.3 Registry

```go
// tools/registry.go
package tools

type Dependencies struct {
    IPPool    *usecase.IPPoolUseCase
    Firewall  *usecase.FirewallUseCase
    Interface *usecase.InterfaceUseCase
    Hotspot   *usecase.HotspotUseCase
    Queue     *usecase.QueueUseCase
    System    *usecase.SystemUseCase
}

func RegisterAll(s *server.MCPServer, deps Dependencies) {
    RegisterIPPoolTools(s, deps.IPPool)
    RegisterFirewallTools(s, deps.Firewall)
    RegisterInterfaceTools(s, deps.Interface)
    RegisterHotspotTools(s, deps.Hotspot)
    RegisterQueueTools(s, deps.Queue)
    RegisterSystemTools(s, deps.System)
}
```

---

## 10. Event Bus & Realtime

Untuk data realtime (traffic monitoring, log streaming), digunakan pola Event Bus sederhana berbasis channel Go.

```go
// pkg/eventbus/eventbus.go
package eventbus

type EventBus struct {
    subs map[string][]chan any
    mu   sync.RWMutex
}

func New() *EventBus {
    return &EventBus{subs: make(map[string][]chan any)}
}

func (eb *EventBus) Subscribe(topic string) chan any {
    ch := make(chan any, 100)
    eb.mu.Lock()
    eb.subs[topic] = append(eb.subs[topic], ch)
    eb.mu.Unlock()
    return ch
}

func (eb *EventBus) Unsubscribe(topic string, ch chan any) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    subs := eb.subs[topic]
    for i, s := range subs {
        if s == ch {
            eb.subs[topic] = append(subs[:i], subs[i+1:]...)
            close(ch)
            return
        }
    }
}

func (eb *EventBus) Publish(topic string, data any) {
    eb.mu.RLock()
    defer eb.mu.RUnlock()
    for _, ch := range eb.subs[topic] {
        select {
        case ch <- data:
        default: // jangan blocking jika subscriber lambat
        }
    }
}
```

**Alur realtime tool `watch_traffic`:**

```
1. AI memanggil: watch_traffic(interface="ether1", seconds=10)
2. Tool subscribe ke eventbus topic "traffic.ether1"
3. Jika listener belum jalan → start goroutine ListenTraffic(ctx, "ether1", publishCh)
4. Listener goroutine publish ke eventbus setiap ada data dari RouterOS
5. Tool kumpulkan events selama 10 detik
6. Unsubscribe & return hasil sebagai JSON ke AI
```

---

## 11. Konfigurasi & Keamanan

### 11.1 config.yaml

```yaml
mikrotik:
  host: "192.168.88.1"
  port: 8728            # 8729 untuk TLS
  username: "admin"
  password: "${MIKROTIK_PASSWORD}"  # ambil dari env var
  use_tls: false
  reconnect_interval: 5s
  timeout: 10s

mcp:
  transport: "stdio"    # stdio | sse
  port: 8080            # hanya untuk transport SSE
  read_only: false      # true = hanya GET/list, tidak bisa write

log:
  level: "info"         # debug | info | warn | error
  format: "json"        # json | console
```

### 11.2 Keamanan

| Aspek | Implementasi |
|-------|--------------|
| Read-only Mode | Flag di config — tools yang bersifat write return error `"operation not permitted in read-only mode"` |
| Konfirmasi Destruktif | Tool `reboot_router`, `delete_firewall_rule` memerlukan parameter `confirm=true` secara eksplisit |
| TLS ke RouterOS | Gunakan port 8729 dengan `use_tls: true` di production |
| Env Vars untuk Secrets | Password dari environment variable, bukan dari file config |
| Audit Log | Setiap tool call dicatat: timestamp, tool name, input args, result — simpan di SQLite |
| Context Timeout | Semua operasi ke RouterOS dibungkus `context.WithTimeout` untuk mencegah hanging |

### 11.3 Audit Log (SQLite)

```go
// internal/audit/audit.go — hanya dicatat, tidak blocking alur utama

type AuditLog struct {
    ID        int64
    Timestamp time.Time
    ToolName  string
    InputArgs string // JSON
    Success   bool
    Error     string
    Duration  time.Duration
}

func (a *Auditor) Log(ctx context.Context, entry AuditLog) {
    // simpan ke SQLite secara async, tidak block response ke AI
    go func() {
        _ = a.db.ExecContext(ctx,
            `INSERT INTO audit_logs (timestamp, tool_name, input_args, success, error, duration_ms)
             VALUES (?, ?, ?, ?, ?, ?)`,
            entry.Timestamp, entry.ToolName, entry.InputArgs,
            entry.Success, entry.Error, entry.Duration.Milliseconds(),
        )
    }()
}
```

---

## 12. Testing Strategy

| Level | Target | Pendekatan |
|-------|--------|------------|
| Unit Test | Use case logic | Testify + mock repository via interface |
| Unit Test | DTO validation | Test langsung struct validation |
| Unit Test | Entity parsing | Test `parseTrafficSentence`, `parseIPPool`, dsb |
| Integration Test | MikroTik adapter | Router real atau MikroTik CHR di VM/container |
| E2E Test | MCP Tool → RouterOS | Jalankan server, panggil tool via MCP client |

**Contoh Unit Test dengan Mock:**

```go
// internal/usecase/ip_pool_usecase_test.go

type mockIPPoolRepo struct {
    mock.Mock
}

func (m *mockIPPoolRepo) GetAll(ctx context.Context) ([]entity.IPPool, error) {
    args := m.Called(ctx)
    return args.Get(0).([]entity.IPPool), args.Error(1)
}

func TestListPools(t *testing.T) {
    mockRepo := new(mockIPPoolRepo)
    mockRepo.On("GetAll", mock.Anything).Return([]entity.IPPool{
        {ID: "*1", Name: "pool-1", Ranges: "192.168.1.100-192.168.1.200"},
    }, nil)

    uc := NewIPPoolUseCase(mockRepo, zap.NewNop())
    result, err := uc.ListPools(context.Background())

    assert.NoError(t, err)
    assert.Equal(t, 1, result.Total)
    assert.Equal(t, "pool-1", result.Pools[0].Name)
    mockRepo.AssertExpectations(t)
}
```

---

## 13. Rencana Pengembangan (Fase)

| Fase | Scope | Output |
|------|-------|--------|
| **Fase 1** — Foundation | `client.go`, domain entities + DTOs + interfaces, ip_pool adapter + usecase + tool | MCP server bisa list/add/delete IP Pool via AI |
| **Fase 2** — Core Features | Firewall, Interface, Queue adapter + usecase + tools | AI bisa manage firewall & lihat interface |
| **Fase 3** — Realtime | `listener.go`, EventBus, `watch_traffic` tool | AI bisa monitor traffic realtime |
| **Fase 4** — Hotspot & System | Hotspot, System resource + logs + reboot | Full network management via AI |
| **Fase 5** — Hardening | Audit log SQLite, read-only mode, konfirmasi destruktif, retry & reconnect | Production-ready |
| **Fase 6** — Advanced | Multi-router support, SSE transport, historical stats InfluxDB (opsional) | Scalable untuk ISP / multi-site |

---

## Ringkasan Prinsip Arsitektur

| Prinsip | Implementasi |
|---------|--------------|
| **Clean Architecture** | Domain tidak bergantung ke siapapun, dependency rule terjaga |
| **Separation of Concerns** | `mikrotik/` hanya bicara ke router, `tools/` hanya definisi MCP |
| **Interface-based Design** | Semua repository berupa interface, mudah di-mock dan di-swap |
| **DTO di Domain** | Request/response use case didefinisikan di domain agar portable |
| **Goroutine + Channel** | Realtime data menggunakan pola Go yang idiomatis |
| **Context Everywhere** | Semua operasi I/O menerima `context.Context` untuk cancellation & timeout |
| **Security by Default** | Read-only mode, konfirmasi destruktif, audit log |

---

*Dengan arsitektur ini, proyek akan mudah ditest, mudah diperluas (tambah modul baru tanpa ubah yang lama), dan siap diintegrasikan dengan berbagai AI model melalui protokol MCP yang standar.*