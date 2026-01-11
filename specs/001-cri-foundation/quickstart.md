# Quickstart: MasCRI Foundation (Feature 001)

## Prerequisites

- **Go 1.22+** installed.
- **crictl** installed (`brew install crictl` or download binary).

## Building

```bash
# From project root
go mod tidy
go build -o bin/mascri cmd/mascri/main.go
```

## Running the Server

```bash
# Start server in foreground
./bin/mascri --socket /tmp/mascri.sock --debug
```

## Verifying (Client)

In a separate terminal:

```bash
# Configure crictl to talk to our socket
export CONTAINER_RUNTIME_ENDPOINT=unix:///tmp/mascri.sock
export IMAGE_SERVICE_ENDPOINT=unix:///tmp/mascri.sock

# 1. Check Version (P1)
crictl info
# Expected: Info about MasCRI 0.1.0

# 2. Test Pod Sandbox (P2)
crictl runp --runtime-config-file=./fixtures/pod-config.yaml
# Expected: "Sandbox ID: mock-sandbox-id"
```
