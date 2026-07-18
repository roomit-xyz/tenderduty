# Config Reference — TenderDuty v2 Roomit

This document covers every configuration option in `config.yml`.

**Quick start:** Run `tenderduty -example-config > config.yml` to generate a commented stub.
See [example-config.yml](../example-config.yml) for a complete annotated example.

---

## Table of Contents

* [General Settings](#general-settings)
* [PagerDuty Settings](#pagerduty-settings)
* [Discord Settings](#discord-settings)
* [Telegram Settings](#telegram-settings)
* [Slack Settings](#slack-settings)
* [Gotify Settings](#gotify-settings)
* [Healthcheck Settings](#healthcheck-settings)
* [Chain Settings](#chain-settings)
* [Chain Alerting](#chain-alerting)
* [Node Settings](#node-settings)

---

## General Settings

| Setting                    | Description |
|----------------------------|-------------|
| `enable_dashboard`         | Enable / disable the web dashboard. |
| `listen_port`              | TCP port the dashboard listens on. |
| `hide_logs`                | Hide log feed & obscure node details (useful for public dashboards). |
| `node_down_alert_minutes`  | Minutes before alerting a node is down. |
| `prometheus_enabled`       | Enable Prometheus exporter. See [prometheus.md](prometheus.md). |
| `prometheus_listen_port`   | Port for the Prometheus exporter. |

---

## PagerDuty Settings

| Setting                     | Description |
|-----------------------------|-------------|
| `pagerduty.enabled`          | Master toggle; overrides per-chain. |
| `pagerduty.api_key`          | PagerDuty API key. See [pagerduty.md](pagerduty.md). |
| `pagerduty.default_severity` | (future) Default escalation severity. |

---

## Discord Settings

| Setting              | Description |
|----------------------|-------------|
| `discord.enabled`    | Master toggle; overrides per-chain. |
| `discord.webhook`    | Discord webhook URL. See [discord.md](discord.md). |

---

## Telegram Settings

| Setting              | Description |
|----------------------|-------------|
| `telegram.enabled`   | Master toggle; overrides per-chain. |
| `telegram.api_key`   | Bot API key from @BotFather. |
| `telegram.channel`   | Chat ID for alerts. See [telegram.md](telegram.md). |

---

## Slack Settings

| Setting           | Description |
|-------------------|-------------|
| `slack.enabled`   | Master toggle; overrides per-chain. |
| `slack.webhook`   | Slack webhook URL. |

---

## Gotify Settings

| Setting              | Description |
|----------------------|-------------|
| `gotify.enabled`     | Master toggle; overrides per-chain. |
| `gotify.server`      | Gotify server URL (include protocol & port). |
| `gotify.token`       | Application token (create in Gotify UI). |
| `gotify.priority`    | Message priority 0-10. Default: 5. |

---

## Healthcheck Settings

*(Dead-man's-switch — pings a healthcheck endpoint at `ping_rate` seconds.)*

| Setting              | Description |
|----------------------|-------------|
| `healthcheck.enabled` | Enable / disable ping. |
| `healthcheck.ping_url` | URL to ping (e.g. healthchecks.io). |
| `healthcheck.ping_rate` | Interval in seconds. |

---

## Chain Settings

Each chain entry uses `blockchain_kind` to select the correct signing parser.
No more manually converting valcons addresses.

| Setting                     | Required | Description |
|-----------------------------|----------|-------------|
| `chain.<name>`              | yes      | Friendly display name (quote if it has spaces). |
| `chain.<name>.chain_id`     | yes      | Chain ID, validated against RPC. |
| `chain.<name>.blockchain_kind` | no    | One of `tendermint` (default), `atomone`, `gnoland`. |
| `chain.<name>.chain_type`   | no       | Set to `gno` for Gno.land chains. |
| `chain.<name>.gno_valopers_realm` | no | Gno.land realm path for valoper query. |
| `chain.<name>.valoper_address` | yes   | Validator operator address (bech32). |
| `chain.<name>.public_fallback` | no    | Use public API endpoints if all RPC nodes fail. |

### `blockchain_kind` reference

| Value        | Use for                                                      |
|--------------|--------------------------------------------------------------|
| `tendermint` | Tendermint / Cosmos SDK chains (default if omitted).         |
| `atomone`    | AtomOne (CometBFT fork, custom signing parser).              |
| `gnoland`    | Gno.land (TM2 polling; also set `chain_type: gno`).          |

---

## Chain Alerting

*All fields live under `chain.<name>.alerts:`*

| Setting                       | Description |
|-------------------------------|-------------|
| `stalled_enabled`             | Alert if chain stops producing blocks. |
| `stalled_minutes`             | Minutes before stalled alert fires. |
| `consecutive_enabled`         | Alert on consecutive missed blocks. |
| `consecutive_missed`          | Number of missed blocks to trigger. |
| `consecutive_priority`        | (future) PagerDuty severity. |
| `percentage_enabled`          | Alert if missed-block percentage exceeds threshold. |
| `percentage_missed`           | Percentage threshold. |
| `percentage_priority`         | (future) PagerDuty severity. |
| `alert_if_inactive`           | Alert if validator is jailed / tombstoned / unbonding. |
| `alert_if_no_servers`         | Alert if all RPC nodes are down. |

### Per-chain notifier overrides

Each notifier block supports `enabled`, and channel-specific fields.
Blank values fall back to the global settings.

```yaml
alerts:
  telegram:
    enabled: yes
    api_key: ""
    channel: ""
  gotify:
    enabled: yes
    server: ""
    token: ""
    priority: 0
  discord:
    enabled: yes
    webhook: ""
  slack:
    enabled: yes
    webhook: ""
  pagerduty:
    enabled: yes
    api_key: ""
```

---

## Node Settings

*Array of RPC servers, tried in order.*

| Setting                  | Description |
|--------------------------|-------------|
| `nodes[].url`            | RPC endpoint `http(s)://host:port`. |
| `nodes[].alert_if_down`  | Trigger alert when this specific node goes down. |
