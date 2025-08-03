package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

func init() {
	registerBuiltinTool("todo_read", &TodoReadTool{})
	registerBuiltinTool("todo_write", &TodoWriteTool{})
}

// Todo 任务项
type Todo struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`   // pending, in_progress, completed, cancelled
	Priority string `json:"priority"` // high, medium, low
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

// TodoManager 任务管理器
type TodoManager struct {
	todos    []Todo
	mu       sync.RWMutex
	filePath string
}

var globalTodoManager *TodoManager
var todoManagerOnce sync.Once

// getTodoManager 获取全局任务管理器
func getTodoManager() *TodoManager {
	todoManagerOnce.Do(func() {
		cwd, _ := os.Getwd()
		todoPath := filepath.Join(cwd, "storage", "todos.json")
		globalTodoManager = &TodoManager{
			todos:    make([]Todo, 0),
			filePath: todoPath,
		}
		globalTodoManager.load()
	})
	return globalTodoManager
}

// load 加载任务
func (tm *TodoManager) load() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !utils.FileExists(tm.filePath) {
		return nil
	}

	content, err := utils.ReadFileContent(tm.filePath)
	if err != nil {
		return err
	}

	var todos []Todo
	if err := json.Unmarshal([]byte(content), &todos); err != nil {
		return err
	}

	tm.todos = todos
	return nil
}

// save 保存任务
func (tm *TodoManager) save() error {
	data, err := utils.JSONMarshal(tm.todos)
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := utils.EnsureDir(filepath.Dir(tm.filePath)); err != nil {
		return err
	}

	return utils.WriteFileContent(tm.filePath, string(data))
}

// getTodos 获取所有任务
func (tm *TodoManager) getTodos() []Todo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 返回副本
	todos := make([]Todo, len(tm.todos))
	copy(todos, tm.todos)
	return todos
}

// updateTodos 更新任务列表
func (tm *TodoManager) updateTodos(newTodos []Todo) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 设置时间戳
	now := time.Now().Format("2006-01-02 15:04:05")
	for i := range newTodos {
		if newTodos[i].Created == "" {
			newTodos[i].Created = now
		}
		newTodos[i].Updated = now
	}

	tm.todos = newTodos
	return tm.save()
}

// TodoReadTool 任务读取工具
type TodoReadTool struct{}

func NewTodoReadTool() *TodoReadTool {
	return &TodoReadTool{}
}

func (t *TodoReadTool) Name() string {
	return "todo_read"
}

func (t *TodoReadTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	// 这个工具不需要参数
	manager := getTodoManager()
	todos := manager.getTodos()

	if len(todos) == 0 {
		return &types.ToolCallResult{
			Success: true,
			Content: "No todos found. Todo list is empty.",
		}
	}

	// 按状态分组
	groups := map[string][]Todo{
		"pending":     {},
		"in_progress": {},
		"completed":   {},
		"cancelled":   {},
	}

	for _, todo := range todos {
		groups[todo.Status] = append(groups[todo.Status], todo)
	}

	var result string
	result += fmt.Sprintf("Current Todo List (%d items):\n\n", len(todos))

	// 显示进行中的任务
	if len(groups["in_progress"]) > 0 {
		result += "🔄 IN PROGRESS:\n"
		for _, todo := range groups["in_progress"] {
			result += fmt.Sprintf("  - [%s] %s (Priority: %s)\n", todo.ID, todo.Content, todo.Priority)
		}
		result += "\n"
	}

	// 显示待处理的任务
	if len(groups["pending"]) > 0 {
		result += "⏳ PENDING:\n"
		for _, todo := range groups["pending"] {
			result += fmt.Sprintf("  - [%s] %s (Priority: %s)\n", todo.ID, todo.Content, todo.Priority)
		}
		result += "\n"
	}

	// 显示已完成的任务
	if len(groups["completed"]) > 0 {
		result += "✅ COMPLETED:\n"
		for _, todo := range groups["completed"] {
			result += fmt.Sprintf("  - [%s] %s\n", todo.ID, todo.Content)
		}
		result += "\n"
	}

	// 显示已取消的任务
	if len(groups["cancelled"]) > 0 {
		result += "❌ CANCELLED:\n"
		for _, todo := range groups["cancelled"] {
			result += fmt.Sprintf("  - [%s] %s\n", todo.ID, todo.Content)
		}
		result += "\n"
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result,
	}
}

func (t *TodoReadTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "todo_read",
			Description: "Read the current todo list with status and priorities",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}
}

func (t *TodoReadTool) IsConcurrencySafe() bool {
	return true
}

// TodoWriteTool 任务写入工具
type TodoWriteTool struct{}

func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{}
}

func (t *TodoWriteTool) Name() string {
	return "todo_write"
}

func (t *TodoWriteTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Todos []Todo `json:"todos"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	if len(params.Todos) < 2 {
		return &types.ToolCallResult{
			Success: false,
			Error:   "at least 2 todo items are required",
		}
	}

	// 验证每个任务
	for i, todo := range params.Todos {
		if todo.Content == "" {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: content is required", i+1),
			}
		}

		if todo.ID == "" {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: id is required", i+1),
			}
		}

		if todo.Status == "" {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: status is required", i+1),
			}
		}

		if todo.Priority == "" {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: priority is required", i+1),
			}
		}

		// 验证状态值
		validStatuses := map[string]bool{
			"pending":     true,
			"in_progress": true,
			"completed":   true,
			"cancelled":   true,
		}
		if !validStatuses[todo.Status] {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: invalid status '%s'", i+1, todo.Status),
			}
		}

		// 验证优先级值
		validPriorities := map[string]bool{
			"high":   true,
			"medium": true,
			"low":    true,
		}
		if !validPriorities[todo.Priority] {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("todo %d: invalid priority '%s'", i+1, todo.Priority),
			}
		}
	}

	// 更新任务列表
	manager := getTodoManager()
	if err := manager.updateTodos(params.Todos); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to save todos: %v", err),
		}
	}

	// 统计状态
	statusCounts := make(map[string]int)
	for _, todo := range params.Todos {
		statusCounts[todo.Status]++
	}

	result := fmt.Sprintf("Successfully updated todo list with %d items:\n", len(params.Todos))
	for status, count := range statusCounts {
		result += fmt.Sprintf("  - %s: %d\n", status, count)
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result,
	}
}

func (t *TodoWriteTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "todo_write",
			Description: "Create and manage structured task list for current coding session",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"todos": map[string]any{
						"type":     "array",
						"minItems": 2,
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"content": map[string]any{
									"type":        "string",
									"minLength":   1,
									"description": "The description/content of the todo item",
								},
								"id": map[string]any{
									"type":        "string",
									"description": "Unique identifier for the todo item",
								},
								"priority": map[string]any{
									"type":        "string",
									"enum":        []string{"high", "medium", "low"},
									"description": "Priority level of the todo item",
								},
								"status": map[string]any{
									"type":        "string",
									"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
									"description": "Current status of the todo item",
								},
							},
							"required": []string{"content", "status", "priority", "id"},
						},
						"description": "Array of todo items to write to the workspace",
					},
				},
				"required": []string{"todos"},
			},
		},
	}
}

func (t *TodoWriteTool) IsConcurrencySafe() bool {
	return false
}
