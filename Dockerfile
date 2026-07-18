# TenderDuty v2 Roomit — Multi-chain validator monitor
# Supports: AtomOne (CometBFT), GNO (TM2 polling), General Tendermint
# FHS: bin/ conf/ var/www/td2-v2/

# Stage 1: build
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git gcc libc-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o tenderduty ./main.go

# Stage 2: runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates curl tzdata && \
    addgroup -g 26657 -S tenderduty && \
    adduser -u 26657 -S -G tenderduty -h /app tenderduty

# FHS layout
RUN mkdir -p /app/bin /app/conf /app/var/www/td2-v2

COPY --from=builder /build/tenderduty /app/bin/tenderduty
COPY --from=builder /build/td2/static/index.html /app/var/www/td2-v2/
COPY --from=builder /build/example-config.yml /app/conf/config.yml.example

# Symlink legacy path for backward compat
RUN ln -sf /app/var/www/td2-v2 /app/td2/static

RUN chown -R tenderduty:tenderduty /app

USER tenderduty
WORKDIR /app
EXPOSE 8888 28686
ENTRYPOINT ["/app/bin/tenderduty"]
