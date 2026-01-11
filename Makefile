
BINARY_NAME=mascri
SOCKET_PATH=/tmp/mascri.sock

.PHONY: all build run clean test

all: build

build:
	go build -o bin/$(BINARY_NAME) cmd/mascri/main.go

# 运行 MasCRI，开启 Debug 模式
run: build
	./bin/$(BINARY_NAME) --socket $(SOCKET_PATH) --debug

# 清理构建产物
clean:
	rm -f bin/$(BINARY_NAME)
	rm -f $(SOCKET_PATH)

# 使用 crictl 验证 (需要 crictl 已安装)
verify-info:
	@echo "Checking CRI Version/Status..."
	crictl --runtime-endpoint unix://$(SOCKET_PATH) info

verify-runp:
	@echo "Simulating Pod Creation..."
	echo '{"metadata": {"name": "nginx-sandbox", "namespace": "default", "uid": "1", "attempt": 1}}' > /tmp/sandbox-config.json
	crictl --runtime-endpoint unix://$(SOCKET_PATH) runp /tmp/sandbox-config.json
	rm /tmp/sandbox-config.json
