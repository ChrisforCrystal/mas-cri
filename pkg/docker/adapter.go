package docker

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DockerAdapter 封装了所有对 Docker CLI 的调用
type DockerAdapter struct{}

// NewAdapter 创建一个新的 DockerAdapter
func NewAdapter() *DockerAdapter {
	return &DockerAdapter{}
}

// SandboxConfig 定义了创建一个 Pod Sandbox 也就是 Pause 容器需要的配置
type SandboxConfig struct {
	Name      string
	Namespace string
	Uid       string
	Image     string // 通常是 pause 镜像
}

// ContainerConfig 定义了创建一个普通业务容器需要的配置
type ContainerConfig struct {
	Name      string
	Image     string
	Command   []string
	Args      []string
}

// Container 代表 docker ps 返回的容器信息
type Container struct {
	ID    string `json:"ID"`
	Names string `json:"Names"`
	Image string `json:"Image"`
	State string `json:"State"` // e.g., "running", "exited"
}

// ==========================================
// Image Operations
// ==========================================

// PullImage 调用 docker pull
func (d *DockerAdapter) PullImage(image string) error {
	logrus.Infof("[Docker] Pulling image: %s", image)
	cmd := exec.Command("docker", "pull", image)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker pull failed: %s (%w)", string(out), err)
	}
	return nil
}

// InspectImage 调用 docker inspect 检查镜像是否存在
func (d *DockerAdapter) InspectImage(image string) error {
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("image not found: %s", string(out))
	}
	return nil
}

// ListImages 调用 docker images (简单版，这里我们暂时只作为透传，实际上CRI需要返回列表)
// 为了简单起见，我们在 server 层可能不会直接用这个，而是主要用于 debug。
// CRI 的 ListImages 比较复杂，我们先集中精力在运行容器上。
func (d *DockerAdapter) ListImages() ([]string, error) {
	// TODO: 实现真正的 ListImages
	return []string{}, nil
}

// ==========================================
// Container Operations
// ==========================================

// RunSandbox 启动一个 Pause 容器作为 Pod 的 Sandbox
// 对应: docker run -d --name k8s_POD_<name>_<ns>_<uid> --net=none <image>
func (d *DockerAdapter) RunSandbox(config *SandboxConfig) (string, error) {
	containerName := fmt.Sprintf("k8s_POD_%s_%s_%s", config.Name, config.Namespace, config.Uid)
	
	args := []string{
		"run", "-d",
		"--name", containerName,
		"--net=none", // 暂时使用 none，不配置网络，或者 host
		config.Image,
	}

	logrus.Infof("[Docker] Running Sandbox: docker %v", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run sandbox: %s (%w)", string(out), err)
	}

	// Docker return full ID on success, trim whitespace
	return strings.TrimSpace(string(out)), nil
}

// CreateContainer 创建一个业务容器，加入 Sandbox 的网络命名空间
// 对应: docker create --name k8s_<name>_<pod_name>... --net=container:<sandbox_id> <image>
// CreateContainer 调用 docker create
// 注意：这里使用的是 'create' 而不是 'run'。
// 因为 CRI 把创建(Create)和启动(Start)分成了两步。
// Kubelet 希望先确保容器资源分配成功（Create），然后再择机启动它（Start）。
func (d *DockerAdapter) CreateContainer(sandboxID string, config *ContainerConfig) (string, error) {
	// 1. 生成容器名: k8s_<容器名>_<SandboxID前6位>
	// 这样我们在 docker ps 里能看出来这个容器属于哪个 Pod
	containerName := fmt.Sprintf("k8s_%s_%s", config.Name, sandboxID[:6])

	args := []string{
		"create",                 // 只创建，不启动 (状态为 Created)
		"--name", containerName,  // 指定名字
		
		// ！！！核心魔法！！！
		// --net=container:<ID> 让这个新容器直接复用 Sandbox (Pause) 容器的网络栈。
		// 这意味着：
		// 1. 它们共享同一个 IP 地址。
		// 2. 它们共享同一个 localhost (一个容器监听 localhost:80，另一个能访问到)。
		// 3. 它们共享同一个端口范围 (不能同时监听 80 端口)。
		fmt.Sprintf("--net=container:%s", sandboxID), 
		
		// 未来还可以加上 IPC 和 PID 共享，那时候它们就真的像在一个虚拟机里了
		// fmt.Sprintf("--ipc=container:%s", sandboxID),
		
		config.Image, // 镜像名 (e.g., nginx:alpine)
	}
	
	// 把用户定义的命令 (Command) 和参数 (Args) 追加到 docker 命令后面
	// 就像 docker create nginx:alpine /bin/sh -c "echo hello"
	args = append(args, config.Command...)
	args = append(args, config.Args...)

	logrus.Infof("[Docker] Creating Container: docker %v", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %s (%w)", string(out), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// StartContainer 启动已创建的容器
// 对应: docker start <id>
func (d *DockerAdapter) StartContainer(containerID string) error {
	logrus.Infof("[Docker] Starting Container: %s", containerID)
	cmd := exec.Command("docker", "start", containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start container: %s (%w)", string(out), err)
	}
	return nil
}

// StopContainer 停止容器
// 对应: docker stop <container_id>
func (d *DockerAdapter) StopContainer(containerID string) error {
	logrus.Infof("[Docker] Stopping Container: %s", containerID)
	cmd := exec.Command("docker", "stop", containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop container: %s (%w)", string(out), err)
	}
	return nil
}

// RemoveContainer 删除容器
// 对应: docker rm <container_id>
// RemovePodSandbox 时需要强制删除(Force)吗？CRI 规范通常先 Stop 再 Remove。
// 这里我们简单实现 docker rm -f (force) 以防万一
func (d *DockerAdapter) RemoveContainer(containerID string) error {
	logrus.Infof("[Docker] Removing Container: %s", containerID)
	cmd := exec.Command("docker", "rm", "-f", containerID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove container: %s (%w)", string(out), err)
	}
	return nil
}


// ListContainers 列出所有容器，用于实现 ListPodSandbox 和 ListContainers
// 对应: docker ps -a --format '{{json .}}'
func (d *DockerAdapter) ListContainers() ([]Container, error) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	var containers []Container
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var c Container
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			logrus.Warnf("Failed to parse docker ps line: %s", line)
			continue // Skip bad lines
		}
		containers = append(containers, c)
	}
	return containers, nil
}

// GetNetNS 获取容器的网络命名空间路径
// 对应: docker inspect -f '{{.NetworkSettings.SandboxKey}}' <container_id>
// 返回值类似: /var/run/docker/netns/xxxx
func (d *DockerAdapter) GetNetNS(containerID string) (string, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.NetworkSettings.SandboxKey}}", containerID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get netns: %s (%w)", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetContainerCreatedAt 获取容器创建时间 (Unix Nano)
func (d *DockerAdapter) GetContainerCreatedAt(containerID string) (int64, error) {
	// docker inspect -f '{{.Created}}' returns RFC3339 format, e.g. "2023-10-27T10:00:00.123456789Z"
	cmd := exec.Command("docker", "inspect", "-f", "{{.Created}}", containerID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to inspect created: %s (%w)", string(out), err)
	}
	
	tsStr := strings.TrimSpace(string(out))
	t, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		// Try without Nano if failed
		t, err = time.Parse(time.RFC3339, tsStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time %s: %w", tsStr, err)
		}
	}
	
	return t.UnixNano(), nil
}
