# RoomDuty (TenderDuty Fork)
A multi-chain validator dashboard.

## Features
- Multi-chain monitoring: AtomOne, Gno.land, Tendermint/Cosmos
- Blockchain-kind flag based signing parser
- Row-based responsive layout

## Config

Add `blockchain_kind` to each chain in `config.json`:

```json
{
  "chains": [
    {
      "name": "AtomOne",
      "rpc": "http://localhost:16711",
      "blockchain_kind": "atomone"
    },
    {
      "name": "Gno",
      "rpc": "http://localhost:26657",
      "blockchain_kind": "gno"
    },
    {
      "name": "Cosmos",
      "rpc": "http://localhost:26657",
      "blockchain_kind": "cosmos"
    }
  ]
}
```

### Supported blockchain_kind values
- `atomone` — AtomOne chain (custom signing parser)
- `gno` — Gno.land chain (custom signing parser)
- `cosmos` — Cosmos SDK (default)

## Usage

Run validator monitor:
```bash
npm start
```

Monitor will connect to all configured chains and display real-time status.
