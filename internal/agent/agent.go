package agent

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

// Agent 主要Agent实现
type Agent struct {
	config         *Config
	llmManager     types.LLMClient
	toolEngine     types.ToolEngine
	contextManager types.ContextManager
	promptManager  types.PromptManager
	logger         log.Logger
}

// Config Agent配置
type Config struct {
	MaxLoops           int `mapstructure:"max_loops"`
	ContextWindow      int `mapstructure:"context_window"`
	MaxToolConcurrency int `mapstructure:"max_tool_concurrency"`
}

// NewAgent 创建Agent
func NewAgent(
	config *Config,
	llmManager types.LLMClient,
	toolEngine types.ToolEngine,
	contextManager types.ContextManager,
	promptManager types.PromptManager,
	logger log.Logger,
) *Agent {
	return &Agent{
		config:         config,
		llmManager:     llmManager,
		toolEngine:     toolEngine,
		contextManager: contextManager,
		promptManager:  promptManager,
		logger:         logger,
	}
}

// Chat 处理聊天请求
func (a *Agent) Chat(ctx context.Context, request types.ChatRequest) (*types.ChatResponse, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = utils.GenerateID()
	}

	// 添加用户消息到上下文
	userMessage := types.Message{
		ID:        utils.GenerateID(),
		Role:      types.RoleUser,
		Content:   request.Message,
		Metadata:  request.Metadata,
		Timestamp: time.Now(),
	}

	if err := a.contextManager.AddMessage(ctx, sessionID, userMessage); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// 执行Agent循环
	response, usage, err := a.runAgentLoop(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("agent loop failed: %w", err)
	}

	return &types.ChatResponse{
		SessionID: sessionID,
		Response:  response,
		Finished:  true,
		Usage:     usage,
		Metadata: map[string]interface{}{
			"loop_completed": true,
		},
	}, nil
}

// ChatStream 处理流式聊天请求
func (a *Agent) ChatStream(ctx context.Context, request types.ChatRequest) (<-chan types.ChatResponse, error) {
	a.logger.Debugf("ChatStream request: %+v", request)

	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = utils.GenerateID()
	}

	// 添加用户消息到上下文
	userMessage := types.Message{
		ID:        utils.GenerateID(),
		Role:      types.RoleUser,
		Content:   request.Message,
		Metadata:  request.Metadata,
		Timestamp: time.Now(),
	}

	if err := a.contextManager.AddMessage(ctx, sessionID, userMessage); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// 创建响应通道
	responseChan := make(chan types.ChatResponse, 10)

	// 启动流式处理
	go func() {
		defer close(responseChan)

		usage, err := a.runAgentLoopStream(ctx, sessionID, responseChan)
		if err != nil {
			responseChan <- types.ChatResponse{
				SessionID: sessionID,
				Response:  fmt.Sprintf("Error: %v", err),
				Finished:  true,
				Usage:     usage,
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		// 发送最终响应
		responseChan <- types.ChatResponse{
			SessionID: sessionID,
			Response:  "",
			Finished:  true,
			Usage:     usage,
			Metadata: map[string]interface{}{
				"loop_completed": true,
			},
		}
	}()

	return responseChan, nil
}

// GetState 获取Agent状态
func (a *Agent) GetState(sessionID string) (*types.AgentState, error) {
	sessionContext, err := a.contextManager.GetSessionContext(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session context: %w", err)
	}

	activeTools := make([]string, 0)
	for _, tool := range a.toolEngine.GetToolDefinitions() {
		activeTools = append(activeTools, tool.Function.Name)
	}

	return &types.AgentState{
		SessionID:         sessionID,
		Status:            "ready",
		CurrentLoop:       0,
		Messages:          sessionContext.Messages,
		CompressedHistory: sessionContext.CompressedHistory,
		ActiveTools:       activeTools,
		LastActivity:      sessionContext.LastActivity,
	}, nil
}

// runAgentLoop 运行Agent主循环
func (a *Agent) runAgentLoop(ctx context.Context, sessionID string) (string, types.Usage, error) {
	var totalUsage types.Usage
	var finalResponse string

	for loop := 0; loop < a.config.MaxLoops; loop++ {
		a.logger.Debugf("Agent loop %d/%d for session %s", loop+1, a.config.MaxLoops, sessionID)

		// 构建LLM请求
		llmRequest, err := a.buildLLMRequest(ctx, sessionID)
		if err != nil {
			return "", totalUsage, fmt.Errorf("failed to build LLM request: %w", err)
		}

		// 调用LLM
		llmResponse, err := a.llmManager.Chat(ctx, *llmRequest)
		if err != nil {
			return "", totalUsage, fmt.Errorf("LLM call failed: %w", err)
		}

		// 累积使用量
		totalUsage.PromptTokens += llmResponse.Usage.PromptTokens
		totalUsage.CompletionTokens += llmResponse.Usage.CompletionTokens
		totalUsage.TotalTokens += llmResponse.Usage.TotalTokens

		// 添加助手响应到上下文
		assistantMessage := types.Message{
			ID:        utils.GenerateID(),
			Role:      types.RoleAssistant,
			Content:   llmResponse.Content,
			ToolCalls: llmResponse.ToolCalls,
			Timestamp: time.Now(),
		}

		if err := a.contextManager.AddMessage(ctx, sessionID, assistantMessage); err != nil {
			return "", totalUsage, fmt.Errorf("failed to add assistant message: %w", err)
		}

		finalResponse = llmResponse.Content

		// 如果没有工具调用，结束循环
		if len(llmResponse.ToolCalls) == 0 {
			break
		}

		// 执行工具调用
		if err := a.executeToolCalls(ctx, sessionID, llmResponse.ToolCalls); err != nil {
			a.logger.Errorf("Tool execution failed: %v", err)
			// 继续循环，让LLM处理错误
		}
	}

	return finalResponse, totalUsage, nil
}

// runAgentLoopStream 运行流式Agent循环
func (a *Agent) runAgentLoopStream(ctx context.Context, sessionID string, responseChan chan<- types.ChatResponse) (types.Usage, error) {
	var totalUsage types.Usage

	for loop := 0; loop < a.config.MaxLoops; loop++ {
		a.logger.Debugf("Agent stream loop %d/%d for session %s", loop+1, a.config.MaxLoops, sessionID)

		// 构建LLM请求
		llmRequest, err := a.buildLLMRequest(ctx, sessionID)
		if err != nil {
			return totalUsage, fmt.Errorf("failed to build LLM request: %w", err)
		}

		// 设置流式请求
		llmRequest.Stream = true

		// 调用LLM流式API
		llmStream, err := a.llmManager.ChatStream(ctx, *llmRequest)
		if err != nil {
			return totalUsage, fmt.Errorf("LLM stream call failed: %w", err)
		}

		var streamContent strings.Builder
		var toolCalls []types.ToolCall

		// 处理流式响应
		for streamResp := range llmStream {
			if streamResp.Content != "" {
				streamContent.WriteString(streamResp.Content)

				// 发送增量响应
				responseChan <- types.ChatResponse{
					SessionID: sessionID,
					Response:  streamResp.Content,
					Finished:  false,
					Usage:     streamResp.Usage,
				}
			}

			// 处理工具调用
			if len(streamResp.ToolCalls) > 0 {
				toolCalls = append(toolCalls, streamResp.ToolCalls...)
			}

			// 累积使用量
			totalUsage.PromptTokens += streamResp.Usage.PromptTokens
			totalUsage.CompletionTokens += streamResp.Usage.CompletionTokens
			totalUsage.TotalTokens += streamResp.Usage.TotalTokens
		}

		// 添加助手响应到上下文
		assistantMessage := types.Message{
			ID:        utils.GenerateID(),
			Role:      types.RoleAssistant,
			Content:   streamContent.String(),
			ToolCalls: toolCalls,
			Timestamp: time.Now(),
		}

		if err := a.contextManager.AddMessage(ctx, sessionID, assistantMessage); err != nil {
			return totalUsage, fmt.Errorf("failed to add assistant message: %w", err)
		}

		// 如果没有工具调用，结束循环
		if len(toolCalls) == 0 {
			a.logger.Debugf("No tool calls found, ending loop for session %s", sessionID)
			break
		}

		// 执行工具调用
		if err := a.executeToolCalls(ctx, sessionID, toolCalls); err != nil {
			a.logger.Errorf("Tool execution failed: %v", err)
		}
	}

	return totalUsage, nil
}

// buildLLMRequest 构建LLM请求
func (a *Agent) buildLLMRequest(ctx context.Context, sessionID string) (*types.LLMRequest, error) {
	// 获取系统提示词
	systemPrompt, err := a.promptManager.GetPromptWithData("system", map[string]any{
		"model_provider": a.llmManager.GetProvider(),
	})
	if err != nil {
		a.logger.Warnf("Failed to get system prompt: %v", err)
		systemPrompt = "You are a helpful AI assistant."
	}

	// 获取用户信息提示词
	pwd, err := os.Getwd()
	if err != nil {
		a.logger.Warnf("Failed to get current working directory: %v", err)
		pwd = "unknown"
	}
	fileStructure, err := utils.BFSDirectoryTraversal(pwd, 200)
	if err != nil {
		a.logger.Warnf("Failed to get file structure: %v", err)
		fileStructure = "unknown"
	}
	userInfoPrompt, err := a.promptManager.GetPromptWithData("user_info", map[string]interface{}{
		"os":             runtime.GOOS,
		"pwd":            pwd,
		"shell":          os.Getenv("SHELL"),
		"date":           time.Now().Format("2006-01-02 15:04:05"),
		"file_structure": fileStructure,
	})
	if err != nil {
		a.logger.Warnf("Failed to get user info prompt: %v", err)
		userInfoPrompt = ""
	}

	// 获取历史消息
	messages, err := a.contextManager.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	id := utils.GenerateID()
	// 构建消息列表
	llmMessages := []types.Message{
		{
			ID:      id,
			Role:    types.RoleSystem,
			Content: systemPrompt,
		},
		{
			ID:      id,
			Role:    types.RoleUser,
			Content: userInfoPrompt,
		},
	}

	// 添加历史消息
	llmMessages = append(llmMessages, messages...)

	// 获取工具定义
	tools := a.toolEngine.GetToolDefinitions()

	return &types.LLMRequest{
		Messages: llmMessages,
		Tools:    tools,
		Stream:   false,
	}, nil
}

// executeToolCalls 执行工具调用
func (a *Agent) executeToolCalls(ctx context.Context, sessionID string, toolCalls []types.ToolCall) error {
	if len(toolCalls) == 0 {
		return nil
	}

	a.logger.Debugf("Executing %d tool calls for session %s", len(toolCalls), sessionID)

	// 执行工具
	results := a.toolEngine.ExecuteTools(ctx, toolCalls)

	// 为每个工具调用添加结果消息
	for i, result := range results {
		if i < len(toolCalls) {
			toolMessage := types.Message{
				ID:      utils.GenerateID(),
				Role:    types.RoleTool,
				Content: a.formatToolResult(toolCalls[i], result),
				Metadata: map[string]string{
					"tool_call_id": toolCalls[i].ID,
					"tool_name":    toolCalls[i].Function.Name,
					"success":      fmt.Sprintf("%t", result.Success),
				},
				Timestamp: time.Now(),
			}

			if err := a.contextManager.AddMessage(ctx, sessionID, toolMessage); err != nil {
				a.logger.Errorf("Failed to add tool result message: %v", err)
			}
		}
	}

	return nil
}

// formatToolResult 格式化工具执行结果
func (a *Agent) formatToolResult(call types.ToolCall, result types.ToolCallResult) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Tool: %s\n", call.Function.Name))
	output.WriteString(fmt.Sprintf("Success: %t\n", result.Success))

	if result.Error != "" {
		output.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
	}

	if result.Content != "" {
		output.WriteString(fmt.Sprintf("Output:\n%s", result.Content))
	}

	return output.String()
}
