package network

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/sirupsen/logrus"
)

// CNIManager manages the CNI plugins and network configuration
// CNIManager manages the CNI plugins and network configuration
// CNIManager 负责管理 CNI 插件交互和网络配置加载。
// 它封装了底层 libcni 库的复杂性，提供给 Runtime 一个简单的 SetUp/TearDown 接口。
type CNIManager struct {
	// libcni 提供的核心配置对象，所有的插件执行（exec）都由它发起。
	// 它知道插件在哪里 (Path)，缓存怎么记 (CacheDir)。
	// libcni.CNIConfig 是 CNI 官方库 (github.com/containernetworking/cni/libcni) 提供的核心接口。
	// 它的地位相当于 libcontainer.Factory：
	// - libcontainer 负责调用 Kernel 创建容器。
	// - libcni 负责调用 CNI 插件（二进制文件）配置网络。
	// 
	// 它的工作流程：
	// 1. 读取 CNI 配置文件（如 10-bridge.conf）。
	// 2. 找到对应的插件二进制（如 /opt/cni/bin/bridge）。
	// 3. 设置好环境变量（CNI_COMMAND=ADD, CNI_NETNS=/proc/...）。
	// 4. fork/exec 执行该插件，并把结果（IP 地址等）返回给我们。
	cniConfig libcni.CNIConfig

	// 配置文件目录路径 (e.g. /etc/cni/net.d)。
	// 我们需要扫描这个目录来找到用户定义的网络配置列表。
	netConfigDir string

	// 插件二进制文件的搜索路径列表 (e.g. [/opt/cni/bin])。
	// 当我们解析配置发现需要 "bridge" 插件时，就会去这些目录里找名为 "bridge" 的可执行文件。
	binDirs []string
}

// NewCNIManager creates a new CNIManager
// netConfigDir: where CNI config files are located (e.g., /etc/cni/net.d)
// binDirs: where CNI plugin binaries are located (e.g., /opt/cni/bin)
// cacheDir: where CNI caches network results (e.g., /var/lib/cni)
// NewCNIManager 创建一个 CNI 管理器实例
// netConfigDir: CNI 配置文件目录 (e.g., /etc/cni/net.d)，存放 "施工图纸"
// binDirs:      CNI 插件二进制目录 (e.g., /opt/cni/bin)，存放 "工具箱"
// cacheDir:     CNI 缓存目录       (e.g., /var/lib/cni)，存放 "记账本" (记录 IP 分配情况)
func NewCNIManager(netConfigDir string, binDirs []string, cacheDir string) (*CNIManager, error) {
	// 使用 libcni 提供的构造函数来初始化配置
	// libcni.NewCNIConfigWithCacheDir 是一个工厂方法：
	// 参数 1 (binDirs): 告诉它去哪里找插件可执行文件
	// 参数 2 (cacheDir): 告诉它把缓存文件写在哪里
	// 参数 3 (nil): 代表使用默认的执行器 (exec.Command) 来运行插件
	config := libcni.NewCNIConfigWithCacheDir(binDirs, cacheDir, nil)
	
	// 返回我们的包装对象
	return &CNIManager{
		cniConfig:    *config,      // 保存这个初始化好的配置对象
		netConfigDir: netConfigDir, // 记住配置文件在哪里，后面 loadNetworkConfig 要用
		binDirs:      binDirs,      // 记住二进制目录
	}, nil
}

// loadNetworkConfig finds and loads the CNI configuration from the directory.
// It creates a NetworkConfigList because usually we handle a chain of plugins (e.g. bridge + loopback).
// loadNetworkConfig 加载 CNI 网络配置
// 它的任务是去 cni-conf-dir (比如 /etc/cni/net.d) 找到第一个合法的配置文件
// 并把它解析成 libcni 能理解的 NetworkConfigList 对象。
func (m *CNIManager) loadNetworkConfig() (*libcni.NetworkConfigList, error) {
	// 1. 扫描目录寻找配置文件
	// 支持的后缀名：.conf, .conflist, .json
	files, err := libcni.ConfFiles(m.netConfigDir, []string{".conf", ".conflist", ".json"})
	if err != nil {
		return nil, fmt.Errorf("failed to list CNI config files: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no CNI config files found in %s", m.netConfigDir)
	}

	// 2. 排序
	// Kubernetes 标准行为：按文件名与其字典序排序，取第一个文件作为默认网络配置。
	// 这确保了哪怕目录下有多个文件，MasCRI 每次启动选择的网络都是确定的。
	// 例如：10-bridge.conf 会排在 99-loopback.conf 之前。
	sort.Strings(files)
	
	// 3. 加载第一个文件
	filename := files[0]
	
	// 情况 A: 这是一个配置列表 (.conflist)
	// .conflist 允许你把多个插件串起来用（链式调用）。
	// 比如：先调用 bridge 分配 IP，再调用 portmap 做端口映射，最后调用 firewall 做防火墙。
	if filepath.Ext(filename) == ".conflist" {
		confList, err := libcni.ConfListFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load CNI config list %s: %v", filename, err)
		}
		return confList, nil
	}

	// 情况 B: 这是一个单插件配置 (.conf / .json)
	// 比如只用一个 bridge 插件。
	// 为了统一处理，libcni 会把它包装成一个只包含一项的列表。
	conf, err := libcni.ConfFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load CNI config %s: %v", filename, err)
	}
	
	// 这里的 ConfListFromConf 就是做这个包装工作的
	confList, err := libcni.ConfListFromConf(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to convert CNI config to list: %v", err)
	}
	return confList, nil
}

// SetUpPod sets up the network for a Pod Sandbox
// id: Pod Sandbox ID
// netns: Path to the network namespace (e.g. /var/run/netns/xxx)
// SetUpPod 为 Pod 配置网络 (CNI ADD 操作)
// id:    容器 ID
// netns: 容器的网络命名空间路径 (e.g. /var/run/docker/netns/xxx)
func (m *CNIManager) SetUpPod(ctx context.Context, id string, netns string) (types.Result, error) {
	// 1. 加载 CNI 配置 (找到图纸)
	confList, err := m.loadNetworkConfig()
	if err != nil {
		return nil, err
	}

	// 2. 准备运行时配置 (RuntimeConf)
	// 这些是传递给 CNI 插件的环境变量参数
	rtConf := &libcni.RuntimeConf{
		ContainerID: id,    // CNI_CONTAINERID
		NetNS:       netns, // CNI_NETNS: 告诉插件去哪个 Namespace 干活
		IfName:      "eth0", // CNI_IFNAME: 我们希望在 Pod 里面叫什么网卡名 (通常是 eth0)
		// 未来可以在这里添加 Capability Args (如端口映射 PortMappings)
	}

	logrus.Infof("[CNI] Adding network for pod %s (netns: %s) using config %s", id, netns, confList.Name)

	// 3. 调用 CNI 插件执行 ADD 操作
	// libcni 会自动帮我们：
	// - 找到插件二进制
	// - 设置环境变量 (CNI_COMMAND=ADD, ...)
	// - 把配置 JSON 喂给插件的 Stdin
	// - 解析插件 Stdout 返回的结果 (Result)
	res, err := m.cniConfig.AddNetworkList(ctx, confList, rtConf)
	if err != nil {
		return nil, fmt.Errorf("CNI add network failed: %v", err)
	}

	// 4. 返回结果
	// 这个 Result 里面就包含了分配到的 IP 地址等信息
	logrus.Infof("[CNI] Success. IP Result: %+v", res)
	return res, nil
}

// TearDownPod removes the network for a Pod Sandbox
func (m *CNIManager) TearDownPod(ctx context.Context, id string, netns string) error {
	confList, err := m.loadNetworkConfig()
	if err != nil {
		return err
	}

	rtConf := &libcni.RuntimeConf{
		ContainerID: id,
		NetNS:       netns,
		IfName:      "eth0",
	}

	logrus.Infof("[CNI] Removing network for pod %s", id)

	// Call CNI DEL
	if err := m.cniConfig.DelNetworkList(ctx, confList, rtConf); err != nil {
		return fmt.Errorf("CNI del network failed: %v", err)
	}
	
	return nil
}
