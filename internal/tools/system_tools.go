package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/zboya/nala-coder/pkg/types"
)

func init() {
	registerBuiltinTool("bash", &BashTool{})
}

// BashTool 系统命令执行工具
type BashTool struct{}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Command     string `json:"command"`
		Description string `json:"description,omitempty"`
		Timeout     int    `json:"timeout,omitempty"` // milliseconds
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	if params.Command == "" {
		return &types.ToolCallResult{
			Success: false,
			Error:   "command is required",
		}
	}

	// 设置超时
	timeout := 120000 // 默认2分钟
	if params.Timeout > 0 {
		timeout = params.Timeout
	}
	if timeout > 600000 { // 最大10分钟
		timeout = 600000
	}

	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// 创建命令
	var cmd *exec.Cmd
	if strings.Contains(params.Command, "&&") || strings.Contains(params.Command, "||") || strings.Contains(params.Command, ";") {
		// 复杂命令使用shell执行
		cmd = exec.CommandContext(cmdCtx, "bash", "-c", params.Command)
	} else {
		// 简单命令直接执行
		parts := strings.Fields(params.Command)
		if len(parts) == 0 {
			return &types.ToolCallResult{
				Success: false,
				Error:   "empty command",
			}
		}
		cmd = exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	}

	// 设置工作目录
	cwd, err := os.Getwd()
	if err == nil {
		cmd.Dir = cwd
	}

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	startTime := time.Now()
	err = cmd.Run()
	duration := time.Since(startTime)

	// 构建结果
	var result strings.Builder

	if params.Description != "" {
		result.WriteString(fmt.Sprintf("Description: %s\n", params.Description))
	}

	result.WriteString(fmt.Sprintf("Command: %s\n", params.Command))
	result.WriteString(fmt.Sprintf("Duration: %v\n", duration))

	if err != nil {
		// 检查是否是超时错误
		if cmdCtx.Err() == context.DeadlineExceeded {
			result.WriteString("Status: TIMEOUT\n")
			result.WriteString(fmt.Sprintf("Error: Command timed out after %d ms\n", timeout))
		} else {
			result.WriteString("Status: FAILED\n")
			if exitError, ok := err.(*exec.ExitError); ok {
				result.WriteString(fmt.Sprintf("Exit Code: %d\n", exitError.ExitCode()))
			}
			result.WriteString(fmt.Sprintf("Error: %v\n", err))
		}
	} else {
		result.WriteString("Status: SUCCESS\n")
		result.WriteString("Exit Code: 0\n")
	}

	// 添加输出
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if stdoutStr != "" {
		// 限制输出长度
		if len(stdoutStr) > 30000 {
			stdoutStr = stdoutStr[:30000] + "\n... (output truncated)"
		}
		result.WriteString(fmt.Sprintf("\nStdout:\n%s\n", stdoutStr))
	}

	if stderrStr != "" {
		if len(stderrStr) > 30000 {
			stderrStr = stderrStr[:30000] + "\n... (output truncated)"
		}
		result.WriteString(fmt.Sprintf("\nStderr:\n%s\n", stderrStr))
	}

	return &types.ToolCallResult{
		Success: err == nil,
		Content: result.String(),
		Error:   "",
	}
}

func (t *BashTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "bash",
			Description: "Execute bash commands in a persistent shell session with timeout and safety measures. Dont include any newlines in the command.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The bash command to execute",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Optional description of what the command does (5-10 words)",
					},
					"timeout": map[string]any{
						"type":        "integer",
						"description": "Timeout in milliseconds (default: 120000, max: 600000)",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (t *BashTool) IsConcurrencySafe() bool {
	return false // 命令执行可能有副作用
}
