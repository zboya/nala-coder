package context

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

// ContextManager 上下文管理器
type ContextManager struct {
	config         *Config
	sessions       map[string]*types.SessionContext
	promptManager  types.PromptManager
	compressionLLM types.LLMClient
	storage        SessionStorage
	mu             sync.RWMutex
	logger         log.Logger
}

// Config 上下文管理器配置
type Config struct {
	HistoryLimit         int         `mapstructure:"history_limit"`
	StoragePath          string      `mapstructure:"storage_path"`
	StorageType          StorageType `mapstructure:"storage_type"`
	PersistenceFile      string      `mapstructure:"persistence_file"`
	CompressionThreshold float64     `mapstructure:"compression_threshold"`
}

// NewContextManager 创建上下文管理器
func NewContextManager(config *Config, promptManager types.PromptManager, compressionLLM types.LLMClient, logger log.Logger) (*ContextManager, error) {
	// 设置默认存储类型
	if config.StorageType == "" {
		config.StorageType = GetDefaultStorageType()
	}

	// 创建存储实例
	storage, err := NewSessionStorage(config.StorageType, config.StoragePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	cm := &ContextManager{
		config:         config,
		sessions:       make(map[string]*types.SessionContext),
		promptManager:  promptManager,
		compressionLLM: compressionLLM,
		storage:        storage,
		logger:         logger,
	}

	// 加载持久化数据
	if err := cm.loadSessions(); err != nil {
		cm.logger.Warnf("Failed to load sessions: %v", err)
	}

	return cm, nil
}

// AddMessage 添加消息到会话
func (cm *ContextManager) AddMessage(ctx context.Context, sessionID string, message types.Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session := cm.getOrCreateSession(sessionID)

	// 添加消息
	session.Messages = append(session.Messages, message)
	session.LastActivity = time.Now()

	// 计算token使用量
	tokens := utils.CountTokens(message.Content)
	session.TotalTokens += tokens

	// 检查是否需要压缩
	if cm.needsCompression(session) {
		if err := cm.compressSessionHistory(ctx, session); err != nil {
			cm.logger.Errorf("Failed to compress session history: %v", err)
		}
	}

	// 限制消息数量
	cm.limitSessionMessages(session)

	// 保存会话
	return cm.saveSession(ctx, session)
}

// GetMessages 获取会话消息
func (cm *ContextManager) GetMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return []types.Message{}, nil
	}

	return session.Messages, nil
}

// CompressHistory 压缩会话历史
func (cm *ContextManager) CompressHistory(ctx context.Context, sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return cm.compressSessionHistory(ctx, session)
}

// LoadPersistentContext 加载持久化上下文
func (cm *ContextManager) LoadPersistentContext(ctx context.Context, sessionID string) (string, error) {
	persistentPath := filepath.Join(cm.config.StoragePath, cm.config.PersistenceFile)

	if !utils.FileExists(persistentPath) {
		return "", nil
	}

	content, err := utils.ReadFileContent(persistentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read persistent context: %w", err)
	}

	return content, nil
}

// SavePersistentContext 保存持久化上下文
func (cm *ContextManager) SavePersistentContext(ctx context.Context, sessionID string, content string) error {
	persistentPath := filepath.Join(cm.config.StoragePath, cm.config.PersistenceFile)

	if err := utils.WriteFileContent(persistentPath, content); err != nil {
		return fmt.Errorf("failed to save persistent context: %w", err)
	}

	return nil
}

// GetSessionContext 获取完整的会话上下文
func (cm *ContextManager) GetSessionContext(sessionID string) (*types.SessionContext, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// 复制会话以避免并发修改
	sessionCopy := *session
	sessionCopy.Messages = make([]types.Message, len(session.Messages))
	copy(sessionCopy.Messages, session.Messages)

	return &sessionCopy, nil
}

// getOrCreateSession 获取或创建会话
func (cm *ContextManager) getOrCreateSession(sessionID string) *types.SessionContext {
	session, exists := cm.sessions[sessionID]
	if !exists {
		session = &types.SessionContext{
			ID:           sessionID,
			Messages:     make([]types.Message, 0),
			Metadata:     make(map[string]string),
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		cm.sessions[sessionID] = session
	}
	return session
}

func (cm *ContextManager) getContextWindow() int {
	return cm.compressionLLM.GetConfig().MaxTokens
}

// needsCompression 判断是否需要压缩
func (cm *ContextManager) needsCompression(session *types.SessionContext) bool {
	threshold := int(float64(cm.getContextWindow()) * cm.config.CompressionThreshold)
	if threshold <= 0 {
		return false
	}
	cm.logger.Debugf("Session %s total tokens: %d, threshold: %d", session.ID, session.TotalTokens, threshold)
	return session.TotalTokens > threshold
}

// compressSessionHistory 压缩会话历史
func (cm *ContextManager) compressSessionHistory(ctx context.Context, session *types.SessionContext) error {
	if len(session.Messages) <= 2 {
		return nil // 消息太少，无需压缩
	}

	// 构建历史消息文本
	var historyText string
	for _, msg := range session.Messages[:len(session.Messages)-1] { // 保留最后一条消息
		historyText += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// 准备压缩数据
	data := map[string]any{
		"conversation_history": historyText,
		"token_limit":          cm.getContextWindow() / 4, // 压缩到1/4大小
	}

	// 获取压缩提示词
	compressionPrompt, err := cm.promptManager.GetPromptWithData("compression", data)
	if err != nil {
		return fmt.Errorf("failed to get compression prompt: %w", err)
	}

	// 创建压缩请求
	compressionRequest := types.LLMRequest{
		Messages: []types.Message{
			{
				ID:      utils.GenerateID(),
				Role:    types.RoleUser,
				Content: compressionPrompt,
			},
		},
	}

	// 执行压缩
	response, err := cm.compressionLLM.Chat(ctx, compressionRequest)
	if err != nil {
		return fmt.Errorf("failed to compress history: %w", err)
	}

	// 保存压缩后的历史
	if session.CompressedHistory != "" {
		session.CompressedHistory += "\n\n" + response.Content
	} else {
		session.CompressedHistory = response.Content
	}

	// 保留最近的几条消息
	keepCount := cm.config.HistoryLimit
	if len(session.Messages) > keepCount {
		session.Messages = session.Messages[len(session.Messages)-keepCount:]
	}

	// 重新计算token数量
	session.TotalTokens = utils.CountTokens(session.CompressedHistory)
	for _, msg := range session.Messages {
		session.TotalTokens += utils.CountTokens(msg.Content)
	}

	cm.logger.Infof("Compressed session %s history, new token count: %d", session.ID, session.TotalTokens)
	return nil
}

// limitSessionMessages 限制会话消息数量
func (cm *ContextManager) limitSessionMessages(session *types.SessionContext) {
	if len(session.Messages) > cm.config.HistoryLimit*2 {
		// 保留最近的消息
		session.Messages = session.Messages[len(session.Messages)-cm.config.HistoryLimit:]

		// 重新计算token数量
		session.TotalTokens = utils.CountTokens(session.CompressedHistory)
		for _, msg := range session.Messages {
			session.TotalTokens += utils.CountTokens(msg.Content)
		}
	}
}

// saveSession 保存会话
func (cm *ContextManager) saveSession(ctx context.Context, session *types.SessionContext) error {
	return cm.storage.SaveSession(ctx, session)
}

// loadSessions 加载所有会话
func (cm *ContextManager) loadSessions() error {
	ctx := context.Background()
	sessions, err := cm.storage.LoadAllSessions(ctx)
	if err != nil {
		return err
	}

	cm.sessions = sessions
	cm.logger.Infof("Loaded %d sessions from storage", len(sessions))
	return nil
}

// Close 关闭上下文管理器，清理资源
func (cm *ContextManager) Close() error {
	if cm.storage != nil {
		return cm.storage.Close()
	}
	return nil
}
