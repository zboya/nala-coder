# NaLa Coder 部署指南

## 概述

NaLa Coder 现在支持**嵌入式部署**，所有Web资源（HTML、CSS、JS等）都被嵌入到Go二进制文件中，实现**单文件部署**，极大简化了部署过程。

## 🚀 快速部署

### 方法一：使用 Makefile（推荐）

```bash
# 完整部署流程（清理 + 构建 + 验证）
make deploy

# 或者分步骤执行
make clean           # 清理之前的构建
make build-embedded  # 构建嵌入式版本
make verify-embedded # 验证部署
```

### 方法二：手动构建

```bash
# 1. 准备嵌入式资源
mkdir -p pkg/embedded/web
cp -r web/* pkg/embedded/web/

# 2. 构建二进制文件
go build -o bin/nala-coder-server cmd/server/main.go

# 3. 运行服务器
./bin/nala-coder-server
```

## 📁 部署文件说明

### 必需文件
```
nala-coder-server    # 主服务器可执行文件（包含所有Web资源）
configs/config.yaml  # 配置文件
```

### 可选文件
```
logs/               # 日志目录（自动创建）
storage/            # 数据存储目录（自动创建）
```

## ⚙️ 配置说明

1. **复制配置文件**：
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   ```

2. **修改关键配置**：
   ```yaml
   # 服务配置
   server:
     port: 8888
     host: "0.0.0.0"
   
   # LLM配置（根据需要选择）
   llm:
     default_provider: "deepseek"  # 或 openai、claude、ollama
     deepseek:
       api_key: "your-api-key-here"
   ```

## 🌐 启动和访问

### 启动服务器
```bash
./bin/nala-coder-server
```

### 访问Web界面
- Web界面：http://localhost:8888/
- API文档：http://localhost:8888/api/health

### API端点
- `POST /api/chat` - 普通聊天
- `POST /api/chat/stream` - 流式聊天
- `GET /api/health` - 健康检查
- `GET /api/tools` - 可用工具列表

## 📦 部署到生产环境

### 1. 系统服务（systemd）

创建服务文件 `/etc/systemd/system/nala-coder.service`：

```ini
[Unit]
Description=NaLa Coder Server
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/nala-coder
ExecStart=/path/to/nala-coder/bin/nala-coder-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable nala-coder
sudo systemctl start nala-coder
```

### 2. 反向代理（Nginx）

配置文件 `/etc/nginx/sites-available/nala-coder`：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8888;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 支持Server-Sent Events
        proxy_buffering off;
        proxy_cache off;
    }
}
```

## 🐳 Docker部署

### 构建镜像
```bash
docker build -t nala-coder .
```

### 运行容器
```bash
docker run -d \
  --name nala-coder \
  -p 8888:8888 \
  -v $(PWD)/configs:/app/configs \
  -v $(PWD)/storage:/app/storage \
  nala-coder
```

## 🔧 验证部署

### 自动验证
```bash
make verify-embedded
```

### 手动验证
```bash
# 健康检查
curl http://localhost:8888/api/health

# Web界面
curl http://localhost:8888/

# 聊天功能
curl -X POST http://localhost:8888/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, NaLa Coder!"}'
```

## 📊 部署优势

### 传统部署 vs 嵌入式部署

| 特性 | 传统部署 | 嵌入式部署 |
|------|----------|------------|
| 文件数量 | 多个文件/目录 | **单个可执行文件** |
| 部署复杂度 | 需要保持目录结构 | **极简部署** |
| 依赖管理 | 需要Web资源文件 | **零外部依赖** |
| 二进制大小 | ~20MB | ~23MB |
| 启动速度 | 正常 | **更快** |
| 维护成本 | 较高 | **极低** |

## 🚨 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   # 检查端口占用
   lsof -i :8888
   
   # 修改配置文件中的端口
   vi configs/config.yaml
   ```

2. **配置文件错误**
   ```bash
   # 重新生成配置文件
   cp configs/config.yaml.example configs/config.yaml
   ```

3. **权限问题**
   ```bash
   # 给可执行文件执行权限
   chmod +x bin/nala-coder-server
   ```

4. **Web界面无法访问**
   ```bash
   # 检查嵌入资源
   make verify-embedded
   
   # 重新构建
   make clean && make build-embedded
   ```

## 📝 性能优化

### 生产环境建议

1. **设置生产模式**：
   ```bash
   export GIN_MODE=release
   ```

2. **调整日志级别**：
   ```yaml
   logging:
     level: "warn"  # 减少日志输出
   ```

3. **配置反向代理缓存**（如使用Nginx）

4. **监控资源使用**：
   - CPU使用率
   - 内存占用
   - 响应时间

## 🔄 更新部署

```bash
# 1. 停止服务
sudo systemctl stop nala-coder

# 2. 备份当前版本
cp bin/nala-coder-server bin/nala-coder-server.backup

# 3. 部署新版本
make deploy

# 4. 启动服务
sudo systemctl start nala-coder
```

---

**✅ 现在你可以通过单个可执行文件部署完整的 NaLa Coder 服务！**