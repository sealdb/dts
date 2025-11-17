# Makefile for PostgreSQL DTS (Data Transfer Service)

# 变量定义
APP_NAME := dts
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO := go
GOFMT := gofmt
GOVET := go vet
GOLINT := golangci-lint
GOCOVER := go tool cover

# 目录定义
CMD_DIR := cmd/server
BIN_DIR := bin
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# 编译标志
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -w -s

# 默认目标
.PHONY: all
all: fmt vet build

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build      - 编译项目"
	@echo "  make run        - 运行项目"
	@echo "  make fmt        - 格式化代码"
	@echo "  make vet        - 静态检查"
	@echo "  make lint       - 代码检查（需要安装 golangci-lint）"
	@echo "  make test       - 运行测试"
	@echo "  make test-race  - 运行竞态检测测试"
	@echo "  make coverage   - 生成测试覆盖率报告"
	@echo "  make clean      - 清理生成的文件"
	@echo "  make deps       - 下载依赖"
	@echo "  make tidy       - 整理依赖"
	@echo "  make install    - 安装到 GOPATH/bin"
	@echo "  make docker     - 构建 Docker 镜像"

# 编译
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

# 编译（开发模式，包含调试信息）
.PHONY: build-dev
build-dev:
	@echo "Building $(APP_NAME) (dev mode)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME)-dev ./$(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)-dev"

# 运行
.PHONY: run
run:
	$(GO) run ./$(CMD_DIR)

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Format complete"

# 检查代码格式
.PHONY: fmt-check
fmt-check:
	@echo "Checking code format..."
	@if [ $$($(GOFMT) -l . | wc -l) -ne 0 ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		$(GOFMT) -d .; \
		exit 1; \
	fi
	@echo "Code format check passed"

# 静态检查
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "Vet check complete"

# 代码检查（需要安装 golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest）
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	@echo "Tests complete"

# 运行测试（快速模式，无竞态检测）
.PHONY: test-fast
test-fast:
	@echo "Running tests (fast mode)..."
	$(GO) test -v ./...
	@echo "Tests complete"

# 运行测试（仅当前包）
.PHONY: test-short
test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...
	@echo "Short tests complete"

# 运行竞态检测测试
.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	$(GO) test -race -v ./...
	@echo "Race detection tests complete"

# 生成测试覆盖率报告
.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOCOVER) -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@echo "Coverage percentage:"
	@$(GOCOVER) -func=$(COVERAGE_FILE) | tail -1

# 查看覆盖率（终端）
.PHONY: coverage-text
coverage-text:
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCOVER) -func=$(COVERAGE_FILE)

# 清理
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -rf $(COVERAGE_DIR)
	$(GO) clean -cache
	@echo "Clean complete"

# 深度清理（包括依赖缓存）
.PHONY: clean-all
clean-all: clean
	@echo "Deep cleaning..."
	$(GO) clean -modcache
	@echo "Deep clean complete"

# 下载依赖
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "Dependencies downloaded"

# 整理依赖
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "Dependencies tidied"

# 验证依赖
.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify
	@echo "Dependencies verified"

# 安装到 GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(APP_NAME)..."
	$(GO) install -ldflags "$(LDFLAGS)" ./$(CMD_DIR)
	@echo "Install complete"

# 交叉编译（Linux）
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 ./$(CMD_DIR)
	@echo "Linux build complete: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

# 交叉编译（Windows）
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "Windows build complete: $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe"

# 交叉编译（macOS）
.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 ./$(CMD_DIR)
	@echo "macOS build complete"

# 构建所有平台
.PHONY: build-all
build-all: build-linux build-windows build-darwin
	@echo "All platform builds complete"

# Docker 构建（需要 Dockerfile）
.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest
	@echo "Docker image built: $(APP_NAME):$(VERSION)"

# 开发环境检查
.PHONY: check
check: fmt-check vet
	@echo "All checks passed"

# CI/CD 检查（用于 CI 环境）
.PHONY: ci
ci: fmt-check vet test-race
	@echo "CI checks complete"

# 生成文档
.PHONY: docs
docs:
	@echo "Generating documentation..."
	$(GO) doc -all ./... > docs/API.md 2>/dev/null || echo "Documentation generation skipped"
	@echo "Documentation generated"

# 显示版本信息
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"






