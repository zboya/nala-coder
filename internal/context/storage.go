package context

import (
	"context"
	"fmt"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// StorageType 存储类型
type StorageType string

const (
	// StorageTypeJSON JSON文件存储
	StorageTypeJSON StorageType = "json"
	// StorageTypeSQLite SQLite数据库存储
	StorageTypeSQLite StorageType = "sqlite"
)

// SessionStorage 会话存储接口
type SessionStorage interface {
	// SaveSession 保存会话
	SaveSession(ctx context.Context, session *types.SessionContext) error

	// LoadSession 加载单个会话
	LoadSession(ctx context.Context, sessionID string) (*types.SessionContext, error)

	// LoadAllSessions 加载所有会话
	LoadAllSessions(ctx context.Context) (map[string]*types.SessionContext, error)

	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, sessionID string) error

	// Close 关闭存储连接
	Close() error
}

// NewSessionStorage 创建会话存储
func NewSessionStorage(storageType StorageType, storagePath string, logger log.Logger) (SessionStorage, error) {
	switch storageType {
	case StorageTypeJSON:
		return NewJSONStorage(storagePath, logger)
	case StorageTypeSQLite:
		return NewSQLiteStorage(storagePath, logger)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// GetDefaultStorageType 获取默认存储类型
func GetDefaultStorageType() StorageType {
	return StorageTypeSQLite
}
