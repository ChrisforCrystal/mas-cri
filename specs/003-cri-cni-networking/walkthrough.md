# Feature 003: CNI Networking 验证指南

本指南将验证 MasCRI 如何集成 CNI 插件为 Pod 分配 IP 地址。
由于 macOS Docker Desktop 的限制，我们将使用一个 **Mock CNI 插件** 来模拟 Linux 网络插件的行为。

## 架构

`MasCRI` -> `libcni` -> `Mock Plugin` -> 返回 Fake IP

## 1. 准备环境

MasCRI 需要知道 CNI 配置文件和插件二进制在哪里。我们在项目目录下创建模拟环境：

```bash
# 1. 创建目录
mkdir -p cni/bin cni/net.d cni/cache

# 2. 创建 Mock 插件 (模拟 Bridge 插件)
# 这个脚本会假装自己给容器分配了 10.88.0.2
cat <<EOF > cni/bin/loopback
#!/bin/sh
cat <<JSON
{
    "cniVersion": "0.3.1",
    "interfaces": [
        {
            "name": "eth0",
            "mac": "02:00:00:00:00:01",
            "sandbox": "\$CNI_NETNS"
        }
    ],
    "ips": [
        {
            "version": "4",
            "address": "10.88.0.2/16",
            "gateway": "10.88.0.1",
            "interface": 0
        }
    ],
    "dns": {}
}
JSON
exit 0
EOF
chmod +x cni/bin/loopback

# 3. 创建 CNI 配置文件
cat <<EOF > cni/net.d/10-loopback.conf
{
	"cniVersion": "0.3.1",
	"name": "lo",
	"type": "loopback"
}
EOF
```

## 2. 编译并启动 MasCRI

```bash
# 编译
make build

# 启动 (指定本地 CNI 路径)
./bin/mascri \
  --socket /tmp/mascri.sock \
  --debug \
  --cni-bin-dir $(pwd)/cni/bin \
  --cni-conf-dir $(pwd)/cni/net.d \
  --cni-cache-dir $(pwd)/cni/cache
```

## 3. 验证 Pod IP

在另一个终端执行：

```bash
# 1. 准备 Sandbox 配置
echo '{"metadata": {"name": "cni-test", "namespace": "default", "uid": "cni", "attempt": 1}}' > sandbox.json

# 2. 运行 Pod
# (注意：如果之前运行过失败的，请先 docker rm -f k8s_POD_cni-test_default_cni)
crictl --runtime-endpoint unix:///tmp/mascri.sock -t 20s runp sandbox.json

# 3. 检查 Pod IP
export POD_ID=$(crictl --runtime-endpoint unix:///tmp/mascri.sock pods -q | head -n 1)
crictl --runtime-endpoint unix:///tmp/mascri.sock inspectp $POD_ID
```

### 预期结果

`inspectp` 的输出 JSON 中应该包含：

```json
    "network": {
      "ip": "10.88.0.2"
    },
```

看到 `10.88.0.2` 字样即表示 MasCRI 成功调用了我们的 CNI 插件！
