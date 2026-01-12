# Feature 003: CNI Networking Integration

## 1. Goal

Implement true Pod networking by integrating **CNI (Container Network Interface)**.
Currently, our `RunPodSandbox` uses `--net=none` (or `--net=host` implicitly if not set), which means Pods don't have their own IP addresses.
The goal is that after `RunPodSandbox`, the Pod behaves like a real VM on the network: it has its own `eth0`, its own IP (e.g., `10.88.0.2`), and can talk to other Pods via a Bridge.

## 2. User Stories

- **As a User**, when I run `crictl runp`, I want the resulting Pod to have an IP address assigned from a CNI subnet (e.g., 10.88.0.0/16).
- **As a User**, I want to be able to use `crictl inspectp` to see the assigned IP. (Currently it shows empty or host IP).

## 3. Technical Requirements

### 3.1 CNI Plugin Integration

- We will NOT write a CNI plugin from scratch (that's a separate project).
- We will USE standard CNI plugins (`bridge`, `loopback`, `host-local`).
- We need to import `github.com/containernetworking/cni/libcni` in our code.

### 3.2 Workflow Update

The `RunPodSandbox` flow will change:

1.  **Stop Docker Networking**: continue using `docker run --net=none`.
2.  **Get NetNS**: Find the path to the Pause container's Network Namespace (e.g., `/var/run/netns/cni-1234...`).
3.  **Call CNI ADD**: Use `libcni` to load configuration from `/etc/cni/net.d/` and invoke the plugins to attach `eth0` to that NetNS.
4.  **Record IP**: Capture the IP returned by CNI and store it (in memory or annotation).

### 3.3 Configuration

- MasCRI needs to know where CNI configs live (default `/etc/cni/net.d`) and where binaries live (default `/opt/cni/bin`).

## 4. Success Criteria

1.  **Setup**: Install standard CNI plugins (`brew install cni-plugins` on Mac or download binaries).
2.  **Config**: Create a simple `10-bridge.conf` in `/etc/cni/net.d/`.
3.  **Run**: `crictl runp sandbox.json`.
4.  **Verify**:
    - `crictl inspectp <ID>` shows an IP address.
    - `docker exec <ID> ip addr` shows `eth0` with that IP.
