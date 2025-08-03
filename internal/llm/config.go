package llm

import (
	"fmt"

	"github.com/zboya/nala-coder/pkg/types"
)

// Config LLM模块配置
type Config struct {
	DefaultProvider types.LLMProvider `mapstructure:"default_provider"`
	OpenAI          types.LLMConfig   `mapstructure:"openai"`
	DeepSeek        types.LLMConfig   `mapstructure:"deepseek"`
	Claude          types.LLMConfig   `mapstructure:"claude"`
	Ollama          types.LLMConfig   `mapstructure:"ollama"`
}

// GetProviderConfigs 获取所有提供商配置
func (c *Config) GetProviderConfigs() map[types.LLMProvider]types.LLMConfig {
	configs := make(map[types.LLMProvider]types.LLMConfig)

	// 只添加有效配置的提供商
	if c.OpenAI.APIKey != "" || c.OpenAI.BaseURL != "" {
		openaiConfig := c.OpenAI
		openaiConfig.Provider = types.ProviderOpenAI
		configs[types.ProviderOpenAI] = openaiConfig
	}

	if c.DeepSeek.APIKey != "" || c.DeepSeek.BaseURL != "" {
		deepseekConfig := c.DeepSeek
		deepseekConfig.Provider = types.ProviderDeepSeek
		configs[types.ProviderDeepSeek] = deepseekConfig
	}

	if c.Claude.APIKey != "" || c.Claude.BaseURL != "" {
		claudeConfig := c.Claude
		claudeConfig.Provider = types.ProviderClaude
		configs[types.ProviderClaude] = claudeConfig
	}

	if c.Ollama.BaseURL != "" {
		ollamaConfig := c.Ollama
		ollamaConfig.Provider = types.ProviderOllama
		configs[types.ProviderOllama] = ollamaConfig
	}

	return configs
}

// ValidateConfig 验证配置
func (c *Config) ValidateConfig() error {
	configs := c.GetProviderConfigs()

	if len(configs) == 0 {
		return fmt.Errorf("no valid LLM providers configured")
	}

	if _, exists := configs[c.DefaultProvider]; !exists {
		return fmt.Errorf("default provider %s is not configured", c.DefaultProvider)
	}

	return nil
}
