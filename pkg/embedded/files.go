package embedded

import (
	"embed"
	"html/template"
	"io/fs"
)

// 嵌入web构建后的静态文件
//
//go:embed web/dist
var webFiles embed.FS

// GetWebFS 返回嵌入的web文件系统
func GetWebFS() fs.FS {
	return webFiles
}

// GetTemplates 返回解析后的HTML模板（已废弃，React应用不需要）
func GetTemplates() (*template.Template, error) {
	return template.ParseFS(webFiles, "web/dist/*.html")
}

// GetTemplatesFS 返回模板文件系统（用于gin）
func GetTemplatesFS() fs.FS {
	return webFiles
}
