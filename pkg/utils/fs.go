package utils

import (
	"os"
	"path/filepath"
	"strings"
)

var defaultMaxItems = 200

var omitDirs = []string{
	"node_modules",
	"vendor",
	"target",
	"build",
	"dist",
	"log",
	"tmp",
	".git",
	".gitignore",
}

func isOmitDir(dir string) bool {
	for _, d := range omitDirs {
		if dir == d {
			return true
		}
	}
	return false
}

// BFSDirectoryTraversal 广度遍历所有目录和文件
/*
cursor_test/
  - code
  - cpp/
    - test.cpp
  - py/
    - test.py
*/
func BFSDirectoryTraversal(root string, maxItems int) (string, error) {
	if maxItems == 0 {
		maxItems = defaultMaxItems
	}

	var result strings.Builder
	itemCount := 0

	type dirInfo struct {
		path  string
		depth int
	}

	// 递归处理每个目录
	var processDir func(dirPath string, depth int) error
	processDir = func(dirPath string, depth int) error {
		if itemCount >= maxItems {
			return nil
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}

		// 分别收集文件和目录
		var files []os.DirEntry
		var dirs []os.DirEntry

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if isOmitDir(entry.Name()) {
				continue
			}

			if entry.IsDir() {
				dirs = append(dirs, entry)
			} else {
				files = append(files, entry)
			}
		}

		// 先输出当前目录的所有文件
		for _, file := range files {
			if itemCount >= maxItems {
				break
			}
			indent := strings.Repeat("  ", depth)
			result.WriteString(indent + "- " + file.Name() + "\n")
			itemCount++
		}

		// 然后递归处理子目录
		for _, dir := range dirs {
			if itemCount >= maxItems {
				break
			}
			indent := strings.Repeat("  ", depth)
			result.WriteString(indent + dir.Name() + "/\n")
			itemCount++

			fullPath := filepath.Join(dirPath, dir.Name())
			if err := processDir(fullPath, depth+1); err != nil {
				return err
			}
		}

		return nil
	}

	// 处理根目录
	if err := processDir(root, 0); err != nil {
		return "", err
	}

	return result.String(), nil
}
