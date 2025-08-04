package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenerateID 生成唯一ID
func GenerateID() string {
	return uuid.New().String()
}

// GenerateShortID 生成短ID
func GenerateShortID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CountTokens 简单的token计数估算
func CountTokens(text string) int {
	// 简单估算: 1 token ≈ 4个字符 (英文) 或 1.5个中文字符
	chars := len([]rune(text))
	chineseCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		}
	}

	englishCount := chars - chineseCount
	return int(float64(englishCount)/4 + float64(chineseCount)/1.5)
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ReadFileContent 读取文件内容
func ReadFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFileContent 写入文件内容
func WriteFileContent(path, content string) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// JSONMarshal 美化JSON序列化
func JSONMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// JSONUnmarshal JSON反序列化
func JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SafeString 安全字符串处理，移除控制字符
func SafeString(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// TruncateString 截断字符串
func TruncateString(s string, maxLength int) string {
	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}
	return string(runes[:maxLength]) + "..."
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// PathJoin 连接路径
func PathJoin(paths ...string) string {
	return filepath.Join(paths...)
}

// AbsPath 获取绝对路径
func AbsPath(path string) (string, error) {
	return filepath.Abs(path)
}

// ExpandPath 扩展路径，处理 ~ 符号
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // 如果获取失败，返回原路径
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// IsAbsPath 判断是否为绝对路径
func IsAbsPath(path string) bool {
	return filepath.IsAbs(path)
}

// ExtractFileExtension 提取文件扩展名
func ExtractFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// SanitizeFilename 清理文件名，移除非法字符
func SanitizeFilename(filename string) string {
	// 移除路径分隔符和其他非法字符
	illegal := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range illegal {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// ParseJSONArguments 解析JSON参数
func ParseJSONArguments(args string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(args), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON arguments: %w", err)
	}
	return result, nil
}

// MergeMap 合并map
func MergeMap(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Contains 检查切片是否包含元素
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates 移除重复元素
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}
