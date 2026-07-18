# Installing TenderDuty v2 Roomit

* [Docker Container](#docker-container)
* [Docker Compose](#docker-compose)
* [Build From Source](#building-from-source)
* [Native FHS Install](#native-fhs-install)
* [Run as a systemd service](#run-as-a-systemd-service)

## Docker Container

```shell
mkdir tenderduty && cd tenderduty
docker run --rm ghcr.io/blockpane/tenderduty:latest -example-config >config.yml
# edit config.yml
docker run -d --name tenderduty -p "8888:8888" -p "28686:28686" --restart unless-stopped \
  -v $(pwd)/config.yml:/app/conf/config.yml:ro \
  -e CONFIG=/app/conf/config.yml \
  -e TENDERDUTY_STATIC_DIR=/app/var/www/td2-v2 \
  ghcr.io/blockpane/tenderduty:latest
docker logs -f --tail 20 tenderduty
```

## Docker Compose

```shell
git clone https://github.com/roomit-xyz/tenderduty
cd tenderduty
mkdir -p conf
cp example-config.yml conf/config.yml
# edit conf/config.yml
docker compose up -d --build
docker compose logs -f --tail 20
```

See [docker-compose.yml](../docker-compose.yml) for the full FHS layout.

## Building from source

Requires Go 1.18+ (1.22 recommended).

```shell
git clone https://github.com/roomit-xyz/tenderduty
cd tenderduty
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o tenderduty ./main.go
```

## Native FHS Install

Recommended layout:

```
.
├── bin/
│   └── tenderduty
├── conf/
│   └── config.yml
├── var/
│   ├── log/
│   └── www/td2-v2/
│       └── index.html
└── chains.d/          # optional per-chain overrides
```

Setup:

```shell
# Build
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o tenderduty ./main.go

# Create FHS layout
install -d bin conf var/log var/www/td2-v2 chains.d
install -m 755 tenderduty bin/
install -m 644 td2/static/index.html var/www/td2-v2/
install -m 600 example-config.yml conf/config.yml
# edit conf/config.yml

# Run
export CONFIG=$(pwd)/conf/config.yml
export TENDERDUTY_STATIC_DIR=$(pwd)/var/www/td2-v2
nohup ./bin/tenderduty > var/log/tenderduty.log 2>&1 &
```

## Run as a systemd service

```shell
# Create the system user
sudo addgroup --system tenderduty
sudo adduser --ingroup tenderduty --system --home /opt/tenderduty tenderduty

# Install the FHS layout
sudo install -d /opt/tenderduty/bin /opt/tenderduty/conf /opt/tenderduty/var/log /opt/tenderduty/var/www/td2-v2
sudo install -m 755 bin/tenderduty /opt/tenderduty/bin/
sudo install -m 644 var/www/td2-v2/index.html /opt/tenderduty/var/www/td2-v2/
sudo install -m 600 conf/config.yml /opt/tenderduty/conf/

sudo tee /etc/systemd/system/tenderduty.service <<EOF
[Unit]
Description=TenderDuty v2
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tenderduty
Group=tenderduty
WorkingDirectory=/opt/tenderduty
Environment=CONFIG=/opt/tenderduty/conf/config.yml
Environment=TENDERDUTY_STATIC_DIR=/opt/tenderduty/var/www/td2-v2
ExecStart=/opt/tenderduty/bin/tenderduty
Restart=always
RestartSec=5
TimeoutSec=180
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now tenderduty
sudo journalctl -fu tenderduty
```
