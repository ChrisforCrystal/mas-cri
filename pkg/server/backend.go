package server

import (
	"context"

	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// RuntimeBackend 定义了 MasCRI 底层容器运行时需要实现的行为。
// 无论是 Docker 还是 Native (libcontainer)，都必须从这里接入。
type RuntimeBackend interface {
	// Image Service
	PullImage(ctx context.Context, image string) (string, error)
	ListImages() ([]string, error)
	InspectImage(image string) error

	// Pod Sandbox
	RunPodSandbox(ctx context.Context, config *cri.PodSandboxConfig, runtimeHandler string) (string, error)
	StopPodSandbox(ctx context.Context, podID string) error
	RemovePodSandbox(ctx context.Context, podID string) error
	PodSandboxStatus(ctx context.Context, podID string) (*cri.PodSandboxStatus, error)
	ListPodSandbox(ctx context.Context, filter *cri.PodSandboxFilter) ([]*cri.PodSandbox, error)

	// Container Lifecycle
	CreateContainer(ctx context.Context, podID string, config *cri.ContainerConfig, sandboxConfig *cri.PodSandboxConfig) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout int64) error
	RemoveContainer(ctx context.Context, containerID string) error
	ContainerStatus(ctx context.Context, containerID string) (*cri.ContainerStatus, error)
	ListContainers(ctx context.Context, filter *cri.ContainerFilter) ([]*cri.Container, error)

	// Helpers
	GetContainerCreatedAt(containerID string) (int64, error)
	GetNetNS(containerID string) (string, error)
}
