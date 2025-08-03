package llm

import (
	"context"
	"fmt"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// Manager LLM管理器
type Manager struct {
	clients         map[types.LLMProvider]types.LLMClient
	defaultProvider types.LLMProvider
	logger          log.Logger
}

// NewManager 创建LLM管理器
func NewManager(defaultProvider types.LLMProvider, logger log.Logger) *Manager {
	return &Manager{
		clients:         make(map[types.LLMProvider]types.LLMClient),
		defaultProvider: defaultProvider,
		logger:          logger,
	}
}

// RegisterClient 注册LLM客户端
func (m *Manager) RegisterClient(provider types.LLMProvider, client types.LLMClient) {
	m.clients[provider] = client
}

// GetClient 获取LLM客户端
func (m *Manager) GetClient(provider types.LLMProvider) (types.LLMClient, error) {
	if provider == "" {
		provider = m.defaultProvider
	}

	client, exists := m.clients[provider]
	if !exists {
		return nil, fmt.Errorf("LLM provider %s not found", provider)
	}

	return client, nil
}

// GetDefaultClient 获取默认LLM客户端
func (m *Manager) GetDefaultClient() (types.LLMClient, error) {
	return m.GetClient(m.defaultProvider)
}

// Chat 使用默认客户端进行对话
func (m *Manager) Chat(ctx context.Context, request types.LLMRequest) (*types.LLMResponse, error) {
	client, err := m.GetDefaultClient()
	if err != nil {
		return nil, err
	}

	return client.Chat(ctx, request)
}

// ChatStream 使用默认客户端进行流式对话
func (m *Manager) ChatStream(ctx context.Context, request types.LLMRequest) (<-chan types.LLMResponse, error) {
	client, err := m.GetDefaultClient()
	if err != nil {
		return nil, err
	}

	return client.ChatStream(ctx, request)
}

// ChatWithProvider 使用指定提供商进行对话
func (m *Manager) ChatWithProvider(ctx context.Context, provider types.LLMProvider, request types.LLMRequest) (*types.LLMResponse, error) {
	client, err := m.GetClient(provider)
	if err != nil {
		return nil, err
	}

	return client.Chat(ctx, request)
}

// ListProviders 列出所有可用的提供商
func (m *Manager) ListProviders() []types.LLMProvider {
	providers := make([]types.LLMProvider, 0, len(m.clients))
	for provider := range m.clients {
		providers = append(providers, provider)
	}
	return providers
}
