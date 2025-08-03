package context

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

// JSONStorage JSON文件存储实现
type JSONStorage struct {
	storagePath string
	logger      log.Logger
}

// NewJSONStorage 创建JSON存储
func NewJSONStorage(storagePath string, logger log.Logger) (*JSONStorage, error) {
	// 确保存储目录存在
	if err := utils.EnsureDir(storagePath); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &JSONStorage{
		storagePath: storagePath,
		logger:      logger,
	}, nil
}

// SaveSession 保存会话
func (js *JSONStorage) SaveSession(ctx context.Context, session *types.SessionContext) error {
	sessionPath := filepath.Join(js.storagePath, fmt.Sprintf("session_%s.json", session.ID))

	data, err := utils.JSONMarshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return utils.WriteFileContent(sessionPath, string(data))
}

// LoadSession 加载单个会话
func (js *JSONStorage) LoadSession(ctx context.Context, sessionID string) (*types.SessionContext, error) {
	sessionPath := filepath.Join(js.storagePath, fmt.Sprintf("session_%s.json", sessionID))

	if !utils.FileExists(sessionPath) {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	content, err := utils.ReadFileContent(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session types.SessionContext
	if err := json.Unmarshal([]byte(content), &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// LoadAllSessions 加载所有会话
func (js *JSONStorage) LoadAllSessions(ctx context.Context) (map[string]*types.SessionContext, error) {
	sessions := make(map[string]*types.SessionContext)

	pattern := filepath.Join(js.storagePath, "session_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	for _, path := range matches {
		content, err := utils.ReadFileContent(path)
		if err != nil {
			js.logger.Warnf("Failed to read session file %s: %v", path, err)
			continue
		}

		var session types.SessionContext
		if err := json.Unmarshal([]byte(content), &session); err != nil {
			js.logger.Warnf("Failed to parse session file %s: %v", path, err)
			continue
		}

		sessions[session.ID] = &session
		js.logger.Debugf("Loaded session: %s", session.ID)
	}

	return sessions, nil
}

// DeleteSession 删除会话
func (js *JSONStorage) DeleteSession(ctx context.Context, sessionID string) error {
	sessionPath := filepath.Join(js.storagePath, fmt.Sprintf("session_%s.json", sessionID))

	if !utils.FileExists(sessionPath) {
		return nil // 文件不存在，认为已删除
	}

	return os.Remove(sessionPath)
}

// Close 关闭存储连接（JSON存储无需关闭）
func (js *JSONStorage) Close() error {
	return nil
}
