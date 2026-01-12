# Feature 004: Streaming IO (Logs, Exec, Attach)

## Goal

实现 CRI 的 Streaming API，支持 `kubectl logs`, `kubectl exec` 和 `kubectl attach` 操作。让用户能够查看容器日志并进入容器内部执行命令。

## Background

CRI 中的流式操作（Streaming）与普通 RPC 不同。

1. 普通 RPC (如 `RunPodSandbox`) 是直接返回结果。
2. 流式 RPC (如 `Exec`) **不直接执行命令**，而是返回一个 **URL**。
3. Kubelet 拿到这个 URL 后，会向该 URL 发起 HTTP 请求，并升级为 SPDY/WebSocket 长连接。
4. Runtime 需要启动一个 **HTTP Server** 来处理这些长连接请求，并将数据流转发(Stream) 给底层的容器。

## Technical Requirements

- **Streaming Server**: 需要启动一个额外的 HTTP Server (不同于 gRPC Server)。
- **Docker Integration**:
  - `logs`: 调用 `docker logs` 获取日志。
  - `exec`: 调用 `docker exec -it` 并通过 `hijack` 劫持 TCP 连接来转发 Stdin/Stdout/Stderr。
- **Library**: 使用 `k8s.io/kubelet/pkg/cri/streaming` 库（K8s 官方提供的标准库）来简化实现。

## User Stories

- 作为用户，如果 Pod 启动失败或运行异常，我希望通过 `crictl logs <ID>` 查看容器标准输出，以便排查问题。
- 作为用户，我希望通过 `crictl exec -it <ID> sh` 进入容器内部 shell，查看文件或调试网络。

## Key Design Decisions

- **Reuse K8s Library**: 直接使用 Kubernetes 官方的 `streaming` 包，避免手写复杂的 SPDY/流式协议处理。
- **Port**: Streaming Server 将监听一个随机端口或固定端口 (e.g. 10010)。

## Verification Plan

1. `crictl logs <container_id>`: 应该能看到容器打印的 "Hello"。
2. `crictl exec -it <container_id> ls`: 应该能列出容器内文件。
