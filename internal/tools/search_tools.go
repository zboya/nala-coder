package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zboya/nala-coder/pkg/grep"
	"github.com/zboya/nala-coder/pkg/types"
)

func init() {
	registerBuiltinTool("glob", &GlobTool{})
	registerBuiltinTool("grep", &GrepTool{})
	registerBuiltinTool("ls", &LSTool{})
}

// GlobTool 文件模式匹配工具
type GlobTool struct{}

func NewGlobTool() *GlobTool {
	return &GlobTool{}
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path,omitempty"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	searchPath := params.Path
	if searchPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return &types.ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("failed to get working directory: %v", err),
			}
		}
		searchPath = cwd
	}

	// 构建完整的模式
	fullPattern := filepath.Join(searchPath, params.Pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("glob pattern error: %v", err),
		}
	}

	// 获取文件信息并排序
	type fileInfo struct {
		path    string
		modTime int64
		isDir   bool
	}

	var files []fileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			path:    match,
			modTime: info.ModTime().Unix(),
			isDir:   info.IsDir(),
		})
	}

	// 按修改时间降序排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	if len(files) > 10 {
		files = files[:10]
	}

	var result strings.Builder
	if len(files) == 0 {
		result.WriteString("No files found matching pattern")
	} else {
		result.WriteString(fmt.Sprintf("Found %d file(s) matching pattern:\n", len(files)))
		for _, file := range files {
			if file.isDir {
				result.WriteString(fmt.Sprintf("d %s\n", file.path))
			} else {
				result.WriteString(fmt.Sprintf("f %s\n", file.path))
			}
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result.String(),
	}
}

func (t *GlobTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "glob",
			Description: "Fast file search based on fuzzy matching against file path. Use if you know part of the file path but don't know where it's located exactly. Response will be capped to 10 results. Make your query more specific if need to filter results further.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern to match files",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Directory to search in (optional, defaults to current directory)",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (t *GlobTool) IsConcurrencySafe() bool {
	return true
}

// GrepTool 内容搜索工具
type GrepTool struct{}

func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Explanation   string `json:"explanation"`
		CaseSensitive bool   `json:"case_sensitive"`
		Include       string `json:"include_pattern"`
		Exclude       string `json:"exclude_pattern"`
		Query         string `json:"query"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	config := grep.DefaultConfig()
	config.Pattern = params.Query
	config.CaseSensitive = params.CaseSensitive
	config.EnableColors = false
	if params.Exclude != "" {
		config.ExcludePatterns = []string{params.Exclude}
	}
	if params.Include != "" {
		config.IncludePatterns = []string{params.Include}
	}
	config.ShowContext = 2
	config.MaxResults = 10

	searcher, err := grep.NewRipgrepClone(config)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create searcher: %v", err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err = searcher.Search(ctx, ".")
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to search: %v", err),
		}
	}
	cost := time.Since(start)
	results := searcher.PrintResults(cost)

	var result strings.Builder
	if len(results) == 0 {
		result.WriteString("No matches found")
	} else {
		result.WriteString(results)
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result.String(),
	}
}

func (t *GrepTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "grep",
			Description: "### Instructions:\nThis is best for finding exact text matches or regex patterns.\nThis is preferred over semantic search when we know the exact symbol/function name/etc. to search in some set of directories/file types.\n\nUse this tool to run fast, exact regex searches over text files using the `ripgrep` engine.\nTo avoid overwhelming output, the results are capped at 50 matches.\nUse the include or exclude patterns to filter the search scope by file type or specific paths.\n\n- Always escape special regex characters: ( ) [ ] { } + * ? ^ $ | . \\\n- Use `\\` to escape any of these characters when they appear in your search string.\n- Do NOT perform fuzzy or semantic matches.\n- Return only a valid regex pattern string.\n\n### Examples:\n| Literal               | Regex Pattern            |\n|-----------------------|--------------------------|\n| function(             | function\\(              |\n| value[index]          | value\\[index\\]         |\n| file.txt               | file\\.txt                |\n| user|admin            | user\\|admin             |\n| path\\to\\file         | path\\\\to\\\\file        |\n| hello world           | hello world              |\n| foo\\(bar\\)          | foo\\\\(bar\\\\)         |",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"case_sensitive": map[string]any{
						"type":        "boolean",
						"description": "Whether the search should be case sensitive",
					},
					"exclude_pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern for files to exclude",
					},
					"explanation": map[string]any{
						"type":        "string",
						"description": "One sentence explanation as to why this tool is being used, and how it contributes to the goal.",
					},
					"include_pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern for files to include (e.g. '*.ts' for TypeScript files)",
					},
					"query": map[string]any{
						"type":        "string",
						"description": "The regex pattern to search for",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (t *GrepTool) IsConcurrencySafe() bool {
	return true
}

// LSTool 目录列举工具
type LSTool struct{}

func NewLSTool() *LSTool {
	return &LSTool{}
}

func (t *LSTool) Name() string {
	return "ls"
}

func (t *LSTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Path   string   `json:"path"`
		Ignore []string `json:"ignore,omitempty"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	// 检查路径是否存在
	info, err := os.Stat(params.Path)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("path does not exist: %v", err),
		}
	}

	if !info.IsDir() {
		return &types.ToolCallResult{
			Success: false,
			Error:   "path is not a directory",
		}
	}

	// 读取目录内容
	entries, err := os.ReadDir(params.Path)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read directory: %v", err),
		}
	}

	// 编译忽略模式
	var ignorePatterns []*regexp.Regexp
	for _, pattern := range params.Ignore {
		// 将glob模式转换为正则表达式
		globRegex := strings.ReplaceAll(pattern, "*", ".*")
		globRegex = strings.ReplaceAll(globRegex, "?", ".")
		regex, err := regexp.Compile(globRegex)
		if err == nil {
			ignorePatterns = append(ignorePatterns, regex)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Contents of directory: %s\n\n", params.Path))

	for _, entry := range entries {
		name := entry.Name()

		// 检查是否应该忽略
		shouldIgnore := false
		for _, pattern := range ignorePatterns {
			if pattern.MatchString(name) {
				shouldIgnore = true
				break
			}
		}

		if shouldIgnore {
			continue
		}

		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("d %s/\n", name))
		} else {
			// 获取文件大小
			fullPath := filepath.Join(params.Path, name)
			if fileInfo, err := os.Stat(fullPath); err == nil {
				size := fileInfo.Size()
				result.WriteString(fmt.Sprintf("f %s (%d bytes)\n", name, size))
			} else {
				result.WriteString(fmt.Sprintf("f %s\n", name))
			}
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result.String(),
	}
}

func (t *LSTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "ls",
			Description: "List contents of a directory",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the directory to list",
					},
					"ignore": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Glob patterns to ignore",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *LSTool) IsConcurrencySafe() bool {
	return true
}
