import { useState, useEffect, useRef, useCallback } from 'react';

interface UseSpeechRecognitionOptions {
  wakeWords?: string | string[];
  lang?: string;
  continuous?: boolean;
}

export const useSpeechRecognition = (options: UseSpeechRecognitionOptions = {}) => {
  const {
    wakeWords = ['小娜', '小助手'],
    lang = 'zh-CN',
    continuous = true
  } = options;

  // 将唤醒词统一转换为数组格式
  const wakeWordList = Array.isArray(wakeWords) ? wakeWords : [wakeWords];

  const [isListening, setIsListening] = useState(false);
  const [isWakeWordListening, setIsWakeWordListening] = useState(false);
  const [isAwake, setIsAwake] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [error, setError] = useState<string | null>(null);

  const recognitionRef = useRef<any>(null);
  const wakeWordRecognitionRef = useRef<any>(null);
  const isStartingRef = useRef(false);
  const wakeWordStartingRef = useRef(false);
  const isAwakeRef = useRef(false);
  const isWakeWordListeningRef = useRef(false);
  const silenceTimerRef = useRef<NodeJS.Timeout | null>(null);
  const lastSpeechTimeRef = useRef<number>(0);
  const isProcessingRef = useRef(false);

  // 同步 ref 和 state
  useEffect(() => {
    isAwakeRef.current = isAwake;
  }, [isAwake]);

  useEffect(() => {
    isWakeWordListeningRef.current = isWakeWordListening;
  }, [isWakeWordListening]);

  const startWakeWordListening = useCallback(() => {
    console.log('🎤 [Wake Word] Attempting to start wake word listening...');

    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
      console.error('❌ [Wake Word] Browser does not support speech recognition');
      setError('浏览器不支持语音识别');
      return;
    }

    // 防止重复启动
    if (wakeWordStartingRef.current || isWakeWordListeningRef.current) {
      console.warn('⚠️ [Wake Word] Already starting or listening, skipping...');
      return;
    }

    // 停止之前的识别
    if (wakeWordRecognitionRef.current) {
      console.log('🛑 [Wake Word] Stopping previous recognition...');
      try {
        wakeWordRecognitionRef.current.stop();
      } catch (e) {
        console.warn('⚠️ [Wake Word] Error stopping previous recognition:', e);
      }
    }

    wakeWordStartingRef.current = true;
    console.log('🚀 [Wake Word] Starting wake word recognition...');

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = true;
    recognition.interimResults = false;
    recognition.lang = lang;

    recognition.onstart = () => {
      console.log('✅ [Wake Word] Wake word recognition started successfully');
      setIsWakeWordListening(true);
      setError(null);
      wakeWordStartingRef.current = false;
    };

    recognition.onresult = (event) => {
      console.log('🎯 [Wake Word] Recognition result received');
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          const text = result[0].transcript.trim();
          console.log(`📝 [Wake Word] Final transcript: "${text}"`);

          // 检查是否包含任何一个唤醒词
          const detectedWakeWord = wakeWordList.find(word => text.includes(word));
          if (detectedWakeWord) {
            console.log(`🎉 [Wake Word] Wake word "${detectedWakeWord}" detected! Activating assistant...`);
            console.log(`📋 [Wake Word] Available wake words: [${wakeWordList.join(', ')}]`);
            setIsAwake(true);
            try {
              recognition.stop();
            } catch (e) {
              console.warn('⚠️ [Wake Word] Error stopping recognition after wake word:', e);
            }
            break;
          } else {
            console.log(`🔍 [Wake Word] No wake word found in: "${text}"`);
            console.log(`📋 [Wake Word] Looking for: [${wakeWordList.join(', ')}]`);
          }
        }
      }
    };

    recognition.onerror = (event) => {
      console.error('❌ [Wake Word] Recognition error:', event.error);
      // no-speech 错误是正常的静默状态，不需要显示错误提示
      if (event.error !== 'no-speech') {
        setError(`唤醒词识别错误: ${event.error}`);
      }
      wakeWordStartingRef.current = false;
    };

    recognition.onend = () => {
      console.log('🏁 [Wake Word] Wake word recognition ended');
      setIsWakeWordListening(false);
      wakeWordStartingRef.current = false;

      // 使用 setTimeout 确保状态更新完成后再检查
      setTimeout(() => {
        if (!isAwakeRef.current && !wakeWordStartingRef.current) {
          console.log('💤 [Wake Word] Assistant not awake, restarting wake word listening in 1s...');
          // 重新开始监听唤醒词，添加延迟避免重复启动
          setTimeout(() => {
            if (!wakeWordStartingRef.current && !isAwakeRef.current) {
              startWakeWordListening();
            }
          }, 1000);
        } else {
          console.log('🎉 [Wake Word] Assistant is awake, not restarting wake word listening');
        }
      }, 100);
    };

    wakeWordRecognitionRef.current = recognition;

    try {
      recognition.start();
    } catch (error) {
      console.error('❌ [Wake Word] Failed to start wake word recognition:', error);
      setError('启动唤醒词识别失败');
      wakeWordStartingRef.current = false;
    }
  }, [wakeWordList, lang]);

  const startListening = useCallback(() => {
    console.log('🎤 [Speech] Attempting to start speech recognition...');

    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
      console.error('❌ [Speech] Browser does not support speech recognition');
      setError('浏览器不支持语音识别');
      return;
    }

    // 防止重复启动
    if (isStartingRef.current || isListening) {
      console.warn('⚠️ [Speech] Already starting or listening, skipping...');
      return;
    }

    // 停止之前的识别
    if (recognitionRef.current) {
      console.log('🛑 [Speech] Stopping previous recognition...');
      try {
        recognitionRef.current.stop();
      } catch (e) {
        console.warn('⚠️ [Speech] Error stopping previous recognition:', e);
      }
    }

    isStartingRef.current = true;
    console.log('🚀 [Speech] Starting speech recognition...');

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = continuous;
    recognition.interimResults = true;
    recognition.lang = lang;

    recognition.onstart = () => {
      console.log('✅ [Speech] Speech recognition started successfully');
      setIsListening(true);
      setError(null);
      isStartingRef.current = false;
    };

    recognition.onresult = (event) => {
      console.log('🎯 [Speech] Recognition result received');
      let finalTranscript = '';
      let interimTranscript = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          finalTranscript += result[0].transcript;
          console.log(`📝 [Speech] Final transcript: "${result[0].transcript}"`);
        } else {
          interimTranscript += result[0].transcript;
          console.log(`📝 [Speech] Interim transcript: "${result[0].transcript}"`);
        }
      }

      const currentTranscript = finalTranscript || interimTranscript;
      setTranscript(currentTranscript);
      console.log(`🔄 [Speech] Updated transcript: "${currentTranscript}"`);

      // 更新最后说话时间
      lastSpeechTimeRef.current = Date.now();

      // 清除之前的静音计时器
      if (silenceTimerRef.current) {
        clearTimeout(silenceTimerRef.current);
        console.log('⏰ [Speech] Cleared previous silence timer');
      }

      // 如果有实际内容，设置静音检测计时器
      if (currentTranscript.trim()) {
        console.log('⏰ [Speech] Setting silence timer for 3 seconds...');
        silenceTimerRef.current = setTimeout(() => {
          const timeSinceLastSpeech = Date.now() - lastSpeechTimeRef.current;
          console.log(`🔇 [Speech] Silence detected. Time since last speech: ${timeSinceLastSpeech}ms`);

          if (timeSinceLastSpeech >= 2500) {
            console.log('✅ [Speech] Speech completed, processing transcript...');
            isProcessingRef.current = true;

            // 停止语音识别
            if (recognitionRef.current) {
              try {
                recognitionRef.current.stop();
              } catch (e) {
                console.warn('⚠️ [Speech] Error stopping recognition after silence:', e);
              }
            }
          } else {
            console.log('⏰ [Speech] Not enough silence time, continuing to listen...');
          }
        }, 3000);
      }
    };

    recognition.onerror = (event) => {
      console.error('❌ [Speech] Recognition error:', event.error);
      // no-speech 错误是正常的静默状态，不需要显示错误提示
      if (event.error !== 'no-speech') {
        setError(`语音识别错误: ${event.error}`);
      }
      setIsListening(false);
      isStartingRef.current = false;
    };

    recognition.onend = () => {
      console.log('🏁 [Speech] Speech recognition ended');
      setIsListening(false);
      isStartingRef.current = false;

      // 清除静音计时器
      if (silenceTimerRef.current) {
        clearTimeout(silenceTimerRef.current);
        silenceTimerRef.current = null;
        console.log('⏰ [Speech] Cleared silence timer on recognition end');
      }

      // 如果是因为处理完成而结束，不需要重新启动
      if (isProcessingRef.current) {
        console.log('✅ [Speech] Recognition ended after processing, transcript ready');
        isProcessingRef.current = false;
      }
    };

    recognitionRef.current = recognition;

    try {
      recognition.start();
    } catch (error) {
      console.error('❌ [Speech] Failed to start speech recognition:', error);
      setError('启动语音识别失败');
      isStartingRef.current = false;
    }
  }, [continuous, lang, isListening]);

  const stopListening = useCallback(() => {
    console.log('🛑 [Speech] Stopping speech recognition...');
    if (recognitionRef.current) {
      try {
        recognitionRef.current.stop();
        console.log('✅ [Speech] Speech recognition stopped successfully');
      } catch (e) {
        console.warn('⚠️ [Speech] Error stopping recognition:', e);
      }
    }
  }, []);

  const resetTranscript = useCallback(() => {
    // 如果正在处理语音输入，不允许重置
    if (isListening && !isProcessingRef.current) {
      console.log('⚠️ [Speech] Cannot reset transcript while actively listening');
      return;
    }
    console.log('🔄 [Speech] Resetting transcript');
    setTranscript('');
  }, [isListening]);

  const sleep = useCallback(() => {
    // 如果正在处理语音输入且还没有完成，延迟进入睡眠模式
    if (isListening && !isProcessingRef.current) {
      console.log('⚠️ [Assistant] Cannot sleep while actively listening, will retry in 2s...');
      setTimeout(() => {
        if (!isListening || isProcessingRef.current) {
          sleep();
        }
      }, 2000);
      return;
    }

    console.log('💤 [Assistant] Going to sleep mode...');
    setIsAwake(false);
    setTranscript('');

    // 清除静音计时器
    if (silenceTimerRef.current) {
      clearTimeout(silenceTimerRef.current);
      silenceTimerRef.current = null;
      console.log('⏰ [Assistant] Cleared silence timer on sleep');
    }

    // 停止当前的语音识别
    if (recognitionRef.current) {
      console.log('🛑 [Assistant] Stopping current speech recognition...');
      try {
        recognitionRef.current.stop();
      } catch (e) {
        console.warn('⚠️ [Assistant] Error stopping speech recognition:', e);
      }
    }

    // 重新开始监听唤醒词，添加延迟和状态检查
    console.log('🔄 [Assistant] Restarting wake word listening in 1s...');
    setTimeout(() => {
      if (!wakeWordStartingRef.current && !isWakeWordListeningRef.current) {
        startWakeWordListening();
      }
    }, 1000);
  }, [startWakeWordListening, isListening]);

  useEffect(() => {
    console.log('🎬 [Init] Initializing speech recognition hook...');
    // 初始化时启动唤醒词监听
    if (!wakeWordStartingRef.current && !isWakeWordListening) {
      console.log('🎤 [Init] Starting initial wake word listening...');
      startWakeWordListening();
    }

    return () => {
      console.log('🧹 [Cleanup] Cleaning up speech recognition resources...');
      // 清理资源
      if (wakeWordRecognitionRef.current) {
        try {
          wakeWordRecognitionRef.current.stop();
          console.log('✅ [Cleanup] Wake word recognition stopped');
        } catch (e) {
          console.warn('⚠️ [Cleanup] Error stopping wake word recognition:', e);
        }
      }
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
          console.log('✅ [Cleanup] Speech recognition stopped');
        } catch (e) {
          console.warn('⚠️ [Cleanup] Error stopping speech recognition:', e);
        }
      }

      // 清理计时器
      if (silenceTimerRef.current) {
        clearTimeout(silenceTimerRef.current);
        console.log('✅ [Cleanup] Silence timer cleared');
      }
    };
  }, []);

  useEffect(() => {
    if (isAwake && !isListening && !isStartingRef.current) {
      console.log('🎉 [Assistant] Assistant is awake, starting speech recognition...');
      startListening();
    }
  }, [isAwake, isListening, startListening]);

  return {
    isListening,
    isWakeWordListening,
    isAwake,
    transcript,
    error,
    startListening,
    stopListening,
    resetTranscript,
    sleep
  };
};