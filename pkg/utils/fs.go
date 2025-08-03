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

	queue := []dirInfo{{path: root, depth: 0}}

	for len(queue) > 0 && itemCount < maxItems {
		current := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(current.path)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			if itemCount >= maxItems {
				break
			}

			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if isOmitDir(entry.Name()) {
				continue
			}

			fullPath := filepath.Join(current.path, entry.Name())
			indent := strings.Repeat("  ", current.depth)

			if entry.IsDir() {
				result.WriteString(indent + entry.Name() + "/\n")
				queue = append(queue, dirInfo{path: fullPath, depth: current.depth + 1})
			} else {
				result.WriteString(indent + "- " + entry.Name() + "\n")
			}
			itemCount++
		}
	}

	return result.String(), nil
}
