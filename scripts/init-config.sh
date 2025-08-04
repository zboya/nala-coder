#!/bin/bash

# NaLa Coder 配置初始化脚本
# 将配置文件和提示词复制到 ~/.nala-coder 目录

set -e

NALA_DIR="$HOME/.nala-coder"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 初始化 NaLa Coder 配置目录..."

# 创建主目录
mkdir -p "$NALA_DIR"
echo "✅ 创建目录: $NALA_DIR"

# 创建子目录
mkdir -p "$NALA_DIR/storage"
mkdir -p "$NALA_DIR/logs"
mkdir -p "$NALA_DIR/prompts"
echo "✅ 创建子目录: storage, logs, prompts"

# 复制配置文件
if [ ! -f "$NALA_DIR/config.yaml" ]; then
    cp "$PROJECT_ROOT/configs/config.yaml.example" "$NALA_DIR/config.yaml"
    echo "✅ 复制配置文件: config.yaml"
else
    echo "⚠️  配置文件已存在，跳过复制: config.yaml"
fi

# 复制提示词文件
if [ -d "$PROJECT_ROOT/prompts" ]; then
    cp -r "$PROJECT_ROOT/prompts/"* "$NALA_DIR/prompts/"
    echo "✅ 复制提示词文件到: $NALA_DIR/prompts/"
else
    echo "⚠️  源提示词目录不存在: $PROJECT_ROOT/prompts"
fi

echo ""
echo "🎉 配置初始化完成！"
echo ""
echo "配置目录结构："
echo "  ~/.nala-coder/"
echo "  ├── config.yaml      # 主配置文件"
echo "  ├── storage/          # 数据存储目录"
echo "  ├── logs/             # 日志文件目录"
echo "  └── prompts/          # 提示词文件目录"
echo "      ├── en/           # 英文提示词"
echo "      └── ch/           # 中文提示词"
echo ""
echo "下一步："
echo "1. 编辑配置文件: vi ~/.nala-coder/config.yaml"
echo "2. 设置你的 API 密钥"
echo "3. 运行 NaLa Coder: make run 或 ./nala-coder"
echo ""