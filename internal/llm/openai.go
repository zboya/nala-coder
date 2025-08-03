package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// OpenAIClient OpenAI客户端
type OpenAIClient struct {
	client *openai.Client
	config types.LLMConfig
	logger log.Logger
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(config types.LLMConfig, logger log.Logger) *OpenAIClient {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	return &OpenAIClient{
		client: openai.NewClientWithConfig(clientConfig),
		config: config,
		logger: logger,
	}
}

// GetConfig 获取配置
func (c *OpenAIClient) GetConfig() types.LLMConfig {
	return c.config
}

// GetProvider 获取提供商
func (c *OpenAIClient) GetProvider() types.LLMProvider {
	return types.ProviderOpenAI
}

// Chat 对话
func (c *OpenAIClient) Chat(ctx context.Context, request types.LLMRequest) (*types.LLMResponse, error) {
	messages := c.convertMessages(request.Messages)
	tools := c.convertTools(request.Tools)

	req := openai.ChatCompletionRequest{
		Model:       c.getModel(request.Model),
		Messages:    messages,
		MaxTokens:   c.getMaxTokens(request.MaxTokens),
		Temperature: c.getTemperature(request.Temperature),
		Stream:      false,
	}

	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	jsonReq, _ := json.Marshal(req)
	c.logger.Infof("OpenAI request: %s", string(jsonReq))
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	return c.convertResponse(resp), nil
}

// ChatStream 流式对话
func (c *OpenAIClient) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	messages := c.convertMessages(request.Messages)
	tools := c.convertTools(request.Tools)

	req := openai.ChatCompletionRequest{
		Model:       c.getModel(request.Model),
		Messages:    messages,
		MaxTokens:   c.getMaxTokens(request.MaxTokens),
		Temperature: c.getTemperature(request.Temperature),
		Stream:      true,
	}

	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	jsonReq, _ := json.Marshal(req)
	c.logger.Infof("OpenAI stream request: %s", string(jsonReq))
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI stream API error: %w", err)
	}

	responseChan := make(chan types.LLMResponse, 10)

	go func() {
		defer close(responseChan)
		defer stream.Close()

		var fullContent string
		// 使用map来跟踪正在构建的工具调用，key是index
		toolCallsMap := make(map[int]*types.ToolCall)
		var finalResponse *openai.ChatCompletionStreamResponse

		for {
			response, err := stream.Recv()
			if err == io.EOF {
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
				final := types.LLMResponse{
					ID:        responseID,
					Content:   fullContent,
					Role:      "assistant",
					ToolCalls: toolCalls,
				}
				responseChan <- final
				return
			}

			if err != nil {
				c.logger.Errorf("OpenAI stream error: %v", err)
				return
			}

			finalResponse = &response

			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				delta := choice.Delta

				if delta.Content != "" {
					fullContent += delta.Content
				}

				// 处理工具调用流式数据
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						index := *tc.Index // OpenAI工具调用的index

						// 如果是新的工具调用，初始化
						if _, exists := toolCallsMap[index]; !exists {
							toolCallsMap[index] = &types.ToolCall{
								ID:   tc.ID,
								Type: string(tc.Type),
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
							toolCallsMap[index].Type = string(tc.Type)
						}
						if tc.Function.Name != "" {
							toolCallsMap[index].Function.Name = tc.Function.Name
						}
					}
				}

				// 发送增量响应
				streamResp := types.LLMResponse{
					ID:      response.ID,
					Content: delta.Content,
					Role:    "assistant",
				}
				responseChan <- streamResp
			}
		}
	}()

	return responseChan, nil
}

// convertMessages 转换消息格式
func (c *OpenAIClient) convertMessages(messages []types.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))

	for i, msg := range messages {
		result[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// 处理工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			result[i].ToolCalls = toolCalls
		}
	}

	return result
}

// convertTools 转换工具格式
func (c *OpenAIClient) convertTools(tools []types.Tool) []openai.Tool {
	result := make([]openai.Tool, len(tools))

	for i, tool := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolType(tool.Type),
			Function: openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	return result
}

// convertResponse 转换响应格式
func (c *OpenAIClient) convertResponse(resp openai.ChatCompletionResponse) *types.LLMResponse {
	if len(resp.Choices) == 0 {
		return &types.LLMResponse{
			ID: resp.ID,
			Usage: types.Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]

	response := &types.LLMResponse{
		ID:      resp.ID,
		Content: choice.Message.Content,
		Role:    choice.Message.Role,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// 处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]types.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			toolCalls[i] = types.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: types.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		response.ToolCalls = toolCalls
	}

	return response
}

// getModel 获取模型名称
func (c *OpenAIClient) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return c.config.Model
}

// getMaxTokens 获取最大token数
func (c *OpenAIClient) getMaxTokens(requestMaxTokens int) int {
	if requestMaxTokens > 0 {
		return requestMaxTokens
	}
	return c.config.MaxTokens
}

// getTemperature 获取温度参数
func (c *OpenAIClient) getTemperature(requestTemperature float64) float32 {
	if requestTemperature > 0 {
		return float32(requestTemperature)
	}
	return float32(c.config.Temperature)
}
