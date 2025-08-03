package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

func init() {
	registerBuiltinTool("web_search", &WebSearchTool{})
	registerBuiltinTool("web_fetch", &WebFetchTool{})
}

// SearchResult 搜索结果结构
type SearchResult struct {
	Title   string
	URL     string
	Summary string
}

// WebSearchTool 网络搜索工具
type WebSearchTool struct{}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

// duckduckgoSearch 执行DuckDuckGo搜索
func (t *WebSearchTool) duckduckgoSearch(ctx context.Context, query string) ([]SearchResult, error) {
	// 构建搜索URL
	searchURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置User-Agent，模拟浏览器请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search request failed with status: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析搜索结果
	return t.parseSearchResults(string(body)), nil
}

// parseSearchResults 解析DuckDuckGo搜索结果HTML
func (t *WebSearchTool) parseSearchResults(html string) []SearchResult {
	var results []SearchResult

	// 更简单的方式：直接匹配包含链接的模式
	// 匹配DuckDuckGo搜索结果中的链接和标题
	linkPattern := regexp.MustCompile(`<a[^>]*rel="nofollow"[^>]*href="([^"]*)"[^>]*>([^<]+)</a>`)

	// 匹配搜索结果摘要文本
	snippetPattern := regexp.MustCompile(`<a[^>]*class="[^"]*snippet[^"]*"[^>]*>([^<]+)</a>`)

	linkMatches := linkPattern.FindAllStringSubmatch(html, -1)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, -1)

	// 创建摘要映射
	snippetMap := make(map[string]string)
	for _, snippetMatch := range snippetMatches {
		if len(snippetMatch) >= 2 {
			snippet := t.cleanHTMLText(snippetMatch[1])
			// 使用摘要的前50个字符作为key
			key := snippet
			if len(key) > 50 {
				key = key[:50]
			}
			snippetMap[key] = snippet
		}
	}

	for _, linkMatch := range linkMatches {
		if len(linkMatch) < 3 {
			continue
		}

		url := strings.TrimSpace(linkMatch[1])
		title := strings.TrimSpace(linkMatch[2])

		// 跳过DuckDuckGo内部链接和广告
		if strings.Contains(url, "duckduckgo.com") || strings.Contains(url, "/y.js?") {
			continue
		}

		// 验证URL格式
		if !strings.HasPrefix(url, "http") {
			continue
		}

		// 清理HTML实体
		title = t.cleanHTMLText(title)

		// 尝试找到对应的摘要
		summary := ""
		for _, snippet := range snippetMap {
			if len(snippet) > 10 { // 只使用有实际内容的摘要
				summary = snippet
				break
			}
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     url,
			Summary: summary,
		})

		// 限制结果数量
		if len(results) >= 10 {
			break
		}
	}

	return results
}

// cleanHTMLText 清理HTML文本
func (t *WebSearchTool) cleanHTMLText(text string) string {
	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	text = re.ReplaceAllString(text, "")

	// 解码HTML实体
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#x27;", "'")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}

// filterResultsByDomain 根据域名过滤搜索结果
func (t *WebSearchTool) filterResultsByDomain(results []SearchResult, allowedDomains, blockedDomains []string) []SearchResult {
	var filtered []SearchResult

	for _, result := range results {
		parsedURL, err := url.Parse(result.URL)
		if err != nil {
			continue
		}

		domain := parsedURL.Hostname()

		// 检查是否被阻止
		blocked := false
		for _, blockedDomain := range blockedDomains {
			if strings.Contains(domain, blockedDomain) {
				blocked = true
				break
			}
		}

		if blocked {
			continue
		}

		// 检查是否在允许列表中（如果有指定）
		if len(allowedDomains) > 0 {
			allowed := false
			for _, allowedDomain := range allowedDomains {
				if strings.Contains(domain, allowedDomain) {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	return filtered
}

func (t *WebSearchTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		Query          string   `json:"query"`
		AllowedDomains []string `json:"allowed_domains,omitempty"`
		BlockedDomains []string `json:"blocked_domains,omitempty"`
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	if len(params.Query) < 2 {
		return &types.ToolCallResult{
			Success: false,
			Error:   "query must be at least 2 characters long",
		}
	}

	// 执行DuckDuckGo搜索
	searchResults, err := t.duckduckgoSearch(ctx, params.Query)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("search failed: %v", err),
		}
	}

	// 应用域名过滤
	if len(params.AllowedDomains) > 0 || len(params.BlockedDomains) > 0 {
		searchResults = t.filterResultsByDomain(searchResults, params.AllowedDomains, params.BlockedDomains)
	}

	// 构建结果输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("DuckDuckGo search results for: %s\n", params.Query))

	if len(params.AllowedDomains) > 0 {
		result.WriteString(fmt.Sprintf("Allowed domains: %s\n", strings.Join(params.AllowedDomains, ", ")))
	}
	if len(params.BlockedDomains) > 0 {
		result.WriteString(fmt.Sprintf("Blocked domains: %s\n", strings.Join(params.BlockedDomains, ", ")))
	}

	result.WriteString(fmt.Sprintf("\nFound %d results:\n\n", len(searchResults)))

	if len(searchResults) == 0 {
		result.WriteString("No search results found.")
	} else {
		for i, searchResult := range searchResults {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, searchResult.Title))
			result.WriteString(fmt.Sprintf("   URL: %s\n", searchResult.URL))
			if searchResult.Summary != "" {
				result.WriteString(fmt.Sprintf("   Summary: %s\n", searchResult.Summary))
			}
			result.WriteString("\n")
		}
	}

	return &types.ToolCallResult{
		Success: true,
		Content: result.String(),
	}
}

func (t *WebSearchTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "web_search",
			Description: "Search the web for real-time information and return formatted search results",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"minLength":   2,
						"description": "The search query to use",
					},
					"allowed_domains": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Only include results from these domains",
					},
					"blocked_domains": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"description": "Never include results from these domains",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (t *WebSearchTool) IsConcurrencySafe() bool {
	return true
}

// WebFetchTool 网页内容获取工具
type WebFetchTool struct{}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Execute(ctx context.Context, call types.ToolCall) *types.ToolCallResult {
	var params struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers,omitempty"`
		Timeout int               `json:"timeout,omitempty"` // seconds
	}

	if err := json.Unmarshal([]byte(call.Function.Arguments), &params); err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse arguments: %v", err),
		}
	}

	// 验证URL
	parsedURL, err := url.Parse(params.URL)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("invalid URL: %v", err),
		}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &types.ToolCallResult{
			Success: false,
			Error:   "only HTTP and HTTPS URLs are supported",
		}
	}

	// 设置超时
	timeout := 30 // 默认30秒
	if params.Timeout > 0 {
		timeout = params.Timeout
	}
	if timeout > 120 { // 最大2分钟
		timeout = 120
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}
	}

	// 设置默认User-Agent
	req.Header.Set("User-Agent", "nala-coder/1.0 (Web Fetch Tool)")

	// 设置自定义头部
	for key, value := range params.Headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to fetch URL: %v", err),
		}
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &types.ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read response body: %v", err),
		}
	}

	// 限制内容长度
	content := string(body)
	if len(content) > 50000 {
		content = content[:50000] + "\n... (content truncated)"
	}

	// 清理内容
	content = utils.SafeString(content)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("URL: %s\n", params.URL))
	result.WriteString(fmt.Sprintf("Status: %d %s\n", resp.StatusCode, resp.Status))
	result.WriteString(fmt.Sprintf("Content-Type: %s\n", resp.Header.Get("Content-Type")))
	result.WriteString(fmt.Sprintf("Content-Length: %d bytes\n", len(body)))
	result.WriteString(fmt.Sprintf("Fetch Time: %v\n", duration))

	// 添加重要的响应头
	if server := resp.Header.Get("Server"); server != "" {
		result.WriteString(fmt.Sprintf("Server: %s\n", server))
	}
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		result.WriteString(fmt.Sprintf("Last-Modified: %s\n", lastModified))
	}

	result.WriteString(fmt.Sprintf("\nContent:\n%s", content))

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &types.ToolCallResult{
		Success: success,
		Content: result.String(),
		Error:   "",
	}
}

func (t *WebFetchTool) GetDefinition() types.Tool {
	return types.Tool{
		Type: "function",
		Function: types.ToolFunction{
			Name:        "web_fetch",
			Description: "Fetch content from a web URL and return the response",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch content from (must be HTTP or HTTPS)",
					},
					"headers": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "string",
						},
						"description": "Optional HTTP headers to include in the request",
					},
					"timeout": map[string]any{
						"type":        "integer",
						"description": "Timeout in seconds (default: 30, max: 120)",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

func (t *WebFetchTool) IsConcurrencySafe() bool {
	return true
}
