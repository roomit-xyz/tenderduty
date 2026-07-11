# TenderDuty 🛰️

> Fork dari [blockpane/tenderduty](https://github.com/blockpane/tenderduty) v2 dengan tambahan **dukungan Gno.land (TM2)** dan **dashboard mobile responsive**.

Validator monitoring dashboard dengan alert multi-channel (Telegram, Discord, Slack, PagerDuty, Gotify) untuk chain Cosmos SDK **dan Gno.land (gno.land/test13, dll)**.

---

## ✨ Fitur

### Original TenderDuty
- ✅ Real-time monitoring validator (signed / proposed / missed blocks)
- ✅ Alert: stalled, consecutive missed, percentage missed, inactive
- ✅ Multi-channel: Telegram, Discord, Slack, PagerDuty, Gotify
- ✅ Dashboard web (Canvas block visualization)
- ✅ Prometheus metrics exporter
- ✅ Healthcheck ping (Uptime Kuma compatible)
- ✅ Encrypted config (Argon2id + AES-256-CBC)

### 🆕 Roomit Fork (v2.1)
- ✅ **Gno.land (TM2) provider** — monitoring chain Gno.land via HTTP polling
- ✅ **Gno-specific:** valoper address resolve via `vm/qeval` abci_query
- ✅ **Gno-specific:** consensus address lookup via `gno_valopers_realm`
- ✅ **Graceful handling:** VM panic "valoper does not exist" tidak bikin node down
- ✅ **Responsive dashboard** — mobile-friendly (table hidden cols, scrollable canvas)
- ✅ **Light/Dark theme** (Roomit green)
- ✅ **Auto-pruning** blocks window (configurable)
- ✅ **Multi-chain support** — Cosmos + Gno side-by-side

---

## 🚀 Quick Start

### 1. Build

```bash
# Install Go 1.19+ (jika belum ada)
wget https://go.dev/dl/go1.19.13.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.19.13.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build binary
git clone <this-repo> tenderduty
cd tenderduty
go build -o tenderduty main.go

# Atau via Docker
docker compose up -d
```

### 2. Buat Config

Lihat `example-config.yml` untuk template lengkap. Minimal:

```yaml
enable_dashboard: yes
listen_port: 8888

chains:
  my-chain:
    chain_id: my-chain-1
    valoper_address: myvaloper1xxxx
    nodes:
      - url: http://localhost:26657
        alert_if_down: yes
    alerts:
      stalled_enabled: yes
      consecutive_enabled: yes
      percentage_enabled: yes
```

### 3. Run

```bash
./tenderduty -f config.yml
```

Dashboard: `http://localhost:8888`
Prometheus: `http://localhost:28686/metrics`

---

## 🌍 Gno.land Support

TenderDuty mendukung monitoring **chain Gno.land (TM2)** via `chain_type: gno`.

### Konfigurasi

```yaml
chains:
  gno-test13:
    chain_id: test-13
    chain_type: gno                                    # ⬅ WAJIB
    gno_valopers_realm: gno.land/r/gnops/valopers      # ⬅ custom realm opsional
    valoper_address: g1zyk4gkw68lzx9yfgcda2ur36yy6dfdtyzglvsc
    nodes:
      - url: http://10.35.4.196:26657
        alert_if_down: yes
    alerts:
      stalled_minutes: 10
      consecutive_missed: 5
      percentage_missed: 10
```

### Field Wajib untuk Gno
| Field | Wajib | Default | Keterangan |
|---|---|---|---|
| `chain_type` | ✅ ya | `cosmos` | Set `gno` untuk Gno.land |
| `gno_valopers_realm` | ❌ | `gno.land/r/gnops/valopers` | Path realm di gno.land |
| `valoper_address` | ✅ ya | - | Validator address (g1...) |
| `chain_id` | ✅ ya | - | Network ID (e.g. `test-13`) |

### Bagaimana Cara Kerjanya

```
TenderDuty (Gno)                 Gno.land Node
    │                                 │
    ├── GET /status ────────────────► │ (health, height, catching_up)
    │                                 │
    ├── GET /validators ────────────► │ (active set, consensus address)
    │                                 │
    ├── abci_query vm/qeval ──────►  │ (resolve g1xxx → moniker via realm)
    │                                 │
    └── GET /block (poll 5s) ──────► │ (block signing status)
                                      │
```

Berbeda dengan Cosmos (WebSocket subscription), Gno polling `/block` tiap 5 detik.

### Scaling untuk Production

| Use case | Setting |
|---|---|
| Testnet Gno.land (test13) | `chain_type: gno`, `chain_id: test-13` |
| Mainnet Gno.land | `chain_type: gno`, `chain_id: gno.land` |
| Custom valopers realm | `gno_valopers_realm: your.realm/path` |
| Multiple Gno nodes | Tambah entry di `nodes:` list |

---

## 📱 Responsive Dashboard

Dashboard otomatis adapt:
- **Desktop** (>1024px): full table, semua kolom
- **Tablet** (768-1024px): font lebih kecil, padding ketat
- **Mobile** (≤768px): kolom Uptime disembunyikan, scroll horizontal
- **Phone** (≤480px): kolom Uptime + Nodes disembunyikan, font compact

---

## 🛠️ Tech Stack

- **Go 1.19+**
- **Cosmos SDK v0.45.11** (Tendermint v0.34.24)
- **Gno TM2** (HTTP /status, /validators, abci_query)
- **gorilla/websocket**
- **prometheus/client_golang**
- **UIkit 3.16** + custom Roomit theme

---

## 📜 License

Original: AGPL-3.0 (blockpane/tenderduty)
Fork: Same license.

---

## 🙏 Credits

- Original: [blockpane/tenderduty](https://github.com/blockpane/tenderduty) by [@blockpane](https://github.com/blockpane)
- Gno.land integration: [@roomit-xyz](https://github.com/roomit-xyz)
- Maintainer: PT Roomit Trimiko Digital
