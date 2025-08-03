# NaLa Coder 前端

这是一个现代化的AI代码助手前端界面，具有以下功能：

## 功能特性

- 🎤 **语音唤醒和语音转文字** - 支持语音输入，自动转换为文字发送给后端
- 💬 **智能聊天界面** - 与AI代码助手进行对话
- 📁 **文件浏览器** - 浏览项目文件结构
- 💻 **代码语法高亮** - 支持多种编程语言的语法高亮
- 🎨 **现代化UI设计** - 深色主题，响应式布局
- ⚡ **实时流式响应** - 支持流式聊天响应

## 技术栈

- **HTML5** - 语义化标记
- **CSS3** - 现代化样式，深色主题
- **JavaScript ES6+** - 原生JavaScript，无框架依赖
- **Web Speech API** - 语音识别和合成
- **Prism.js** - 代码语法高亮
- **Font Awesome** - 图标库

## 文件结构

```
frontend/
├── index.html      # 主页面
├── styles.css      # 样式文件
├── app.js          # 应用逻辑
└── README.md       # 说明文档
```

## 使用方法

1. 确保后端服务正在运行
2. 访问 `http://localhost:8080/frontend/` 或 `http://localhost:8080/`
3. 开始与AI代码助手对话

## 语音功能

- 点击麦克风图标开始语音输入
- 支持中文语音识别
- 语音输入会自动转换为文字并发送

## API接口

前端使用以下后端API：

- `POST /api/chat` - 发送聊天消息
- `POST /api/chat/stream` - 流式聊天
- `GET /api/files/tree` - 获取文件树
- `GET /api/files/content` - 获取文件内容
- `GET /api/speech/config` - 获取语音配置

## 浏览器兼容性

- Chrome 66+ (推荐)
- Firefox 60+
- Safari 14+
- Edge 79+

注意：语音识别功能需要HTTPS环境或localhost。

## 开发说明

- 所有代码都是原生JavaScript，无需构建工具
- 使用现代CSS特性，支持响应式设计
- 代码结构清晰，易于维护和扩展 