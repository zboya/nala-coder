import { useState, useRef, useEffect } from 'react';
import { Send, Bot, User, Mic, MicOff, Volume2, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Card, CardContent } from '@/components/ui/card';
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition';
import { getSpeechConfig } from '@/services/api';
import { cn } from '@/lib/utils';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

interface ChatBoxProps {
  onSendMessage: (message: string) => void;
  messages: Message[];
}

export const ChatBox = ({ onSendMessage, messages }: ChatBoxProps) => {
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);
  const [wakeWord, setWakeWord] = useState('小助手');
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 语音识别功能
  const {
    isListening,
    isWakeWordListening,
    isAwake,
    transcript,
    error,
    sleep,
    resetTranscript
  } = useSpeechRecognition();

  // 加载语音配置
  useEffect(() => {
    const loadSpeechConfig = async () => {
      try {
        const config = await getSpeechConfig();
        if (config.wake_words && config.wake_words.length > 0) {
          setWakeWord(config.wake_words[0]);
        }
      } catch (error) {
        console.error('Failed to load speech config:', error);
        // 使用默认唤醒词
      }
    };

    loadSpeechConfig();
  }, []);

  // 处理语音转录结果
  useEffect(() => {
    if (transcript && transcript.trim()) {
      setInput(transcript);
      resetTranscript();
      sleep();
    }
  }, [transcript, resetTranscript, sleep]);

  // 处理动画状态
  useEffect(() => {
    if (isListening || isWakeWordListening) {
      setIsAnimating(true);
    } else {
      setIsAnimating(false);
    }
  }, [isListening, isWakeWordListening]);

  const handleSend = async () => {
    if (!input.trim() || isLoading) return;

    const message = input.trim();
    setInput('');
    setIsLoading(true);

    try {
      await onSendMessage(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const getVoiceIcon = () => {
    if (isAwake && isListening) return <Volume2 className="h-4 w-4" />;
    if (isWakeWordListening) return <Mic className="h-4 w-4" />;
    return <MicOff className="h-4 w-4" />;
  };

  const getVoiceTooltip = () => {
    if (isAwake && isListening) return '正在听...';
    if (isWakeWordListening) return `等待唤醒词"${wakeWord}"`;
    return '语音助手';
  };

  // 自动滚动到底部
  useEffect(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
  }, [messages]);

  // 自动调整文本框高度
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 120)}px`;
    }
  }, [input]);

  return (
    <div className="flex flex-col h-full bg-background border-r">
      {/* 头部 */}
      <div className="p-4 border-b bg-card">
        <h2 className="text-lg font-semibold text-card-foreground">智能助手</h2>
        <p className="text-sm text-muted-foreground">代码编辑助手，支持语音交互</p>
      </div>

      {/* 消息列表 */}
      <ScrollArea ref={scrollAreaRef} className="flex-1 p-4">
        <div className="space-y-4">
          {messages.length === 0 && (
            <div className="text-center py-8">
              <Bot className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
              <p className="text-muted-foreground">
                你好！我是你的代码编辑助手。<br />
                你可以通过文字或语音与我交流。
              </p>
            </div>
          )}
          
          {messages.map((message) => (
            <div
              key={message.id}
              className={`flex gap-3 ${
                message.type === 'user' ? 'justify-end' : 'justify-start'
              }`}
            >
              {message.type === 'assistant' && (
                <Avatar className="h-8 w-8 mt-1">
                  <AvatarFallback className="bg-primary text-primary-foreground">
                    <Bot className="h-4 w-4" />
                  </AvatarFallback>
                </Avatar>
              )}
              
              <Card className={`max-w-[80%] ${
                message.type === 'user' 
                  ? 'bg-primary text-primary-foreground' 
                  : 'bg-muted'
              }`}>
                <CardContent className="p-3">
                  <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                  <span className="text-xs opacity-70 mt-1 block">
                    {message.timestamp.toLocaleTimeString()}
                  </span>
                </CardContent>
              </Card>

              {message.type === 'user' && (
                <Avatar className="h-8 w-8 mt-1">
                  <AvatarFallback className="bg-secondary text-secondary-foreground">
                    <User className="h-4 w-4" />
                  </AvatarFallback>
                </Avatar>
              )}
            </div>
          ))}

          {isLoading && (
            <div className="flex gap-3 justify-start">
              <Avatar className="h-8 w-8 mt-1">
                <AvatarFallback className="bg-primary text-primary-foreground">
                  <Bot className="h-4 w-4" />
                </AvatarFallback>
              </Avatar>
              <Card className="bg-muted">
                <CardContent className="p-3">
                  <div className="flex space-x-1">
                    <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                    <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                    <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </ScrollArea>

      {/* 输入框 */}
      <div className="p-4 border-t bg-card">
        {/* 语音状态提示 */}
        {(transcript || error) && (
          <div className="mb-2">
            {transcript && (
              <div className="bg-accent text-accent-foreground px-3 py-2 rounded-lg text-sm mb-1">
                转录中: {transcript}
              </div>
            )}
            {error && (
              <div className="bg-destructive text-destructive-foreground px-3 py-1 rounded-lg text-sm">
                {error}
              </div>
            )}
          </div>
        )}
        
        <div className="flex gap-2 items-end">
          <Textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="输入消息或说'小助手'唤醒语音..."
            className="min-h-[40px] max-h-[120px] resize-none"
            disabled={isLoading}
          />
          
          {/* 语音助手按钮 */}
          <Button
            onClick={sleep}
            disabled={!isWakeWordListening && !isAwake}
            size="icon"
            variant="outline"
            className={cn(
              "h-10 w-10 shrink-0 transition-all duration-300",
              isAnimating && "scale-110",
              isAwake && "bg-accent hover:bg-accent/90"
            )}
            title={getVoiceTooltip()}
          >
            {isAnimating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              getVoiceIcon()
            )}
          </Button>
          
          <Button
            onClick={handleSend}
            disabled={!input.trim() || isLoading}
            size="icon"
            className="h-10 w-10 shrink-0"
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
};