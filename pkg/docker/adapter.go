package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// DockerAdapter 封装了所有对 Docker CLI 的调用
type DockerAdapter struct{}

// NewAdapter 创建一个新的 DockerAdapter
func NewAdapter() *DockerAdapter {
	return &DockerAdapter{}
}

// ==========================================
// Image Operations
// ==========================================

// PullImage 调用 docker pull
func (d *DockerAdapter) PullImage(ctx context.Context, image string) (string, error) {
	logrus.Infof("[Docker] Pulling image: %s", image)
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker pull failed: %s (%w)", string(out), err)
	}
	// Return image ref as is for now
	return image, nil
}

// InspectImage 调用 docker inspect 检查镜像是否存在 (Internal helper, not in main interface but used by List/Status if needed)
func (d *DockerAdapter) InspectImage(image string) error {
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("image not found: %s", string(out))
	}
	return nil
}

func (d *DockerAdapter) ListImages() ([]string, error) {
    return []string{}, nil
}

// ==========================================
// Pod Sandbox Operations
// ==========================================

// RunPodSandbox 启动一个 Pause 容器作为 Pod 的 Sandbox
func (d *DockerAdapter) RunPodSandbox(ctx context.Context, config *cri.PodSandboxConfig, runtimeHandler string) (string, error) {
	name := config.Metadata.Name
	namespace := config.Metadata.Namespace
	uid := config.Metadata.Uid
	// Image := config.Image // Error: PodSandboxConfig has no Image field.
	image := "registry.k8s.io/pause:3.9" // Hardcoded default for Docker backend

	// Format: k8s_POD_<name>_<ns>_<uid>
	containerName := fmt.Sprintf("k8s_POD_%s_%s_%s", name, namespace, uid)
	
	// Ensure image is present (simple check)
	// In real CRI, image service handles this, but here adapter does a sanity check?
	// Actually runtime calls PullImage first usually.

	args := []string{
		"run", "-d",
		"--name", containerName,
		"--net=none", // Use CNI
		image,
	}

	logrus.Infof("[Docker] Running Sandbox: docker %v", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run sandbox: %s (%w)", string(out), err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (d *DockerAdapter) StopPodSandbox(ctx context.Context, podID string) error {
	return d.StopContainer(ctx, podID, 10)
}

func (d *DockerAdapter) RemovePodSandbox(ctx context.Context, podID string) error {
	return d.RemoveContainer(ctx, podID)
}

func (d *DockerAdapter) PodSandboxStatus(ctx context.Context, podID string) (*cri.PodSandboxStatus, error) {
	// Simple implementation reusing ContainerStatus logic concept
	// Get createdAt
	created, err := d.GetContainerCreatedAt(podID)
	if err != nil {
		return nil, err
	}
	
	// We need to parse metadata from name if possible, or just return basic
	// For now, let's keep it simple. Real impl would parse `docker inspect` labels/name.
	// Assuming podID is the container ID.
	
	return &cri.PodSandboxStatus{
		Id: podID,
		State: cri.PodSandboxState_SANDBOX_READY,
		CreatedAt: created,
		Metadata: &cri.PodSandboxMetadata{
			Name: "unknown", // Todo: parse from docker name
		},
		Network: &cri.PodSandboxNetworkStatus{
			Ip: "127.0.0.1", // Mocked for now until CNI integ flows back
		},
	}, nil
}

func (d *DockerAdapter) ListPodSandbox(ctx context.Context, filter *cri.PodSandboxFilter) ([]*cri.PodSandbox, error) {
	// Not implemented fully yet
	return []*cri.PodSandbox{}, nil
}


// ==========================================
// Container Operations
// ==========================================

func (d *DockerAdapter) CreateContainer(ctx context.Context, podID string, config *cri.ContainerConfig, sandboxConfig *cri.PodSandboxConfig) (string, error) {
	// docker create --net=container:<pod_id> ...
	
	containerName := config.Metadata.Name
	// k8s_<container_name>_<pod_name>_<ns>_<pod_uid>_0
	dockerContainerName := fmt.Sprintf("k8s_%s_%s_%s_%s_0", 
		containerName, 
		sandboxConfig.Metadata.Name, 
		sandboxConfig.Metadata.Namespace, 
		sandboxConfig.Metadata.Uid)

	args := []string{
		"create",
		"--name", dockerContainerName,
		fmt.Sprintf("--net=container:%s", podID),
	}
	
	if config.Command != nil {
		args = append(args, "--entrypoint", config.Command[0])
	}
	
	args = append(args, config.Image.Image)
	
	if config.Command != nil && len(config.Command) > 1 {
		args = append(args, config.Command[1:]...)
	}
	if config.Args != nil {
		args = append(args, config.Args...)
	}

	logrus.Infof("[Docker] Creating Container: docker %v", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s (%w)", string(out), err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (d *DockerAdapter) StartContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "start", containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start container: %s (%w)", string(out), err)
	}
	return nil
}

func (d *DockerAdapter) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", fmt.Sprintf("%d", timeout), containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Ignore if already stopped/gone? No, return error
		return fmt.Errorf("failed to stop container: %s (%w)", string(out), err)
	}
	return nil
}

func (d *DockerAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove container: %s (%w)", string(out), err)
	}
	return nil
}

func (d *DockerAdapter) ContainerStatus(ctx context.Context, containerID string) (*cri.ContainerStatus, error) {
	// Call docker inspect
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	// Parse status... simplified
	state := cri.ContainerState_CONTAINER_UNKNOWN
	if strings.Contains(string(out), "\"Running\": true") {
		state = cri.ContainerState_CONTAINER_RUNNING
	} else {
		state = cri.ContainerState_CONTAINER_EXITED
	}
	
	return &cri.ContainerStatus{
		Id: containerID,
		State: state,
	}, nil
}

func (d *DockerAdapter) ListContainers(ctx context.Context, filter *cri.ContainerFilter) ([]*cri.Container, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.State}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	
	var containers []*cri.Container
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" { continue }
		parts := strings.Split(line, "|")
		if len(parts) < 4 { continue }
		
		// Map logic to CRI container
		// Simplification: just list them
		c := &cri.Container{
			Id: parts[0],
			Metadata: &cri.ContainerMetadata{Name: parts[1]},
			Image: &cri.ImageSpec{Image: parts[2]},
			State: cri.ContainerState_CONTAINER_UNKNOWN,
		}
		containers = append(containers, c)
	}
	return containers, nil
}


// ==========================================
// Helpers
// ==========================================

func (d *DockerAdapter) GetNetNS(containerID string) (string, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.NetworkSettings.SandboxKey}}", containerID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get netns: %s (%w)", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *DockerAdapter) GetContainerCreatedAt(containerID string) (int64, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.Created}}", containerID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to inspect created: %s (%w)", string(out), err)
	}
	
	tsStr := strings.TrimSpace(string(out))
	t, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		t, err = time.Parse(time.RFC3339, tsStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time %s: %w", tsStr, err)
		}
	}
	return t.UnixNano(), nil
}
