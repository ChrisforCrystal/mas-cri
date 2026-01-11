package server

import (
	"context"

	"mascri/pkg/version"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// Version 接口实现
// Kubelet 调用此接口来确认 Runtime 的名称和版本，以及支持的 API 版本。
func (s *MasCRIServer) Version(ctx context.Context, req *runtimeapi.VersionRequest) (*runtimeapi.VersionResponse, error) {
	// 这里我们返回一个简单的版本信息
	return &runtimeapi.VersionResponse{
		Version:           version.APIVersion,  // CRI API Version
		RuntimeName:       version.ProgramName, // 我们的名字 "MasCRI"
		RuntimeVersion:    version.Version,     // 我们的版本 "0.1.0"
		RuntimeApiVersion: version.APIVersion,  // 再次确认 API 版本
	}, nil
}

// Status 接口实现
// Kubelet 会定期调用 Status 来检查 Runtime 的健康状况 (Network, RuntimeReady)。
func (s *MasCRIServer) Status(ctx context.Context, req *runtimeapi.StatusRequest) (*runtimeapi.StatusResponse, error) {
	// 对于 MasCRI 早期阶段，我们假装一切都好 (Fake it till you make it)
	return &runtimeapi.StatusResponse{
		Status: &runtimeapi.RuntimeStatus{
			Conditions: []*runtimeapi.RuntimeCondition{
				{
					Type:    runtimeapi.RuntimeReady,
					Status:  true,
					Reason:  "MasCRIIsReady",
					Message: "MasCRI is ready to rock",
				},
				{
					Type:    runtimeapi.NetworkReady,
					Status:  true,
					Reason:  "NetworkIsFake",
					Message: "Network is mocked",
				},
			},
		},
	}, nil
}

// RunPodSandbox 接口实现 (Stub)
// 这是创建 Pod 的第一步。Kubelet 会把 Pod 的配置（Metadata, DNS, PortMappings 等）发过来。
// 在 MasCRI 0.1.0 中，我们只是记录日志（由 Interceptor 自动完成），然后返回一个假的 ID。
func (s *MasCRIServer) RunPodSandbox(ctx context.Context, req *runtimeapi.RunPodSandboxRequest) (*runtimeapi.RunPodSandboxResponse, error) {
	// 注意：虽然函数体内我们什么都没写，但得益于 Interceptor，
	// 你会在控制台看到 req.Config 的完整 JSON 结构！这就是我们想要的效果。

	// 返回一个假的 ID，骗过 Kubelet
	fakePodID := "sandbox-fake-12345"

	return &runtimeapi.RunPodSandboxResponse{
		PodSandboxId: fakePodID,
	}, nil
}

// 其他必需的 RuntimeService 方法的存根 (Stub)
// 如果不实现这些，虽然编译没问题，但在运行时如果 Kubelet 调用了它们，
// 会因为嵌入了 UnimplementedRuntimeServiceServer 而返回错误，这是预期行为。
// 不过为了让 runp 流程完整，我们对 Stop/Remove/List 也稍微做个假。

func (s *MasCRIServer) StopPodSandbox(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) (*runtimeapi.StopPodSandboxResponse, error) {
	return &runtimeapi.StopPodSandboxResponse{}, nil
}

func (s *MasCRIServer) RemovePodSandbox(ctx context.Context, req *runtimeapi.RemovePodSandboxRequest) (*runtimeapi.RemovePodSandboxResponse, error) {
	return &runtimeapi.RemovePodSandboxResponse{}, nil
}

func (s *MasCRIServer) PodSandboxStatus(ctx context.Context, req *runtimeapi.PodSandboxStatusRequest) (*runtimeapi.PodSandboxStatusResponse, error) {
	return &runtimeapi.PodSandboxStatusResponse{
		Status: &runtimeapi.PodSandboxStatus{
			Id:    req.PodSandboxId,
			State: runtimeapi.PodSandboxState_SANDBOX_READY, // 永远是 Ready 的
		},
	}, nil
}

func (s *MasCRIServer) ListPodSandbox(ctx context.Context, req *runtimeapi.ListPodSandboxRequest) (*runtimeapi.ListPodSandboxResponse, error) {
	return &runtimeapi.ListPodSandboxResponse{
		Items: []*runtimeapi.PodSandbox{}, // 返回空列表
	}, nil
}
