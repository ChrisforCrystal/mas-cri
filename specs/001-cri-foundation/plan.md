# Implementation Plan: Basic MasCRI Foundation with Traceability

**Branch**: `001-cri-foundation` | **Date**: 2026-01-11 | **Spec**: [001-cri-foundation/spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-cri-foundation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement the initial "Hello World" CRI Shim server ("MasCRI") that listens on a Unix socket, implements the required `RuntimeService` and `ImageService` gRPC interfaces using `k8s.io/cri-api`, and provides a logging interceptor to trace all Kubelet interactions for educational purposes.

**Future Roadmap (User Requested):**

1.  **Phase 1 (Current)**: Foundation & Traceability (Mock Backend).
2.  **Phase 2**: Docker Backend (Proxy to Docker Daemon).
3.  **Phase 3**: Runc/Containerd Backend (Direct OCI manipulation).
4.  **Phase 4**: Secure/VM Backend (gVisor/Firecracker exploration).

## Technical Context

**Language/Version**: Go 1.22 (Latest Stable, typical for Cloud Native)
**Primary Dependencies**:

- `k8s.io/cri-api` (Standard Interfaces)
- `google.golang.org/grpc` (Communication)
- `github.com/sirupsen/logrus` or similar (Structured Logging)
- `github.com/urfave/cli/v2` (CLI Framework, standard in Go ecosystem)
  **Storage**: N/A (In-memory mock for Phase 1)
  **Testing**: `go test`, `crictl` (E2E verification)
  **Target Platform**: Linux/macOS (Development), Kubernetes Node (Deployment)
  **Project Type**: System Daemon / CLI
  **Performance Goals**: N/A (Educational focus, 1s response constraint is loose)
  **Constraints**: Must use Unix Domain Sockets; Code must be heavily commented in Chinese.
  **Scale/Scope**: Single binary, minimal external deps.

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

- [x] **I. Cloud Native Native**: Uses `k8s.io/cri-api` and standard gRPC.
- [x] **II. Giant's Shoulders**: Reuses upstream `cri-api` rather than redefining protos.
- [x] **III. Clarity over Performance**: Plan prioritizes logging middleware over raw throughput.
- [x] **IV. Traceability**: gRPC Interceptor is a core requirement (FR-003).
- [x] **V. Test Driven Learning**: Verification relies on `crictl`.

## Project Structure

### Documentation (this feature)

```text
specs/001-cri-foundation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (Proto definitions if custom, or usage docs)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
cmd/
└── mascri/              # Main entrypoint
    └── main.go

pkg/
├── server/              # gRPC Server Implementation
│   ├── server.go        # Server struct & Start logic
│   ├── runtime.go       # RPC Implementations (Version, RunPodSandbox)
│   ├── image.go         # RPC Implementations (ImageService stubs)
│   └── interceptor.go   # Logging Middleware (FR-003)
└── version/             # Version info

go.mod                   # Dependency definitions
Makefile                 # Build and Test commands
```

**Structure Decision**: Standard Go Project Layout (cmd/pkg pattern) to allow cleaner separation of the library logic (pkg) from the executable (cmd), supporting future growth.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --------- | ---------- | ------------------------------------ |
| N/A       |            |                                      |
