## API接口文档

### 1. 流式聊天接口

#### `POST /api/chat/stream`

**功能描述**：以Server-Sent Events (SSE) 方式流式返回AI助手的回复内容。

**请求格式**：
```json
{
  "message": "用户输入的消息内容",
  "session_id": "可选，会话ID",
  "stream": true,
  "metadata": {
    "key": "value"
  }
}
```

**响应格式**：SSE流式响应，每个事件格式如下：
```json
event: message
data: {
  "session_id": "会话ID",
  "response": "AI助手的部分回复内容",
  "finished": false,
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  },
  "metadata": {}
}
```

### 2. 获取会话详情

#### `GET /api/session/:id`

**功能描述**：获取指定会话的详细信息和状态。

**路径参数**：
- `id` (string, required): 会话ID

**响应格式**：
```json
{
  "session_id": "会话ID",
  "messages": [...],
  "metadata": {},
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:30:00Z"
}
```

**错误响应**：
- `400 Bad Request`: 缺少会话ID
- `404 Not Found`: 会话不存在

### 3. 获取会话列表

#### `GET /api/sessions`

**功能描述**：获取所有会话的列表（当前版本返回空列表）。

**响应格式**：
```json
{
  "sessions": [],
  "message": "Session listing not implemented yet",
  "count": 0
}
```

### 4. 获取文件树

#### `GET /api/files/tree`

**功能描述**：获取指定路径的文件和目录树形结构。

**查询参数**：
- `path` (string, optional): 要浏览的目录路径，默认为当前工作目录
- `depth` (int, optional): 目录深度限制，默认为5层

**响应格式**：
```json
{
  "tree": {
    "name": "目录名称",
    "path": "完整路径",
    "type": "directory",
    "size": 4096,
    "mod_time": "2024-01-01T12:00:00Z",
    "children": [
      {
        "name": "文件名",
        "path": "完整路径",
        "type": "file",
        "size": 1024,
        "mod_time": "2024-01-01T12:00:00Z"
      }
    ]
  },
  "path": "/当前/浏览/路径"
}
```

过滤规则：自动跳过以下文件和目录：

隐藏文件（以.开头）
常见忽略目录：node_modules, vendor, target, build, dist, logs, .git, __pycache__等


### 5. 获取文件内容

#### `GET /api/files/content`

**功能描述**：获取指定文本文件的内容和元数据。

**查询参数**：
path (string, required): 文件完整路径

**响应格式**：
```json
{
  "path": "/完整/文件/路径",
  "content": "文件内容文本",
  "language": "go",
  "size": 1024,
  "mod_time": "2024-01-01 12:00:00"
}
```

**限制条件**：
- 文件大小限制：最大1MB
- 仅支持文本文件
- 不支持目录路径

**支持的文件类型**：
- 编程语言：Go, Python, JavaScript, TypeScript, Java, C/C++, Rust等
- 标记语言：Markdown, HTML, CSS, JSON, YAML, XML等
- 配置文件：Dockerfile, Makefile, .env等

**错误响应**：
- `400 Bad Request`: 缺少路径参数或文件过大
- `404 Not Found`: 文件不存在
- `400 Bad Request`: 路径是目录或不是文本文件

### 6. 获取语音配置

#### `GET /api/speech/config`

**功能描述**：获取语音助手的配置信息。

**响应格式**：
```json
{
  "enabled": true,
  "wake_words": ["小助手", "助手"],
  "wake_timeout": 30,
  "language": "zh-CN"
}
```

**默认值**：
默认值：

enabled: 根据系统配置
wake_words: ["小助手", "助手"]
wake_timeout: 30秒
language: "zh-CN"
