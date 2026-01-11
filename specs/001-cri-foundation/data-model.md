# Data Model: Basic MasCRI Foundation

## Internal Entities

### ServerConfig

Configuration passed to the server on startup.

| Field      | Type   | Description                                               |
| ---------- | ------ | --------------------------------------------------------- |
| SocketPath | string | Path to the Unix Domain Socket (e.g., `/tmp/mascri.sock`) |
| Debug      | bool   | Enable verbose debug logging                              |

### Server Implementation

`pkg/server/server.go`

```go
type MasCRIServer struct {
    // UnimplementedRuntimeServiceServer must be embedded to have forward compatible implementations.
    v1.UnimplementedRuntimeServiceServer
    v1.UnimplementedImageServiceServer

    config ServerConfig
}
```

## External Entities (CRI API)

_Managed by `k8s.io/cri-api`_

- **VersionRequest / VersionResponse**
- **RunPodSandboxRequest / RunPodSandboxResponse**
