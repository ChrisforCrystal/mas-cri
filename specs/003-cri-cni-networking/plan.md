# Plan: Feature 003 CNI Networking

## 1. Summary

We will integrate `libcni` into MasCRI. We'll introduce a new package `pkg/network` (or `pkg/cni`) to handle CNI interactions.
The `MasCRIServer` will initialize a `CNIManager` on startup.
`RunPodSandbox` will call `CNIManager.SetUpPod()`.
`StopPodSandbox` will call `CNIManager.TearDownPod()`.

## 2. Dependencies

- **Go Module**: `github.com/containernetworking/cni`
- **System**: CNI Plugins must be installed in `/opt/cni/bin`.
- **Config**: CNI Config must exist in `/etc/cni/net.d`.

## 3. Component Design

### 3.1 `pkg/cni` Package

- `CNIManager` struct:

  - Holds `libcni.CNIConfig`
  - Scans `/etc/cni/net.d` for the default network list.

- `SetUpPod(id string, netnsPath string, podConfig *runtimeapi.PodSandboxConfig) (Result, error)`:

  - Constucts `libcni.RuntimeConf` (ContainerID, NetNS, IfName="eth0").
  - Calls `cniConfig.AddNetworkList`.
  - Returns the IP result.

- `TearDownPod(id string, netnsPath string, podConfig...)`:
  - Calls `cniConfig.DelNetworkList`.

### 3.2 `pkg/server/runtime.go` Updates

- **RunPodSandbox**:

  1.  Call Docker Adapter -> Get `PodID`.
  2.  Ask Docker Adapter -> Get `NetNS Path` (needs new Adapter method `InspectNetNS`).
  3.  Call `CNIManager.SetUpPod(PodID, NetNS)`.
  4.  Return `PodID`.

- **StopPodSandbox**:
  1.  Ask Docker Adapter -> Get `NetNS Path`.
  2.  Call `CNIManager.TearDownPod`.
  3.  Call Docker Adapter -> `StopContainer`.

### 3.3 `pkg/docker` Updates

- `InspectContainer`: We need to parse more details, specifically `NetworkSettings.SandboxKey` (which is the NetNS path).

## 4. Complexity & Risks

- **NetNS on macOS/Docker Desktop**: This is the biggest risk. On Docker Desktop for Mac, the docker daemon runs in a VM. The "NetNS path" is inside the VM, not on the host mac.
- **Workaround for Mac**: MasCRI running on Mac specific host CANNOT directly configure NetNS inside the Docker VM via CNI.
- **Pivot**: Since we are on Mac, unless we run MasCRI _inside_ the Docker VM (or a Linux VM), standard CNI won't work easily because `libcni` expects to see the `/proc/.../ns/net` file.
- **Alternative**: For this learning project on Mac, we might need to fallback to:
  - **Option A**: Run MasCRI itself in a Docker container (bind mounting docker sock). **Recommended for authenticity**.
  - **Option B**: Simulate CNI logic but just print what it _would_ do (Mock CNI).
  - **Option C**: Use `docker network connect` as a cheat implementation of "CNI".

**Decision**: Let's aim for **Option A**. The user is running `make run` on Mac locally. This implies standard CNI won't work directly.
**Wait**: If we use Option C (`docker network connect`), we mimic the _effect_ (IP assignment) without the _pain_ of cross-OS NetNS manipulation.
User wants to learn **CNI Principle**. If we cheat with `docker network`, we don't learn CNI.
**Bold Move**: We will implement the `libcni` code. But to verify it, we might need to run MasCRI inside a linux environment (like a Codespace or a Dev Container). Or, for now, we write the code, and if it fails on Mac, we explain why (OS limitation).
_Actually_, let's check if the user is willing to use a dev container. The user context shows `OS version is mac`.

**Refined Plan**: We will write the **Real CNI Code**.
If verification fails on Mac, that is a valid learning outcome ("Why CNI needs Linux Kernel Namespace access").
We can add a "Cnless Mode" or "Docker Bridge Mode" as a fallback later.
Let's stick to the "Real Way" first.

## 5. Tasks

1.  Import `libcni`.
2.  Create `pkg/network/cni_manager.go`.
3.  Implement `SetUpPod` / `TearDownPod`.
4.  Integrate into `RunPodSandbox`.
