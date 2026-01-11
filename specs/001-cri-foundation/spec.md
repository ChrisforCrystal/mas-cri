# Feature Specification: Basic MasCRI Foundation with Traceability

**Feature Branch**: `001-cri-foundation`
**Created**: 2026-01-11
**Status**: Draft
**Input**: User description: "设定具体的方针" (Define specific guidelines/policies - Interpreted as: Implement the basic CRI Shim architecture and first "Hello World" functional loop)

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Kubelet/CRI Verification (Priority: P1)

As a Kubernetes Administrator (or Learning Developer), I want to be able to start the MasCRI server and have `crictl` successfully connect to it, so that I can verify the basic communication channel is open and tracing is working.

**Why this priority**: Without a working connection, no other functionality can be built or learned from. This establishes the "Hello World" of our gRPC service.

**Independent Test**: Can be tested without a full K8s cluster using `crictl --runtime-endpoint ... info`.

**Acceptance Scenarios**:

1. **Given** the MasCRI server is running, **When** I run `crictl info` against its socket, **Then** I should see the server version information.
2. **Given** the MasCRI server is running, **When** I run `crictl info`, **Then** the server logs should output the exact API request received (Traceability).

---

### User Story 2 - Basic Pod Sandbox Stub (Priority: P2)

As a Developer, I want to see how `RunPodSandbox` requests look when triggered, even if the implementation is a mock, so that I can study the data fields Kubelet sends for a Pod.

**Why this priority**: `RunPodSandbox` is the first step in the Pod lifecycle. Seeing the data is crucial for the "Test Driven Learning" principle.

**Independent Test**: Use `crictl runp sandbox-config.yaml` and observe logs.

**Acceptance Scenarios**:

1. **Given** MasCRI is running, **When** I send a `RunPodSandbox` request via `crictl`, **Then** the server logs should verify receipt of the complex Pod configuration struct.
2. **Given** MasCRI is running, **When** I send the request, **Then** it should return a fake Sandbox ID, allowing the client to think it succeeded (Mock implementation).

---

### Edge Cases

- What happens when `crictl` sends a version we don't support? (Should fail gracefully or negotiate)
- What happens if the socket file is already occupied? (Should cleanup or error out)

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST implement a gRPC server listening on a Unix Domain Socket (default UNIX path per K8s standards).
- **FR-002**: System MUST implement the `RuntimeService` and `ImageService` interfaces defined in `k8s.io/cri-api`.
- **FR-003**: System MUST provide a "Trace Logging" middleware that intercepts ALL incoming gRPC calls and logs the full Method Name and Request Payload (as JSON) to stdout/stderr.
- **FR-004**: System MUST implement the `Version` RPC to return compliant Runtime Name ("MasCRI") and Version ("0.1.0").
- **FR-005**: All other required RPC methods (RunPodSandbox, CreateContainer, etc.) MUST be implemented as stubs that log usage and return "Not Implemented" (or Mock Success for P2) to prevent gRPC errors on the wire.

### Key Entities

- **RuntimeServer**: The main gRPC server struct.
- **Interceptor**: The logging logic wrapper.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `crictl info` returns successfully in under 1 second.
- **SC-002**: Developer can see the full JSON body of a `RunPodSandbox` request in the logs.
- **SC-003**: The project compiles with `go build` without errors.
