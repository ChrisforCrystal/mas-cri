# Plan: Feature 005 Native Runtime

## Prerequisites

- [ ] 确保开发环境是 Linux (Lima)。
- [ ] 引入 `github.com/opencontainers/runc` (libcontainer) 依赖。

## Phase 1: Image & Rootfs (The Filesystem)

容器跑起来得先有文件系统。我们暂时不写复杂的镜像分层下载器。

- [ ] 实现 `ImageService`: 简单的 tar 包解压器。
- [ ] 用户提供一个 `busybox.tar`。
- [ ] `PullImage`: 把 tar 包解压到 `/var/lib/mascri/images/busybox`。
- [ ] `CreateContainer`: 把镜像目录 copy (或 overlay mount) 到 `/var/lib/mascri/containers/<id>/rootfs`。

## Phase 2: The Native Adapter

替换掉 `DockerAdapter`。

- [ ] 创建 `pkg/native/adapter.go`。
- [ ] 定义 `Container` 结构体，持有 `libcontainer.Container` 对象。

## Phase 3: Implementing Lifecycle

- [ ] **Create**: 使用 `libcontainer.New()` 创建 factory。
  - 配置 `Rootfs` 路径。
  - 配置 `Namespaces` (PID, IPC, UTS, Mount)。
  - 配置 `Cgroups` (Memory, CPU)。
- [ ] **Start**: 调用 `container.Run(process)`。
  - 这里是真正 fork 进程的地方。
- [ ] **Stop**: 发送 Signal 给进程。
- [ ] **Remove**: 删除目录。

## Phase 4: Integration

- [ ] 修改 `main.go`，增加 `--runtime-mode=native` 开关。
- [ ] 在 `NewMasCRIServer` 里根据开关选择 Adapter。

## Verification

- [ ] 在 Lima Linux 上运行。
- [ ] `crictl runp` (Pause)
- [ ] `crictl create` (Busybox)
- [ ] `crictl start`
- [ ] `ls /proc` 在容器内应该只能看到很少的进程。
