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
  const [wakeWords, setWakeWords] = useState<string[]>(['小娜', '小助手']);
  const [configLoaded, setConfigLoaded] = useState(false);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 加载语音配置
  useEffect(() => {
    const loadSpeechConfig = async () => {
      try {
        const config = await getSpeechConfig();
        if (config.wake_words && config.wake_words.length > 0) {
          setWakeWords(config.wake_words);
          console.log('✅ [ChatBox] Loaded wake words from config:', config.wake_words);
        }
      } catch (error) {
        console.error('❌ [ChatBox] Failed to load speech config:', error);
        // 使用默认唤醒词
      } finally {
        setConfigLoaded(true);
      }
    };

    loadSpeechConfig();
  }, []);

  // 语音识别功能
  const {
    isListening,
    isWakeWordListening,
    isAwake,
    transcript,
    error,
    sleep,
    resetTranscript
  } = useSpeechRecognition({
    wakeWords: configLoaded ? wakeWords : ['小娜', '小助手']
  });

  // 处理语音转录结果
  useEffect(() => {
    // 只有当语音识别完成（不再监听）且有有效内容时才处理
    if (transcript && transcript.trim() && !isListening && isAwake && !isLoading) {
      console.log('📨 [ChatBox] Processing completed speech transcript:', transcript);
      const message = transcript.trim();

      // 直接发送消息给后端
      const sendSpeechMessage = async () => {
        setIsLoading(true);
        try {
          await onSendMessage(message);
          console.log('✅ [ChatBox] Speech message sent successfully:', message);
        } catch (error) {
          console.error('❌ [ChatBox] Failed to send speech message:', error);
          // 如果发送失败，将内容放到输入框让用户手动发送
          setInput(message);
        } finally {
          setIsLoading(false);
        }
      };

      // 清理语音识别状态
      resetTranscript();
      sleep();

      // 发送消息
      sendSpeechMessage();
    }
  }, [transcript, isListening, isAwake, isLoading, onSendMessage, resetTranscript, sleep]);

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
    if (isWakeWordListening) {
      const displayWords = wakeWords.length > 2
        ? `${wakeWords.slice(0, 2).join('", "')}...`
        : wakeWords.join('", "');
      return `等待唤醒词"${displayWords}"`;
    }
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
              className={`flex gap-3 ${message.type === 'user' ? 'justify-end' : 'justify-start'
                }`}
            >
              {message.type === 'assistant' && (
                <Avatar className="h-8 w-8 mt-1">
                  <AvatarFallback className="bg-primary text-primary-foreground">
                    <Bot className="h-4 w-4" />
                  </AvatarFallback>
                </Avatar>
              )}

              <Card className={`max-w-[80%] ${message.type === 'user'
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
            placeholder={`输入消息或说'${wakeWords[0]}'唤醒语音...`}
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
              "h-10 w-10 shrink-0 transition-all duration-300 relative border-2",
              isAnimating && "scale-110",
              isAwake && isListening && "bg-green-100 border-green-400 hover:bg-green-200 dark:bg-green-900/30 dark:border-green-500",
              isWakeWordListening && !isAwake && "bg-blue-100 border-blue-400 hover:bg-blue-200 dark:bg-blue-900/30 dark:border-blue-500",
              !isWakeWordListening && !isAwake && "bg-gray-100 border-gray-300 hover:bg-gray-200 dark:bg-gray-800 dark:border-gray-600"
            )}
            title={getVoiceTooltip()}
          >
            {/* 底层话筒标识 - 居中显示 */}
            <div className={cn(
              "absolute inset-0 flex items-center justify-center transition-colors duration-300",
              isAwake && isListening && "text-green-400 opacity-40",
              isWakeWordListening && !isAwake && "text-blue-400 opacity-40",
              !isWakeWordListening && !isAwake && "text-gray-400 opacity-30"
            )}>
              <Mic className="h-5 w-5" />
            </div>
            {/* 前景状态图标 */}
            <div className={cn(
              "relative z-10 transition-colors duration-300",
              isAwake && isListening && "text-green-600 dark:text-green-400",
              isWakeWordListening && !isAwake && "text-blue-600 dark:text-blue-400",
              !isWakeWordListening && !isAwake && "text-gray-500 dark:text-gray-400"
            )}>
              {isAnimating ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                getVoiceIcon()
              )}
            </div>
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