import { useState, useEffect } from 'react';
import { Mic, MicOff, Volume2, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition';
import { getSpeechConfig } from '@/services/api';
import { cn } from '@/lib/utils';

interface VoiceAssistantProps {
  onTranscript: (text: string) => void;
}

export const VoiceAssistant = ({ onTranscript }: VoiceAssistantProps) => {
  const [isAnimating, setIsAnimating] = useState(false);
  const [wakeWords, setWakeWords] = useState<string[]>(['小助手', '你好助手', 'hey assistant']);
  
  // 加载语音配置
  useEffect(() => {
    const loadSpeechConfig = async () => {
      try {
        const config = await getSpeechConfig();
        if (config.wake_words && config.wake_words.length > 0) {
          setWakeWords(config.wake_words);
        }
      } catch (error) {
        console.error('Failed to load speech config:', error);
        // 使用默认唤醒词
      }
    };

    loadSpeechConfig();
  }, []);

  const {
    isListening,
    isWakeWordListening,
    isAwake,
    transcript,
    error,
    sleep,
    resetTranscript
  } = useSpeechRecognition({ wakeWords });

  useEffect(() => {
    if (transcript && transcript.trim()) {
      onTranscript(transcript);
      resetTranscript();
      sleep();
    }
  }, [transcript, onTranscript, resetTranscript, sleep]);

  useEffect(() => {
    if (isListening || isWakeWordListening) {
      setIsAnimating(true);
    } else {
      setIsAnimating(false);
    }
  }, [isListening, isWakeWordListening]);

  const getVoiceState = () => {
    if (isAwake && isListening) return 'listening';
    if (isWakeWordListening) return 'wake-listening';
    return 'idle';
  };

  const getStateText = () => {
    if (isAwake && isListening) return '正在听...';
    if (isWakeWordListening) {
      const displayWords = wakeWords.length > 2 
        ? `${wakeWords.slice(0, 2).join('", "')}...` 
        : wakeWords.join('", "');
      return `等待唤醒词"${displayWords}"`;
    }
    return '语音助手';
  };

  const getStateIcon = () => {
    if (isAwake && isListening) return <Volume2 className="h-6 w-6" />;
    if (isWakeWordListening) return <Mic className="h-6 w-6" />;
    return <MicOff className="h-6 w-6" />;
  };

  return (
    <div className="fixed bottom-6 right-6 z-50">
      <div className="relative">
        {/* 主按钮 */}
        <Button
          size="lg"
          className={cn(
            "h-16 w-16 rounded-full shadow-lg transition-all duration-300",
            "bg-primary hover:bg-primary/90",
            isAnimating && "scale-110",
            isAwake && "bg-accent hover:bg-accent/90"
          )}
          onClick={sleep}
          disabled={!isWakeWordListening && !isAwake}
        >
          {isAnimating ? (
            <Loader2 className="h-6 w-6 animate-spin" />
          ) : (
            getStateIcon()
          )}
        </Button>

        {/* 动画圆环 */}
        {isAnimating && (
          <div className="absolute inset-0 rounded-full border-2 border-primary/30 animate-ping" />
        )}

        {/* 状态文本 */}
        <div className="absolute -top-12 left-1/2 transform -translate-x-1/2 whitespace-nowrap">
          <div className="bg-popover text-popover-foreground px-3 py-1 rounded-lg text-sm shadow-md">
            {getStateText()}
          </div>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="absolute -top-20 left-1/2 transform -translate-x-1/2 whitespace-nowrap">
            <div className="bg-destructive text-destructive-foreground px-3 py-1 rounded-lg text-sm shadow-md max-w-48 truncate">
              {error}
            </div>
          </div>
        )}

        {/* 实时转录显示 */}
        {transcript && (
          <div className="absolute -top-24 left-1/2 transform -translate-x-1/2 max-w-64">
            <div className="bg-accent text-accent-foreground px-3 py-2 rounded-lg text-sm shadow-md">
              {transcript}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};