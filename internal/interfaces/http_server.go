package interfaces

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboya/nala-coder/pkg/embedded"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	agent        types.Agent
	logger       log.Logger
	speechConfig types.SpeechConfig
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(agent types.Agent, logger log.Logger, speechConfig types.SpeechConfig) *HTTPServer {
	return &HTTPServer{
		agent:        agent,
		logger:       logger,
		speechConfig: speechConfig,
	}
}

// SetupRoutes 设置路由
func (s *HTTPServer) SetupRoutes() *gin.Engine {
	router := gin.New()

	// 中间件
	router.Use(s.loggingMiddleware())
	router.Use(s.corsMiddleware())
	router.Use(gin.Recovery())

	// API路由组
	api := router.Group("/api")
	{
		// 聊天接口
		api.POST("/chat", s.handleChat)
		api.POST("/chat/stream", s.handleChatStream)

		// 会话管理
		api.GET("/session/:id", s.handleGetSession)
		api.GET("/sessions", s.handleListSessions)

		// 文件浏览
		api.GET("/files/tree", s.handleGetFileTree)
		api.GET("/files/content", s.handleGetFileContent)

		// 语音配置（保留基本配置接口）
		api.GET("/speech/config", s.handleGetSpeechConfig)

		// 系统信息
		api.GET("/health", s.handleHealth)
		api.GET("/tools", s.handleGetTools)
	}

	// 设置嵌入式静态文件 - React构建后的资源
	distFS, err := fs.Sub(embedded.GetWebFS(), "web/dist")
	if err != nil {
		s.logger.Error("Failed to create dist filesystem", "error", err)
		// 如果子文件系统创建失败，使用原始方式
		router.StaticFS("/assets", http.FS(embedded.GetWebFS()))
	} else {
		// 创建assets子文件系统，指向dist/assets目录
		assetsFS, err := fs.Sub(distFS, "assets")
		if err != nil {
			s.logger.Error("Failed to create assets filesystem", "error", err)
			// 回退到直接使用distFS
			router.StaticFS("/assets", http.FS(distFS))
		} else {
			// 静态资源路由 - 将/assets路径映射到dist/assets目录
			router.StaticFS("/assets", http.FS(assetsFS))
		}

		// 处理其他静态文件（如favicon等）
		router.GET("/favicon.ico", func(c *gin.Context) {
			file, err := distFS.Open("favicon.ico")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			c.DataFromReader(http.StatusOK, stat.Size(), "image/x-icon", file, nil)
		})

		// 处理其他根级静态文件
		router.GET("/robots.txt", func(c *gin.Context) {
			file, err := distFS.Open("robots.txt")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			c.DataFromReader(http.StatusOK, stat.Size(), "text/plain", file, nil)
		})
	}

	// 前端文件路由 - 处理所有 /frontend/* 请求（开发模式）
	router.Static("/frontend", "./frontend")

	// React SPA 路由处理
	router.NoRoute(func(c *gin.Context) {
		// 如果是API请求，返回404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// 检查是否存在开发模式的前端文件
		if _, err := os.Stat("./frontend/index.html"); err == nil {
			// 如果存在前端文件，重定向到前端页面
			c.Redirect(http.StatusMovedPermanently, "/frontend/")
			return
		}

		// 否则使用嵌入式React应用
		if distFS != nil {
			indexFile, err := distFS.Open("index.html")
			if err != nil {
				s.logger.Error("Failed to open index.html", "error", err)
				c.String(http.StatusInternalServerError, "Failed to load application")
				return
			}
			defer indexFile.Close()

			indexContent, err := ioutil.ReadAll(indexFile)
			if err != nil {
				s.logger.Error("Failed to read index.html", "error", err)
				c.String(http.StatusInternalServerError, "Failed to load application")
				return
			}

			c.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
		} else {
			c.String(http.StatusInternalServerError, "Application not available")
		}
	})

	// 默认页面 - 重定向到React应用
	router.GET("/", func(c *gin.Context) {
		// 检查是否存在开发模式的前端文件
		if _, err := os.Stat("./frontend/index.html"); err == nil {
			// 如果存在前端文件，重定向到前端页面
			c.Redirect(http.StatusMovedPermanently, "/frontend/")
			return
		}

		// 否则使用嵌入式React应用
		if distFS != nil {
			indexFile, err := distFS.Open("index.html")
			if err != nil {
				s.logger.Error("Failed to open index.html", "error", err)
				c.String(http.StatusInternalServerError, "Failed to load application")
				return
			}
			defer indexFile.Close()

			indexContent, err := ioutil.ReadAll(indexFile)
			if err != nil {
				s.logger.Error("Failed to read index.html", "error", err)
				c.String(http.StatusInternalServerError, "Failed to load application")
				return
			}

			c.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
		} else {
			c.String(http.StatusInternalServerError, "Application not available")
		}
	})

	return router
}

// ChatRequest HTTP聊天请求
type ChatRequest struct {
	Message   string            `json:"message" binding:"required"`
	SessionID string            `json:"session_id,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ChatResponse HTTP聊天响应
type ChatResponse struct {
	SessionID string                 `json:"session_id"`
	Response  string                 `json:"response"`
	Finished  bool                   `json:"finished"`
	Usage     types.Usage            `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// SpeechConfigResponse 语音配置响应
type SpeechConfigResponse struct {
	Enabled     bool     `json:"enabled"`
	WakeWords   []string `json:"wake_words"`
	WakeTimeout int      `json:"wake_timeout"`
	Language    string   `json:"language"`
}

// FileNode 文件节点
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "directory"
	Size     int64       `json:"size"`
	ModTime  time.Time   `json:"mod_time"`
	Children []*FileNode `json:"children,omitempty"`
}

// FileContentResponse 文件内容响应
type FileContentResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time"`
}

// handleChat 处理聊天请求
func (s *HTTPServer) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为内部类型
	query := fmt.Sprintf("<user_query>\n%s\n</user_query>", req.Message)
	agentReq := types.ChatRequest{
		Message:   query,
		SessionID: req.SessionID,
		Stream:    false,
		Metadata:  req.Metadata,
	}

	// 调用Agent
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	response, err := s.agent.Chat(ctx, agentReq)
	if err != nil {
		s.logger.Errorf("Chat failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换响应
	httpResp := ChatResponse{
		SessionID: response.SessionID,
		Response:  response.Response,
		Finished:  response.Finished,
		Usage:     response.Usage,
		Metadata:  response.Metadata,
	}

	c.JSON(http.StatusOK, httpResp)
}

// handleChatStream 处理流式聊天请求
func (s *HTTPServer) handleChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置SSE头部
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 转换为内部类型
	query := fmt.Sprintf("<user_query>\n%s\n</user_query>", req.Message)
	agentReq := types.ChatRequest{
		Message:   query,
		SessionID: req.SessionID,
		Stream:    true,
		Metadata:  req.Metadata,
	}

	// 调用Agent流式API
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	stream, err := s.agent.ChatStream(ctx, agentReq)
	if err != nil {
		s.logger.Errorf("Chat stream failed: %v", err)
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	// 发送流式响应
	for response := range stream {
		httpResp := ChatResponse{
			SessionID: response.SessionID,
			Response:  response.Response,
			Finished:  response.Finished,
			Usage:     response.Usage,
			Metadata:  response.Metadata,
		}

		c.SSEvent("message", httpResp)
		c.Writer.Flush()

		if response.Finished {
			break
		}
	}

	c.SSEvent("end", nil)
}

// handleGetSession 获取会话信息
func (s *HTTPServer) handleGetSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	state, err := s.agent.GetState(sessionID)
	if err != nil {
		s.logger.Errorf("Failed to get session state: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, state)
}

// handleListSessions 列出所有会话
func (s *HTTPServer) handleListSessions(c *gin.Context) {
	// 这里需要实现会话列表功能
	// 暂时返回空列表
	c.JSON(http.StatusOK, gin.H{
		"sessions": []interface{}{},
		"message":  "Session listing not implemented yet",
	})
}

// handleHealth 健康检查
func (s *HTTPServer) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// handleGetTools 获取可用工具列表
func (s *HTTPServer) handleGetTools(c *gin.Context) {
	// 这里需要通过Agent获取工具列表
	// 暂时返回固定列表
	tools := []gin.H{
		{"name": "read", "description": "Read file content"},
		{"name": "write", "description": "Write file content"},
		{"name": "edit", "description": "Edit file content"},
		{"name": "glob", "description": "Find files by pattern"},
		{"name": "grep", "description": "Search text in files"},
		{"name": "bash", "description": "Execute bash commands"},
		{"name": "todo_read", "description": "Read todo list"},
		{"name": "todo_write", "description": "Update todo list"},
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": tools,
		"count": len(tools),
	})
}

// loggingMiddleware 日志中间件
func (s *HTTPServer) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		s.logger.WithFields(log.Fields{
			"status":     param.StatusCode,
			"method":     param.Method,
			"path":       param.Path,
			"ip":         param.ClientIP,
			"latency":    param.Latency,
			"user_agent": param.Request.UserAgent(),
		}).Info("HTTP request")
		return ""
	})
}

// corsMiddleware CORS中间件
func (s *HTTPServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// handleGetSpeechConfig 获取语音配置
func (s *HTTPServer) handleGetSpeechConfig(c *gin.Context) {
	// 构建语音配置响应
	config := SpeechConfigResponse{
		Enabled:     s.speechConfig.Enabled,
		WakeWords:   s.speechConfig.WakeWords,
		WakeTimeout: s.speechConfig.WakeTimeout,
		Language:    s.speechConfig.Language,
	}

	// 设置默认值
	if len(config.WakeWords) == 0 {
		config.WakeWords = []string{"小助手", "助手", "hello", "hey"}
	}
	if config.WakeTimeout == 0 {
		config.WakeTimeout = 30
	}
	if config.Language == "" {
		config.Language = "zh-CN"
	}

	c.JSON(http.StatusOK, config)
}

// handleGetFileTree 获取文件树
func (s *HTTPServer) handleGetFileTree(c *gin.Context) {
	// 获取查询参数
	path := c.Query("path")
	if path == "" {
		// 获取当前工作目录
		currentDir, err := os.Getwd()
		if err != nil {
			s.logger.Error("Failed to get current directory", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current directory"})
			return
		}
		path = currentDir
	}

	// 构建文件树
	root, err := s.buildFileTree(path, 0, 20) // 限制深度为20
	if err != nil {
		s.logger.Error("Failed to build file tree", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tree": root,
		"path": path,
	})
}

// handleGetFileContent 获取文件内容
func (s *HTTPServer) handleGetFileContent(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		s.logger.Error("Failed to stat file", "path", filePath, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 检查是否为文件
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory, not a file"})
		return
	}

	// 检查文件大小（限制为1MB）
	if info.Size() > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 1MB)"})
		return
	}

	// 检查是否为文本文件
	if !s.isTextFile(filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is not a text file"})
		return
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		s.logger.Error("Failed to read file", "path", filePath, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// 检测编程语言
	language := s.detectLanguage(filePath)

	response := FileContentResponse{
		Path:     filePath,
		Content:  string(content),
		Language: language,
		Size:     info.Size(),
		ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, response)
}

// buildFileTree 构建文件树
func (s *HTTPServer) buildFileTree(path string, currentDepth, maxDepth int) (*FileNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &FileNode{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	if info.IsDir() {
		node.Type = "directory"
		node.Name = filepath.Base(path)
		if node.Name == "." {
			node.Name = filepath.Base(filepath.Dir(path))
		}

		// 如果还未达到最大深度，继续构建子目录
		if currentDepth < maxDepth {
			entries, err := ioutil.ReadDir(path)
			if err != nil {
				s.logger.Warn("Failed to read directory", "path", path, "error", err)
				return node, nil
			}

			var children []*FileNode
			for _, entry := range entries {
				// 跳过隐藏文件和一些常见的忽略目录
				if s.shouldSkipFile(entry.Name()) {
					continue
				}

				childPath := filepath.Join(path, entry.Name())
				child, err := s.buildFileTree(childPath, currentDepth+1, maxDepth)
				if err != nil {
					s.logger.Warn("Failed to build child tree", "path", childPath, "error", err)
					continue
				}
				children = append(children, child)
			}

			// 按类型和名称排序：目录在前，文件在后
			sort.Slice(children, func(i, j int) bool {
				if children[i].Type != children[j].Type {
					return children[i].Type == "directory"
				}
				return children[i].Name < children[j].Name
			})

			node.Children = children
		}
	} else {
		node.Type = "file"
	}

	return node, nil
}

// shouldSkipFile 判断是否应该跳过文件/目录
func (s *HTTPServer) shouldSkipFile(name string) bool {
	// 跳过隐藏文件
	if strings.HasPrefix(name, ".") {
		return true
	}

	// 跳过常见的忽略目录
	skipDirs := []string{
		"node_modules", "vendor", "target", "build", "dist",
		"logs", "log", "tmp", "temp", ".git", ".svn",
		"__pycache__", ".pytest_cache", ".coverage",
	}

	for _, skipDir := range skipDirs {
		if name == skipDir {
			return true
		}
	}

	return false
}

// isTextFile 判断是否为文本文件
func (s *HTTPServer) isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 首先通过扩展名快速判断已知的文本文件类型
	textExts := []string{
		".txt", ".md", ".markdown", ".rst", ".adoc",
		".go", ".py", ".js", ".ts", ".jsx", ".tsx",
		".html", ".htm", ".css", ".scss", ".sass", ".less",
		".json", ".xml", ".yaml", ".yml", ".toml", ".ini",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx",
		".java", ".kt", ".scala", ".clj", ".cljs",
		".rb", ".php", ".pl", ".r", ".sql",
		".vim", ".lua", ".dart", ".swift", ".rs",
		".dockerfile", ".makefile", ".gitignore", ".gitattributes",
		".env", ".properties", ".conf", ".config",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	// 检查一些无扩展名的常见文本文件
	base := strings.ToLower(filepath.Base(filePath))
	textFiles := []string{
		"makefile", "dockerfile", "license", "readme",
		"changelog", "authors", "contributors", "copying",
	}

	for _, textFile := range textFiles {
		if base == textFile {
			return true
		}
	}

	// 已知的二进制文件扩展名，直接返回false
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".lib",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".tiff", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".wav", ".ogg",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".bin", ".dat", ".db", ".sqlite", ".sqlite3",
		".class", ".jar", ".war", ".ear",
		".o", ".obj", ".pyc", ".pyo",
	}

	for _, binaryExt := range binaryExts {
		if ext == binaryExt {
			return false
		}
	}

	// 对于未知扩展名的文件，读取前1KB内容进行二进制检测
	return s.isTextFileByContent(filePath)
}

// isTextFileByContent 通过读取文件内容判断是否为文本文件
func (s *HTTPServer) isTextFileByContent(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		s.logger.Warn("Failed to open file for content detection", "path", filePath, "error", err)
		return false
	}
	defer file.Close()

	// 读取前1KB内容
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		s.logger.Warn("Failed to read file for content detection", "path", filePath, "error", err)
		return false
	}

	// 如果文件为空，认为是文本文件
	if n == 0 {
		return true
	}

	// 检查是否包含NULL字节，这是二进制文件的强烈指示
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	// 统计非打印字符的比例
	nonPrintableCount := 0
	for i := 0; i < n; i++ {
		b := buffer[i]
		// 允许的控制字符：制表符(9)、换行符(10)、回车符(13)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintableCount++
		} else if b > 126 {
			// 检查是否为有效的UTF-8字符
			if !s.isValidUTF8Byte(buffer, i, n) {
				nonPrintableCount++
			}
		}
	}

	// 如果非打印字符比例超过30%，认为是二进制文件
	threshold := float64(n) * 0.3
	return float64(nonPrintableCount) <= threshold
}

// isValidUTF8Byte 检查从指定位置开始是否为有效的UTF-8字符
func (s *HTTPServer) isValidUTF8Byte(buffer []byte, start, length int) bool {
	if start >= length {
		return false
	}

	b := buffer[start]

	// ASCII字符
	if b <= 127 {
		return true
	}

	// UTF-8多字节字符
	var expectedBytes int
	if b >= 0xC0 && b <= 0xDF {
		expectedBytes = 2
	} else if b >= 0xE0 && b <= 0xEF {
		expectedBytes = 3
	} else if b >= 0xF0 && b <= 0xF7 {
		expectedBytes = 4
	} else {
		return false
	}

	// 检查是否有足够的字节
	if start+expectedBytes > length {
		return false
	}

	// 检查后续字节是否符合UTF-8格式
	for i := 1; i < expectedBytes; i++ {
		if start+i >= length {
			return false
		}
		nextByte := buffer[start+i]
		if nextByte < 0x80 || nextByte > 0xBF {
			return false
		}
	}

	return true
}

// detectLanguage 检测编程语言
func (s *HTTPServer) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	langMap := map[string]string{
		".go":       "go",
		".py":       "python",
		".js":       "javascript",
		".ts":       "typescript",
		".jsx":      "javascript",
		".tsx":      "typescript",
		".html":     "html",
		".htm":      "html",
		".css":      "css",
		".scss":     "scss",
		".sass":     "sass",
		".less":     "less",
		".json":     "json",
		".xml":      "xml",
		".yaml":     "yaml",
		".yml":      "yaml",
		".toml":     "toml",
		".ini":      "ini",
		".sh":       "bash",
		".bash":     "bash",
		".zsh":      "zsh",
		".fish":     "fish",
		".ps1":      "powershell",
		".bat":      "batch",
		".cmd":      "batch",
		".c":        "c",
		".cpp":      "cpp",
		".cc":       "cpp",
		".cxx":      "cpp",
		".h":        "c",
		".hpp":      "cpp",
		".hxx":      "cpp",
		".java":     "java",
		".kt":       "kotlin",
		".scala":    "scala",
		".clj":      "clojure",
		".cljs":     "clojure",
		".rb":       "ruby",
		".php":      "php",
		".pl":       "perl",
		".r":        "r",
		".sql":      "sql",
		".vim":      "vim",
		".lua":      "lua",
		".dart":     "dart",
		".swift":    "swift",
		".rs":       "rust",
		".md":       "markdown",
		".markdown": "markdown",
		".rst":      "rst",
		".adoc":     "asciidoc",
		".txt":      "text",
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	// 检查一些特殊的文件名
	base := strings.ToLower(filepath.Base(filePath))
	if base == "dockerfile" {
		return "dockerfile"
	}
	if base == "makefile" {
		return "makefile"
	}

	return "text"
}
