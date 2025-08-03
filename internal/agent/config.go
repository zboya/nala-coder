package agent

import (
	"fmt"

	"github.com/zboya/nala-coder/internal/context"
	"github.com/zboya/nala-coder/internal/llm"
	"github.com/zboya/nala-coder/internal/tools"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// AppConfig 应用程序配置
type AppConfig struct {
	Server  ServerConfig       `mapstructure:"server"`
	LLM     llm.Config         `mapstructure:"llm"`
	Agent   Config             `mapstructure:"agent"`
	Tools   tools.Config       `mapstructure:"tools"`
	Context context.Config     `mapstructure:"context"`
	Prompts PromptsConfig      `mapstructure:"prompts"`
	Logging LoggingConfig      `mapstructure:"logging"`
	Speech  types.SpeechConfig `mapstructure:"speech"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// PromptsConfig 提示词配置
type PromptsConfig struct {
	Directory string `mapstructure:"directory"`
	HotReload bool   `mapstructure:"hot_reload"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Builder Agent构建器
type Builder struct {
	config         *AppConfig
	logger         log.Logger
	llmManager     *llm.Manager
	toolEngine     *tools.Engine
	contextManager *context.ContextManager
	promptManager  *context.PromptManager
}

// NewBuilder 创建Agent构建器
func NewBuilder(config *AppConfig, logger log.Logger) *Builder {
	return &Builder{
		config: config,
		logger: logger,
	}
}

// BuildLLMManager 构建LLM管理器
func (b *Builder) BuildLLMManager() error {
	// 验证配置
	if err := b.config.LLM.ValidateConfig(); err != nil {
		return err
	}

	// 创建LLM管理器
	providerConfigs := b.config.LLM.GetProviderConfigs()
	manager, err := llm.CreateManagerFromConfigs(providerConfigs, b.config.LLM.DefaultProvider, b.logger)
	if err != nil {
		return err
	}

	b.llmManager = manager
	return nil
}

// BuildPromptManager 构建提示词管理器
func (b *Builder) BuildPromptManager() error {
	manager, err := context.NewPromptManager(
		b.config.Prompts.Directory,
		b.config.Prompts.HotReload,
		b.logger,
	)
	if err != nil {
		return err
	}

	b.promptManager = manager
	return nil
}

// BuildToolEngine 构建工具引擎
func (b *Builder) BuildToolEngine() error {
	engine := tools.NewEngine(&b.config.Tools, b.logger)
	b.toolEngine = engine
	return nil
}

// BuildContextManager 构建上下文管理器
func (b *Builder) BuildContextManager() error {
	// 需要LLM管理器进行压缩
	if b.llmManager == nil {
		return fmt.Errorf("LLM manager must be built before context manager")
	}

	compressionLLM, err := b.llmManager.GetDefaultClient()
	if err != nil {
		return fmt.Errorf("failed to get compression LLM client: %w", err)
	}

	manager, err := context.NewContextManager(
		&b.config.Context,
		b.promptManager,
		compressionLLM,
		b.logger,
	)
	if err != nil {
		return err
	}

	b.contextManager = manager
	return nil
}

// Build 构建Agent
func (b *Builder) Build() (*Agent, error) {
	// 按依赖顺序构建组件
	if err := b.BuildLLMManager(); err != nil {
		return nil, fmt.Errorf("failed to build LLM manager: %w", err)
	}

	if err := b.BuildPromptManager(); err != nil {
		return nil, fmt.Errorf("failed to build prompt manager: %w", err)
	}

	if err := b.BuildToolEngine(); err != nil {
		return nil, fmt.Errorf("failed to build tool engine: %w", err)
	}

	if err := b.BuildContextManager(); err != nil {
		return nil, fmt.Errorf("failed to build context manager: %w", err)
	}

	// 获取默认LLM客户端
	defaultLLM, err := b.llmManager.GetDefaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get default LLM client: %w", err)
	}

	// 创建Agent
	agent := NewAgent(
		&b.config.Agent,
		defaultLLM,
		b.toolEngine,
		b.contextManager,
		b.promptManager,
		b.logger,
	)

	return agent, nil
}

// GetComponents 获取构建的组件（用于依赖注入）
func (b *Builder) GetComponents() (
	*llm.Manager,
	*tools.Engine,
	*context.ContextManager,
	*context.PromptManager,
) {
	return b.llmManager, b.toolEngine, b.contextManager, b.promptManager
}
