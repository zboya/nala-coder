package tools

import "github.com/zboya/nala-coder/pkg/types"

var builtinTools = map[string]types.ToolExecutor{}

func registerBuiltinTool(name string, tool types.ToolExecutor) {
	builtinTools[name] = tool
}

func getBuiltinTool(name string) types.ToolExecutor {
	return builtinTools[name]
}
