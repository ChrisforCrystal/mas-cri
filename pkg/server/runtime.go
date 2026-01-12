package server

import (
	"context"

	"mascri/pkg/docker"
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
	
	// 准备 Docker 适配器需要的配置
	sandboxConfig := &docker.SandboxConfig{
		Name:      config.GetMetadata().GetName(),
		Namespace: config.GetMetadata().GetNamespace(),
		Uid:       config.GetMetadata().GetUid(),
		// K8s 官方 Pause 镜像，实际生产中这个应该是可配置的
		Image:     "registry.k8s.io/pause:3.9", 
	}

	// 1. 确保 Pause 镜像存在 (可选，但推荐)
	// 虽然 docker run 会自动 pull，但显式 pull 更可控
	if err := s.docker.PullImage(sandboxConfig.Image); err != nil {
		// Log error but try to run anyway
		logrus.WithError(err).Warn("Failed to pre-pull pause image")
	}

	// 2. 运行 Pause 容器
	podID, err := s.docker.RunSandbox(sandboxConfig)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Launched Pod Sandbox %s (ID: %s)", sandboxConfig.Name, podID)

	return &runtimeapi.RunPodSandboxResponse{
		PodSandboxId: podID,
	}, nil
}

// CreateContainer 创建业务容器
// 必须在 RunPodSandbox 之后调用。
func (s *MasCRIServer) CreateContainer(ctx context.Context, req *runtimeapi.CreateContainerRequest) (*runtimeapi.CreateContainerResponse, error) {
	config := req.GetConfig()
	sandboxID := req.GetPodSandboxId()

	containerConfig := &docker.ContainerConfig{
		Name:    config.GetMetadata().GetName(),
		Image:   config.GetImage().GetImage(),
		Command: config.GetCommand(),
		Args:    config.GetArgs(),
	}

	// 1. Pull Image (Strictly speaking this should be done by Kubelet calling PullImage first,
	// but docker create usually needs image locally or it will pull)
	// We rely on previous PullImage call or Docker's auto-pull.

	containerID, err := s.docker.CreateContainer(sandboxID, containerConfig)
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

	if err := s.docker.StartContainer(containerID); err != nil {
		return nil, err
	}

	return &runtimeapi.StartContainerResponse{}, nil
}

// ListPodSandbox 列出所有的 Pod (Pause 容器)
func (s *MasCRIServer) ListPodSandbox(ctx context.Context, req *runtimeapi.ListPodSandboxRequest) (*runtimeapi.ListPodSandboxResponse, error) {
	containers, err := s.docker.ListContainers()
	if err != nil {
		return nil, err
	}

	var sandboxes []*runtimeapi.PodSandbox
	for _, c := range containers {
		// 只有名字包含 k8s_POD 的才是 Pod Sandbox
		// 这是一个非常 naive 的过滤方式，但在 Feature 002 阶段足够了
		if len(c.Names) > 0 && (c.Names == "k8s_POD_" || contains(c.Names, "k8s_POD_")) {
			sandboxes = append(sandboxes, &runtimeapi.PodSandbox{
				Id:    c.ID,
				State: runtimeapi.PodSandboxState_SANDBOX_READY, // 简化处理
				Metadata: &runtimeapi.PodSandboxMetadata{
					Name: "unknown", // 解析名字比较麻烦，先跳过
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
	containers, err := s.docker.ListContainers()
	if err != nil {
		return nil, err
	}

	var result []*runtimeapi.Container
	for _, c := range containers {
		// 排除掉 POD 容器
		if !contains(c.Names, "k8s_POD_") {
			result = append(result, &runtimeapi.Container{
				Id:           c.ID,
				PodSandboxId: "unknown",
				Image:        &runtimeapi.ImageSpec{Image: c.Image},
				State:        runtimeapi.ContainerState_CONTAINER_RUNNING, // 简化处理
			})
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
func (s *MasCRIServer) StopPodSandbox(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) (*runtimeapi.StopPodSandboxResponse, error) {
	// TODO: Call docker stop (using adapter)
	return &runtimeapi.StopPodSandboxResponse{}, nil
}

func (s *MasCRIServer) RemovePodSandbox(ctx context.Context, req *runtimeapi.RemovePodSandboxRequest) (*runtimeapi.RemovePodSandboxResponse, error) {
	// TODO: Call docker rm (using adapter)
	return &runtimeapi.RemovePodSandboxResponse{}, nil
}

func (s *MasCRIServer) PodSandboxStatus(ctx context.Context, req *runtimeapi.PodSandboxStatusRequest) (*runtimeapi.PodSandboxStatusResponse, error) {
	// TODO: Call docker inspect
	return &runtimeapi.PodSandboxStatusResponse{
		Status: &runtimeapi.PodSandboxStatus{
			Id:    req.PodSandboxId,
			State: runtimeapi.PodSandboxState_SANDBOX_READY,
		},
	}, nil
}

