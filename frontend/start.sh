#!/bin/bash

# NaLa Coder 前端启动脚本

echo "🚀 启动 NaLa Coder 前端测试..."

# 检查后端服务是否运行
echo "📡 检查后端服务..."
if curl -s http://localhost:8888/api/health > /dev/null 2>&1; then
    echo "✅ 后端服务正在运行"
else
    echo "❌ 后端服务未运行，请先启动后端服务"
    echo "💡 提示: 运行 'go run cmd/server/main.go' 启动后端"
    exit 1
fi

# 打开浏览器
echo "🌐 打开浏览器..."
if command -v open > /dev/null; then
    # macOS
    open http://localhost:8888/frontend/
    open http://localhost:8888/frontend/test.html
elif command -v xdg-open > /dev/null; then
    # Linux
    xdg-open http://localhost:8888/frontend/
    xdg-open http://localhost:8888/frontend/test.html
elif command -v start > /dev/null; then
    # Windows
    start http://localhost:8888/frontend/
    start http://localhost:8888/frontend/test.html
else
    echo "📋 请手动打开以下链接:"
    echo "   主页面: http://localhost:8888/frontend/"
    echo "   测试页面: http://localhost:8888/frontend/test.html"
fi

echo "✨ 前端已启动!"
echo ""
echo "📝 使用说明:"
echo "   1. 主页面包含完整的AI代码助手功能"
echo "   2. 测试页面可以验证各项功能是否正常"
echo "   3. 语音功能需要允许浏览器麦克风权限"
echo "   4. 建议使用Chrome浏览器以获得最佳体验"
echo ""
echo "🔧 故障排除:"
echo "   - 如果页面无法加载，请检查后端服务是否运行"
echo "   - 如果语音功能不工作，请检查浏览器权限设置"
echo "   - 如果API调用失败，请检查网络连接" 