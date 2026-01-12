# Plan: Feature 004 Streaming IO

## Phase 1: Foundation (Streaming Server)

- [ ] 引入 `k8s.io/kubelet` 依赖 (如果需要 update go.mod)。
- [ ] 创建 `pkg/streaming` 包，实现 `streaming.Runtime` 接口。
- [ ] 在 `MasCRIServer` 中初始化 `streaming.Server`。
- [ ] 在 `Start()` 中启动 Streaming HTTP Server。

## Phase 2: Implement Logs

- [ ] 实现 `GetContainerLogs` 方法 (调用 `docker logs`)。
- [ ] 只有这一步其实不需要 Streaming Server，是直接读取文件或 Docker API 返回。
- [ ] _Correction_: `GetContainerLogs` 是普通 RPC，不算 Streaming，但我们要在这里一并做掉。

## Phase 3: Implement Exec (True Streaming)

- [ ] 实现 `Exec` 接口 (CRI RPC)。
  - **不是执行命令**，而是生成一个重定向 URL (e.g. `http://localhost:10010/exec/token`).
- [ ] 实现 `streaming.Runtime.Exec` 方法 (Server 回调)。
  - 这是真正干活的地方。
  - 调用 `docker exec` 并通过 API 劫持连接 (Hijack)。
  - 将数据流对接：`Browser(Kubelet) <-> Streaming Server <-> Docker Daemon <-> Container`。

## Phase 4: Verification

- [ ] 启动一个打印日志的容器。
- [ ] 验证 `crictl logs`。
- [ ] 验证 `crictl exec ls`。
