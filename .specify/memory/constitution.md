<!--
SYNC IMPACT REPORT
Version Change: [TEMPLATE] -> 0.1.0
Modified Principles:
- [PRINCIPLE_1_NAME] -> I. Cloud Native Native (云原生优先)
- [PRINCIPLE_2_NAME] -> II. Giant's Shoulders (站在巨人的肩膀上)
- [PRINCIPLE_3_NAME] -> III. Clarity over Performance (清晰优于性能)
- [PRINCIPLE_4_NAME] -> IV. Traceability & Observability (可追溯与可观测)
- [PRINCIPLE_5_NAME] -> V. Test Driven Learning (测试驱动学习)
Added Sections: None
Removed Sections: [SECTION_2_NAME], [SECTION_3_NAME]
Templates Requiring Updates:
- .specify/templates/plan-template.md (⚠ pending - check Constitution Check section)
Follow-up TODOs: None
-->

# MasCRI Constitution

<!-- Project: MasCRI - Mastering Container Runtime Interface -->

## Core Principles

### I. Cloud Native Native (云原生优先)

<!-- Focus on Cloud Native standards and understanding -->

MasCRI 的设计与实现必须遵循 Cloud Native Computing Foundation (CNCF) 的标准与最佳实践。我们的核心目标不仅是“跑通”，而是通过实现 CRI 来深度理解 Kubernetes 的 LifeCycle、资源隔离与调度机制。任何设计决策都应参考 Kubernetes 官方文档与设计原则。

### II. Giant's Shoulders (站在巨人的肩膀上)

<!-- Leverage existing robust frameworks/libraries -->

不重复造轮子。我们应当充分利用 `k8s.io/cri-api` 定义标准接口，使用 `go-grpc` 处理通信，以及利用成熟的 OCI 运行时工具（如 `runc` 或直接调用 `docker`/`podman` 命令行作为底层执行器）。代码应专注于 CRI 层的逻辑编排与转换，而非底层的系统调用细节，除非是为了教学目的特定的深入探索。

### III. Clarity over Performance (清晰优于性能)

<!-- Educational focus: Readability notes and extensive comments -->

MasCRI 是一个学习型项目。代码的可读性、架构的清晰度以及详尽的中文注释（解释“为什么这样做”）远比运行时的微秒级性能重要。所有的关键逻辑路径都必须包含解释性的注释，帮助阅读者理解 CRI 的工作流。

### IV. Traceability & Observability (可追溯与可观测)

<!-- Detailed logging for learning -->

为了让开发者“看见” Kubelet 与 Runtime 的对话，系统必须具备极高的可观测性。每一个 gRPC 接口调用（Request/Response）都必须被结构化地记录（Log），包括参数细节。通过日志应当能完整复原一个 Pod 创建的全过程。

### V. Test Driven Learning (测试驱动学习)

<!-- Verify understanding through tests -->

利用 `crictl` 工具和 Kubernetes E2E 测试集作为检验标准。每一个功能的实现都应伴随着对其行为的验证。不仅要跑通 Happy Path，也要通过测试去理解各种 Error Code 的含义。

## Governance

<!-- Constitution supersedes all other practices -->

本章程确立了 MasCRI 项目的核心价值观。任何对项目的重大架构变更或引入新的依赖，都必须首先通过“宪法检查”（System Check），确保其符合上述原则。特别是当引入复杂性时（例如为了性能牺牲可读性），必须有充分的理由并获得特别批准。

**Version**: 0.1.0 | **Ratified**: 2026-01-11 | **Last Amended**: 2026-01-11
