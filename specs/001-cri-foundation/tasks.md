# Tasks: Basic MasCRI Foundation with Traceability

**Feature Branch**: `001-cri-foundation`
**Feature Spec**: [Link to spec.md](./spec.md)

## Phase 1: Setup (Project Initialization)

_Goal: Initialize the Go project structure and install dependencies._

- [x] T001 Initialize Go module `mascri` and `go.mod` in root
- [x] T002 Install dependencies: `k8s.io/cri-api` (requires specific tag logic usually, or just latest), `google.golang.org/grpc`, `github.com/urfave/cli/v2`, `github.com/sirupsen/logrus`
- [x] T003 Create directory structure: `cmd/mascri`, `pkg/server`, `pkg/version`
- [x] T004 Create `Makefile` with `build`, `run`, `test` targets

## Phase 2: Foundational (Traceability & gRPC Shell)

_Goal: Establish the running gRPC server with the "Trace Logging" interceptor (FR-003) which is critical for all subsequent steps._

- [x] T005 [P] Implement `pkg/version` with constant string "0.1.0" and Name "MasCRI"
- [x] T006 Implement `pkg/server/interceptor.go`: Create `UnaryServerInterceptor` that logs Method and JSON-marshaled Request to logrus
- [x] T007 Define `MasCRIServer` struct in `pkg/server/server.go` embedding `UnimplementedRuntimeServiceServer` and `UnimplementedImageServiceServer`
- [x] T008 Implement `Start(socketPath string)` in `pkg/server/server.go` that:
  - Removes existing socket file if present (Safe Clean)
  - Listens on Unix socket
  - Registers Service with Interceptor
  - Serving loop
- [x] T009 Implement `cmd/mascri/main.go` using `urfave/cli` to parse `--socket` and `--debug` flags and call `server.Start`

## Phase 3: User Story 1 - Kubelet/CRI Verification (Priority: P1)

_Goal: "Hello World" - Make `crictl info` work (FR-004)._
_Independent Test: `crictl -r unix:///tmp/mascri.sock info` returns version info._

- [x] T010 [US1] Implement `Version` RPC method in `pkg/server/runtime.go`
  - Returns `kubelet.RuntimeVersion` formatted struct
- [x] T011 [US1] Implement `Status` RPC method in `pkg/server/runtime.go` (Required by `info` usually)
  - Return `RuntimeReady` status condition (true)
- [x] T012 [US1] Manual Verification: Build and run `mascri`, then run `crictl info` against it. Capture logs proving Traceability.

## Phase 4: User Story 2 - Basic Pod Sandbox Stub (Priority: P2)

_Goal: Observe Pod Data (Test Driven Learning) via `RunPodSandbox`._
_Independent Test: `crictl -r ... runp config.yaml` returns success ID._

- [x] T013 [P] [US2] Implement `RunPodSandbox` RPC stub in `pkg/server/runtime.go`
  - Must NOT return "Not Implemented"
  - Must log the `config` argument (covered by interceptor, but add specific "Implementing..." log)
  - Return a random/static Sandbox ID (e.g., "sandbox-123")
- [x] T014 [US2] Ensure `StopPodSandbox`, `RemovePodSandbox`, `PodSandboxStatus`, `ListPodSandbox` are at least stubs that don't crash (return "Not Implemented" is fine for now, or simple success for `Stop`/`Remove`)
- [x] T015 [US2] Manual Verification: Run `crictl runp` and verify full JSON payload is visible in logs.

## Final Phase: Polish

- [x] T016 Review comment quality (Ensure complex logic has Chinese explanations per Constitution)
- [x] T017 Update `README.md` with usage instructions derived from `quickstart.md`

## Dependencies

1. **Setup** -> **Foundational**
2. **Foundational** (T008/T009) -> **US1** (Server must run for `crictl` to connect)
3. **US1** -> **US2** (Connection established, now adding features)

## Parallel Execution Examples

- T005 (Version pkg) and T006 (Interceptor) can be done in parallel.
- T013 (RunPodSandbox) and T014 (Other Sandbox Stubs) can be done in parallel once struct is ready.
