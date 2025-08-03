package context

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

// SQLiteStorage SQLite数据库存储实现
type SQLiteStorage struct {
	db          *sql.DB
	storagePath string
	logger      log.Logger
}

// NewSQLiteStorage 创建SQLite存储
func NewSQLiteStorage(storagePath string, logger log.Logger) (*SQLiteStorage, error) {
	// 确保存储目录存在
	if err := utils.EnsureDir(storagePath); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 数据库文件路径
	dbPath := filepath.Join(storagePath, "sessions.db")

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{
		db:          db,
		storagePath: storagePath,
		logger:      logger,
	}

	// 初始化数据库表
	if err := storage.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return storage, nil
}

// initTables 初始化数据库表
func (ss *SQLiteStorage) initTables() error {
	createSessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		messages TEXT NOT NULL,
		compressed_history TEXT,
		metadata TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		last_activity DATETIME NOT NULL,
		total_tokens INTEGER NOT NULL DEFAULT 0
	)`

	if _, err := ss.db.Exec(createSessionsTable); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	return nil
}

// SaveSession 保存会话
func (ss *SQLiteStorage) SaveSession(ctx context.Context, session *types.SessionContext) error {
	// 序列化消息
	messagesJSON, err := json.Marshal(session.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	// 序列化元数据
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO sessions (
		id, messages, compressed_history, metadata, 
		created_at, last_activity, total_tokens
	) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = ss.db.ExecContext(ctx, query,
		session.ID,
		string(messagesJSON),
		session.CompressedHistory,
		string(metadataJSON),
		session.CreatedAt,
		session.LastActivity,
		session.TotalTokens,
	)

	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	ss.logger.Debugf("Saved session to SQLite: %s", session.ID)
	return nil
}

// LoadSession 加载单个会话
func (ss *SQLiteStorage) LoadSession(ctx context.Context, sessionID string) (*types.SessionContext, error) {
	query := `
	SELECT id, messages, compressed_history, metadata, 
		   created_at, last_activity, total_tokens
	FROM sessions WHERE id = ?`

	row := ss.db.QueryRowContext(ctx, query, sessionID)

	var session types.SessionContext
	var messagesJSON, metadataJSON string

	err := row.Scan(
		&session.ID,
		&messagesJSON,
		&session.CompressedHistory,
		&metadataJSON,
		&session.CreatedAt,
		&session.LastActivity,
		&session.TotalTokens,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session %s not found", sessionID)
		}
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	// 反序列化消息
	if err := json.Unmarshal([]byte(messagesJSON), &session.Messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	// 反序列化元数据
	if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	ss.logger.Debugf("Loaded session from SQLite: %s", session.ID)
	return &session, nil
}

// LoadAllSessions 加载所有会话
func (ss *SQLiteStorage) LoadAllSessions(ctx context.Context) (map[string]*types.SessionContext, error) {
	sessions := make(map[string]*types.SessionContext)

	query := `
	SELECT id, messages, compressed_history, metadata, 
		   created_at, last_activity, total_tokens
	FROM sessions ORDER BY last_activity DESC`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var session types.SessionContext
		var messagesJSON, metadataJSON string

		err := rows.Scan(
			&session.ID,
			&messagesJSON,
			&session.CompressedHistory,
			&metadataJSON,
			&session.CreatedAt,
			&session.LastActivity,
			&session.TotalTokens,
		)

		if err != nil {
			ss.logger.Warnf("Failed to scan session row: %v", err)
			continue
		}

		// 反序列化消息
		if err := json.Unmarshal([]byte(messagesJSON), &session.Messages); err != nil {
			ss.logger.Warnf("Failed to unmarshal messages for session %s: %v", session.ID, err)
			continue
		}

		// 反序列化元数据
		if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
			ss.logger.Warnf("Failed to unmarshal metadata for session %s: %v", session.ID, err)
			continue
		}

		sessions[session.ID] = &session
		ss.logger.Debugf("Loaded session: %s", session.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	ss.logger.Infof("Loaded %d sessions from SQLite", len(sessions))
	return sessions, nil
}

// DeleteSession 删除会话
func (ss *SQLiteStorage) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`

	result, err := ss.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		ss.logger.Debugf("Session %s not found for deletion", sessionID)
	} else {
		ss.logger.Debugf("Deleted session from SQLite: %s", sessionID)
	}

	return nil
}

// Close 关闭数据库连接
func (ss *SQLiteStorage) Close() error {
	if ss.db != nil {
		return ss.db.Close()
	}
	return nil
}
