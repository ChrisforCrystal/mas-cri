# Feature 005: Native Container Runtime (Bye Bye Docker)

## Goal

移除对 Docker Daemon 的依赖，直接在 Linux 上实现容器的创建和运行。这相当于我们要手写一个简易版的 `runc` 或 `containerd-shim`。

## Scope

为了保持可行性且不至于陷入造轮子的深渊，建议分两步走：

1. **Level 1 (Hard)**: 使用 `opencontainers/runc/libcontainer` 库。这是 Docker 和 Kubernetes 底层的核心库，它封装了 OS 细节，但控制权仍在 Go 代码里。
2. **Level 2 (Nightmare)**: 纯手写 Syscall (Clone, PivotRoot, Cgroups)。这对教学非常有意义，但代码量巨大且极易出错。

**建议选择 Level 1**，既能学到内核原理（Namespace/Cgroup 配置），又能保证写出来的东西真的能跑。

## Architecture Change

- **Old**: `MasCRIServer` -> `DockerAdapter` -> `Docker Daemon` -> `Container`
- **New**: `MasCRIServer` -> `NativeAdapter` -> `libcontainer` -> `Container Process`

## Technical Considerations (Linux Only)

- **Rootfs**: 我们需要一个真正的 rootfs 目录。
  - 方案：简单的 `docker export` 导出一个 busybox tar 包，解压到 `/var/lib/mascri/containers/<id>/rootfs`。
- **Cgroups**: 需要管理 `/sys/fs/cgroup`。
- **Namespace**: 需要配置 PID, Network, IPC, UTS, Mount 命名空间。

## User Stories

- 作为开发者，我希望 MasCRI 可以在没有安装 Docker 的 Linux 机器上运行容器。
- 作为学习者，我希望看到代码里是如何显式配置 `CLONE_NEWPID` 和挂载 `/proc` 文件系统的。
