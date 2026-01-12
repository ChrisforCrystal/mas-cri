package server

import (
	"fmt"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"mascri/pkg/docker"
	"mascri/pkg/native"
	"mascri/pkg/network"
)

// MasCRIServer 是我们 CRI 的核心服务结构体。
// 它组合了 RuntimeService 和 ImageService 的 stub，
// 这样我们也就可以只实现部分方法，而不用一次性实现所有接口（未实现的方法会返回 Unimplemented 错误）。
type MasCRIServer struct {
	runtimeapi.UnimplementedRuntimeServiceServer
	runtimeapi.UnimplementedImageServiceServer

	socketPath string
	backend    RuntimeBackend // Replaced specific docker adapter with interface
	cni        *network.CNIManager
}

// NewMasCRIServer 创建一个新的服务器实例
// socketPath:   gRPC 监听的 Unix Socket 路径 (e.g. /tmp/mascri.sock)
// cniConfigDir: CNI 插件的配置文件目录 (e.g. /etc/cni/net.d)
// cniBinDirs:   CNI 插件的可执行文件目录列表 (e.g. /opt/cni/bin)
// cniCacheDir:  CNI 插件的缓存目录 (e.g. /var/lib/cni)
// runtimeMode:  运行时模式，"native" 或 "docker"
func NewMasCRIServer(socketPath string, cniConfigDir string, cniBinDirs []string, cniCacheDir string, runtimeMode string) *MasCRIServer {
	// 1. 初始化 CNI 管理器 (Networking)
	// CNI 是 Kubernetes 的标准网络接口。我们需要它来给 Pod 分配 IP。
	// 这里通过 network 包加载配置，准备好随时被 Runtime 调用。
	cniMgr, err := network.NewCNIManager(cniConfigDir, cniBinDirs, cniCacheDir)
	if err != nil {
		// 如果 CNI 初始化失败，我们只打印警告，不强制退出。
		// 这样至少还可以运行 HostNetwork 的 Pod 或者仅用于非网络测试。
		logrus.Warnf("Failed to initialize CNI manager: %v. Networking will not work.", err)
	}

	var backend RuntimeBackend
	var initErr error

	// 2. 选择并初始化运行时后端 (Runtime Backend)
	// 策略模式：根据用户传参决定 MasCRI 的“心脏”是谁。
	switch runtimeMode {
	case "native":
		// Native 模式：直接基于 libcontainer (runc) 操作内核。
		// 这是我们自己实现的“简易版 Docker”。
		// 我们将 cniMgr 注入进去，让 NativeAdapter 有能力配置网络。
		logrus.Info("Initializing NATIVE runtime backend (libcontainer)...")
		// /var/lib/mascri 是默认的容器数据存储根目录
		backend, initErr = native.NewNativeAdapter("/var/lib/mascri", cniMgr)
	default: // "docker"
		// Docker 模式：作为 Docker Daemon 的代理。
		// 这种模式下，网络由 Docker 自己管理（bridge network），所以不需要传 cniMgr。
		logrus.Info("Initializing DOCKER runtime backend...")
		backend = docker.NewAdapter()
	}

	if initErr != nil {
		// 如果心脏启动失败（比如 native 模式下无法创建目录），直接 Fatal 退出。
		logrus.Fatalf("Failed to initialize backend %s: %v", runtimeMode, initErr)
	}

	// 3. 返回构造好的 Server 对象
	// 这个对象随后会被注册到 gRPC Server 中。
	return &MasCRIServer{
		socketPath: socketPath,
		backend:    backend,
		cni:        cniMgr,
	}
}

// Start 启动 gRPC 服务器并阻塞等待
func (s *MasCRIServer) Start() error {
	// 1. 准备 Unix Domain Socket
	// 如果 socket 文件已经存在，必须先删除，否则 Listen 会报错 "bind: address already in use"
	if _, err := os.Stat(s.socketPath); err == nil {
		logrus.Infof("Removing existing socket file: %s", s.socketPath)
		if err := os.Remove(s.socketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket: %w", err)
		}
	}

	// 2. 监听 Socket
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket %s: %w", s.socketPath, err)
	}
	logrus.Infof("MasCRI listening on %s", s.socketPath)

	// 3. 创建 gRPC Server 并注册拦截器
	// WithUnaryInterceptor 将我们的日志记录器注入到服务器中
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryInterceptor),
	)

	// 4. 注册 CRI 服务
	// 告诉 gRPC 框架：在这个 Server 上，RuntimeService 和 ImageService 由 s (MasCRIServer) 来处理
	runtimeapi.RegisterRuntimeServiceServer(grpcServer, s)
	runtimeapi.RegisterImageServiceServer(grpcServer, s)

	// 5. 开始服务
	return grpcServer.Serve(listener)
}
