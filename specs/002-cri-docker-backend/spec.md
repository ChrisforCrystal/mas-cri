# Feature Specification: Docker Backend Integration

**Feature Branch**: `002-cri-docker-backend`
**Created**: 2026-01-12
**Status**: Draft
**Input**: Implement Docker Backend for CRI. Implement RunPodSandbox, PullImage, CreateContainer, StartContainer using Docker CLI/API.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pull Image via CRI (Priority: P1)

As a User, I want `crictl pull nginx:alpine` to work, which should trigger the backend to pull the image from Docker Hub, so that I can have images ready for my containers.

**Why this priority**: You can't run a container without an image. `ImageService` is the prerequisite for `RuntimeService`.

**Independent Test**:

```bash
crictl pull nginx:alpine
crictl images
# Should show nginx:alpine
docker images
# Should also show nginx:alpine (proving we used Docker)
```

**Acceptance Scenarios**:

1. **Given** MasCRI is running, **When** I run `crictl pull nginx:alpine`, **Then** it should return success.
2. **Given** image is pulled, **When** I run `crictl images`, **Then** the list should match `docker images`.

---

### User Story 2 - Run a Pod with a Container (Priority: P2)

As a User, I want to run a real Sandbox (Pod) and then start a real Container inside it using `crictl`, so that I can see an actual process running.

**Why this priority**: This is the core "Runtime" functionality. It transitions us from "Fake ID" to "Real Container".

**Independent Test**:

```bash
crictl runp sandbox-config.json
# Returns POD_ID
crictl create POD_ID container-config.json sandbox-config.json
# Returns CONTAINER_ID
crictl start CONTAINER_ID
# Returns Success
crictl ps
# Should show running container
```

**Acceptance Scenarios**:

1. **RunPodSandbox**: Should create a "pause" container in Docker (standard K8s pattern) or at least a placeholder container sharing namespaces.
2. **CreateContainer**: Should creating a container (e.g., nginx) that joins the Sandbox's namespaces.
3. **StartContainer**: Should actually start the process.
4. **ListContainers**: `crictl ps` should show the containers managed by MasCRI.

---

### Edge Cases

- **Image Pull Fail**: If image doesn't exist, should return proper gRPC error.
- **Name Conflict**: If Docker already has a container with the generated name, should handle it (or fail with "AlreadyExists").

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Implement `ImageService.PullImage` by calling `docker pull`.
- **FR-002**: Implement `ImageService.ListImages` by calling `docker images` and converting format.
- **FR-003**: Implement `RuntimeService.RunPodSandbox`.
  - MUST pull the "pause" image (e.g., `registry.k8s.io/pause:3.9`) if missing.
  - MUST running a "pause" container via Docker to hold the namespaces.
  - MUST return the Docker Container ID as the PodSandboxId.
- **FR-004**: Implement `RuntimeService.CreateContainer`.
  - MUST interpret `PodSandboxId` as the parent pause container ID.
  - MUST create the container with `--net=container:<PauseID>` (Network Namespace Sharing) and IPC/PID sharing if requested.
- **FR-005**: Implement `RuntimeService.StartContainer` by calling `docker start`.
- **FR-006**: Implement `RuntimeService.ListContainers` and `ListPodSandbox` by filtering `docker ps`.

### Key Entities

- **DockerAdapter**: A utility struct/package to wrap `exec.Command("docker", ...)` calls.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `crictl pull nginx:alpine` succeeds in <30s (depending on network).
- **SC-002**: `crictl runp` creates a visible container in `docker ps`.
- **SC-003**: `crictl create` + `start` launches an Nginx container that is reachable (networking might be simple host-network for now if CNI is not ready, start with HostNetwork or standard Docker bridge).
