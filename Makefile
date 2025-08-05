# NaLa Coder Makefile

.PHONY: default help build test clean install deps fmt lint build-embedded deploy server build-web build-web-dev install-web clean-web init-config

# 默认目标
default: server

help:
	@echo "NaLa Coder - AI-powered coding assistant"
	@echo ""
	@echo "Available commands:"
	@echo "  init-config   - Initialize ~/.nala-coder configuration directory"
	@echo "  build         - Build all binaries"
	@echo "  build-embedded- Build binaries with embedded web assets"
	@echo "  build-web     - Build React application (production)"
	@echo "  build-web-dev - Build React application (development)"
	@echo "  install-web   - Install web dependencies"
	@echo "  clean-web     - Clean web build artifacts"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean all build artifacts"
	@echo "  install       - Install dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  deploy        - Build embedded assets"
	@echo "  server        - Build embedded assets and start the server"

# 初始化配置目录
init-config:
	@echo "Initializing NaLa Coder configuration..."
	./scripts/init-config.sh

# 构建所有二进制文件
build:
	@echo "Building ..."
	go build -o bin/nala-coder cmd/*.go
	@echo "Build complete!"

# 运行测试
test:
	go test -v ./...

# 清理构建产物
clean: clean-web
	rm -rf bin/
	rm -rf storage/
	go clean

# 安装二进制文件到系统
install: build-embedded
	sudo cp bin/nala-coder /usr/local/bin/
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
	go run cmd/*.go

# 构建Docker镜像
docker-build:
	docker build -t nala-coder .

# 运行Docker容器
docker-run:
	docker run -p 8888:8888 -v $(PWD)/configs:/app/configs nala-coder

# 生成API文档
docs:
	@echo "API documentation available at /api endpoints when server is running"

# 构建嵌入式版本（包含所有web资源）
build-embedded:
	@echo "Building React application..."
	@cd web && npm run build
	@echo "Preparing embedded web assets..."
	@mkdir -p pkg/embedded/web
	@cp -r web/dist pkg/embedded/web/ 2>/dev/null || true
	@echo "Building server with embedded assets..."
	go build -o bin/nala-coder cmd/*.go
	@echo "Embedded build complete!"
	@echo "Binary size: $(du -h bin/nala-coder | cut -f1)"

# 完整部署流程
deploy: clean init-config install-web build-embedded 
	@echo "🚀 Deployment ready!"
	@echo "Start the server with: ./bin/nala-coder"

# 运行服务器
server: deploy
	@echo "Starting server, Access the web interface at: http://localhost:8888"
	./bin/nala-coder

# 构建React应用（开发模式）
build-web-dev:
	@echo "Building React application in development mode..."
	@cd web && npm run build:dev

# 构建React应用（生产模式）
build-web:
	@echo "Building React application in production mode..."
	@cd web && npm run build

# 安装Web依赖
install-web:
	@echo "Installing web dependencies..."
	@cd web && npm install

# 开发模式构建（自动嵌入资源）
dev-build:
	@echo "Development build with auto-embedded assets..."
	@make build-embedded
	@echo "Development build ready!"

# 清理Web构建产物
clean-web:
	@echo "Cleaning web build artifacts..."
	@rm -rf web/dist
	@rm -rf pkg/embedded/web