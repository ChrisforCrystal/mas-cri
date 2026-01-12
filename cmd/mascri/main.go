package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"mascri/pkg/server"
	"mascri/pkg/version"
)

func main() {
	// 使用 urfave/cli 库构建命令行应用程序
	app := &cli.App{
		Name:    version.ProgramName, // 程序名称，来自 version 包
		Version: version.Version,     // 程序版本
		Usage:   "A simple, educational Kubernetes Container Runtime Interface implementation", // 简短的使用说明
		
		// 定义命令行参数 (Flags)
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "socket",              // 参数名称，通过 --socket 使用
				Aliases: []string{"s"},         // 参数别名，通过 -s 使用
				Value:   "/tmp/mascri.sock",    // 默认值：如果我们不指定，就监听这个文件
				Usage:   "Path to the Unix Domain Socket", // 参数说明
				EnvVars: []string{"MASCRI_SOCKET"},      // 也可以通过环境变量设置
			},
			&cli.BoolFlag{
				Name:    "debug",          // Debug 模式开关
				Aliases: []string{"d"},    // 别名 -d
				Usage:   "Enable debug logging", // 开启后会打印更多细节
			},
			&cli.StringFlag{
				Name:  "cni-bin-dir",
				Value: "/opt/cni/bin",// 这里面存放着各种编译好的 CNI 插件二进制文件，比如 bridge（造桥）、loopback（回环）、host-local（分配 IP）等。
				Usage: "Path to CNI plugin binaries",
			},
			&cli.StringFlag{
				Name:  "cni-conf-dir",
				Value: "/etc/cni/net.d", // 这里存放 CNI 的配置文件（通常是 .conf 或 .conflist 结尾的 JSON）。
				Usage: "Path to CNI configuration files",
			},
			&cli.StringFlag{
				Name:  "cni-cache-dir",
				Value: "/var/lib/cni", // CNI 插件需要在这个目录里记录哪些 IP 已经被分配给谁了，防止 IP 冲突。
				Usage: "Path to CNI cache directory",
			},
		},
		
		// 应用程序的主逻辑入口
		Action: func(c *cli.Context) error {
			// 1. 初始化日志配置
			// 如果用户指定了 --debug，则开启调试级别的日志
			if c.Bool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}
			// 设置日志格式为带时间戳的文本格式，方便阅读
			logrus.SetFormatter(&logrus.TextFormatter{
				FullTimestamp: true,
			})

			// 2. 获取参数
			// 注意：这里必须和上面 StringFlag 的 Name 保持一致
			socketPath := c.String("socket")
			
			// 3. 创建并启动 MasCRI 服务器
			// 这一步会初始化 gRPC 服务，并开始监听 Unix Socket
			cniBinDir := c.String("cni-bin-dir")
			cniConfDir := c.String("cni-conf-dir")
			cniCacheDir := c.String("cni-cache-dir")
			srv := server.NewMasCRIServer(socketPath, cniConfDir, []string{cniBinDir}, cniCacheDir)
			
			// Start() 是一个阻塞操作，直到程序退出或出错
			if err := srv.Start(); err != nil {
				logrus.Fatalf("Server failed: %v", err)
				return err
			}
			return nil
		},
	}

	// 运行应用程序，os.Args 包含了命令行输入的所有参数
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
