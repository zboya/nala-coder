package llm

import (
	"fmt"
	"strings"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// CreateClient 根据配置创建LLM客户端
func CreateClient(provider types.LLMProvider, config types.LLMConfig, logger log.Logger) (types.LLMClient, error) {
	provider = types.LLMProvider(strings.ToLower(string(provider)))
	switch provider {
	case types.ProviderOpenAI:
		return NewOpenAIClient(config, logger), nil
	case types.ProviderDeepSeek:
		return NewDeepSeekClient(config, logger), nil
	case types.ProviderClaude:
		return NewClaudeClient(config, logger), nil
	case types.ProviderOllama:
		return NewOllamaClient(config, logger), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

// CreateManagerFromConfigs 从配置创建LLM管理器
func CreateManagerFromConfigs(configs map[types.LLMProvider]types.LLMConfig, defaultProvider types.LLMProvider, logger log.Logger) (*Manager, error) {
	manager := NewManager(defaultProvider, logger)

	for provider, config := range configs {
		client, err := CreateClient(provider, config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for provider %s: %w", provider, err)
		}
		manager.RegisterClient(provider, client)
	}

	// 确保默认提供商存在
	if _, err := manager.GetClient(defaultProvider); err != nil {
		return nil, fmt.Errorf("default provider %s not configured: %w", defaultProvider, err)
	}

	return manager, nil
}
