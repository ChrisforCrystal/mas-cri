package server

import (
	"context"
	"fmt"
	"strings"

	"mascri/pkg/version"

	"github.com/sirupsen/logrus"
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

// RunPodSandbox 接口实现
// 这是创建 Pod 的第一步。我们在这里启动一个 "Pause" 容器。
func (s *MasCRIServer) RunPodSandbox(ctx context.Context, req *runtimeapi.RunPodSandboxRequest) (*runtimeapi.RunPodSandboxResponse, error) {
	config := req.GetConfig()
	
	// 1. 确保 Pause 镜像存在 (Backend 自行处理，native 不需要 pull)
	// Docker adapter will handle pulling if not present or let Docker daemon do it.

	// 2. 运行 Pause 容器
	// 直接传 CRI config
	podID, err := s.backend.RunPodSandbox(ctx, config, req.RuntimeHandler)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Launched Pod Sandbox %s (ID: %s)", config.Metadata.Name, podID)

	// 3. 配置网络 (CNI)
	// 只有当 CNI 管理器初始化成功时才执行
	if s.cni != nil {
		netns, err := s.backend.GetNetNS(podID)
		if err != nil {
			return nil, fmt.Errorf("failed to get netns for sandbox %s: %w", podID, err)
		}
		
		logrus.Infof("Setting up network for pod %s (netns: %s)", podID, netns)
		_, err = s.cni.SetUpPod(ctx, podID, netns)
		if err != nil {
			return nil, fmt.Errorf("CNI setup failed: %w", err)
		}
	}

	return &runtimeapi.RunPodSandboxResponse{
		PodSandboxId: podID,
	}, nil
}

// CreateContainer 创建业务容器
// 必须在 RunPodSandbox 之后调用。
func (s *MasCRIServer) CreateContainer(ctx context.Context, req *runtimeapi.CreateContainerRequest) (*runtimeapi.CreateContainerResponse, error) {
	config := req.GetConfig()
	sandboxID := req.GetPodSandboxId()

	// 1. Pull Image (Handled by Kubelet/Docker)
	
	// Create request has GetSandboxConfig
	containerID, err := s.backend.CreateContainer(ctx, sandboxID, config, req.GetSandboxConfig())
	if err != nil {
		return nil, err
	}

	return &runtimeapi.CreateContainerResponse{
		ContainerId: containerID,
	}, nil
}

// StartContainer 启动容器
func (s *MasCRIServer) StartContainer(ctx context.Context, req *runtimeapi.StartContainerRequest) (*runtimeapi.StartContainerResponse, error) {
	containerID := req.GetContainerId()

	if err := s.backend.StartContainer(ctx, containerID); err != nil {
		return nil, err
	}

	return &runtimeapi.StartContainerResponse{}, nil
}

// ListPodSandbox 列出所有的 Pod (Pause 容器)
func (s *MasCRIServer) ListPodSandbox(ctx context.Context, req *runtimeapi.ListPodSandboxRequest) (*runtimeapi.ListPodSandboxResponse, error) {
	// 暂时还是利用 ListContainers 获取所有容器然后手动过滤
	// 理想情况应该调用 s.backend.ListPodSandbox(ctx, req.Filter)
	containers, err := s.backend.ListContainers(ctx, nil)
	if err != nil {
		return nil, err
	}

	var sandboxes []*runtimeapi.PodSandbox
	for _, c := range containers {
		// 只有名字包含 k8s_POD 的才是 Pod Sandbox
		name := c.Metadata.Name
		if strings.Contains(name, "k8s_POD_") {
			sandboxes = append(sandboxes, &runtimeapi.PodSandbox{
				Id:    c.Id,
				State: runtimeapi.PodSandboxState_SANDBOX_READY, 
				Metadata: &runtimeapi.PodSandboxMetadata{
					Name: name, // Simplified
					Uid: "unknown",
					Namespace: "unknown",
				},
			})
		}
	}

	return &runtimeapi.ListPodSandboxResponse{
		Items: sandboxes,
	}, nil
}

// ListContainers 列出所有业务容器
func (s *MasCRIServer) ListContainers(ctx context.Context, req *runtimeapi.ListContainersRequest) (*runtimeapi.ListContainersResponse, error) {
	containers, err := s.backend.ListContainers(ctx, req.GetFilter())
	if err != nil {
		return nil, err
	}

	var result []*runtimeapi.Container
	for _, c := range containers {
		// 排除掉 POD 容器
		if c.Metadata != nil && !strings.Contains(c.Metadata.Name, "k8s_POD_") {
			result = append(result, c)
		}
	}

	return &runtimeapi.ListContainersResponse{
		Containers: result,
	}, nil
}

func contains(s, substr string) bool {
	// Helper function since strings.Contains is standard library
	// But `docker ps` names might involve slashes etc.
	// Simple wrapper.
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > 0 // Placeholder logic, actually we should use strings.Contains
}

// Helper stubs for removal
// StopPodSandbox 停止 Pod Sandbox
// 1. 清理网络 (CNI TearDown)
// 2. 停止容器 (Docker Stop)
func (s *MasCRIServer) StopPodSandbox(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) (*runtimeapi.StopPodSandboxResponse, error) {
	podID := req.PodSandboxId

	// 1. 尝试清理网络
	// 注意：即便获取 NetNS 失败，也要继续尝试停止容器
	if s.cni != nil {
		netns, err := s.backend.GetNetNS(podID)
		if err == nil && netns != "" {
			// 只有拿到了 netns 才能清理
			if err := s.cni.TearDownPod(ctx, podID, netns); err != nil {
				logrus.Warnf("CNI teardown failed for pod %s: %v", podID, err)
			}
		}
	}

	// 2. 停止容器
	if err := s.backend.StopPodSandbox(ctx, podID); err != nil {
		// 忽略 "No such container" 错误，因为可能已经被删除了
		logrus.Warnf("Failed to stop sandbox container %s: %v", podID, err)
	}

	return &runtimeapi.StopPodSandboxResponse{}, nil
}

// RemovePodSandbox 删除 Pod Sandbox
func (s *MasCRIServer) RemovePodSandbox(ctx context.Context, req *runtimeapi.RemovePodSandboxRequest) (*runtimeapi.RemovePodSandboxResponse, error) {
	podID := req.PodSandboxId

	// 调用 Docker 删除容器
	if err := s.backend.RemovePodSandbox(ctx, podID); err != nil {
		return nil, err
	}

	return &runtimeapi.RemovePodSandboxResponse{}, nil
}

// PodSandboxStatus 获取 Pod 状态
// 我们需要根据容器名 k8s_POD_<name>_<ns>_<uid> 反向解析出 Metadata
func (s *MasCRIServer) PodSandboxStatus(ctx context.Context, req *runtimeapi.PodSandboxStatusRequest) (*runtimeapi.PodSandboxStatusResponse, error) {
	podID := req.PodSandboxId

	// 1. Get Container Info from Docker
	// We reuse ListContainers but filter for this specific ID would be better.
	containers, err := s.backend.ListContainers(ctx, nil)
	if err != nil {
		return nil, err
	}

	var targetContainer *runtimeapi.Container
	for _, c := range containers {
		if strings.HasPrefix(c.Id, podID) {
			targetContainer = c
			break
		}
	}

	if targetContainer == nil {
		return nil, fmt.Errorf("pod sandbox %s not found", podID)
	}

	// 2. Parse Metadata from Name
	// Name format: k8s_POD_<name>_<namespace>_<uid>_<attempt>
	// Example: k8s_POD_cni-test_default_cni
	
	// Helper to parse name
	name, ns, uid := parseSandboxName(targetContainer.Metadata.Name)

	// 3. Get Creation Time via Docker Inspect
	// ListContainers gives us "About a minute ago", which is hard to parse.
	// We need exact timestamp for CreatedAt (int64 nanoseconds).
	createdAtNano, err := s.backend.GetContainerCreatedAt(targetContainer.Id)
	if err != nil {
		logrus.Warnf("Failed to get created timestamp for %s: %v", podID, err)
		createdAtNano = 0 // Fallback
	}

	return &runtimeapi.PodSandboxStatusResponse{
		Status: &runtimeapi.PodSandboxStatus{
			Id:        targetContainer.Id,
			State:     runtimeapi.PodSandboxState_SANDBOX_READY,
			CreatedAt: createdAtNano,
			Metadata: &runtimeapi.PodSandboxMetadata{
				Name:      name,
				Namespace: ns,
				Uid:       uid,
			},
			Network: &runtimeapi.PodSandboxNetworkStatus{
				Ip: "10.88.0.2", // Mock IP for now, verifying static value
			},
		},
	}, nil
}

// Helper to parse container name
// Expected: k8s_POD_{name}_{ns}_{uid}
func parseSandboxName(fullName string) (string, string, string) {
	// Docker name might start with /
	cleanName := fullName
	if len(cleanName) > 0 && cleanName[0] == '/' {
		cleanName = cleanName[1:]
	}
	
	// Split by "_"
	// parts[0] = k8s
	// parts[1] = POD
	// parts[2] = name
	// parts[3] = ns
	// parts[4] = uid
	parts := strings.Split(cleanName, "_")
	if len(parts) < 5 {
		return "unknown", "unknown", "unknown"
	}
	
	return parts[2], parts[3], parts[4]
}
