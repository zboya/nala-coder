package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// OllamaClient Ollama客户端
type OllamaClient struct {
	config     types.LLMConfig
	httpClient *http.Client
	logger     log.Logger
}

// OllamaMessage Ollama消息格式
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaRequest Ollama请求格式
type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse Ollama响应格式
type OllamaResponse struct {
	Model     string        `json:"model"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
	CreatedAt string        `json:"created_at"`
}

// NewOllamaClient 创建Ollama客户端
func NewOllamaClient(config types.LLMConfig, logger log.Logger) *OllamaClient {
	return &OllamaClient{
		config:     config,
		httpClient: &http.Client{},
		logger:     logger,
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
	ollamaReq := c.convertRequest(request)
	ollamaReq.Stream = false

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(ollamaResp), nil
}

// ChatStream 流式对话
func (c *OllamaClient) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	ollamaReq := c.convertRequest(request)
	ollamaReq.Stream = true

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama stream API error: %w", err)
	}

	responseChan := make(chan types.LLMResponse, 10)

	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		var fullContent string

		for {
			var ollamaResp OllamaResponse
			if err := decoder.Decode(&ollamaResp); err != nil {
				if err == io.EOF {
					break
				}
				return
			}

			content := ollamaResp.Message.Content
			fullContent += content

			streamResp := types.LLMResponse{
				Content: content,
				Role:    "assistant",
			}
			responseChan <- streamResp

			if ollamaResp.Done {
				// 发送最终响应
				final := types.LLMResponse{
					Content: fullContent,
					Role:    "assistant",
				}
				responseChan <- final
				break
			}
		}
	}()

	return responseChan, nil
}

// convertRequest 转换请求格式
func (c *OllamaClient) convertRequest(request types.LLMRequest) OllamaRequest {
	messages := make([]OllamaMessage, len(request.Messages))

	for i, msg := range request.Messages {
		messages[i] = OllamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	options := make(map[string]interface{})
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

	return OllamaRequest{
		Model:    c.getModel(request.Model),
		Messages: messages,
		Options:  options,
	}
}

// convertResponse 转换响应格式
func (c *OllamaClient) convertResponse(resp OllamaResponse) *types.LLMResponse {
	return &types.LLMResponse{
		Content: resp.Message.Content,
		Role:    "assistant",
		Usage: types.Usage{
			// Ollama 通常不返回token使用情况
			TotalTokens: 0,
		},
	}
}

// getModel 获取模型名称
func (c *OllamaClient) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return c.config.Model
}
