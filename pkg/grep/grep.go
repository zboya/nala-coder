package grep

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ANSI 颜色代码
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// SearchConfig 搜索配置
type SearchConfig struct {
	Pattern         string   // 搜索模式
	IsRegex         bool     // 是否使用正则表达式
	CaseSensitive   bool     // 是否区分大小写
	WholeWord       bool     // 是否匹配整个单词
	ShowLineNumbers bool     // 是否显示行号
	ShowContext     int      // 显示上下文行数
	MaxResults      int      // 最大结果数
	IncludePatterns []string // 包含的文件模式
	ExcludePatterns []string // 排除的文件模式
	ExcludeDirs     []string // 排除的目录
	MaxDepth        int      // 最大搜索深度
	Workers         int      // 并发工作数
	EnableColors    bool     // 是否启用颜色
	ShowFilenames   bool     // 是否显示文件名
	InvertMatch     bool     // 反向匹配
}

// MatchResult 匹配结果
type MatchResult struct {
	Filename    string
	TotalLines  int
	LineNumber  int
	Line        string
	MatchStart  int
	MatchEnd    int
	ContextPrev []string
	ContextNext []string
	ModTime     time.Time // 文件修改时间
}

// RipgrepClone ripgrep克隆
type RipgrepClone struct {
	config  *SearchConfig
	regex   *regexp.Regexp
	mu      sync.Mutex
	results []*MatchResult
}

// NewRipgrepClone 创建新的搜索实例
func NewRipgrepClone(config *SearchConfig) (*RipgrepClone, error) {
	rg := &RipgrepClone{
		config:  config,
		results: make([]*MatchResult, 0),
	}

	// 编译正则表达式
	pattern := config.Pattern
	if !config.IsRegex {
		pattern = regexp.QuoteMeta(pattern)
	}

	if config.WholeWord {
		pattern = `\b` + pattern + `\b`
	}

	flags := ""
	if !config.CaseSensitive {
		flags = "(?i)"
	}

	regex, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	rg.regex = regex

	return rg, nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *SearchConfig {
	return &SearchConfig{
		IsRegex:         false,
		CaseSensitive:   true,
		WholeWord:       false,
		ShowLineNumbers: true,
		ShowContext:     0,
		MaxResults:      1000,
		ExcludeDirs: []string{
			".git", ".svn", ".hg",
			"node_modules", "vendor", "target",
			"build", "dist", ".vscode",
		},
		MaxDepth:      50,
		Workers:       runtime.NumCPU(),
		EnableColors:  true,
		ShowFilenames: true,
		InvertMatch:   false,
	}
}

// Search 执行搜索
func (rg *RipgrepClone) Search(ctx context.Context, rootPath string) error {
	// 创建工作通道
	fileChan := make(chan string, 100)
	resultChan := make(chan *MatchResult, 100)

	// 启动文件遍历goroutine
	go rg.walkFiles(ctx, rootPath, fileChan)

	// 启动worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < rg.config.Workers; i++ {
		wg.Add(1)
		go rg.searchWorker(ctx, fileChan, resultChan, &wg)
	}

	// 启动结果收集goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		rg.mu.Lock()
		rg.results = append(rg.results, result)
		if len(rg.results) >= rg.config.MaxResults {
			rg.mu.Unlock()
			break
		}
		rg.mu.Unlock()
	}

	return nil
}

// walkFiles 遍历文件
func (rg *RipgrepClone) walkFiles(ctx context.Context, rootPath string, fileChan chan<- string) {
	defer close(fileChan)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 忽略错误，继续遍历
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 检查目录深度
		depth := strings.Count(strings.TrimPrefix(path, rootPath), string(os.PathSeparator))
		if depth > rg.config.MaxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// 跳过排除的目录
		if d.IsDir() && rg.shouldExcludeDir(d.Name()) {
			return fs.SkipDir
		}

		// 只处理文件
		if d.IsDir() {
			return nil
		}

		// 检查文件是否匹配包含/排除模式
		if rg.shouldIncludeFile(path) {
			select {
			case fileChan <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})

	if err != nil && err != context.Canceled {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}

// searchWorker 搜索工作协程
func (rg *RipgrepClone) searchWorker(ctx context.Context, fileChan <-chan string, resultChan chan<- *MatchResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case filename, ok := <-fileChan:
			if !ok {
				return
			}
			rg.searchInFile(filename, resultChan)
		}
	}
}

// searchInFile 在文件中搜索
func (rg *RipgrepClone) searchInFile(filename string, resultChan chan<- *MatchResult) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	// 获取文件修改时间
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}
	modTime := fileInfo.ModTime()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var lines []string

	// 读取所有行（用于上下文显示）
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return
	}

	// 搜索匹配
	for i, line := range lines {
		lineNumber = i + 1

		matches := rg.regex.FindAllStringIndex(line, -1)
		hasMatch := len(matches) > 0

		// 反向匹配逻辑
		if rg.config.InvertMatch {
			hasMatch = !hasMatch
		}

		if hasMatch {
			for _, match := range matches {
				result := &MatchResult{
					Filename:   filename,
					TotalLines: len(lines),
					LineNumber: lineNumber,
					Line:       line,
					MatchStart: match[0],
					MatchEnd:   match[1],
					ModTime:    modTime,
				}

				// 添加上下文
				if rg.config.ShowContext > 0 {
					result.ContextPrev = rg.getContextLines(lines, i, -rg.config.ShowContext, 0)
					result.ContextNext = rg.getContextLines(lines, i, 1, rg.config.ShowContext+1)
				}

				select {
				case resultChan <- result:
				default:
					return
				}
			}
		}
	}
}

// getContextLines 获取上下文行
func (rg *RipgrepClone) getContextLines(lines []string, currentIndex, start, end int) []string {
	var context []string

	startIdx := currentIndex + start
	endIdx := currentIndex + end

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	for i := startIdx; i < endIdx; i++ {
		if i != currentIndex {
			context = append(context, lines[i])
		}
	}

	return context
}

// shouldExcludeDir 检查是否应该排除目录
func (rg *RipgrepClone) shouldExcludeDir(dirname string) bool {
	for _, exclude := range rg.config.ExcludeDirs {
		if dirname == exclude {
			return true
		}
	}
	return false
}

// shouldIncludeFile 检查是否应该包含文件
func (rg *RipgrepClone) shouldIncludeFile(filename string) bool {
	// 检查排除模式
	for _, pattern := range rg.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filename)); matched {
			return false
		}
	}

	// 检查包含模式
	if len(rg.config.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range rg.config.IncludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filename)); matched {
			return true
		}
	}

	return false
}

// PrintResults 打印搜索结果
func (rg *RipgrepClone) PrintResults(cost time.Duration) string {
	results := ""
	// 按修改时间倒序排序
	sort.Slice(rg.results, func(i, j int) bool {
		if !rg.results[i].ModTime.Equal(rg.results[j].ModTime) {
			return rg.results[i].ModTime.After(rg.results[j].ModTime)
		}
		// 如果修改时间相同，则按行号排序
		return rg.results[i].LineNumber < rg.results[j].LineNumber
	})

	currentFile := ""
	for _, result := range rg.results {
		// 打印文件名（如果是新文件）
		if rg.config.ShowFilenames && result.Filename != currentFile {
			currentFile = result.Filename
			if rg.config.EnableColors {
				results += fmt.Sprintf("%s%s%s total_lines: %d %s\n", ColorBold, ColorPurple, result.Filename, result.TotalLines, ColorReset)
			} else {
				results += fmt.Sprintf("%s total_lines: %d\n", result.Filename, result.TotalLines)
			}
		}

		// 打印上下文（之前）
		for i, contextLine := range result.ContextPrev {
			lineNum := result.LineNumber - len(result.ContextPrev) + i
			results += rg.printLine(lineNum, contextLine, false)
		}

		// 打印匹配行
		results += rg.printLine(result.LineNumber, result.Line, true)

		// 打印上下文（之后）
		for i, contextLine := range result.ContextNext {
			lineNum := result.LineNumber + i + 1
			results += rg.printLine(lineNum, contextLine, false)
		}

		if len(result.ContextPrev) > 0 || len(result.ContextNext) > 0 {
			results += "--"
		}
	}
	totalMatches, fileStats := rg.GetStats()
	results += fmt.Sprintf("\nFound %d matches in %d files (%.3fs)\n",
		totalMatches, len(fileStats), cost.Seconds())
	return results
}

// printLine 打印单行结果
func (rg *RipgrepClone) printLine(lineNumber int, line string, isMatch bool) string {
	result := ""
	prefix := ""

	if rg.config.ShowLineNumbers {
		if rg.config.EnableColors {
			if isMatch {
				prefix = fmt.Sprintf("%s%s%d%s:", ColorBold, ColorGreen, lineNumber, ColorReset)
			} else {
				prefix = fmt.Sprintf("%s%d%s-", ColorBlue, lineNumber, ColorReset)
			}
		} else {
			if isMatch {
				prefix = fmt.Sprintf("%d:", lineNumber)
			} else {
				prefix = fmt.Sprintf("%d-", lineNumber)
			}
		}
	}

	if isMatch && rg.config.EnableColors {
		// 高亮匹配的文本
		highlightedLine := rg.highlightMatches(line)
		result += fmt.Sprintf("%s%s\n", prefix, highlightedLine)
	} else {
		result += fmt.Sprintf("%s%s\n", prefix, line)
	}
	return result
}

// highlightMatches 高亮匹配的文本
func (rg *RipgrepClone) highlightMatches(line string) string {
	if !rg.config.EnableColors {
		return line
	}

	matches := rg.regex.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// 添加匹配前的文本
		result.WriteString(line[lastEnd:match[0]])
		// 添加高亮的匹配文本
		result.WriteString(ColorBold + ColorRed)
		result.WriteString(line[match[0]:match[1]])
		result.WriteString(ColorReset)
		lastEnd = match[1]
	}

	// 添加剩余的文本
	result.WriteString(line[lastEnd:])
	return result.String()
}

// GetStats 获取搜索统计
func (rg *RipgrepClone) GetStats() (int, map[string]int) {
	fileCount := make(map[string]int)
	for _, result := range rg.results {
		fileCount[result.Filename]++
	}
	return len(rg.results), fileCount
}

// 示例使用
func main() {
	config := DefaultConfig()
	config.Pattern = "newInputLambda"
	// config.IncludePatterns = []string{"*.go"}
	config.ShowContext = 2
	config.MaxResults = 50

	searcher, err := NewRipgrepClone(config)
	if err != nil {
		fmt.Printf("Error creating searcher: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Searching for '%s' in Go files...\n\n", config.Pattern)

	start := time.Now()
	err = searcher.Search(ctx, ".")
	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}

	duration := time.Since(start)

	result := searcher.PrintResults(duration)
	fmt.Println(result)
}
