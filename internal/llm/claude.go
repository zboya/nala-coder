package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// ClaudeClient Claude客户端
type ClaudeClient struct {
	config     types.LLMConfig
	httpClient *http.Client
	logger     log.Logger
}

// ClaudeMessage Claude消息格式
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest Claude请求格式
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
}

// ClaudeResponse Claude响应格式
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewClaudeClient 创建Claude客户端
func NewClaudeClient(config types.LLMConfig, logger log.Logger) *ClaudeClient {
	return &ClaudeClient{
		config:     config,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

// GetConfig 获取配置
func (c *ClaudeClient) GetConfig() types.LLMConfig {
	return c.config
}

// GetProvider 获取提供商
func (c *ClaudeClient) GetProvider() types.LLMProvider {
	return types.ProviderClaude
}

// Chat 对话
func (c *ClaudeClient) Chat(ctx context.Context, request types.LLMRequest) (*types.LLMResponse, error) {
	claudeReq := c.convertRequest(request)

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Claude API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(claudeResp), nil
}

// ChatStream 流式对话
func (c *ClaudeClient) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	claudeReq := c.convertRequest(request)
	claudeReq.Stream = true

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Claude stream API error: %w", err)
	}

	responseChan := make(chan types.LLMResponse, 10)

	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var fullContent strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}

				var event map[string]interface{}
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}

				if event["type"] == "content_block_delta" {
					if delta, ok := event["delta"].(map[string]interface{}); ok {
						if text, ok := delta["text"].(string); ok {
							fullContent.WriteString(text)

							streamResp := types.LLMResponse{
								Content: text,
								Role:    "assistant",
							}
							responseChan <- streamResp
						}
					}
				}
			}
		}

		// 发送最终响应
		final := types.LLMResponse{
			Content: fullContent.String(),
			Role:    "assistant",
		}
		responseChan <- final
	}()

	return responseChan, nil
}

// convertRequest 转换请求格式
func (c *ClaudeClient) convertRequest(request types.LLMRequest) ClaudeRequest {
	messages := make([]ClaudeMessage, 0)
	var systemMessage string

	for _, msg := range request.Messages {
		if msg.Role == types.RoleSystem {
			systemMessage = msg.Content
		} else {
			messages = append(messages, ClaudeMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	return ClaudeRequest{
		Model:     c.getModel(request.Model),
		MaxTokens: c.getMaxTokens(request.MaxTokens),
		Messages:  messages,
		System:    systemMessage,
		Stream:    request.Stream,
	}
}

// convertResponse 转换响应格式
func (c *ClaudeClient) convertResponse(resp ClaudeResponse) *types.LLMResponse {
	var content string
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &types.LLMResponse{
		ID:      resp.ID,
		Content: content,
		Role:    resp.Role,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// getModel 获取模型名称
func (c *ClaudeClient) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return c.config.Model
}

// getMaxTokens 获取最大token数
func (c *ClaudeClient) getMaxTokens(requestMaxTokens int) int {
	if requestMaxTokens > 0 {
		return requestMaxTokens
	}
	return c.config.MaxTokens
}
