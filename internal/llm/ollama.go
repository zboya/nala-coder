package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// OllamaClient Ollama客户端
type OllamaClient struct {
	config types.LLMConfig
	client *api.Client
	logger log.Logger
}

// NewOllamaClient 创建Ollama客户端
func NewOllamaClient(config types.LLMConfig, logger log.Logger) *OllamaClient {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		baseURL, _ = url.Parse("http://localhost:11434")
	}

	client := api.NewClient(baseURL, &http.Client{})
	return &OllamaClient{
		config: config,
		client: client,
		logger: logger,
	}
}

// GetConfig 获取配置
func (c *OllamaClient) GetConfig() types.LLMConfig {
	return c.config
}

// GetProvider 获取提供商
func (c *OllamaClient) GetProvider() types.LLMProvider {
	return types.ProviderOllama
}

// Chat 对话
func (c *OllamaClient) Chat(ctx context.Context, request types.LLMRequest) (*types.LLMResponse, error) {
	messages := c.convertMessages(request.Messages)

	chatRequest := &api.ChatRequest{
		Model:    c.getModel(request.Model),
		Messages: messages,
		Options:  c.buildOptions(request),
	}

	if len(request.Tools) > 0 {
		chatRequest.Tools = c.convertTools(request.Tools)
	}

	var response api.ChatResponse
	err := c.client.Chat(ctx, chatRequest, func(resp api.ChatResponse) error {
		response = resp
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Ollama chat error: %w", err)
	}

	return c.convertResponse(response), nil
}

// ChatStream 流式对话
func (c *OllamaClient) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	messages := c.convertMessages(request.Messages)

	chatRequest := &api.ChatRequest{
		Model:    c.getModel(request.Model),
		Messages: messages,
		Options:  c.buildOptions(request),
		Stream:   new(bool),
	}

	if len(request.Tools) > 0 {
		chatRequest.Tools = c.convertTools(request.Tools)
	}

	*chatRequest.Stream = true

	responseChan := make(chan types.LLMResponse, 10)

	go func() {
		defer close(responseChan)

		var fullContent string

		err := c.client.Chat(ctx, chatRequest, func(resp api.ChatResponse) error {
			content := resp.Message.Content
			fullContent += content

			streamResp := types.LLMResponse{
				Content: content,
				Role:    string(resp.Message.Role),
			}

			if resp.Message.ToolCalls != nil {
				streamResp.ToolCalls = c.convertToolCalls(resp.Message.ToolCalls)
			}

			responseChan <- streamResp

			if resp.Done {
				final := types.LLMResponse{
					Content: fullContent,
					Role:    string(resp.Message.Role),
					Usage: types.Usage{
						PromptTokens:     resp.PromptEvalCount,
						CompletionTokens: resp.EvalCount,
						TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
					},
				}
				if resp.Message.ToolCalls != nil {
					final.ToolCalls = c.convertToolCalls(resp.Message.ToolCalls)
				}
				responseChan <- final
			}
			return nil
		})

		if err != nil {
			c.logger.Error("Ollama stream chat error", "error", err)
		}
	}()

	return responseChan, nil
}

// convertMessages 转换消息格式
func (c *OllamaClient) convertMessages(messages []types.Message) []api.Message {
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = api.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// 转换工具调用
		if len(msg.ToolCalls) > 0 {
			ollamaMessages[i].ToolCalls = c.convertToolCallsToOllama(msg.ToolCalls)
		}
	}
	return ollamaMessages
}

// convertTools 转换工具格式
func (c *OllamaClient) convertTools(tools []types.Tool) []api.Tool {
	ollamaTools := make([]api.Tool, len(tools))
	for i, tool := range tools {
		oTool := api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters: struct {
					Type       string   `json:"type"`
					Defs       any      `json:"$defs,omitempty"`
					Items      any      `json:"items,omitempty"`
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type        api.PropertyType `json:"type"`
						Items       any              `json:"items,omitempty"`
						Description string           `json:"description"`
						Enum        []any            `json:"enum,omitempty"`
					} `json:"properties"`
				}{},
			},
		}
		properties := tool.Function.Parameters["properties"].(map[string]any)
		oTool.Function.Parameters.Type = "object"
		for k, v := range properties {
			v := v.(map[string]any)
			oTool.Function.Parameters.Properties[k] = struct {
				Type        api.PropertyType `json:"type"`
				Items       any              `json:"items,omitempty"`
				Description string           `json:"description"`
				Enum        []any            `json:"enum,omitempty"`
			}{
				Type:        api.PropertyType{v["type"].(string)},
				Description: v["description"].(string),
			}
		}

		ollamaTools[i] = oTool
	}
	return ollamaTools
}

// convertToolCalls 转换工具调用格式
func (c *OllamaClient) convertToolCalls(toolCalls []api.ToolCall) []types.ToolCall {
	result := make([]types.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		jsonArgs, _ := json.Marshal(tc.Function.Arguments)
		jsonArgs = []byte(url.QueryEscape(string(jsonArgs)))
		result[i] = types.ToolCall{
			ID: tc.Function.Name, // Ollama uses function name as ID
			Function: types.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: string(jsonArgs),
			},
		}
	}
	return result
}

// convertToolCallsToOllama 转换工具调用格式到Ollama格式
func (c *OllamaClient) convertToolCallsToOllama(toolCalls []types.ToolCall) []api.ToolCall {
	result := make([]api.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		args := api.ToolCallFunctionArguments{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		result[i] = api.ToolCall{
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		}
	}
	return result
}

// buildOptions 构建选项
func (c *OllamaClient) buildOptions(request types.LLMRequest) map[string]any {
	options := make(map[string]any)

	if request.MaxTokens > 0 {
		options["num_predict"] = request.MaxTokens
	} else if c.config.MaxTokens > 0 {
		options["num_predict"] = c.config.MaxTokens
	}

	if request.Temperature > 0 {
		options["temperature"] = request.Temperature
	} else if c.config.Temperature > 0 {
		options["temperature"] = c.config.Temperature
	}

	return options
}

// convertResponse 转换响应格式
func (c *OllamaClient) convertResponse(resp api.ChatResponse) *types.LLMResponse {
	result := &types.LLMResponse{
		Content: resp.Message.Content,
		Role:    string(resp.Message.Role),
		Usage: types.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}

	if resp.Message.ToolCalls != nil {
		result.ToolCalls = c.convertToolCalls(resp.Message.ToolCalls)
	}

	return result
}

// getModel 获取模型名称
func (c *OllamaClient) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return c.config.Model
}
