package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// UnaryInterceptor 是一个 gRPC 中间件，用于拦截并记录所有的请求。
// 它的作用就像一个海关检查站，所有进出的“货物”（请求）都要在这里登记。
func UnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// 1. 尝试将请求体转换为 JSON 字符串，以便人类阅读
	// 我们关心的不是二进制数据，而是 Kubelet 到底传了什么参数给我们
	jsonBytes, err := json.Marshal(req)
	var reqLog string
	if err != nil {
		reqLog = "failed to marshal request"
	} else {
		reqLog = string(jsonBytes)
	}

	// 2. 打印请求日志
	// FullMethod 例如: /runtime.v1.RuntimeService/RunPodSandbox
	logrus.WithFields(logrus.Fields{
		"method": info.FullMethod,
		"body":   reqLog,
	}).Info("--> [gRPC Request]")

	// 3. 调用真正的处理函数 (handler)
	// 这里的 resp 就是具体的 RPC 方法（如 Version, RunPodSandbox）返回的结果
	resp, err := handler(ctx, req)

	// 4. 记录处理耗时和错误
	duration := time.Since(start)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"method":   info.FullMethod,
			"duration": duration,
			"error":    err,
		}).Error("<-- [gRPC Error]")
	} else {
		// 如果你想看响应内容，也可以在这里 Marshal resp，但通常请求内容更重要
		logrus.WithFields(logrus.Fields{
			"method":   info.FullMethod,
			"duration": duration,
		}).Info("<-- [gRPC Response]")
	}

	return resp, err
}
