//go:build linux

package native

import (
	"context"
	"fmt"
	"mascri/pkg/network"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/sirupsen/logrus"

	cri "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// NativeAdapter implements a container runtime using libcontainer directly on Linux.
type NativeAdapter struct {
	// factory 是 libcontainer 的核心接口。
	// 它封装了所有与 Linux 内核交互的脏活累活。
	// 当我们调用 factory.Create() 时，它并不直接创建进程，而是：
	// 1. 在 /run/mascri/<id> 下创建状态目录。
	// 2. 准备好 Namespace 和 Cgroups 的配置。
	// 真正的 PID 1 进程是在 container.Run() 时通过系统调用 (clone/exec) 产生的。
	factory libcontainer.Factory
	rootDir string // /var/lib/mascri
	cni     *network.CNIManager
}

// NewNativeAdapter creates a new native adapter.
func NewNativeAdapter(rootDir string, cni *network.CNIManager) (*NativeAdapter, error) {
	// State dir: where libcontainer stores process state (json)
	// 1. 初始化状态目录 (State Directory)
	// libcontainer 需要一个地方来存放容器的运行时状态文件（比如 state.json）。
	// 这些文件记录了容器的 PID、创建时间、配置等关键信息，以便容器重启或断电后能恢复状态。
	// 通常位于 /run 目录下（内存文件系统），重启后清空。
	stateDir := "/run/mascri"
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, err
	}
	
	// 2. 获取自身二进制路径 (Self Executable Path)
	// 这里是 libcontainer "Re-exec" 机制的关键。
	// 当我们创建一个新容器时，其实是再次运行了当前这个 mascri 二进制程序，
	// 只不过给了它特殊的参数（"init"），让它去执行容器初始化的逻辑。
	// 必须使用绝对路径，确保在 sudo 或不同工作目录下都能找到自己。
	selfExe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get self executable path: %w", err)
	}

	// 3. 创建 libcontainer 工厂 (Factory)
	// Factory 是创建具体容器实例的工厂。
	// libcontainer.New 需要两个参数：
	// - stateDir: 状态存储位置
	// - InitArgs: 告诉 libcontainer，当需要作为容器内的 "init" 进程启动时，应该运行哪个命令。
	//   【关键点】：这里我们填入 selfExe 和 "init"。
	//   这意味着子进程启动时，实际执行的是 `/path/to/mascri init`。
	//   这就直接导致了 os.Args[1] == "init"，从而触发了 func init() 里的拦截逻辑。
	factory, err := libcontainer.New(stateDir, libcontainer.InitArgs(selfExe, "init"))
	if err != nil {
		return nil, fmt.Errorf("failed to create libcontainer factory: %w", err)
	}

	return &NativeAdapter{
		factory: factory,
		rootDir: rootDir,
		cni:     cni,
	}, nil
}

// 这个函数在 main() 之前执行！
// 问：为什么 init 会执行两次？
// 答：因为它是在**两个完全不同的进程**里分别执行的！
// 
// 第一次执行：你手动启动 ./mascri Server 时。
//   - 进程：Server 进程 (PID 100)
//   - 参数：无
//   - 结果：init() 检查发现没参数，直接通过 -> 接着执行 main() 启动服务器。
//
// 第二次执行：Server 代码调用 Create -> Run 时，fork 了一个新进程。
//   - 进程：Container Init 进程 (PID 200)
//   - 参数：mascri init
//   - 结果：init() 检查发现有参数，**拦截执行流** -> 执行 StartInitialization -> 变成了 /bin/sh -> 永远不执行 main()。
func init() {
	// 检查：我是不是被复用为容器的 Init 进程？
	// 当 NativeAdapter 创建容器时，会执行 /proc/self/exe init
	if len(os.Args) > 1 && os.Args[1] == "init" {
		// 为了调试方便，我们在 /tmp 下记录日志（因为容器启动早期很难看到 stdout）
		f, _ := os.OpenFile("/tmp/mascri-init.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		f.WriteString(fmt.Sprintf("Init called with args: %v\n", os.Args))
		
		// 容器 Init 进程必须是单线程的，并且绑定到 OS 线程。
		// 这是因为 Setns 等 Namespace 操作是线程级别生效的，Go 的多线程调度会破坏这一点。
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		
		// 创建一个空的 factory 实例，仅仅为了调用 StartInitialization
		factory, _ := libcontainer.New("")
		
		// StartInitialization 是 libcontainer 的核心魔术。
		// 
		// 核心概念：
		// 1. 【出生】当代码运行到这里时，当前进程其实**已经**处于一个新的 Namespace 里了。
		//    (这是父进程在 fork 我们的时候，通过 clone 系统调用帮我们创建好的“空房间”)。
		//    
		// 2. 【装修】StartInitialization() 的作用就是根据父进程传过来的配置，
		//    对这个空房间进行“装修”（挂载 /proc，设置 hostname，切断对宿主机的访问）。
		//    
		// 3. 【入住】最后通过 exec 系统调用，把自己替换成用户指定的命令 (如 /bin/sh)。
		// 
		// 注意：如果成功，StartInitialization 永远不会返回！
		if err := factory.StartInitialization(); err != nil {
			f.WriteString(fmt.Sprintf("StartInitialization failed: %v\n", err))
			logrus.Fatal(err)
		}
		
		// 如果代码执行到这里，说明 exec 失败了（比如找不到 /bin/sh）。
		panic("--this line should have never been executed, congratulations--")
	}
}

// CreateContainer prepares the container configuration (namespaces, cgroups, rootfs)
// CreateContainer 准备容器的静态配置，但还不会运行进程。
func (n *NativeAdapter) CreateContainer(ctx context.Context, podID string, config *cri.ContainerConfig, sandboxConfig *cri.PodSandboxConfig) (string, error) {
	containerID := config.Metadata.Name // 简单起见，直接用名字作为 ID
	
	// 1. 准备 Rootfs (根文件系统)
	// 容器需要一个隔离的文件系统视图。
	// 这里我们简化处理，直接解压一个 busybox.tar 到指定目录作为容器的根InspectImage目录。
	// 在生产级 Runtime 中，这里应该调用 GraphDriver (如 overlay2) 来通过镜像分层构建。
	imagePath := "/tmp/busybox.tar" 
	containerRoot := filepath.Join(n.rootDir, "containers", containerID)
	rootfs := filepath.Join(containerRoot, "rootfs")
	
	if err := SetupRootfs(imagePath, rootfs); err != nil {
		return "", fmt.Errorf("failed to setup rootfs: %w", err)
	}

	// 2. 生成 libcontainer 配置 (Define Configuration)
	// 将 CRI 的需求（资源、挂载点等）转换为 libcontainer 能理解的底层配置对象。
	// 这里面包含了 Cgroups 限制、Namespace 设定、Capabilities 权限等。
	lcConfig := n.getLibcontainerConfig(containerID, rootfs)
	
	// 3. 创建容器实例 (Create Container)
	// 这一步是核心：它告诉 libcontainer 工厂“请按照这个配置，在磁盘上把容器建立起来”。
	// 具体做了什么：
	// - 在 /run/mascri/<id>/ 下创建目录。
	// - 把 lcConfig 序列化成 config.json 存进去（持久化）。
	// - 注意：这里**还没有**创建任何 Linux 进程！
	//   它只是把元数据准备好了。真正跑起来要等 StartContainer 或 RunPodSandbox 调用 container.Run()。
	// 如果之前有残留的状态文件，先清理掉。
	// 问：为什么 CreateContainer 和 RunPodSandbox 看起来逻辑一样？
	// 答：
	// 1. RunPodSandbox 创建的是 "Pause" 容器（Infrastructure Container）。
	//    它的任务是创建并持有 Namespaces（特别是 Network NS）。
	// 2. CreateContainer 创建的是 "业务" 容器（Workload Container）。
	// 
	// 【纠正误区】：注意这行 create 代码**不会**克隆进程！
	// 它只是在磁盘上生成了 "state.json" 和 "config.json"，把 New() 时定义的
	// "init" 参数写进去。真正的 fork 动作要等到后面 container.Run()。
	os.RemoveAll(filepath.Join("/run/mascri", containerID))
	
	// 这里只是创建元数据 (Metadata Only)
	_, err := n.factory.Create(containerID, lcConfig)
	//    在成熟实现中，业务容器不会创建新的 Network NS，而是**加入** Pause 容器的 NS。
	// 
	// 目前为了演示方便，我们在 getLibcontainerConfig 里简化了“加入 Namespace”的逻辑，
	// 所以看起来它们都是“创建一个新容器”。但物理上，它们确实是两个独立的 Linux 进程。

	if err != nil {
		return "", fmt.Errorf("libcontainer create failed: %w", err)
	}

	logrus.Infof("[Native] Container %s created successfully", containerID)
	return containerID, nil
}

const defaultMountFlags = syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

func (n *NativeAdapter) StartContainer(ctx context.Context, containerID string) error {
	// 1. 加载容器状态 (Load container)
	// 因为 CreateContainer 只是把配置写到了磁盘上，这里我们需要把那个配置读回来，
	// 恢复出一个 container 对象。
	container, err := n.factory.Load(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}
	
	// 2. 定义要运行的进程 (Define Process)
	// 这里就是真正要在容器里跑的命令。
	// 相当于 Docker 里的 ENTRYPOINT/CMD。
	// 在这个简单的演示里，我们硬编码如下命令：
	// Command: /bin/sh
	// Args:    -c "echo Hello from Native Linux Container! && sleep 3600"
	process := &libcontainer.Process{
		Args:   []string{"/bin/sh", "-c", "echo Hello from Native Linux Container! && sleep 3600"},
		Env:    []string{"PATH=/bin:/usr/bin:/sbin:/usr/sbin"},
		User:   "root",
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Init:   true, // Init: true 表示这是容器内的 1 号进程，它死则容器死。
		Cwd:    "/",
	}
	
	// 3. 运行进程 (Run)
	// 这里的 Run() 到底做了什么？
	// a. 它会 fork 一个子进程（就是我们开头看到的 selfExe "init"）。
	// b. 子进程通过 setns/unshare 系统调用，进入 Create 阶段定义的各种 Namespace。
	// c. 子进程 pivot_root 切换根目录到 rootfs。
	// d. 最后 execve() 替换成上面的 "/bin/sh" 命令。
	// e. 此时，你的终端上应该能看到 "Hello from Native Linux Container!"。
	//
	// 【关键回答】：这里就是“按钮”。
	// 这一行代码执行下去，底层就会触发 clone 系统调用，
	// 一个新的 PID 就会在宿主机上诞生！
	if err := container.Run(process); err != nil {
		// Cleanup if run fails
		container.Destroy()
		return fmt.Errorf("failed to run process: %w", err)
	}
	
	return nil
}

// RunPodSandbox Creates and Starts a Pause container
// RunPodSandbox 本质上是创建一个 "Pause" 容器。
// "Pause" 容器是 Pod 里第一个启动的容器，它的生命周期等同于整个 Pod。
// 它的作用是占住 Linux Namespace（尤其是 Network Namespace），
// 后面创建的业务容器都会加入到这个 Pause 容器的 Namespace 里，从而实现网络共享。
func (n *NativeAdapter) RunPodSandbox(ctx context.Context, config *cri.PodSandboxConfig, runtimeHandler string) (string, error) {
	// 构造 Pod ID。这里为了调试方便，包含了 Name/Namespace/Uid。
	// 实际上 CRI 规范只要求返回这里生成的 ID，后续操作都用这个 ID。
	podID := fmt.Sprintf("k8s_POD_%s_%s_%s", config.Metadata.Name, config.Metadata.Namespace, config.Metadata.Uid)
	
	// 1. 准备 Rootfs (Pause 容器也要有文件系统)
	// 我们再次复用 busybox.tar。Pause 镜像通常非小（几百KB），这里只是为了演示。
	imagePath := "/tmp/busybox.tar" 
	containerRoot := filepath.Join(n.rootDir, "containers", podID)
	// /var/lib/mascri/containers/k8s_POD_nginx_default_a1b2c3d4/rootfs
	/**
		/var/lib/mascri/containers/k8s_POD_.../rootfs/
		├── bin/          <-- 存放常用命令，如 sh, ls, cat, echo
		│   ├── sh
		│   ├── ls
		│   └── ...
		├── dev/          <-- 设备文件目录（初始可能是空的，容器启动时会挂载）
		├── etc/          <-- 配置文件
		│   ├── hostname
		│   └── hosts
		├── proc/         <-- 进程信息（空目录，待挂载）
		├── sys/          <-- 系统信息（空目录，待挂载）
		├── tmp/
		├── usr/
		└── var/
	**/
	rootfs := filepath.Join(containerRoot, "rootfs")
	
	if err := SetupRootfs(imagePath, rootfs); err != nil {
		return "", fmt.Errorf("failed to setup rootfs for sandbox: %w", err)
	}

	// 2. 生成 libcontainer 配置
	// 对于 Pause 容器，这个配置就是整个 Pod 的“基准配置”。
	// 也就是在这里，我们决定了 Pod 的 IP、Cgroup 父节点等。
	lcConfig := n.getLibcontainerConfig(podID, rootfs)
	
	// 3. 创建并启动 Pause 容器
	// 清理旧状态
	os.RemoveAll(filepath.Join("/run/mascri", podID))
	
	container, err := n.factory.Create(podID, lcConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create sandbox container: %w", err)
	}
	
	// 定义 Pause 容器运行的进程。
	// 标准的 pause 镜像会执行一个可以响应信号的死循环汇编程序。
	// 这里我们用 Shell 模拟：既然是占位符，只要不退出就行。
	process := &libcontainer.Process{
		Args: []string{"/bin/sh", "-c", "echo Pod Sandbox (Pause) && sleep infinity"},
		Env:  []string{"PATH=/bin"},
		Init: true, // 这是容器内的 1 号进程
		Cwd:  "/",
	}
	
	// 这里真正调用 StartInitialization -> exec，启动容器进程。
	if err := container.Run(process); err != nil {
		container.Destroy()
		return "", fmt.Errorf("failed to start sandbox process: %w", err)
	}
	
	logrus.Infof("[Native] Pod Sandbox %s launched", podID)
	
	// 4. 配置网络 (CNI Setup)
	// 到了这一步，容器进程已经跑起来了，意味着 Network Namespace 已经创建完毕。
	// 是时候叫 CNI 插件过来插网线了。
	if n.cni != nil {
		// 先获取那个新创建的 Network Namespace 的路径 (/proc/<pid>/ns/net)
		netnsPath, err := n.GetNetNS(podID)
		if err != nil {
			return podID, fmt.Errorf("failed to get netns for sandbox: %w", err)
		}
		if netnsPath != "" {
			// 调用 CNI 插件：把 eth0 插进去，分配 IP。
			if _, err := n.cni.SetUpPod(ctx, podID, netnsPath); err != nil {
				return podID, fmt.Errorf("failed to setup network for sandbox: %w", err)
			}
		}
	}

	return podID, nil
}

func (n *NativeAdapter) StopPodSandbox(ctx context.Context, podID string) error {
	// Teardown Networking first
	if n.cni != nil {
		// We try to get netns even if container is stopped/stopping
		netnsPath, err := n.GetNetNS(podID)
		if err == nil && netnsPath != "" {
			if err := n.cni.TearDownPod(ctx, podID, netnsPath); err != nil {
				logrus.Warnf("Failed to teardown network for pod %s: %v", podID, err)
			}
		}
	}

	return n.StopContainer(ctx, podID, 0)
}

func (n *NativeAdapter) RemovePodSandbox(ctx context.Context, podID string) error {
	return n.RemoveContainer(ctx, podID)
}

func (n *NativeAdapter) ListPodSandbox(ctx context.Context, filter *cri.PodSandboxFilter) ([]*cri.PodSandbox, error) {
	return nil, nil
}

func (n *NativeAdapter) PodSandboxStatus(ctx context.Context, podID string) (*cri.PodSandboxStatus, error) {
	// Not used by runtime.go (it uses ListContainers)
	return nil, nil
}

func (n *NativeAdapter) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	logrus.Infof("[Native] Stopping container %s", containerID)
	container, err := n.factory.Load(containerID)
	if err != nil {
		// If not found, assume stopped
		return nil
	}
	
	// Kill init process
	return container.Signal(syscall.SIGKILL, true)
}

func (n *NativeAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	logrus.Infof("[Native] Removing container %s", containerID)
	// Remove libcontainer state by removing directory
	os.RemoveAll(filepath.Join("/run/mascri", containerID))

	// Remove rootfs files
	containerRoot := filepath.Join(n.rootDir, "containers", containerID)
	os.RemoveAll(containerRoot)
	return nil
}

func (n *NativeAdapter) ContainerStatus(ctx context.Context, containerID string) (*cri.ContainerStatus, error) {
	// Not fully used yet, runtime.go uses ListContainers logic
	return nil, nil
}

func (n *NativeAdapter) ListContainers(ctx context.Context, filter *cri.ContainerFilter) ([]*cri.Container, error) {
	// Implement simple listing by checking state dir
	// State dir is /run/mascri/<id>/state.json
	stateDir := "/run/mascri"
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, err
	}
	
	var containers []*cri.Container
	for _, e := range entries {
		if e.IsDir() {
			id := e.Name()
			// Check if running?
			// Just list everything found in state dir
			containers = append(containers, &cri.Container{
				Id: id,
				Metadata: &cri.ContainerMetadata{
					Name: id, // Use ID as name for native
				},
				State: cri.ContainerState_CONTAINER_RUNNING, // Assume running if state exists
			})
		}
	}
	return containers, nil
}

func (n *NativeAdapter) GetContainerCreatedAt(containerID string) (int64, error) {
	return time.Now().UnixNano(), nil // Mock
}

func (n *NativeAdapter) ListImages() ([]string, error) {
	return []string{"busybox"}, nil
}

func (n *NativeAdapter) InspectImage(image string) error {
	// Assume always valid if it matches our pattern
	return nil
}

// GetNetNS 获取容器的网络 Namespace 路径。
// 这个路径是 CNI 插件工作的必要参数。
func (n *NativeAdapter) GetNetNS(containerID string) (string, error) {
	// 1. 从磁盘 (/run/mascri/<id>/state.json) 加载容器状态。
	// 这里再次体现了 factory 的作用：它是通往容器状态的唯一入口。
	container, err := n.factory.Load(containerID)
	if err != nil {
		return "", err
	}
	
	// 2. 获取容器当前的运行时状态（包含 PID 等）。
	state, err := container.State()
	if err != nil {
		return "", err
	}
	
	// 3. 构建 NetNS 路径。
	// Linux 的每个进程都在 /proc/<pid>/ns/net 下暴露了它的网络 Namespace 文件。
	// CNI 插件只要拿到这个文件路径，就能通过 setns() 系统调用进入这个命名空间，
	// 然后在里面创建 eth0 网卡、配置 IP。
	if state.InitProcessPid > 0 {
		return fmt.Sprintf("/proc/%d/ns/net", state.InitProcessPid), nil
	}
	return "", nil
}

// PullImage 拉取镜像。
// 在 Native 模式下，目前我们**不实现**真正的镜像拉取功能。
// 原因：
// 1. 实现一个完整的 OCI 镜像下载、校验、解压、分层存储 (OverlayFS) 是一个巨大的工程（相当于重写半个 Docker）。
// 2. 这里的重点是展示 Runtime 和 Networking，因此我们简化为使用本地已有的 "/tmp/busybox.tar"。
// 也就是说，无论 Kubelet 叫我们拉什么镜像，我们都假装拉好了，实际上后面 CreateContainer 时只会用那个 tar 包。
func (n *NativeAdapter) PullImage(ctx context.Context, image string) (string, error) {
	// No-op: 假装成功
	return image, nil
}


// Internal helper for config
// getLibcontainerConfig 生成 libcontainer 底层配置。
// 这是 Native Runtime 的核心：即使没有 Docker，我們也能通过 libcontainer
// 精确控制容器的每一个细节（权限、隔离、资源限制、挂载点）。
func (n *NativeAdapter) getLibcontainerConfig(id string, rootfs string) *configs.Config {
	return &configs.Config{
		Rootfs: rootfs,
		
		// 1. Capabilities (权限控制)
		// Linux Capabilities 将 root 的上帝权限拆分成细粒度的权限。
		// 这里列出的是一个标准的“白名单”，允许大多数普通应用运行，
		// 但禁止了某些危险操作（如加载内核模块、系统时间修改等）。
		Capabilities: &configs.Capabilities{
			// Bounding Set 是权限能力的上限（天花板）。
			// 这里列出的是 Docker 默认的允许列表，包含了一些基础管理能力：
			// - CAP_CHOWN/FOWNER/DAC_OVERRIDE: 既然是 root，当然要能随便改文件权限。
			// - CAP_NET_BIND_SERVICE: 允许绑定 80/443 等低端口（普通用户只能绑 1024 以上）。
			// - CAP_NET_RAW: 允许 ping (ICMP)。
			// - CAP_SETUID/SETGID: 允许程序降权运行（比如 sudo 里的逻辑）。
			// - CAP_SYS_CHROOT: 允许 chroot。
			// - CAP_KILL: 允许杀进程。
			// 
			// 关键点在于**没有**什么：
			// - 没有 CAP_SYS_ADMIN: 这是真正的“上帝权限”，能挂载文件系统、配置交换分区等。
			// - 没有 CAP_NET_ADMIN: 不能配置 IP、防火墙 (iptables)。这正是为什么我们需要 CNI 插件在外部替它配好！
			Bounding:    []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD", "CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP", "CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE"},
			Effective:   []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD", "CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP", "CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE"},
			Inheritable: []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD", "CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP", "CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE"},
			Permitted:   []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD", "CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP", "CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE"},
			Ambient:     []string{"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD", "CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP", "CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE"},
		},
		
		// 2. Namespaces (隔离空间)
		// 这里定义了容器需要哪些类型的隔离。
		// NEWNS:  Mount Namespace (文件系统挂载点隔离)
		// NEWUTS: UTS Namespace (Hostname 隔离)
		// NEWIPC: IPC Namespace (信号量、消息队列隔离)
		// NEWPID: PID Namespace (进程编号隔离，容器内 PID 从 1 开始)
		// NEWNET: Network Namespace (网络协议栈隔离)。
		Namespaces: configs.Namespaces([]configs.Namespace{
			{Type: configs.NEWNS},
			{Type: configs.NEWUTS},
			{Type: configs.NEWIPC},
			{Type: configs.NEWPID},
			// {Type: configs.NEWNET}, // 暂时注释掉，方便调试（使用 Host 网络）。若开启则需要 CNI 配合。
		}),
		
		// 3. Cgroups (资源限制)
		// 限制容器的资源使用，防止它耗尽宿主机资源。
		Cgroups: &configs.Cgroup{
			Name:   id,
			Parent: "mascri", // 在 /sys/fs/cgroup/.../mascri 下创建
			Resources: &configs.Resources{
				Memory: 1024 * 1024 * 64, // 限制内存：64MB
			},
		},
		
		// 4. 安全屏蔽 (MaskPaths & ReadonlyPaths)
		// 对于 /proc 下某些泄露宿主机内核信息的敏感路径，我们要么隐藏，要么只读。
		MaskPaths: []string{
			"/proc/kcore",
		},
		ReadonlyPaths: []string{
			"/proc/sys", "/proc/sysrq-trigger", "/proc/irq", "/proc/bus",
		},
		
		// 5. 关键挂载 (Mounts)
		// 除了根文件系统，Linux 容器还需要挂载一些虚拟文件系统才能正常工作。
		Mounts: []*configs.Mount{
			{
				Source:      "proc", // 挂载 procfs
				Destination: "/proc",
				Device:      "proc",
				Flags:       defaultMountFlags,
			},
			{
				Source:      "tmpfs", // 挂载 tmpfs 到 /dev
				Destination: "/dev",
				Device:      "tmpfs",
				Flags:       syscall.MS_NOSUID | syscall.MS_STRICTATIME,
				Data:        "mode=755",
			},
		},
		
		// 6. 设备节点 (Devices)
		// 在容器内的 /dev 目录下创建必要的设备文件，确保常用程序能运行。
		// 如 /dev/null (黑洞), /dev/zero (零生成器), /dev/random (随机数)
		Devices:  []*devices.Device{
			{Rule: devices.Rule{Type: 'c', Major: 1, Minor: 3, Permissions: "rwm", Allow: true}, Path: "/dev/null", FileMode: 0666, Uid: 0, Gid: 0},
			{Rule: devices.Rule{Type: 'c', Major: 1, Minor: 5, Permissions: "rwm", Allow: true}, Path: "/dev/zero", FileMode: 0666, Uid: 0, Gid: 0},
			{Rule: devices.Rule{Type: 'c', Major: 1, Minor: 7, Permissions: "rwm", Allow: true}, Path: "/dev/full", FileMode: 0666, Uid: 0, Gid: 0},
			{Rule: devices.Rule{Type: 'c', Major: 5, Minor: 0, Permissions: "rwm", Allow: true}, Path: "/dev/tty", FileMode: 0666, Uid: 0, Gid: 0},
			{Rule: devices.Rule{Type: 'c', Major: 1, Minor: 8, Permissions: "rwm", Allow: true}, Path: "/dev/random", FileMode: 0666, Uid: 0, Gid: 0},
			{Rule: devices.Rule{Type: 'c', Major: 1, Minor: 9, Permissions: "rwm", Allow: true}, Path: "/dev/urandom", FileMode: 0666, Uid: 0, Gid: 0},
		},
		Hostname: id,
		Version: "1.0.2",
	}
}
