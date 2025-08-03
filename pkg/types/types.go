package types

import (
	"context"
	"time"
)

// Message 代表一条对话消息
type Message struct {
	ID        string            `json:"id"`
	Role      MessageRole       `json:"role"`
	Content   string            `json:"content"`
	ToolCalls []ToolCall        `json:"tool_calls,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// ToolCall 工具调用
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ToolCallFunction       `json:"function"`
	Result   *ToolCallResult        `json:"result,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCallFunction 工具调用函数
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content   string    `json:"content"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// LLMProvider 大模型提供商类型
type LLMProvider string

const (
	ProviderOpenAI   LLMProvider = "openai"
	ProviderDeepSeek LLMProvider = "deepseek"
	ProviderClaude   LLMProvider = "claude"
	ProviderOllama   LLMProvider = "ollama"
)

// LLMConfig 大模型配置
type LLMConfig struct {
	Provider    LLMProvider `mapstructure:"provider"`
	APIKey      string      `mapstructure:"api_key"`
	BaseURL     string      `mapstructure:"base_url"`
	Model       string      `mapstructure:"model"`
	MaxTokens   int         `mapstructure:"max_tokens"`
	Temperature float64     `mapstructure:"temperature"`
}

// LLMRequest 大模型请求
type LLMRequest struct {
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Model       string    `json:"model,omitempty"`
}

// LLMResponse 大模型响应
type LLMResponse struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Role      string     `json:"role"`
	Usage     Usage      `json:"usage"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Usage token使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Tool 工具定义
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction 工具函数定义
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// SpeechConfig 语音识别配置（仅前端配置）
type SpeechConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	WakeWords   []string `mapstructure:"wake_words"`
	WakeTimeout int      `mapstructure:"wake_timeout"`
	Language    string   `mapstructure:"language"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	MaxLoops             int     `mapstructure:"max_loops"`
	ContextWindow        int     `mapstructure:"context_window"`
	CompressionThreshold float64 `mapstructure:"compression_threshold"`
	MaxToolConcurrency   int     `mapstructure:"max_tool_concurrency"`
}

// AgentState Agent状态
type AgentState struct {
	SessionID         string    `json:"session_id"`
	Status            string    `json:"status"`
	CurrentLoop       int       `json:"current_loop"`
	Messages          []Message `json:"messages"`
	CompressedHistory string    `json:"compressed_history,omitempty"`
	ActiveTools       []string  `json:"active_tools"`
	LastActivity      time.Time `json:"last_activity"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Message   string            `json:"message"`
	SessionID string            `json:"session_id,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	SessionID string                 `json:"session_id"`
	Response  string                 `json:"response"`
	Finished  bool                   `json:"finished"`
	Usage     Usage                  `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionContext 会话上下文
type SessionContext struct {
	ID                string            `json:"id"`
	Messages          []Message         `json:"messages"`
	CompressedHistory string            `json:"compressed_history,omitempty"`
	Metadata          map[string]string `json:"metadata"`
	CreatedAt         time.Time         `json:"created_at"`
	LastActivity      time.Time         `json:"last_activity"`
	TotalTokens       int               `json:"total_tokens"`
}

// ContextManager 上下文管理器接口
type ContextManager interface {
	AddMessage(ctx context.Context, sessionID string, message Message) error
	GetMessages(ctx context.Context, sessionID string) ([]Message, error)
	CompressHistory(ctx context.Context, sessionID string) error
	LoadPersistentContext(ctx context.Context, sessionID string) (string, error)
	SavePersistentContext(ctx context.Context, sessionID string, context string) error
	GetSessionContext(sessionID string) (*SessionContext, error)
}

// LLMClient 大模型客户端接口
type LLMClient interface {
	GetProvider() LLMProvider
	Chat(ctx context.Context, request LLMRequest) (*LLMResponse, error)
	ChatStream(ctx context.Context, request LLMRequest) (<-chan LLMResponse, error)
	GetConfig() LLMConfig
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	Name() string
	Execute(ctx context.Context, call ToolCall) *ToolCallResult
	GetDefinition() Tool
	IsConcurrencySafe() bool
}

// ToolEngine 工具引擎接口
type ToolEngine interface {
	RegisterTool(name string, executor ToolExecutor) error
	ExecuteTools(ctx context.Context, calls []ToolCall) []ToolCallResult
	GetToolDefinitions() []Tool
	GetTool(name string) (ToolExecutor, bool)
}

// Agent 主要Agent接口
type Agent interface {
	Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, request ChatRequest) (<-chan ChatResponse, error)
	GetState(sessionID string) (*AgentState, error)
}

// PromptManager 提示词管理器接口
type PromptManager interface {
	GetPrompt(name string) (string, error)
	GetPromptWithData(name string, data map[string]any) (string, error)
	ReloadPrompts() error
	WatchPrompts() error
}
