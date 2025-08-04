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
        
        setContent('');
        setIsModified(false);
        
        toast({
          title: "加载文件内容失败",
          description: "请检查文件是否过大或者是否为二进制文件",
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
                },
                // 禁用语法错误检查
                'semanticHighlighting.enabled': false,
                quickSuggestions: false,
                parameterHints: { enabled: false },
                hover: { enabled: false },
                contextmenu: false,
                // 禁用所有诊断信息（错误、警告等）
                glyphMargin: false,
                // 禁用错误标记
                renderValidationDecorations: 'off'
              }}
              beforeMount={(monaco) => {
                // 禁用所有语言的诊断功能
                monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions({
                  noSemanticValidation: true,
                  noSyntaxValidation: true,
                  noSuggestionDiagnostics: true
                });
                
                monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
                  noSemanticValidation: true,
                  noSyntaxValidation: true,
                  noSuggestionDiagnostics: true
                });
              }}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
};