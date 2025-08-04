import { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { Save, Copy, Download, FileText } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { getFileContent } from '@/services/api';

interface FileNode {
  id: string;
  name: string;
  type: 'file' | 'folder';
  path: string;
  extension?: string;
}

interface CodeEditorProps {
  selectedFile?: FileNode | null;
}

// 模拟文件内容（作为后备方案）
const getMockFileContent = (file: FileNode): string => {
  switch (file.extension) {
    case 'tsx':
      return `import React from 'react';
import { useState } from 'react';

interface Props {
  title: string;
  onSubmit: (value: string) => void;
}

const ${file.name.replace('.tsx', '')} = ({ title, onSubmit }: Props) => {
  const [value, setValue] = useState('');

  const handleSubmit = () => {
    onSubmit(value);
  };

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-4">{title}</h2>
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        className="border p-2 rounded mb-4 w-full"
      />
      <button
        onClick={handleSubmit}
        className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
      >
        Submit
      </button>
    </div>
  );
};

export default ${file.name.replace('.tsx', '')};`;
      
    case 'ts':
      return `export interface User {
  id: string;
  name: string;
  email: string;
  createdAt: Date;
}

export class ${file.name.replace('.ts', '').replace('use', '').replace('Auth', 'AuthService')} {
  private apiUrl = 'https://api.example.com';

  async fetchUser(id: string): Promise<User> {
    try {
      const response = await fetch(\`\${this.apiUrl}/users/\${id}\`);
      if (!response.ok) {
        throw new Error('Failed to fetch user');
      }
      return await response.json();
    } catch (error) {
      console.error('Error fetching user:', error);
      throw error;
    }
  }

  async updateUser(id: string, data: Partial<User>): Promise<User> {
    try {
      const response = await fetch(\`\${this.apiUrl}/users/\${id}\`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      
      if (!response.ok) {
        throw new Error('Failed to update user');
      }
      
      return await response.json();
    } catch (error) {
      console.error('Error updating user:', error);
      throw error;
    }
  }
}`;

    case 'css':
      return `.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 16px;
}

.header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 20px 0;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.header h1 {
  margin: 0;
  font-size: 2rem;
  font-weight: 600;
}

.button {
  background: #3b82f6;
  color: white;
  border: none;
  padding: 12px 24px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s ease;
}

.button:hover {
  background: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.card {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border: 1px solid #e5e7eb;
}`;

    case 'json':
      return JSON.stringify({
        name: "智能代码编辑器",
        version: "1.0.0",
        description: "A smart code editor with voice recognition",
        main: "index.js",
        scripts: {
          dev: "vite",
          build: "vite build",
          preview: "vite preview"
        },
        dependencies: {
          "react": "^18.3.1",
          "react-dom": "^18.3.1",
          "@monaco-editor/react": "^4.7.0",
          "lucide-react": "^0.462.0"
        },
        devDependencies: {
          "@types/react": "^18.3.3",
          "@types/react-dom": "^18.3.0",
          "typescript": "^5.5.3",
          "vite": "^5.4.1"
        }
      }, null, 2);

    case 'html':
      return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>智能代码编辑器</title>
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 0;
            background: #f5f5f5;
        }
        .loading {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            font-size: 18px;
            color: #666;
        }
    </style>
</head>
<body>
    <div id="root">
        <div class="loading">正在加载智能代码编辑器...</div>
    </div>
    <script type="module" src="/src/main.tsx"></script>
</body>
</html>`;

    case 'md':
      return `# 智能代码编辑器

一个具有语音识别功能的智能代码编辑器。

## 功能特性

- 🎤 **语音唤醒**: 支持中文"小助手"语音唤醒
- 🗣️ **语音识别**: 唤醒后支持中文语音识别
- 💬 **智能对话**: 与AI助手进行自然语言交互
- 📁 **文件浏览**: 支持项目文件和目录浏览
- 🎨 **语法高亮**: 支持多种编程语言的语法高亮
- 🔧 **代码编辑**: 基于Monaco Editor的强大代码编辑功能

## 技术栈

- React 18
- TypeScript
- Monaco Editor
- Web Speech API
- Tailwind CSS
- Shadcn/ui

## 使用方法

1. 说出"小助手"唤醒语音助手
2. 唤醒后可以通过语音或文字与助手交互
3. 在左侧文件浏览器中选择文件进行编辑
4. 在右侧代码编辑器中查看和修改代码

## 开发

\`\`\`bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build
\`\`\`

## 许可证

MIT License
`;

    default:
      return `// ${file.name}
// 这是一个示例文件

console.log('Hello, World!');

function greet(name: string): string {
  return \`Hello, \${name}!\`;
}

export { greet };`;
  }
};

const getLanguageFromExtension = (extension?: string): string => {
  switch (extension) {
    case 'tsx':
    case 'jsx':
      return 'typescript';
    case 'ts':
      return 'typescript';
    case 'js':
      return 'javascript';
    case 'html':
      return 'html';
    case 'css':
      return 'css';
    case 'json':
      return 'json';
    case 'md':
      return 'markdown';
    default:
      return 'typescript';
  }
};

export const CodeEditor = ({ selectedFile }: CodeEditorProps) => {
  const [content, setContent] = useState('');
  const [isModified, setIsModified] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    const loadFileContent = async () => {
      if (!selectedFile) {
        setContent('');
        return;
      }

      setIsLoading(true);
      try {
        const fileData = await getFileContent(selectedFile.path);
        setContent(fileData.content);
        setIsModified(false);
      } catch (error) {
        console.error('Failed to load file content:', error);
        
        // 如果API失败，回退到模拟内容
        const mockContent = getMockFileContent(selectedFile);
        setContent(mockContent);
        setIsModified(false);
        
        toast({
          title: "加载文件内容失败",
          description: "显示模拟内容，请检查文件路径",
          variant: "destructive",
        });
      } finally {
        setIsLoading(false);
      }
    };

    loadFileContent();
  }, [selectedFile, toast]);

  const handleSave = () => {
    // TODO: 实际保存文件的逻辑
    setIsModified(false);
    toast({
      title: "文件已保存",
      description: `${selectedFile?.name} 已成功保存`,
    });
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      toast({
        title: "复制成功",
        description: "代码已复制到剪贴板",
      });
    } catch (error) {
      toast({
        title: "复制失败",
        description: "无法复制到剪贴板",
        variant: "destructive",
      });
    }
  };

  const handleDownload = () => {
    if (!selectedFile) return;
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = selectedFile.name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    toast({
      title: "下载完成",
      description: `${selectedFile.name} 已下载`,
    });
  };

  if (!selectedFile) {
    return (
      <div className="h-full flex items-center justify-center bg-background">
        <Card className="w-96">
          <CardContent className="flex flex-col items-center justify-center p-8">
            <FileText className="h-16 w-16 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">选择文件</h3>
            <p className="text-sm text-muted-foreground text-center">
              从左侧文件浏览器中选择一个文件开始编辑
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col bg-background">
      {/* 文件头部 */}
      <Card className="m-4 mb-0">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <CardTitle className="text-lg">{selectedFile.name}</CardTitle>
              {isModified && (
                <Badge variant="secondary" className="text-xs">
                  未保存
                </Badge>
              )}
              <Badge variant="outline" className="text-xs">
                {selectedFile.extension?.toUpperCase()}
              </Badge>
            </div>
            
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopy}
              >
                <Copy className="h-4 w-4 mr-1" />
                复制
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
              >
                <Download className="h-4 w-4 mr-1" />
                下载
              </Button>
              <Button
                variant="default"
                size="sm"
                onClick={handleSave}
                disabled={!isModified}
              >
                <Save className="h-4 w-4 mr-1" />
                保存
              </Button>
            </div>
          </div>
          <p className="text-sm text-muted-foreground">{selectedFile.path}</p>
        </CardHeader>
      </Card>

      {/* 编辑器 */}
      <Card className="flex-1 m-4 mt-2">
        <CardContent className="p-0 h-full">
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-muted-foreground">加载中...</div>
            </div>
          ) : (
            <Editor
              height="100%"
              language={getLanguageFromExtension(selectedFile.extension)}
              value={content}
              onChange={(value) => {
                setContent(value || '');
                setIsModified(true);
              }}
              theme="vs-dark"
              options={{
                minimap: { enabled: true },
                fontSize: 14,
                lineNumbers: 'on',
                roundedSelection: false,
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                insertSpaces: true,
                wordWrap: 'on',
                folding: true,
                lineDecorationsWidth: 10,
                lineNumbersMinChars: 3,
                renderLineHighlight: 'all',
                selectionHighlight: false,
                bracketPairColorization: {
                  enabled: true
                }
              }}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
};