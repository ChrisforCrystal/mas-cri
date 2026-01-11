# Research: Basic MasCRI Foundation

## Decisions

### 1. Request Logging Strategy

- **Decision**: Use a gRPC Unary Interceptor.
- **Rationale**: This is the standard way to inject middleware in gRPC-Go. It allows intercepting the request _before_ the handler, logging the payload, and then invoking the handler.
- **Alternatives Considered**: Logging inside each handler. Rejected because it violates DRY (Don't Repeat Yourself) and makes it hard to ensure 100% coverage.

### 2. CLI Framework

- **Decision**: `github.com/urfave/cli/v2`
- **Rationale**: De-facto standard for Go CLI apps. Easy to manage flags like `--socket-path` and subcommands.
- **Alternatives Considered**: `cobra`. Cobra is also excellent but slightly heavier (often paired with Viper). `urfave/cli` is sufficient for a single-binary daemon.

### 3. CRI Dependency

- **Decision**: `k8s.io/cri-api` (v0.25+ typically, matching K8s versions)
- **Rationale**: Provides the generated Go structs for the protobufs. No need to run `protoc` ourselves unless modifying the spec (which we aren't).

## Unknowns Resolved

- **Socket Path**: Default is usually `/var/run/dockershim.sock` or similar. We will default to `/tmp/mascri.sock` for safe local development (rootless) but allow configuration.
