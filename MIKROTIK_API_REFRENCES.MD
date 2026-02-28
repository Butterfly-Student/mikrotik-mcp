# MikroTik RouterOS — API Command Reference
> Referensi lengkap commands, properties, dan data untuk integrasi via go-routeros v3

---

## Catatan API

Semua command di RouterOS API menggunakan format path seperti CLI dengan `/` sebagai separator.

```
CLI:  /ip firewall filter print
API:  /ip/firewall/filter/print
```

**Universal Commands per menu:**
- `/print` atau `/getall` — ambil semua data
- `/add` — tambah item baru
- `/set` — ubah item (butuh `=.id=`)
- `/remove` — hapus item (butuh `=.id=`)
- `/listen` — stream realtime changes
- `/find` — cari item berdasarkan kondisi (kembalikan `.id`)
- `/enable` & `/disable` — aktifkan/nonaktifkan item

**Konvensi property di API:**
- Semua nama property menggunakan `kebab-case` (bukan camelCase)
- `.id` adalah internal ID (`*1`, `*2`, dst.) — digunakan untuk set/remove
- Property `read-only` tidak bisa di-set saat add/set
- Flag `D` = dynamic (dibuat otomatis sistem), `X` = disabled, `I` = invalid

---

## 1. IP Pool

**Path:** `/ip/pool`

### Commands

| Command | API Path | Keterangan |
|---------|----------|-----------|
| Print semua pool | `/ip/pool/print` | |
| Print used addresses | `/ip/pool/used/print` | |
| Add pool | `/ip/pool/add` | |
| Edit pool | `/ip/pool/set` | butuh `=.id=` |
| Remove pool | `/ip/pool/remove` | butuh `=.id=` |
| Find pool | `/ip/pool/find` | e.g. `?name=my-pool` |

### Properties — `/ip/pool/print`

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID (`*1`, `*2`) |
| `name` | string | R/W | Nama unik pool (**required** saat add) |
| `ranges` | string | R/W | Range IP, format: `from1-to1,from2-to2` (**required** saat add) |
| `next-pool` | string | R/W | Pool fallback jika pool ini habis (opsional) |
| `comment` | string | R/W | Keterangan (opsional) |

### Properties — `/ip/pool/used/print` (Read-Only)

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `pool` | string | Nama pool yang digunakan |
| `address` | string | IP address yang sedang digunakan |
| `owner` | string | Service yang menggunakan (`dhcp`, `ppp`, dll.) |
| `info` | string | Info tambahan (MAC address untuk DHCP, username untuk PPP) |

### Contoh API Calls

```
# Print semua pool
/ip/pool/print

# Add pool baru
/ip/pool/add
=name=my-pool
=ranges=192.168.1.100-192.168.1.200
=comment=Pool untuk hotspot lantai 1

# Edit ranges pool
/ip/pool/set
=.id=*1
=ranges=192.168.1.100-192.168.1.250

# Remove pool
/ip/pool/remove
=.id=*1

# Filter: cari pool berdasarkan nama
/ip/pool/print
?name=my-pool
```

---

## 2. Interface

**Path:** `/interface`

### Commands

| Command | API Path | Keterangan |
|---------|----------|-----------|
| Print semua interface | `/interface/print` | |
| Print stats | `/interface/print` dengan `stats` | |
| Monitor traffic realtime | `/interface/monitor-traffic` | listen/stream |
| Enable interface | `/interface/enable` | butuh `=.id=` |
| Disable interface | `/interface/disable` | butuh `=.id=` |
| Set property | `/interface/set` | butuh `=.id=` |
| Print stats detail | `/interface/print` + `stats-detail` | |
| Print Ethernet | `/interface/ethernet/print` | spesifik ethernet |
| Print Bridge | `/interface/bridge/print` | spesifik bridge |
| Print VLAN | `/interface/vlan/print` | spesifik vlan |

### Properties — `/interface/print`

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Nama interface |
| `type` | string | R | Tipe: `ether`, `wlan`, `bridge`, `vlan`, `loopback`, `pppoe-out`, dll. |
| `mtu` | integer | R/W | MTU size (default: 1500) |
| `mac-address` | string | R | MAC address hardware |
| `disabled` | bool | R/W | `true`/`false` |
| `running` | bool | R | Apakah interface sedang aktif |
| `dynamic` | bool | R | Apakah dibuat secara dinamis oleh sistem |
| `slave` | bool | R | Apakah interface adalah slave dari bridge/bond |
| `comment` | string | R/W | Keterangan |
| `last-link-up-time` | string | R | Waktu terakhir link naik |
| `link-downs` | integer | R | Jumlah link down sejak boot |

### Properties — `/interface/print stats` (Statistik)

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `rx-byte` | integer | Total bytes diterima |
| `tx-byte` | integer | Total bytes dikirim |
| `rx-packet` | integer | Total paket diterima |
| `tx-packet` | integer | Total paket dikirim |
| `rx-error` | integer | Jumlah error saat menerima |
| `tx-error` | integer | Jumlah error saat mengirim |
| `rx-drop` | integer | Paket drop saat menerima |
| `tx-drop` | integer | Paket drop saat mengirim |
| `fp-rx-byte` | integer | FastPath rx bytes |
| `fp-tx-byte` | integer | FastPath tx bytes |
| `fp-rx-packet` | integer | FastPath rx packets |
| `fp-tx-packet` | integer | FastPath tx packets |

### Properties — `/interface/monitor-traffic` (Realtime Stream)

> Command ini menggunakan **listen mode** — stream data terus sampai di-cancel.
> Kirim `=interface=ether1` atau `=interface=all` untuk semua.

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `name` | string | Nama interface |
| `rx-bits-per-second` | integer | Kecepatan download saat ini (bps) |
| `tx-bits-per-second` | integer | Kecepatan upload saat ini (bps) |
| `rx-packets-per-second` | integer | Paket/detik diterima |
| `tx-packets-per-second` | integer | Paket/detik dikirim |
| `rx-drops-per-second` | integer | Drop/detik saat menerima |
| `tx-drops-per-second` | integer | Drop/detik saat mengirim |

### Contoh API Calls

```
# Print semua interface
/interface/print

# Monitor traffic realtime (stream)
/interface/monitor-traffic
=interface=ether1
=once=         # tanpa ini = stream terus-menerus

# Monitor semua interface sekaligus
/interface/monitor-traffic
=interface=all
=once=

# Disable interface
/interface/disable
=.id=ether2

# Set comment
/interface/set
=.id=*3
=comment=WAN Interface

# Filter: hanya ether yang running
/interface/print
?type=ether
?running=true
```

---

## 3. Firewall Filter

**Path:** `/ip/firewall/filter`

### Commands

| Command | API Path | Keterangan |
|---------|----------|-----------|
| Print semua rules | `/ip/firewall/filter/print` | termasuk dynamic |
| Print hanya static | `/ip/firewall/filter/print` + `?dynamic=false` | |
| Add rule | `/ip/firewall/filter/add` | |
| Set/edit rule | `/ip/firewall/filter/set` | butuh `=.id=` |
| Remove rule | `/ip/firewall/filter/remove` | butuh `=.id=` |
| Enable rule | `/ip/firewall/filter/enable` | butuh `=.id=` |
| Disable rule | `/ip/firewall/filter/disable` | butuh `=.id=` |
| Move rule | `/ip/firewall/filter/move` | `=.id=` + `=destination=` |

### Properties saat `print`

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `chain` | string | R/W | `input`, `forward`, `output`, atau custom chain |
| `action` | string | R/W | Lihat tabel action di bawah |
| `disabled` | bool | R/W | `true`/`false` |
| `dynamic` | bool | R | Dibuat sistem (hotspot, dll.) |
| `invalid` | bool | R | Rule tidak valid |
| `comment` | string | R/W | Keterangan |
| `log` | bool | R/W | Apakah log packet yang match |
| `log-prefix` | string | R/W | Prefix untuk log |

### Matchers (digunakan saat `add` / `set`)

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `src-address` | IP/prefix | IP sumber, e.g. `192.168.1.0/24` |
| `dst-address` | IP/prefix | IP tujuan |
| `src-address-list` | string | Nama address-list sumber |
| `dst-address-list` | string | Nama address-list tujuan |
| `protocol` | string | `tcp`, `udp`, `icmp`, `ospf`, `gre`, dll. |
| `src-port` | string | Port sumber, e.g. `80,443` atau `1000-2000` |
| `dst-port` | string | Port tujuan |
| `port` | string | Src atau dst port (shorthand) |
| `in-interface` | string | Interface asal paket masuk |
| `out-interface` | string | Interface tujuan keluar |
| `in-interface-list` | string | Interface list masuk |
| `out-interface-list` | string | Interface list keluar |
| `connection-state` | string | `new`, `established`, `related`, `invalid`, `untracked` |
| `connection-nat-state` | string | `srcnat`, `dstnat`, `!srcnat`, `!dstnat` |
| `icmp-options` | string | ICMP type:code, e.g. `8:0` (echo request) |
| `tcp-flags` | string | `syn`, `ack`, `fin`, `rst`, `psh`, `urg`, `ece`, `cwr` |
| `limit` | string | Rate limit, e.g. `10,5:packet` |
| `src-mac-address` | string | MAC address sumber |
| `content` | string | String yang harus ada di packet payload |
| `layer7-protocol` | string | Layer 7 protocol matcher |
| `p2p` | string | Peer-to-peer traffic matcher |
| `hotspot` | string | HotSpot matcher: `auth`, `from-client`, `to-client`, dll. |
| `ttl` | string | Time To Live value |
| `dscp` | integer | DSCP value (0-63) |
| `packet-size` | string | Ukuran paket, e.g. `100-200` |
| `place-before` | string | Posisi insert sebelum rule tertentu |

### Actions

| Action | Deskripsi |
|--------|-----------|
| `accept` | Terima paket, stop evaluasi rule berikutnya |
| `drop` | Buang paket diam-diam (no reply) |
| `reject` | Buang + kirim ICMP error ke pengirim |
| `jump` | Lompat ke custom chain (`jump-target=nama-chain`) |
| `return` | Kembali dari custom chain |
| `log` | Log paket, lanjutkan ke rule berikutnya |
| `passthrough` | Tidak lakukan apa-apa, terus ke rule berikutnya (untuk counting) |
| `fasttrack-connection` | FastPath untuk koneksi TCP/UDP (IPv4 only) |
| `tarpit` | Tangkap & tahan koneksi TCP (IPv4 only) |

### Statistik (Read-Only) saat `print stats`

| Property | Deskripsi |
|----------|-----------|
| `bytes` | Jumlah bytes yang di-match rule ini |
| `packets` | Jumlah paket yang di-match |

### Contoh API Calls

```
# Print semua rules (termasuk dynamic dari hotspot)
/ip/firewall/filter/print

# Print hanya static rules
/ip/firewall/filter/print
?dynamic=false

# Add rule: drop src address
/ip/firewall/filter/add
=chain=forward
=action=drop
=src-address=192.168.1.50
=comment=Block IP nakal

# Add rule: accept established + related
/ip/firewall/filter/add
=chain=input
=action=accept
=connection-state=established,related
=comment=Allow established

# Add rule: drop port tertentu dari WAN
/ip/firewall/filter/add
=chain=input
=action=drop
=in-interface=ether1
=protocol=tcp
=dst-port=23,3389
=comment=Block Telnet dan RDP dari WAN

# Disable rule
/ip/firewall/filter/disable
=.id=*5

# Remove rule
/ip/firewall/filter/remove
=.id=*5

# Move rule ke posisi tertentu
/ip/firewall/filter/move
=.id=*5
=destination=*2

# Filter: cari rules di chain forward
/ip/firewall/filter/print
?chain=forward
```

---

## 4. Hotspot

**Path:** `/ip/hotspot`

### Sub-menu Hotspot

| Sub-menu | Path | Keterangan |
|----------|------|-----------|
| Hotspot server | `/ip/hotspot` | Server instances |
| Users | `/ip/hotspot/user` | User lokal |
| Active users | `/ip/hotspot/active` | Sesi aktif saat ini |
| Hosts | `/ip/hotspot/host` | Semua host yang terdeteksi |
| User profiles | `/ip/hotspot/user/profile` | Template/profil user |
| Server profiles | `/ip/hotspot/ip/profile` | Template server |
| Walled garden | `/ip/hotspot/walled-garden` | Bypass auth |
| Walled garden IP | `/ip/hotspot/walled-garden/ip` | Bypass by IP |
| IP bindings | `/ip/hotspot/ip-binding` | Binding IP statis |
| Cookie | `/ip/hotspot/cookie` | Cookie authentication |

---

### 4.1 Hotspot Server — `/ip/hotspot`

| Command | API Path |
|---------|----------|
| Print servers | `/ip/hotspot/print` |
| Add server | `/ip/hotspot/add` |
| Set server | `/ip/hotspot/set` |
| Remove server | `/ip/hotspot/remove` |
| Enable / Disable | `/ip/hotspot/enable` / `/ip/hotspot/disable` |

**Properties `print`:**

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Nama server hotspot |
| `interface` | string | R/W | Interface yang digunakan server |
| `address-pool` | string | R/W | Nama IP pool untuk client |
| `profile` | string | R/W | Nama server profile |
| `idle-timeout` | string | R/W | Timeout jika user idle, e.g. `5m` |
| `keepalive-timeout` | string | R/W | Keepalive timeout |
| `login-timeout` | string | R/W | Timeout halaman login |
| `addresses-per-mac` | integer | R/W | Maksimal IP per MAC address |
| `disabled` | bool | R/W | Status aktif/nonaktif |
| `invalid` | bool | R | Konfigurasi invalid |
| `https` | bool | R | Apakah HTTPS aktif |

---

### 4.2 Hotspot Users — `/ip/hotspot/user`

| Command | API Path |
|---------|----------|
| Print semua user | `/ip/hotspot/user/print` |
| Add user | `/ip/hotspot/user/add` |
| Set/edit user | `/ip/hotspot/user/set` |
| Remove user | `/ip/hotspot/user/remove` |
| Find user | `/ip/hotspot/user/find` |

**Properties `print`:**

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Username (**required** saat add) |
| `password` | string | W | Password (tidak tampil di print) |
| `server` | string | R/W | Nama hotspot server (atau `all`) |
| `profile` | string | R/W | Nama user profile |
| `mac-address` | string | R/W | MAC address binding (opsional) |
| `ip-address` | string | R/W | IP statis (opsional, biasanya dari pool) |
| `comment` | string | R/W | Keterangan |
| `disabled` | bool | R/W | Status user |
| `limit-bytes-in` | integer | R/W | Batas download total (bytes, 0=unlimited) |
| `limit-bytes-out` | integer | R/W | Batas upload total (bytes) |
| `limit-bytes-total` | integer | R/W | Batas total traffic |
| `limit-uptime` | string | R/W | Batas waktu total online, e.g. `1d` |
| `uptime` | string | R | Total waktu online yang sudah digunakan |
| `bytes-in` | integer | R | Total bytes download yang digunakan |
| `bytes-out` | integer | R | Total bytes upload yang digunakan |
| `bytes-total` | integer | R | Total traffic yang digunakan |

**Properties saat `add`:**

```
/ip/hotspot/user/add
=name=user1
=password=pass123
=server=hotspot1        # atau "all"
=profile=default
=mac-address=AA:BB:CC:DD:EE:FF   # opsional
=limit-uptime=1d                 # opsional
=limit-bytes-total=1073741824    # 1GB opsional
=comment=User test
```

---

### 4.3 Hotspot Active Users — `/ip/hotspot/active` (Read-Only)

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `.id` | string | Internal ID sesi |
| `server` | string | Nama server hotspot |
| `user` | string | Username |
| `domain` | string | Domain user |
| `address` | string | IP address client |
| `mac-address` | string | MAC address client |
| `login-by` | string | Metode login: `mac`, `http-pap`, `http-chap`, `https`, `trial` |
| `uptime` | string | Durasi sesi aktif |
| `idle-time` | string | Waktu idle sejak aktivitas terakhir |
| `session-time-left` | string | Sisa waktu sesi |
| `bytes-in` | integer | Bytes download sesi ini |
| `bytes-out` | integer | Bytes upload sesi ini |
| `packets-in` | integer | Paket download sesi ini |
| `packets-out` | integer | Paket upload sesi ini |
| `radius` | bool | Apakah autentikasi via RADIUS |
| `idle-timeout` | string | Timeout idle yang berlaku |
| `keepalive-timeout` | string | Keepalive timeout |
| `comment` | string | Keterangan |

**Commands:**

```
# Print active sessions
/ip/hotspot/active/print

# Hapus/kick sesi aktif (paksa logout user)
/ip/hotspot/active/remove
=.id=*A1
```

---

### 4.4 Hotspot User Profile — `/ip/hotspot/user/profile`

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Nama profil |
| `rate-limit` | string | R/W | Bandwidth limit format `Rx/Tx`, e.g. `2M/1M` |
| `shared-users` | integer | R/W | Maks user per akun (default: 1) |
| `idle-timeout` | string | R/W | Idle timeout override |
| `keepalive-timeout` | string | R/W | Keepalive timeout override |
| `session-timeout` | string | R/W | Batas durasi sesi |
| `incoming-filter` | string | R/W | Firewall chain untuk traffic masuk |
| `outgoing-filter` | string | R/W | Firewall chain untuk traffic keluar |
| `address-pool` | string | R/W | Override address pool |
| `open-status-page` | string | R/W | `http-login`, `always`, `never` |
| `parent-queue` | string | R/W | Parent queue untuk QoS |
| `add-mac-cookie` | bool | R/W | Aktifkan MAC cookie |
| `mac-cookie-timeout` | string | R/W | Durasi MAC cookie |

---

### 4.5 Hotspot Hosts — `/ip/hotspot/host` (Read-Only)

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `.id` | string | Internal ID |
| `address` | string | IP host yang terdeteksi |
| `mac-address` | string | MAC address |
| `server` | string | Server hotspot |
| `to-address` | string | IP setelah binding |
| `bridge-port` | string | Bridge port asal |
| `status` | string | `waiting`, `blocked`, `logged-in` |
| `authorized` | bool | Apakah sudah terautorisasi |
| `bypassed` | bool | Apakah di-bypass (walled garden) |
| `uptime` | string | Waktu host terdeteksi |
| `idle-time` | string | Waktu idle |
| `bytes-in` | integer | Total bytes download |
| `bytes-out` | integer | Total bytes upload |

---

## 5. Queue

### 5.1 Simple Queue — `/queue/simple`

| Command | API Path |
|---------|----------|
| Print semua queue | `/queue/simple/print` |
| Print dengan stats | `/queue/simple/print stats` |
| Add queue | `/queue/simple/add` |
| Set/edit queue | `/queue/simple/set` |
| Remove queue | `/queue/simple/remove` |
| Enable / Disable | `/queue/simple/enable` / `/queue/simple/disable` |
| Reset counter | `/queue/simple/reset-counters` |

**Properties `print`:**

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Nama queue (**required**) |
| `target` | string | R/W | IP target, e.g. `192.168.1.100/32` atau interface |
| `dst-address` | string | R/W | IP tujuan spesifik (opsional) |
| `parent` | string | R/W | Nama parent queue atau `none` |
| `priority` | integer | R/W | Prioritas 1-8 (1 = tertinggi) |
| `queue` | string | R/W | Tipe queue upload/download, e.g. `default/default` |
| `max-limit` | string | R/W | Batas maksimal `upload/download`, e.g. `2M/10M` |
| `limit-at` | string | R/W | Guaranteed rate `upload/download`, e.g. `512k/1M` |
| `burst-limit` | string | R/W | Burst max rate `upload/download` |
| `burst-threshold` | string | R/W | Threshold on/off burst |
| `burst-time` | string | R/W | Periode pengukuran burst, e.g. `8s/8s` |
| `packet-marks` | string | R/W | Comma-separated packet marks dari mangle |
| `time` | string | R/W | Jadwal aktif, e.g. `8h-20h,mon,tue,wed` |
| `comment` | string | R/W | Keterangan |
| `disabled` | bool | R/W | Status aktif |
| `invalid` | bool | R | Queue invalid |
| `dynamic` | bool | R | Dibuat sistem |

**Statistik `print stats` (Read-Only):**

| Property | Deskripsi |
|----------|-----------|
| `rate` | Rate upload/download saat ini (bytes/s) |
| `packet-rate` | Rate paket/detik |
| `bytes` | Total bytes yang diproses |
| `packets` | Total paket |
| `dropped` | Total paket yang di-drop |
| `queued-bytes` | Bytes yang sedang antri |
| `queued-packets` | Paket yang sedang antri |
| `borrows` | Paket yang dipinjam dari pool HTB |
| `lends` | Paket yang dipinjamkan ke child queue |

**Contoh `add`:**

```
# Limit user 2Mbps download / 1Mbps upload
/queue/simple/add
=name=user-192.168.1.100
=target=192.168.1.100/32
=max-limit=1M/2M         # upload/download
=limit-at=256k/512k      # guaranteed
=comment=Limit client lantai 1

# Queue dengan burst
/queue/simple/add
=name=burst-user
=target=192.168.1.105/32
=max-limit=1M/5M
=burst-limit=2M/10M
=burst-threshold=512k/2M
=burst-time=8s/8s

# Queue parent (HTB)
/queue/simple/add
=name=dept-IT
=target=192.168.10.0/24
=max-limit=10M/50M

# Queue child dari dept-IT
/queue/simple/add
=name=server-IT
=target=192.168.10.100/32
=parent=dept-IT
=max-limit=5M/20M
=priority=1
```

---

### 5.2 Queue Tree — `/queue/tree`

| Command | API Path |
|---------|----------|
| Print | `/queue/tree/print` |
| Add | `/queue/tree/add` |
| Set | `/queue/tree/set` |
| Remove | `/queue/tree/remove` |
| Stats | `/queue/tree/print stats` |

**Properties `print`:**

| Property | Tipe | R/W | Deskripsi |
|----------|------|-----|-----------|
| `.id` | string | R | Internal ID |
| `name` | string | R/W | Nama queue (**required**) |
| `parent` | string | R/W | Interface atau nama parent queue (`global`, `ether1`, dll.) |
| `packet-mark` | string | R/W | Packet mark dari `/ip/firewall/mangle` |
| `priority` | integer | R/W | Prioritas 1-8 |
| `max-limit` | string | R/W | Batas maksimal bandwidth |
| `limit-at` | string | R/W | Guaranteed rate |
| `burst-limit` | string | R/W | Burst max rate |
| `burst-threshold` | string | R/W | Burst threshold |
| `burst-time` | string | R/W | Burst averaging period |
| `queue` | string | R/W | Tipe queue |
| `comment` | string | R/W | Keterangan |
| `disabled` | bool | R/W | Status aktif |

**Contoh `add`:**

```
# Limit traffic YouTube berdasarkan packet mark dari mangle
/queue/tree/add
=name=limit-youtube
=parent=global
=packet-mark=pmark-youtube
=max-limit=2M
```

---

### 5.3 Queue Types — `/queue/type` (mostly Read-Only)

| Property | Deskripsi |
|----------|-----------|
| `name` | Nama tipe queue |
| `kind` | Algoritma: `pfifo`, `bfifo`, `sfq`, `pcq`, `red`, `mq-pfifo`, `none`, `codel`, `fq-codel`, `cake` |
| `pfifo-limit` | Limit antrian untuk pfifo (packets) |
| `bfifo-limit` | Limit antrian untuk bfifo (bytes) |
| `sfq-perturb` | Perturb interval SFQ |
| `sfq-allot` | Allotment SFQ (bytes per round) |
| `pcq-rate` | Max rate per sub-stream PCQ |
| `pcq-limit` | Queue size per sub-stream PCQ |
| `pcq-classifier` | `src-address`, `dst-address`, `src-port`, `dst-port` |
| `pcq-total-limit` | Total max queue semua sub-stream |
| `pcq-burst-rate` | Burst rate per sub-stream |
| `cake-bandwidth` | Bandwidth untuk CAKE shaper |

---

## 6. System

### 6.1 System Resource — `/system/resource`

| Command | API Path |
|---------|----------|
| Print resource | `/system/resource/print` |
| Monitor CPU (stream) | `/system/resource/cpu/print` |

**Properties — `/system/resource/print` (semua Read-Only):**

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `uptime` | string | Waktu sejak boot, e.g. `3d2h45m10s` |
| `version` | string | Versi RouterOS, e.g. `7.14.3` |
| `build-time` | string | Waktu build firmware |
| `factory-software` | string | Versi factory firmware |
| `free-memory` | integer | RAM bebas (bytes) |
| `total-memory` | integer | Total RAM (bytes) |
| `cpu` | string | Nama/model CPU |
| `cpu-count` | integer | Jumlah core CPU |
| `cpu-frequency` | integer | Frekuensi CPU (MHz) |
| `cpu-load` | integer | Utilisasi CPU (%) |
| `free-hdd-space` | integer | Storage bebas (bytes) |
| `total-hdd-space` | integer | Total storage (bytes) |
| `write-sect-since-reboot` | integer | Sektor ditulis sejak boot |
| `write-sect-total` | integer | Total sektor ditulis |
| `bad-blocks` | integer | Jumlah bad blocks |
| `architecture-name` | string | Arsitektur: `arm`, `x86`, `tile`, dll. |
| `board-name` | string | Nama hardware board |
| `platform` | string | Platform: `MikroTik`, dll. |

---

### 6.2 System Log — `/log`

| Command | API Path |
|---------|----------|
| Print log | `/log/print` |
| Listen log realtime | `/log/listen` (stream) |

**Properties — `/log/print`:**

| Property | Tipe | Deskripsi |
|----------|------|-----------|
| `.id` | string | Internal ID log entry |
| `time` | string | Timestamp log, e.g. `oct/28 14:32:01` |
| `topics` | string | Topik/kategori log (bisa multiple): `info`, `warning`, `error`, `firewall`, `dhcp`, `hotspot`, `system`, `bridge`, dll. |
| `message` | string | Isi pesan log |

**Filter log berdasarkan topic:**

```
# Print semua log
/log/print

# Filter log firewall saja
/log/print
?topics=firewall

# Listen log realtime (stream semua)
/log/listen

# Listen log firewall realtime
/log/listen
?topics=firewall
```

---

### 6.3 System Identity — `/system/identity`

```
# Print hostname
/system/identity/print
# Returns: =name=MikroTik

# Set hostname
/system/identity/set
=name=Router-Kantor
```

---

### 6.4 System Reboot — `/system/reboot`

```
# Reboot router (DESTRUCTIVE — tidak ada konfirmasi)
/system/reboot
```

> ⚠️ Command ini langsung dieksekusi. Implementasi tool MCP **wajib** menambahkan konfirmasi eksplisit sebelum memanggil command ini.

---

### 6.5 System Routerboard — `/system/routerboard`

| Property | Deskripsi |
|----------|-----------|
| `routerboard` | `true`/`false` apakah ini MikroTik hardware |
| `board-name` | Nama board, e.g. `hAP ac²` |
| `model` | Model number |
| `serial-number` | Serial number hardware |
| `firmware-type` | Tipe firmware |
| `factory-firmware` | Versi firmware factory |
| `current-firmware` | Versi firmware saat ini |
| `upgrade-firmware` | Versi firmware yang tersedia untuk upgrade |

---

### 6.6 System Health — `/system/health`

> Tersedia di hardware yang mendukung sensor (temperature, voltage, fan).

| Property | Deskripsi |
|----------|-----------|
| `name` | Nama sensor |
| `type` | Tipe: `temperature`, `voltage`, `current`, `fan-speed` |
| `value` | Nilai sensor |

---

### 6.7 System History — `/system/history`

```
# Melihat history perubahan konfigurasi
/system/history/print
```

---

## 7. API Khusus — Listen / Subscribe

Beberapa endpoint mendukung mode **listen** yang mengalirkan data secara realtime.

| Endpoint | Deskripsi |
|----------|-----------|
| `/interface/listen` | Perubahan status interface |
| `/interface/monitor-traffic` | Traffic stats per detik |
| `/user/active/listen` | User yang login/logout |
| `/ip/hotspot/active/listen` | Hotspot active session changes |
| `/ip/hotspot/host/listen` | Perubahan host hotspot |
| `/log/listen` | Stream log masuk realtime |
| `/ip/firewall/connection/listen` | Perubahan connection table |

**Cara menggunakan listen di go-routeros v3:**

```go
// Start listen
listen, err := client.ListenArgs(routeros.Args{
    "/interface/monitor-traffic",
    "=interface=ether1",
})

// Baca stream
for sentence := range listen.Chan() {
    // proses sentence
}

// Cancel / stop listen
listen.Cancel()
```

**Format reply listen:**
- `!re` — data baru atau update
- `=.dead=yes` — item dihapus/hilang (khusus untuk listen menu list)
- `!done` — stream selesai (setelah Cancel)
- `!trap` + `=category=2` + `=message=interrupted` — listen diinterrupt oleh Cancel

---

## 8. Query / Filter pada Print

Semua print command mendukung query words (`?`) untuk filter data.

```
# Cari berdasarkan nilai exact
?name=ether1

# Cari yang nilai property-nya ada (tidak kosong)
?comment

# Cari yang nilai property-nya kosong
?-comment

# Lebih besar dari
?>max-limit=1000000

# Lebih kecil dari
?<cpu-load=50

# Kombinasi dengan OR
/ip/firewall/filter/print
?chain=input
?chain=forward
?#|           ← OR logic

# Kombinasi dengan AND (default)
/queue/simple/print
?disabled=false
?dynamic=false
```

---

## 9. Penggunaan `.proplist`

Untuk performa lebih baik, gunakan `.proplist` agar API hanya mengembalikan field yang diperlukan.

```
/interface/print
=.proplist=.id,name,type,running,disabled

/queue/simple/print
=.proplist=.id,name,target,max-limit,disabled

/ip/hotspot/active/print
=.proplist=user,address,mac-address,uptime,bytes-in,bytes-out
```

> Tanpa `.proplist`, semua property dikembalikan termasuk yang lambat (seperti file contents, performance counters).

---

## 10. Error Handling

| `!trap` Category | Keterangan |
|-----------------|-----------|
| `0` | Missing item or command |
| `1` | Argument value failure (value tidak valid) |
| `2` | Execution interrupted (Cancel) |
| `3` | Scripting failure |
| `4` | General failure |
| `5` | API related failure |
| `6` | TTY related failure |
| `7` | Return dari :return command |

**Contoh error reply:**

```
!trap
=category=1
=message=input does not match any value of interface
```

---

*Referensi ini disusun berdasarkan dokumentasi resmi MikroTik RouterOS API, Console, dan module-specific documentation.*