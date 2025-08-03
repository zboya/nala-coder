package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
)

// Engine 工具引擎
type Engine struct {
	enabledTools   []string
	tools          map[string]types.ToolExecutor
	maxConcurrency int
	semaphore      chan struct{}
	mu             sync.RWMutex
	logger         log.Logger
	timeouts       map[string]time.Duration
}

// Config 工具引擎配置
type Config struct {
	MaxConcurrency int            `mapstructure:"max_concurrency"`
	EnabledTools   []string       `mapstructure:"enabled_tools"`
	Timeouts       map[string]int `mapstructure:"timeouts"` // milliseconds
}

// NewEngine 创建工具引擎
func NewEngine(config *Config, logger log.Logger) *Engine {
	maxConcurrency := config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	engine := &Engine{
		tools:          make(map[string]types.ToolExecutor),
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
		logger:         logger,
		timeouts:       make(map[string]time.Duration),
	}

	// 设置超时配置
	for tool, timeout := range config.Timeouts {
		engine.timeouts[tool] = time.Duration(timeout) * time.Millisecond
	}

	// 注册内置工具
	engine.registerBuiltinTools(config.EnabledTools)

	return engine
}

// RegisterTool 注册工具
func (e *Engine) RegisterTool(name string, executor types.ToolExecutor) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	e.tools[name] = executor
	e.logger.Infof("Registered tool: %s", name)
	return nil
}

// ExecuteTools 执行多个工具调用
func (e *Engine) ExecuteTools(ctx context.Context, calls []types.ToolCall) []types.ToolCallResult {
	if len(calls) == 0 {
		return []types.ToolCallResult{}
	}

	results := make([]types.ToolCallResult, len(calls))

	// 分组：并发安全的工具和非并发安全的工具
	concurrentCalls := make([]int, 0)
	sequentialCalls := make([]int, 0)

	for i, call := range calls {
		e.mu.RLock()
		tool, exists := e.tools[call.Function.Name]
		e.mu.RUnlock()

		if !exists {
			results[i] = types.ToolCallResult{
				Content:   "",
				Success:   false,
				Error:     fmt.Sprintf("tool %s not found", call.Function.Name),
				Timestamp: time.Now(),
			}
			continue
		}

		if tool.IsConcurrencySafe() {
			concurrentCalls = append(concurrentCalls, i)
		} else {
			sequentialCalls = append(sequentialCalls, i)
		}
	}

	// 先并发执行安全的工具
	if len(concurrentCalls) > 0 {
		e.executeConcurrentTools(ctx, calls, concurrentCalls, results)
	}

	// 然后顺序执行非并发安全的工具
	if len(sequentialCalls) > 0 {
		e.executeSequentialTools(ctx, calls, sequentialCalls, results)
	}

	return results
}

// GetToolDefinitions 获取所有工具定义
func (e *Engine) GetToolDefinitions() []types.Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	definitions := make([]types.Tool, 0, len(e.tools))
	for _, name := range e.enabledTools {
		if tool, exists := e.tools[name]; exists {
			definitions = append(definitions, tool.GetDefinition())
		}
	}

	return definitions
}

// GetTool 获取指定工具
func (e *Engine) GetTool(name string) (types.ToolExecutor, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tool, exists := e.tools[name]
	return tool, exists
}

// executeConcurrentTools 并发执行工具
func (e *Engine) executeConcurrentTools(ctx context.Context, calls []types.ToolCall, indices []int, results []types.ToolCallResult) {
	var wg sync.WaitGroup

	for _, i := range indices {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 获取信号量
			select {
			case e.semaphore <- struct{}{}:
				defer func() { <-e.semaphore }()
			case <-ctx.Done():
				results[index] = types.ToolCallResult{
					Content:   "",
					Success:   false,
					Error:     "context cancelled",
					Timestamp: time.Now(),
				}
				return
			}

			results[index] = e.executeSingleTool(ctx, calls[index])
		}(i)
	}

	wg.Wait()
}

// executeSequentialTools 顺序执行工具
func (e *Engine) executeSequentialTools(ctx context.Context, calls []types.ToolCall, indices []int, results []types.ToolCallResult) {
	for _, i := range indices {
		select {
		case <-ctx.Done():
			results[i] = types.ToolCallResult{
				Content:   "",
				Success:   false,
				Error:     "context cancelled",
				Timestamp: time.Now(),
			}
			return
		default:
			results[i] = e.executeSingleTool(ctx, calls[i])
		}
	}
}

// executeSingleTool 执行单个工具
func (e *Engine) executeSingleTool(ctx context.Context, call types.ToolCall) types.ToolCallResult {
	e.mu.RLock()
	tool, exists := e.tools[call.Function.Name]
	e.mu.RUnlock()

	if !exists {
		return types.ToolCallResult{
			Content:   "",
			Success:   false,
			Error:     fmt.Sprintf("tool %s not found", call.Function.Name),
			Timestamp: time.Now(),
		}
	}

	// 设置超时
	if timeout, exists := e.timeouts[call.Function.Name]; exists && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行工具
	result := tool.Execute(ctx, call)

	// 记录执行时间
	duration := time.Since(startTime)
	e.logger.Debugf("Tool %s executed in %v,result: %+v", call.Function.Name, duration, result)

	return *result
}

// registerBuiltinTools 注册内置工具
func (e *Engine) registerBuiltinTools(enabledTools []string) {
	e.enabledTools = enabledTools
	for _, tool := range enabledTools {
		toolExecutor := getBuiltinTool(tool)
		if toolExecutor != nil {
			e.tools[tool] = toolExecutor
		}
	}
	e.logger.Infof("Registered %d builtin tools", len(e.tools))
}
