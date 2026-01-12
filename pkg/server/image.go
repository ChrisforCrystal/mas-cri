package server

import (
	"context"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// PullImage 下载镜像
// 必须实现，否则无法运行容器
func (s *MasCRIServer) PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error) {
	imageName := req.GetImage().GetImage()
	
	// 如果没有指定 tag，CRI 可能会传 "nginx" 而不是 "nginx:latest"，
	// 但 docker pull 默认处理 "nginx" 为 "nginx:latest"，所以直接透传通常没问题。
	
	if _, err := s.backend.PullImage(ctx, imageName); err != nil {
		return nil, err
	}

	return &runtimeapi.PullImageResponse{
		ImageRef: imageName, // 返回拉取成功的镜像名
	}, nil
}

// ListImages 列出镜像
// 目前只是为了满足接口，不一定会返回完整列表（因为 Adapter 没实现完整解析）
func (s *MasCRIServer) ListImages(ctx context.Context, req *runtimeapi.ListImagesRequest) (*runtimeapi.ListImagesResponse, error) {
	// Adapter 里这个方法是个空的，但我们调用一下，表示逻辑通了
	_, err := s.backend.ListImages()
	if err != nil {
		return nil, err
	}

	return &runtimeapi.ListImagesResponse{
		Images: []*runtimeapi.Image{}, // 返回空列表
	}, nil
}

// ImageStatus 检查镜像状态
// Kubelet 在拉取镜像前会先问一下：“我有这个镜像吗？”
func (s *MasCRIServer) ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error) {
	imageName := req.GetImage().GetImage()
	
	status := &runtimeapi.ImageStatusResponse{
		Image: &runtimeapi.Image{
			Id: imageName,
		},
	}

	// 尝试 inspect，如果报错说明镜像不存在
	if err := s.backend.InspectImage(imageName); err == nil {
		// 存在
		// 严格来说这里应该解析 inspect 输出来填 Size 等字段
		// 但为了 MVP，我们只告诉 Kubelet “有” 就行了
		return status, nil
	}
	
	// 如果不存在，返回 nil Image 字段告诉 Kubelet 没找到
	return &runtimeapi.ImageStatusResponse{
		Image: nil,
	}, nil
}

// ImageFsInfo 返回镜像文件系统的信息
// crictl 在执行很多操作前会先检查这个，用于统计磁盘使用。
func (s *MasCRIServer) ImageFsInfo(ctx context.Context, req *runtimeapi.ImageFsInfoRequest) (*runtimeapi.ImageFsInfoResponse, error) {
	// 暂时返回空，或者 Mock 数据
	return &runtimeapi.ImageFsInfoResponse{
		ImageFilesystems: []*runtimeapi.FilesystemUsage{},
	}, nil
}

