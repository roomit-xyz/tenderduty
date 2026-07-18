# TenderDuty v2 Roomit

A multi-chain validator monitoring dashboard. Fork of
[blockpane/tenderduty](https://github.com/blockpane/tenderduty) v2 with:

- **Multi-chain** — Tendermint / Cosmos SDK, AtomOne (CometBFT fork), Gno.land (TM2 polling)
- **`blockchain_kind` flag** — per-chain signing parser (no manual valcons conversion)
- **Native FHS layout** — no Caddy / reverse proxy required for internal use
- **Single static-file binary** with auto-detected dashboard path

![Dashboard Preview](assets/dashboard.jpg)

---

## Quick start (Docker)

```bash
git clone https://github.com/roomit-xyz/tenderduty.git
cd tenderduty
mkdir -p conf var/www/td2-v2
cp example-config.yml conf/config.yml
# edit conf/config.yml with your chains / notifiers
docker compose up -d --build
```

Dashboard: <http://localhost:8888>
Prometheus exporter: <http://localhost:28686/metrics>

## Quick start (native install)

```bash
# Build
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/tenderduty .

# FHS layout (run as root or sudo)
install -d bin conf var/www/td2-v2 var/log
cp bin/tenderduty bin/
cp td2/static/index.html var/www/td2-v2/
cp example-config.yml conf/config.yml
# edit conf/config.yml

# Run
CONFIG=$(pwd)/conf/config.yml \
  TENDERDUTY_STATIC_DIR=$(pwd)/var/www/td2-v2 \
  nohup ./bin/tenderduty > var/log/tenderduty.log 2>&1 &
```

---

## FHS layout

```
.
├── bin/
│   └── tenderduty              # static binary, single file
├── conf/
│   └── config.yml              # chain / notifier config
├── var/
│   ├── log/                    # runtime logs
│   └── www/
│       └── td2-v2/             # dashboard static files (index.html)
└── chains.d/                   # optional per-chain overrides
```

The binary auto-detects the dashboard path in this order:

1. `TENDERDUTY_STATIC_DIR` env var (highest priority)
2. `<executable>/../var/www/td2-v2`
3. `<cwd>/var/www/td2-v2`
4. `./td2/static` (legacy fallback)

---

## Config

Each chain entry supports a `blockchain_kind` field. This selects the
correct signing/encoding parser and prevents you from manually converting
valcons keys.

```yaml
chains:
  "AtomOne Testnet":
    chain_id: atomone-testnet-1
    blockchain_kind: atomone
    valoper_address: "atonevaloper1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    nodes:
      - url: http://10.35.4.199:16711
        alert_if_down: yes

  "GNO.LAND Testnet-13":
    chain_id: test-13
    blockchain_kind: gnoland
    chain_type: gno
    gno_valopers_realm: gno.land/r/gnops/valopers
    valoper_address: g1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    nodes:
      - url: http://10.35.4.196:26657
        alert_if_down: yes

  "Empe Testnet":
    chain_id: empe-testnet-2
    blockchain_kind: tendermint
    valoper_address: empevaloper1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    nodes:
      - url: http://10.35.4.197:16710
        alert_if_down: yes
```

### Supported `blockchain_kind` values

| Value        | Use for                                              |
|--------------|------------------------------------------------------|
| `tendermint` | Tendermint / Cosmos SDK chains (default if omitted)  |
| `atomone`    | AtomOne (CometBFT fork, custom signing parser)      |
| `gnoland`    | Gno.land (TM2 polling; also set `chain_type: gno`)  |

### Notifiers

Set `enabled: yes` and configure the per-channel fields in `config.yml`:

- **Telegram** — `api_key` from @BotFather, `channel` is the chat ID
- **Gotify** — server URL + application token
- **Healthcheck** — dead-man's-switch URL, pinged every `ping_rate` seconds
- **PagerDuty / Discord / Slack** — see `example-config.yml`

Per-chain overrides live under each chain's `alerts:` block; if a field is
blank, the global setting is used.

---

## Building from source

The build is reproducible from `Dockerfile`:

```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o tenderduty ./main.go
```

The same flags are used in CI to produce a single static binary.

## License

See [LICENSE](LICENSE).
