package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/fsnotify/fsnotify"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/utils"
)

// PromptManager 提示词管理器
type PromptManager struct {
	directory string
	hotReload bool
	prompts   map[string]*template.Template
	mu        sync.RWMutex
	watcher   *fsnotify.Watcher
	stopWatch chan bool
	logger    log.Logger
}

// NewPromptManager 创建提示词管理器
func NewPromptManager(directory string, hotReload bool, logger log.Logger) (*PromptManager, error) {
	pm := &PromptManager{
		directory: directory,
		hotReload: hotReload,
		prompts:   make(map[string]*template.Template),
		stopWatch: make(chan bool),
		logger:    logger,
	}

	// 确保目录存在
	if err := utils.EnsureDir(directory); err != nil {
		return nil, fmt.Errorf("failed to create prompts directory: %w", err)
	}

	// 初始加载提示词
	if err := pm.loadPrompts(); err != nil {
		return nil, fmt.Errorf("failed to load prompts: %w", err)
	}

	// 如果启用热更新，设置文件监听
	if hotReload {
		if err := pm.setupWatcher(); err != nil {
			pm.logger.Warnf("Failed to setup prompt watcher: %v", err)
		}
	}

	return pm, nil
}

// GetPrompt 获取提示词
func (pm *PromptManager) GetPrompt(name string) (string, error) {
	pm.mu.RLock()
	tmpl, exists := pm.prompts[name]
	pm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("prompt '%s' not found", name)
	}

	// 执行模板，使用空的数据
	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GetPromptWithData 使用数据渲染提示词
func (pm *PromptManager) GetPromptWithData(name string, data map[string]any) (string, error) {
	pm.mu.RLock()
	tmpl, exists := pm.prompts[name]
	pm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("prompt '%s' not found", name)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template with data: %w", err)
	}

	return buf.String(), nil
}

// ReloadPrompts 重新加载所有提示词
func (pm *PromptManager) ReloadPrompts() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 清空现有提示词
	pm.prompts = make(map[string]*template.Template)

	// 重新加载
	return pm.loadPromptsLocked()
}

// WatchPrompts 开始监听提示词文件变化
func (pm *PromptManager) WatchPrompts() error {
	if !pm.hotReload {
		return fmt.Errorf("hot reload is disabled")
	}
	return pm.setupWatcher()
}

// Stop 停止提示词管理器
func (pm *PromptManager) Stop() {
	if pm.watcher != nil {
		close(pm.stopWatch)
		pm.watcher.Close()
	}
}

// ListPrompts 列出所有可用的提示词名称
func (pm *PromptManager) ListPrompts() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.prompts))
	for name := range pm.prompts {
		names = append(names, name)
	}
	return names
}

// loadPrompts 加载提示词文件
func (pm *PromptManager) loadPrompts() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.loadPromptsLocked()
}

// loadPromptsLocked 在锁定状态下加载提示词
func (pm *PromptManager) loadPromptsLocked() error {
	return filepath.Walk(pm.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 只处理 .md 文件
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// 读取文件内容
		content, err := utils.ReadFileContent(path)
		if err != nil {
			pm.logger.Errorf("Failed to read prompt file %s: %v", path, err)
			return nil // 继续处理其他文件
		}

		// 创建模板
		name := strings.TrimSuffix(info.Name(), ".md")
		tmpl, err := template.New(name).Parse(content)
		if err != nil {
			pm.logger.Errorf("Failed to parse prompt template %s: %v", name, err)
			return nil // 继续处理其他文件
		}

		pm.prompts[name] = tmpl
		pm.logger.Debugf("Loaded prompt: %s", name)

		return nil
	})
}

// setupWatcher 设置文件监听器
func (pm *PromptManager) setupWatcher() error {
	if pm.watcher != nil {
		pm.watcher.Close()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	pm.watcher = watcher

	// 添加目录到监听列表
	if err := watcher.Add(pm.directory); err != nil {
		return fmt.Errorf("failed to watch prompts directory: %w", err)
	}

	// 启动监听协程
	go pm.watchLoop()

	return nil
}

// watchLoop 文件监听循环
func (pm *PromptManager) watchLoop() {
	for {
		select {
		case event, ok := <-pm.watcher.Events:
			if !ok {
				return
			}

			// 只关心 .txt 文件的创建、写入和删除事件
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			switch {
			case event.Op&fsnotify.Write == fsnotify.Write:
				pm.logger.Infof("Prompt file modified: %s", event.Name)
				pm.reloadSinglePrompt(event.Name)

			case event.Op&fsnotify.Create == fsnotify.Create:
				pm.logger.Infof("Prompt file created: %s", event.Name)
				pm.reloadSinglePrompt(event.Name)

			case event.Op&fsnotify.Remove == fsnotify.Remove:
				pm.logger.Infof("Prompt file removed: %s", event.Name)
				pm.removeSinglePrompt(event.Name)
			}

		case err, ok := <-pm.watcher.Errors:
			if !ok {
				return
			}
			pm.logger.Errorf("Prompt watcher error: %v", err)

		case <-pm.stopWatch:
			return
		}
	}
}

// reloadSinglePrompt 重新加载单个提示词文件
func (pm *PromptManager) reloadSinglePrompt(filePath string) {
	// 读取文件内容
	content, err := utils.ReadFileContent(filePath)
	if err != nil {
		pm.logger.Errorf("Failed to read prompt file %s: %v", filePath, err)
		return
	}

	// 获取提示词名称
	fileName := filepath.Base(filePath)
	name := strings.TrimSuffix(fileName, ".md")

	// 创建模板
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		pm.logger.Errorf("Failed to parse prompt template %s: %v", name, err)
		return
	}

	// 更新提示词
	pm.mu.Lock()
	pm.prompts[name] = tmpl
	pm.mu.Unlock()

	pm.logger.Infof("Reloaded prompt: %s", name)
}

// removeSinglePrompt 移除单个提示词
func (pm *PromptManager) removeSinglePrompt(filePath string) {
	fileName := filepath.Base(filePath)
	name := strings.TrimSuffix(fileName, ".md")

	pm.mu.Lock()
	delete(pm.prompts, name)
	pm.mu.Unlock()

	pm.logger.Infof("Removed prompt: %s", name)
}
