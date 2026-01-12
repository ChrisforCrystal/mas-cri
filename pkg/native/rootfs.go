package native

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// SetupRootfs 将镜像 tar 包解压到指定的 rootfs 目录
// imageTarPath: e.g. /var/lib/mascri/images/busybox.tar
// rootfsPath:   e.g. /var/lib/mascri/containers/abc12345/rootfs
// SetupRootfs 负责将镜像（tar包）解压到指定目录，成为容器的根文件系统。
// 在真实的 Docker/Containerd 实现中，这里极其复杂：
// 1. 并没有一个巨大的 Tar 包，而是多层 Image Layer (OverlayFS)。
// 2. 需要处理 Graph Driver（联合挂载）。
// 3. 我们的 MasCRI 为了演示核心原理，简化为“解压单一 Tar 包”。
func SetupRootfs(imageTarPath string, rootfsPath string) error {
	logrus.Infof("Extracting image %s to %s", imageTarPath, rootfsPath)

	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs dir: %w", err)
	}

	tarFile, err := os.Open(imageTarPath)
	if err != nil {
		return fmt.Errorf("failed to open image tar: %w", err)
	}
	defer tarFile.Close()

	tr := tar.NewReader(tarFile)

	// 遍历 Tar 包中的每一个文件头
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // 解压完毕
		}
		if err != nil {
			return err
		}

		// 拼接目标路径：rootfsPath + 文件在 tar 包内的路径
		target := filepath.Join(rootfsPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// 如果是目录，直接创建
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// 如果是普通文件，创建并写入内容
			// 注意要保留原文件的权限 (header.Mode)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			// 如果是软链接，创建 Symlink
			if err := os.Symlink(header.Linkname, target); err != nil {
				// 在某些受限环境或者文件名冲突时可能会失败，这里选择打 Log 宽容处理
				logrus.Warnf("Failed to create symlink %s -> %s: %v", target, header.Linkname, err)
			}
		default:
			// 忽略其他生僻类型（如 Block Device 等，普通镜像里很少见）
		}
	}

	return nil
}
