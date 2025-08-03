# NaLa Coder Dockerfile

# 使用官方Go镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nala-coder-server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nala-coder-cli cmd/cli/main.go

# 使用alpine作为运行环境
FROM alpine:latest

# 安装必要的工具
RUN apk --no-cache add ca-certificates bash git curl

# 创建工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/nala-coder-server .
COPY --from=builder /app/nala-coder-cli .

# 复制配置文件和提示词
COPY configs/ ./configs/
COPY prompts/ ./prompts/
COPY web/ ./web/

# 创建存储目录
RUN mkdir -p storage logs

# 暴露端口
EXPOSE 8888

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8888/api/health || exit 1

# 默认运行服务器
CMD ["./nala-coder-server"]