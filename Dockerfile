# TenderDuty v2 Roomit — Multi-chain validator monitor
# Supports: AtomOne (CometBFT), GNO (TM2 polling), General Tendermint

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
RUN apk add --no-cache ca-certificates curl && \
    addgroup -g 26657 -S tenderduty && \
    adduser -u 26657 -S -G tenderduty -h /app tenderduty

COPY --from=builder /build/tenderduty /bin/tenderduty
COPY --from=builder /build/td2/static/ /app/td2/static/
COPY --from=builder /build/example-config.yml /app/

# Dashboard reads from /app/td2/static/ (os.DirFS)
USER tenderduty
WORKDIR /app
EXPOSE 8888 28686
ENTRYPOINT ["/bin/tenderduty"]
DEOF
