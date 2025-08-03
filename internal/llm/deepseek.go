package llm

import (
	"context"
	"encoding/json"
	"fmt"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// DeepSeekClient DeepSeek客户端 (使用 github.com/cohesion-org/deepseek-go)
type DeepSeekClient struct {
	client *deepseek.Client
	config types.LLMConfig
	logger log.Logger
}

// NewDeepSeekClient 创建DeepSeek客户端
func NewDeepSeekClient(config types.LLMConfig, logger log.Logger) *DeepSeekClient {
	var client *deepseek.Client

	if config.BaseURL != "" {
		// 使用自定义 BaseURL
		client = deepseek.NewClient(config.APIKey, config.BaseURL)
	} else {
		// 使用默认 DeepSeek API
		client = deepseek.NewClient(config.APIKey)
	}

	if client == nil {
		logger.Error("Failed to create DeepSeek client")
		return nil
	}

	return &DeepSeekClient{
		client: client,
		config: config,
		logger: logger,
	}
}

// GetConfig 获取配置
func (c *DeepSeekClient) GetConfig() types.LLMConfig {
	return c.config
}

// GetProvider 获取提供商
func (c *DeepSeekClient) GetProvider() types.LLMProvider {
	return types.ProviderDeepSeek
}

// Chat 对话
func (c *DeepSeekClient) Chat(ctx context.Context, request types.LLMRequest) (*types.LLMResponse, error) {
	dsRequest := c.convertToDeepSeekRequest(request)

	c.logger.Infof("DeepSeek request: model=%s, messages=%d", dsRequest.Model, len(dsRequest.Messages))

	response, err := c.client.CreateChatCompletion(ctx, dsRequest)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek API error: %w", err)
	}

	return c.convertFromDeepSeekResponse(response), nil
}

// ChatStream 流式对话
func (c *DeepSeekClient) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	dsRequest := c.convertToDeepSeekStreamRequest(request)

	c.logger.Infof("DeepSeek stream request: model=%s, messages=%d", dsRequest.Model, len(dsRequest.Messages))

	dsRequestBytes, _ := json.Marshal(dsRequest)
	c.logger.Debugf("DeepSeek stream request: %s", string(dsRequestBytes))

	stream, err := c.client.CreateChatCompletionStream(ctx, dsRequest)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek stream API error: %w", err)
	}

	responseChan := make(chan types.LLMResponse, 10)

	go func() {
		defer close(responseChan)
		defer stream.Close()

		// 使用map来跟踪正在构建的工具调用，key是index
		toolCallsMap := make(map[int]*types.ToolCall)
		var finalResponse *deepseek.StreamChatCompletionResponse

		buildFinalResponse := func() types.LLMResponse {
			// 构建最终的工具调用数组
			var toolCalls []types.ToolCall
			for i := 0; i < len(toolCallsMap); i++ {
				if tc, exists := toolCallsMap[i]; exists {
					toolCalls = append(toolCalls, *tc)
				}
			}

			// 发送最终响应
			var responseID string
			if finalResponse != nil {
				responseID = finalResponse.ID
			}
			return types.LLMResponse{
				ID:        responseID,
				Content:   "",
				Role:      "assistant",
				ToolCalls: toolCalls,
			}
		}

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					c.logger.Errorf("DeepSeek stream error: %v", err)
				}
				// 发送最终响应
				responseChan <- buildFinalResponse()
				return
			}

			finalResponse = response

			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]
			choiceBytes, _ := json.Marshal(choice)
			c.logger.Debugf("DeepSeek stream choice: %s", string(choiceBytes))

			// 处理工具调用流式数据
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					index := tc.Index // DeepSeek工具调用的index

					// 如果是新的工具调用，初始化
					if _, exists := toolCallsMap[index]; !exists {
						toolCallsMap[index] = &types.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: types.ToolCallFunction{
								Name:      tc.Function.Name,
								Arguments: "",
							},
						}
					}

					// 累积arguments
					if tc.Function.Arguments != "" {
						toolCallsMap[index].Function.Arguments += tc.Function.Arguments
					}

					// 更新其他字段（如果有的话）
					if tc.ID != "" {
						toolCallsMap[index].ID = tc.ID
					}
					if tc.Type != "" {
						toolCallsMap[index].Type = tc.Type
					}
					if tc.Function.Name != "" {
						toolCallsMap[index].Function.Name = tc.Function.Name
					}
				}
			}
			// 发送增量响应（不包含工具调用，避免重复发送未完成的工具调用）
			resp := types.LLMResponse{
				ID:      response.ID,
				Content: choice.Delta.Content,
				Role:    "assistant",
			}

			select {
			case responseChan <- resp:
			case <-ctx.Done():
				return
			}
			// 检查是否完成
			if choice.FinishReason != "" {
				responseChan <- buildFinalResponse()
				return
			}
		}
	}()

	return responseChan, nil
}

// convertToDeepSeekRequest 将内部请求转换为 DeepSeek 请求
func (c *DeepSeekClient) convertToDeepSeekRequest(request types.LLMRequest) *deepseek.ChatCompletionRequest {
	messages := make([]deepseek.ChatCompletionMessage, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = deepseek.ChatCompletionMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.Metadata["tool_call_id"],
		}
		// 转换工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]deepseek.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = deepseek.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: deepseek.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			messages[i].ToolCalls = toolCalls
		}
	}

	// 转换工具定义
	var tools []deepseek.Tool
	if len(request.Tools) > 0 {
		tools = make([]deepseek.Tool, len(request.Tools))
		for i, tool := range request.Tools {
			tools[i] = deepseek.Tool{
				Type: tool.Type,
				Function: deepseek.Function{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  c.convertParameters(tool.Function.Parameters),
				},
			}
		}
	}

	return &deepseek.ChatCompletionRequest{
		Model:       c.getModel(request.Model),
		Messages:    messages,
		MaxTokens:   c.getMaxTokens(request.MaxTokens),
		Temperature: c.getTemperature(request.Temperature),
		Tools:       tools,
	}
}

// convertToDeepSeekStreamRequest 将内部请求转换为 DeepSeek 流式请求
func (c *DeepSeekClient) convertToDeepSeekStreamRequest(request types.LLMRequest) *deepseek.StreamChatCompletionRequest {
	messages := make([]deepseek.ChatCompletionMessage, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = deepseek.ChatCompletionMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.Metadata["tool_call_id"],
		}

		// 转换工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]deepseek.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = deepseek.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: deepseek.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			messages[i].ToolCalls = toolCalls
		}
	}

	// 转换工具定义
	var tools []deepseek.Tool
	if len(request.Tools) > 0 {
		tools = make([]deepseek.Tool, len(request.Tools))
		for i, tool := range request.Tools {
			tools[i] = deepseek.Tool{
				Type: tool.Type,
				Function: deepseek.Function{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  c.convertParameters(tool.Function.Parameters),
				},
			}
		}
	}

	return &deepseek.StreamChatCompletionRequest{
		Model:       c.getModel(request.Model),
		Messages:    messages,
		MaxTokens:   c.getMaxTokens(request.MaxTokens),
		Temperature: c.getTemperature(request.Temperature),
		Tools:       tools,
		Stream:      true,
	}
}

// convertFromDeepSeekResponse 将 DeepSeek 响应转换为内部响应
func (c *DeepSeekClient) convertFromDeepSeekResponse(response *deepseek.ChatCompletionResponse) *types.LLMResponse {
	var content string
	var toolCalls []types.ToolCall

	if len(response.Choices) > 0 {
		choice := response.Choices[0]
		content = choice.Message.Content

		// 转换工具调用
		if len(choice.Message.ToolCalls) > 0 {
			toolCalls = make([]types.ToolCall, len(choice.Message.ToolCalls))
			for i, tc := range choice.Message.ToolCalls {
				toolCalls[i] = types.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: types.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	return &types.LLMResponse{
		ID:      response.ID,
		Content: content,
		Role:    "assistant",
		Usage: types.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
		ToolCalls: toolCalls,
	}
}

// 辅助方法
func (c *DeepSeekClient) getModel(model string) string {
	if model != "" {
		return model
	}
	if c.config.Model != "" {
		return c.config.Model
	}
	return deepseek.DeepSeekChat // 默认模型
}

func (c *DeepSeekClient) getMaxTokens(maxTokens int) int {
	if maxTokens > 0 {
		return maxTokens
	}
	if c.config.MaxTokens > 0 {
		return c.config.MaxTokens
	}
	return 4000 // 默认值
}

func (c *DeepSeekClient) getTemperature(temperature float64) float32 {
	if temperature > 0 {
		return float32(temperature)
	}
	if c.config.Temperature > 0 {
		return float32(c.config.Temperature)
	}
	return 0.7 // 默认值
}

func (c *DeepSeekClient) convertParameters(params interface{}) *deepseek.FunctionParameters {
	if params == nil {
		return nil
	}

	// 简单的类型转换，实际使用中可能需要更复杂的转换逻辑
	if paramsMap, ok := params.(map[string]interface{}); ok {
		result := &deepseek.FunctionParameters{
			Type: "object",
		}

		if properties, exists := paramsMap["properties"]; exists {
			if props, ok := properties.(map[string]interface{}); ok {
				result.Properties = props
			}
		}

		if required, exists := paramsMap["required"]; exists {
			if req, ok := required.([]string); ok {
				result.Required = req
			}
		}

		return result
	}

	return nil
}
