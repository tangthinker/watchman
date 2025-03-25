# 项目名称
PROJECT_NAME := watchman

# Go 相关变量
GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOCLEAN := $(GO) clean
GOGET := $(GO) get

# 二进制文件路径
BINARY_NAME := $(PROJECT_NAME)
BINARY_PATH := ./$(BINARY_NAME)

# 源代码路径
MAIN_PATH := ./cmd/watchman/main.go

# 默认目标
.DEFAULT_GOAL := build

# 构建目标
.PHONY: build
build:
	@echo "Building $(PROJECT_NAME)..."
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete!"

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete!"

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_PATH)
	@echo "Clean complete!"

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@echo "Dependencies installed!"

# 运行程序
.PHONY: run
run: build
	@echo "Running $(PROJECT_NAME)..."
	./$(BINARY_PATH)

# 开发模式（自动重新构建和运行）
.PHONY: dev
dev:
	@echo "Starting development mode..."
	@while true; do \
		make build; \
		./$(BINARY_PATH) & \
		PID=$$!; \
		inotifywait -e modify -e create -e delete -e move -r .; \
		kill $$PID; \
	done

# 安装到系统
.PHONY: install
install: build
	@echo "Installing $(PROJECT_NAME)..."
	install -m 755 $(BINARY_PATH) /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete!"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(PROJECT_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation complete!"

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the project"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build files"
	@echo "  deps       - Install dependencies"
	@echo "  run        - Build and run the program"
	@echo "  dev        - Run in development mode (auto-rebuild)"
	@echo "  install    - Install to system"
	@echo "  uninstall  - Uninstall from system"
	@echo "  help       - Show this help message" 