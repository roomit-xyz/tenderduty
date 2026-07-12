# TenderDuty v2 — Roomit Validator Monitor

Multi-chain validator monitoring dashboard dengan WebSocket live updates, block signature tracking, dan alerting system.

## Supported Chains

| Chain | Type | Parser | Status |
|---|---|---|---|
| AtomOne Testnet | CometBFT v0.38 | Dual (Precommits + Signatures) | ✅ |
| Dora Testnet | Cosmos SDK | Standard Tendermint | ✅ |
| Empe Testnet | Cosmos SDK | Standard Tendermint | ✅ |
| GNO.LAND Testnet-13 | TM2 | Polling (RPC block) | ✅ |

## Features

- **Live Dashboard** — WebSocket‑powered UI di port `8888` dengan animasi blok, legend warna, dan log stream
- **3‑Path Signing Parser** — AtomOne (CometBFT), GNO (TM2 polling), General Tendermint
- **Alerting** — Telegram, Gotify, PagerDuty, Discord
- **Prometheus Metrics** — `/metrics` endpoint
- **Healthcheck Ping** — Push ke healthchecks.io / custom
- **Auto‑failover** — Jika node RPC down, auto switch ke backup node

## Quick Start

```bash
# Build
go build -o tenderduty-manual ./main.go

# Run
./tenderduty-manual -f config.yml
```

Dashboard: `http://localhost:8888`

## Configuration

Edit `config.yml`:

```yaml
enable_dashboard: true
hide_logs: false
listen_port: 8888

chains:
  "AtomOne Testnet":
    chain_id: atomone-testnet-1
    valoper_address: atonevaloper1...
    alerts:
      stalled_enabled: yes
      consecutive_enabled: yes
    nodes:
      - url: http://10.35.4.199:16711
```

## Directory Structure

```
td2/
├── dashboard/
│   ├── server.go    # HTTP + WebSocket server (reads from td2/static/)
│   └── types.go     # Data structures
├── static/
│   └── index.html   # Dashboard UI (served directly, no embed)
├── ws.go            # WebSocket block subscription
├── ws_poll.go       # RPC polling (GNO/AtomOne TM2 chains)
├── run.go           # Main orchestrator
├── alert.go         # Alerting logic
├── prometheus.go    # Metrics
└── ...
```

## UI

Dashboard v2 menampilkan:
- **Stats bar**: Total chains, healthy/warning/critical count
- **Legend**: Signed (🟢 hijau), Proposed (🔵 biru), Missed (🔴 merah), None (⚪ abu‑abu)
- **Chain cards**: Height, missed blocks, signing history (animasi glow)
- **Live Logs**: Sidebar kanan, polling `/logs` setiap 3 detik

## Maintenance

UI update tanpa rebuild:
```bash
# Edit langsung
vim td2/static/index.html

# Restart service
sudo pkill -f tenderduty-manual
nohup ./tenderduty-manual -f config.yml > /var/log/tenderduty.log 2>&1 &
```

## Security

- Dashboard port (`8888`) hanya terbuka di jaringan internal (`10.35.4.0/24`)
- Log messages di‑redact jika `hide_logs: true`
- Tidak ada data sensitif di UI statis
