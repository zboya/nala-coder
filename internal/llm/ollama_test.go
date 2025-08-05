package llm

import (
	"context"
	"testing"

	"github.com/zboya/nala-coder/pkg/types"
)

func TestOllamaClientWithTools(t *testing.T) {
	config := types.LLMConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama3.1",
	}

	client := NewOllamaClient(config, nil)
	if client == nil {
		t.Fatal("Failed to create Ollama client")
	}

	// 测试工具定义
	tools := []types.Tool{
		{
			Type: "function",
			Function: types.ToolFunction{
				Name:        "get_weather",
				Description: "获取指定城市的天气信息",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"city": map[string]interface{}{
							"type":        "string",
							"description": "城市名称",
						},
					},
					"required": []interface{}{"city"},
				},
			},
		},
	}

	// 测试请求
	request := types.LLMRequest{
		Messages: []types.Message{
			{
				Role:    types.RoleUser,
				Content: "今天北京的天气怎么样？",
			},
		},
		Tools: tools,
	}

	// 测试非流式调用
	response, err := client.Chat(context.Background(), request)
	if err != nil {
		t.Logf("Chat error (expected if Ollama not running): %v", err)
	} else {
		t.Logf("Response: %+v", response)
	}

	// 测试流式调用
	stream, err := client.ChatStream(context.Background(), request)
	if err != nil {
		t.Logf("ChatStream error (expected if Ollama not running): %v", err)
	} else {
		for resp := range stream {
			t.Logf("Stream response: %+v", resp)
		}
	}
}
