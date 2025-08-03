# NaLa Coder Makefile

.PHONY: default help build test clean install deps fmt lint build-embedded deploy server

# 默认目标
default: server

help:
	@echo "NaLa Coder - AI-powered coding assistant"
	@echo ""
	@echo "Available commands:"
	@echo "  build         - Build all binaries"
	@echo "  build-embedded- Build binaries with embedded web assets"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  deploy        - Build embedded assets"
	#echo "  server        - Build embedded assets And Start the server"

# 构建所有二进制文件
build:
	@echo "Building CLI..."
	go build -o bin/nala-coder-cli cmd/cli/main.go
	@echo "Building server..."
	go build -o bin/nala-coder-server cmd/server/main.go
	@echo "Build complete!"

# 运行测试
test:
	go test -v ./...

# 清理构建产物
clean:
	rm -rf bin/
	rm -rf storage/
	go clean

# 安装二进制文件到系统
install: build-embedded
	sudo cp bin/nala-coder-cli /usr/local/bin/
	sudo cp bin/nala-coder-server /usr/local/bin/
	@echo "Installed to /usr/local/bin/"

# 下载依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 初始化项目
init:
	mkdir -p storage
	mkdir -p logs
	@echo "Project initialized!"

# 开发模式运行
dev-cli:
	@echo "Starting CLI in development mode..."
	go run cmd/cli/main.go -v

dev-server:
	@echo "Starting server in development mode..."
	go run cmd/server/main.go

# 构建Docker镜像
docker-build:
	docker build -t nala-coder .

# 运行Docker容器
docker-run:
	docker run -p 8888:8888 -v $(PWD)/configs:/app/configs nala-coder

# 生成API文档
docs:
	@echo "API documentation available at /api endpoints when server is running"

# 检查配置
check-config:
	@if [ -f "configs/config.yaml" ]; then \
		echo "✓ Configuration file found"; \
	else \
		echo "✗ Configuration file not found. Creating example..."; \
		cp configs/config.yaml.example configs/config.yaml; \
	fi

# 构建嵌入式版本（包含所有web资源）
build-embedded:
	@echo "Preparing embedded web assets..."
	@mkdir -p pkg/embedded/web
	@cp -r web/* pkg/embedded/web/ 2>/dev/null || true
	@echo "Building server with embedded assets..."
	go build -o bin/nala-coder-server cmd/server/main.go
	@echo "Embedded build complete!"
	@echo "Binary size: $$(du -h bin/nala-coder-server | cut -f1)"

# 完整部署流程
deploy: clean check-config build-embedded 
	@echo "🚀 Deployment ready!"
	@echo "Start the server with: ./bin/nala-coder-server"

# 运行服务器
server: deploy
	@echo "Starting server, Access the web interface at: http://localhost:8888"
	./bin/nala-coder-server

# 开发模式构建（自动嵌入资源）
dev-build:
	@echo "Development build with auto-embedded assets..."
	@make build-embedded
	@echo "Development build ready!"