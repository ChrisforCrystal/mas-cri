# Tasks: Docker Backend Integration

**Feature Branch**: `002-cri-docker-backend`
**Feature Spec**: [Link to spec.md](./spec.md)
**Implementation Plan**: [Link to plan.md](./plan.md)

## Phase 1: Docker Adapter (Infrastructure)

_Goal: Create the "engine room" - a safe wrapper around `exec.Command("docker")`._

- [x] T001 Create `pkg/docker` directory and `adapter.go`
- [x] T002 Implement `Adapter` struct and constructor
- [x] T003 Implement `PullImage(image)`: Wraps `docker pull`
- [x] T004 Implement `InspectImage(image)`: Wraps `docker inspect` (needed for ImageStatus)
- [x] T005 Implement `RunSandbox(config)`: Wraps `docker run` for pause container
  - Arguments: `-d`, `--name`, `--net=none` (or host for now), `registry.k8s.io/pause:3.9`
- [x] T006 Implement `CreateContainer(sandboxID, config)`: Wraps `docker create`
  - Arguments: `--net=container:<sandboxID>`, `--name ...`
- [x] T007 Implement `StartContainer(containerID)`: Wraps `docker start`
- [x] T008 Implement `ListContainers()`: Wraps `docker ps -a --format '{{json .}}'`

## Phase 2: Wiring Image Service

_Goal: Make `crictl pull` work (User Story 1)._

- [x] T009 Refactor `MasCRIServer` in `pkg/server/server.go` to include `*docker.Adapter`
- [x] T010 Create `pkg/server/image.go` and move ImageService methods there
- [x] T011 Implement `PullImage` RPC: Call adapter, handle errors
- [x] T012 Implement `ListImages` and `ImageStatus` RPCs (Basic version)
- [x] T013 Verify: `crictl pull nginx:alpine` works

## Phase 3: Wiring Runtime Service

_Goal: Make `crictl runp` and `create` work (User Story 2)._

- [x] T014 Update `RunPodSandbox` in `pkg/server/runtime.go`:
  - Remove Stub logic
  - Extract name/namespace/uid from config
  - Call `docker.RunSandbox`
  - Return real Docker ID
- [x] T015 Implement `CreateContainer` RPC in `pkg/server/runtime.go`:
  - Call `docker.CreateContainer`
- [x] T016 Implement `StartContainer` RPC in `pkg/server/runtime.go`:
  - Call `docker.StartContainer`
- [x] T017 Implement `ListContainers` RPC:
  - Call `docker.ListContainers` and convert to CRI format
- [x] T018 Implement `ListPodSandbox` RPC (using docker ps filters)

## Phase 4: Verification & Polish

_Goal: Full end-to-end verification._

- [x] T019 Manual Verification: Full Lifecycle (Pull -> RunP -> Create -> Start -> List)
- [x] T020 Update README to reflect Docker prerequisite

## Dependencies

- Phase 1 must be complete before Phase 2/3.
- Phase 2 (Image) is independent of Phase 3 (Runtime), but Image is usually needed first to run anything.
