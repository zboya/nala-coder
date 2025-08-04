import { useState, useEffect } from 'react';
import { ChatBox } from '@/components/ChatBox';
import { FileExplorer } from '@/components/FileExplorer';
import { CodeEditor } from '@/components/CodeEditor';
import { VoiceAssistant } from '@/components/VoiceAssistant';
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '@/components/ui/resizable';
import { streamChat, getFileTree, getSpeechConfig } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

interface FileNode {
  id: string;
  name: string;
  type: 'file' | 'folder';
  children?: FileNode[];
  path: string;
  extension?: string;
}

const Index = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [selectedFile, setSelectedFile] = useState<FileNode | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [sessionId, setSessionId] = useState<string>('');
  const { toast } = useToast();

  const handleSendMessage = async (content: string): Promise<void> => {
    // 添加用户消息
    const userMessage: Message = {
      id: Date.now().toString(),
      type: 'user',
      content,
      timestamp: new Date(),
    };
    
    setMessages(prev => [...prev, userMessage]);

    // 创建助手消息占位符
    const assistantMessageId = (Date.now() + 1).toString();
    const assistantMessage: Message = {
      id: assistantMessageId,
      type: 'assistant',
      content: '',
      timestamp: new Date(),
    };
    
    setMessages(prev => [...prev, assistantMessage]);

    try {
      // 调用流式聊天API
      await streamChat(
        content,
        sessionId,
        (chunk) => {
          // 更新助手消息内容
          setMessages(prev => prev.map(msg => 
            msg.id === assistantMessageId 
              ? { ...msg, content: msg.content + chunk.response }
              : msg
          ));
          
          // 如果是第一次响应且有session_id，保存它
          if (chunk.session_id && !sessionId) {
            setSessionId(chunk.session_id);
          }
        }
      );
    } catch (error) {
      console.error('Chat error:', error);
      toast({
        title: "发送消息失败",
        description: "无法连接到AI助手，请稍后再试",
        variant: "destructive",
      });
      
      // 移除失败的助手消息
      setMessages(prev => prev.filter(msg => msg.id !== assistantMessageId));
    }
  };

  const handleVoiceTranscript = (transcript: string) => {
    handleSendMessage(transcript);
  };

  const handleFileSelect = (file: FileNode) => {
    if (file.type === 'file') {
      setSelectedFile(file);
    }
  };

  // 加载文件树
  useEffect(() => {
    const loadFileTree = async () => {
      try {
        const fileTree = await getFileTree();
        setFiles(fileTree);
      } catch (error) {
        console.error('Failed to load file tree:', error);
        toast({
          title: "加载文件失败",
          description: "无法获取项目文件列表",
          variant: "destructive",
        });
      }
    };

    loadFileTree();
  }, [toast]);

  return (
    <div className="h-screen w-full bg-background">
      <ResizablePanelGroup direction="horizontal" className="min-h-screen">
        {/* 左侧：聊天框和文件浏览器 */}
        <ResizablePanel defaultSize={35} minSize={25} maxSize={50}>
          <ResizablePanelGroup direction="vertical">
            {/* 聊天框 */}
            <ResizablePanel defaultSize={60} minSize={30}>
              <ChatBox 
                messages={messages} 
                onSendMessage={handleSendMessage} 
              />
            </ResizablePanel>
            
            <ResizableHandle withHandle />
            
            {/* 文件浏览器 */}
            <ResizablePanel defaultSize={40} minSize={30}>
            <FileExplorer 
              files={files} 
              selectedFile={selectedFile?.id}
              onFileSelect={handleFileSelect}
            />
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>

        <ResizableHandle withHandle />

        {/* 右侧：代码编辑器 */}
        <ResizablePanel defaultSize={65} minSize={50}>
          <CodeEditor selectedFile={selectedFile} />
        </ResizablePanel>
      </ResizablePanelGroup>

      {/* 悬浮语音助手 */}
      <VoiceAssistant onTranscript={handleVoiceTranscript} />
    </div>
  );
};

export default Index;
