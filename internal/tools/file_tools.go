package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

func init() {
	registerBuiltinTool("read", &ReadTool{})
	registerBuiltinTool("write", &WriteTool{})
	registerBuiltinTool("edit", &EditTool{})
	registerBuiltinTool("multi_edit", &MultiEditTool{})
}

// ReadTool 文件读取工具
type ReadTool struct{}

func NewReadTool() *ReadTool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		FilePath string `json:"file_path"`
		Limit    int    `json:"limit,omitempty"`
		Offset   int    `json:"offset,omitempty"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	content, err := utils.ReadFileContent(params.FilePath)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}
	}

	lines := strings.Split(content, "\n")

	// 处理分页
	start := params.Offset
	end := len(lines)
	if params.Limit > 0 {
		end = start + params.Limit
		if end > len(lines) {
			end = len(lines)
		}
	}

	if start >= len(lines) {
		return &types.ToolCallResult{
			Success: true,
			Content: "File is empty or offset is beyond file length",
		}
	}

	// 格式化输出（类似cat -n）
	var result strings.Builder
	for i := start; i < end; i++ {
		result.WriteString(fmt.Sprintf("%6d→%s\n", i+1, lines[i]))
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result.String(),
	}
}

func (t *ReadTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name: "read",
			Description: `"Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.
		
Usage:
- The file_path parameter must be an absolute path, not a relative path.
- By default, it reads up to 2000 lines starting from the beginning of the file.
- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters.
- Any lines longer than 2000 characters will be truncated.
- Results are returned using cat -n format, with line numbers starting at 1.
- You have the capability to call multiple tools in a single response. It is always better to speculatively read multiple files as a batch that are potentially useful. 
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents."`,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to read",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Number of lines to read (optional)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Line number to start reading from (optional)",
					},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

func (t *ReadTool) IsConcurrencySafe() bool {
	return true
}

// WriteTool 文件写入工具
type WriteTool struct{}

func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	// 检查文件是否存在，如果存在要求先读取
	if utils.FileExists(params.FilePath) {
		return &types.ToolCallResult{
			Success: false,
			Error:   "file already exists, please use read tool first to check existing content",
		}
	}

	if err := utils.WriteFileContent(params.FilePath, params.Content); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.FilePath),
	}
}

func (t *WriteTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "write",
			Description: "Writes a file to the local filesystem.\n\nUsage:\n- This tool will overwrite the existing file if there is one at the provided path.\n- If this is an existing file, you MUST use the Read tool first to read the file's contents. This tool will fail if you did not read the file first.\n- ALWAYS prefer editing existing files in the edit.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to write. Relative paths are not supported.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
	}
}

func (t *WriteTool) IsConcurrencySafe() bool {
	return false
}

// EditTool 文件编辑工具
type EditTool struct{}

func NewEditTool() *EditTool {
	return &EditTool{}
}

func (t *EditTool) Name() string {
	return "edit"
}

type EditToolParams struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (t *EditTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params EditToolParams

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	if params.OldString == params.NewString {
		return &types.ToolCallResult{
			Success: false,
			Error:   "old_string and new_string are identical",
		}
	}

	content, err := utils.ReadFileContent(params.FilePath)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}
	}

	newContent, err := SearchReplace(ctx, content, &params)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	if err := utils.WriteFileContent(params.FilePath, newContent); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: fmt.Sprintf("Successfully replaced in %s", params.FilePath),
	}
}

func SearchReplace(ctx context.Context, fileContent string, params *EditToolParams) (string, error) {
	if params.OldString == params.NewString {
		return "", fmt.Errorf("old_string and new_string cannot be the same")
	}

	if params.OldString == "" { // add new string to end of file
		return fileContent + "\n" + params.NewString, nil
	}

	if !strings.Contains(fileContent, params.OldString) {
		return "", fmt.Errorf("old_string not found in file")
	}

	if params.ReplaceAll {
		return strings.ReplaceAll(fileContent, params.OldString, params.NewString), nil
	}

	newContent := strings.Replace(fileContent, params.OldString, params.NewString, 1)
	return newContent, nil
}

func (t *EditTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "edit",
			Description: "Replaces text within a file. By default, replaces a single occurrence, but can replace multiple occurrences when `expected_replacements` is specified. This tool requires providing significant context around the change to ensure precise targeting. Always use the read_file tool to examine the file's current content before attempting a text replacement.\n\nThe user has the ability to modify the `new_string` content. If modified, this will be stated in the response.\n\nExpectation for required parameters:\n1. `file_path` MUST be an absolute path; otherwise an error will be thrown.\n2. `old_string` MUST be the exact literal text to replace (including all whitespace, indentation, newlines, and surrounding code etc.).\n3. `new_string` MUST be the exact literal text to replace `old_string` with (also including all whitespace, indentation, newlines, and surrounding code etc.). Ensure the resulting code is correct and idiomatic.\n4. NEVER escape `old_string` or `new_string`, that would break the exact literal text requirement.\n**Important:** If ANY of the above are not satisfied, the tool will fail. CRITICAL for `old_string`: Must uniquely identify the single instance to change. Include at least 3 lines of context BEFORE and AFTER the target text, matching whitespace and indentation precisely. If this string matches multiple locations, or does not match exactly, the tool will fail.\n**Multiple replacements:** Set `expected_replacements` to the number of occurrences you want to replace. The tool will replace ALL occurrences that match `old_string` exactly. Ensure the number of replacements matches your expectation.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to edit",
					},
					"old_string": map[string]any{
						"type":        "string",
						"description": "The exact literal text to replace, preferably unescaped. For single replacements (default), include at least 3 lines of context BEFORE and AFTER the target text, matching whitespace and indentation precisely. For multiple replacements, specify expected_replacements parameter. If this string is not the exact literal text (i.e. you escaped it) or does not match exactly, the tool will fail.",
					},
					"new_string": map[string]any{
						"type":        "string",
						"description": "The exact literal text to replace `old_string` with, preferably unescaped. Provide the EXACT text. Ensure the resulting code is correct and idiomatic.",
					},
					"replace_all": map[string]any{
						"type":        "boolean",
						"description": "Replace all occurrences (default: false)",
					},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			},
		},
	}
}

func (t *EditTool) IsConcurrencySafe() bool {
	return false
}

// MultiEditTool 多重编辑工具
type MultiEditTool struct{}

func NewMultiEditTool() *MultiEditTool {
	return &MultiEditTool{}
}

func (t *MultiEditTool) Name() string {
	return "multi_edit"
}

func (t *MultiEditTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		FilePath string           `json:"file_path"`
		Edits    []EditToolParams `json:"edits"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	content, err := utils.ReadFileContent(params.FilePath)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}
	}

	currentContent := content

	// 按顺序应用所有编辑
	for _, edit := range params.Edits {
		newContent, err := SearchReplace(ctx, currentContent, &edit)
		if err != nil {
			return &types.ToolCallResult{
				Success: false,
				Error:   err.Error(),
			}
		}
		currentContent = newContent
	}

	if err := utils.WriteFileContent(params.FilePath, currentContent); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: fmt.Sprintf("Successfully applied %d edits in %s", len(params.Edits), params.FilePath),
	}
}

func (t *MultiEditTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "multi_edit",
			Description: "Apply multiple edits to a single file in sequence",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to edit",
					},
					"edits": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"old_string": map[string]any{
									"type":        "string",
									"description": "Text to replace",
								},
								"new_string": map[string]any{
									"type":        "string",
									"description": "Replacement text",
								},
								"replace_all": map[string]any{
									"type":        "boolean",
									"description": "Replace all occurrences (default: false)",
								},
							},
							"required": []string{"old_string", "new_string"},
						},
						"description": "Array of edit operations to apply sequentially",
					},
				},
				"required": []string{"file_path", "edits"},
			},
		},
	}
}

func (t *MultiEditTool) IsConcurrencySafe() bool {
	return false
}
