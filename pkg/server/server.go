package server

import (
	"fmt"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"mascri/pkg/docker"
)

// MasCRIServer 是我们 CRI 的核心服务结构体。
// 它组合了 RuntimeService 和 ImageService 的 stub，
// 这样我们也就可以只实现部分方法，而不用一次性实现所有接口（未实现的方法会返回 Unimplemented 错误）。
type MasCRIServer struct {
	runtimeapi.UnimplementedRuntimeServiceServer
	runtimeapi.UnimplementedImageServiceServer

	socketPath string
	docker     *docker.DockerAdapter
}

// NewMasCRIServer 创建一个新的服务器实例
func NewMasCRIServer(socketPath string) *MasCRIServer {
	return &MasCRIServer{
		socketPath: socketPath,
		docker:     docker.NewAdapter(),
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
