//go:build !linux

package native

import (
	"context"
	"fmt"

	"mascri/pkg/network"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// NativeAdapter (Stub)
// 这个文件是 NativeAdapter 在非 Linux 系统（如 macOS/Windows）上的“替身”。
//
// 为什么需要它？
// 1. 真正的 adapter.go 依赖 `libcontainer`，它使用了大量 Linux 独有的系统调用（syscall）。
// 2. 如果你在 Mac 上直接编译 adapter.go，编译器会报错说找不到那些 syscall。
// 3. 为了让你在 Mac 上能写代码、跑单元测试、通过 `go build`，我们必须提供一个“哑实现”。
//
// 它的作用：
// 仅仅是为了满足 RuntimeBackend 接口定义，让编译器闭嘴。
// 实际上它什么都做不了，所有方法都只会返回 "not supported" 错误。
type NativeAdapter struct {
	cni *network.CNIManager
}

func NewNativeAdapter(rootPath string, cni *network.CNIManager) (*NativeAdapter, error) {
	return &NativeAdapter{cni: cni}, nil
}

func (n *NativeAdapter) Name() string {
	return "native-stub"
}

func (n *NativeAdapter) PullImage(ctx context.Context, image string) (string, error) {
	return "", fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) RunPodSandbox(ctx context.Context, config *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error) {
	return "", fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) StopPodSandbox(ctx context.Context, podID string) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) RemovePodSandbox(ctx context.Context, podID string) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) PodSandboxStatus(ctx context.Context, podID string) (*runtimeapi.PodSandboxStatus, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) ListPodSandbox(ctx context.Context, filter *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) CreateContainer(ctx context.Context, podSandboxID string, config *runtimeapi.ContainerConfig, sandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
	return "", fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) StartContainer(ctx context.Context, containerID string) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) ContainerStatus(ctx context.Context, containerID string) (*runtimeapi.ContainerStatus, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) ListContainers(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) UpdateContainerResources(ctx context.Context, containerID string, resources *runtimeapi.LinuxContainerResources) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) ExecSync(ctx context.Context, containerID string, cmd []string, timeout int64) (stdout []byte, stderr []byte, err error) {
	return nil, nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) Exec(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) Attach(ctx context.Context, req *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) PortForward(ctx context.Context, req *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) GetContainerCreatedAt(containerID string) (int64, error) {
	return 0, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) ListImages() ([]string, error) {
	return nil, fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) InspectImage(image string) error {
	return fmt.Errorf("native runtime not supported on this OS")
}

func (n *NativeAdapter) GetNetNS(containerID string) (string, error) {
	return "", fmt.Errorf("native runtime not supported on this OS")
}
