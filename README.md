# MasCRI (Mastering CRI)

**MasCRI** 是一个用于学习云原生底层原理的 **Kubernetes Container Runtime Interface (CRI)** 实现。

它的设计目标不是为了生产环境的高性能，而是为了**清晰度 (Clarity)** 和 **可观测性 (Observability)**。通过 MasCRI，你可以即时看到 Kubelet 是如何指挥 Runtime 工作的，每一个 gRPC 请求都会被透明地记录下来。

## 核心原则 (Constitution)

1.  **Cloud Native Native**: 严格遵循 CNCF 标准 (`k8s.io/cri-api`)。
2.  **Giant's Shoulders**: 站在巨人肩膀上，不重复造轮子。
3.  **Traceability**: "看见" 协议。所有的交互都必须可视。
4.  **Test Driven Learning**: 通过 `crictl` 来验证和学习。

## 当前状态 (v0.1.0)

目前处于 **Feature 002: Docker Backend** 完成阶段。

- [x] **Setup**: Go 1.22 项目结构。
- [x] **gRPC Server**: 监听 Unix Socket (`/tmp/mascri.sock`)。
- [x] **Trace Interceptor**: 自动捕获并打印所有请求的 JSON 参数。
- [x] **Docker Backend**: 真正的容器运行时，支持 Pull, RunP, Create, Start, List。

## 快速开始

### 依赖

- Go 1.22+
- `crictl` (推荐 `brew install cri-tools`)
- Docker (后续阶段需要)

### 运行

打开一个终端启动服务器：

```bash
make run
# 输出: MasCRI listening on /tmp/mascri.sock
```

### 验证

打开另一个终端发送 CRI 指令：

```bash
# 1. 检查版本和状态 (Kubelet 握手)
make verify-info

# 2. 模拟创建一个 Pod (观察 Server 端的日志！)
make verify-runp
```

你会看到 Server 端打印出类似这样的日志，这就展示了 Kubelet 发来的 Pod 配置：

```text
INFO --> [gRPC Request] method=/runtime.v1.RuntimeService/RunPodSandbox body="{\"config\":{\"metadata\":...}}"
```

## 路线图

- [x] **Phase 1**: 基础框架与 gRPC 日志拦截
- [x] **Phase 2**: 对接 Docker 后端 (Real Container Implementation) (Current)
- [ ] **Phase 3**: 进阶容器网络与 CNI 探索
- [ ] **Phase 4**: 探索安全容器 (gVisor/Kata)

## License

MIT
