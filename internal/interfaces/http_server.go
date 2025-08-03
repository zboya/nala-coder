package interfaces

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
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

		// 语音配置（保留基本配置接口）
		api.GET("/speech/config", s.handleGetSpeechConfig)

		// 系统信息
		api.GET("/health", s.handleHealth)
		api.GET("/tools", s.handleGetTools)
	}

	// 设置嵌入式HTML模板
	templ := template.Must(template.New("").ParseFS(embedded.GetWebFS(), "web/templates/*"))
	router.SetHTMLTemplate(templ)

	// 设置嵌入式静态文件
	staticFS, err := fs.Sub(embedded.GetWebFS(), "web/static")
	if err != nil {
		s.logger.Error("Failed to create static filesystem", "error", err)
		// 如果子文件系统创建失败，使用原始方式
		router.StaticFS("/static", http.FS(embedded.GetWebFS()))
	} else {
		router.StaticFS("/static", http.FS(staticFS))
	}

	// 默认页面
	router.GET("/", func(c *gin.Context) {
		// 获取当前工作目录
		currentDir, err := os.Getwd()
		if err != nil {
			s.logger.Error("Failed to get current directory", "error", err)
			currentDir = "未知路径"
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":       "NaLa Coder",
			"projectPath": currentDir,
		})
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
