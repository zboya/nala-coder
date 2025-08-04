import { useState, useEffect, useRef, useCallback } from 'react';

interface SpeechRecognitionResult {
  transcript: string;
  confidence: number;
}

interface UseSpeechRecognitionOptions {
  wakeWord?: string;
  lang?: string;
  continuous?: boolean;
}

export const useSpeechRecognition = (options: UseSpeechRecognitionOptions = {}) => {
  const {
    wakeWord = '小助手',
    lang = 'zh-CN',
    continuous = true
  } = options;

  const [isListening, setIsListening] = useState(false);
  const [isWakeWordListening, setIsWakeWordListening] = useState(false);
  const [isAwake, setIsAwake] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [error, setError] = useState<string | null>(null);

  const recognitionRef = useRef<any>(null);
  const wakeWordRecognitionRef = useRef<any>(null);

  const startWakeWordListening = useCallback(() => {
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
      setError('浏览器不支持语音识别');
      return;
    }

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = true;
    recognition.interimResults = false;
    recognition.lang = lang;

    recognition.onstart = () => {
      setIsWakeWordListening(true);
      setError(null);
    };

    recognition.onresult = (event) => {
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          const text = result[0].transcript.trim();
          if (text.includes(wakeWord)) {
            setIsAwake(true);
            recognition.stop();
            break;
          }
        }
      }
    };

    recognition.onerror = (event) => {
      console.error('唤醒词识别错误:', event.error);
      setError(`唤醒词识别错误: ${event.error}`);
    };

    recognition.onend = () => {
      setIsWakeWordListening(false);
      if (!isAwake) {
        // 重新开始监听唤醒词
        setTimeout(() => {
          if (wakeWordRecognitionRef.current) {
            wakeWordRecognitionRef.current.start();
          }
        }, 1000);
      }
    };

    wakeWordRecognitionRef.current = recognition;
    recognition.start();
  }, [wakeWord, lang, isAwake]);

  const startListening = useCallback(() => {
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
      setError('浏览器不支持语音识别');
      return;
    }

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = continuous;
    recognition.interimResults = true;
    recognition.lang = lang;

    recognition.onstart = () => {
      setIsListening(true);
      setError(null);
    };

    recognition.onresult = (event) => {
      let finalTranscript = '';
      let interimTranscript = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          finalTranscript += result[0].transcript;
        } else {
          interimTranscript += result[0].transcript;
        }
      }

      setTranscript(finalTranscript || interimTranscript);
    };

    recognition.onerror = (event) => {
      console.error('语音识别错误:', event.error);
      setError(`语音识别错误: ${event.error}`);
      setIsListening(false);
    };

    recognition.onend = () => {
      setIsListening(false);
    };

    recognitionRef.current = recognition;
    recognition.start();
  }, [continuous, lang]);

  const stopListening = useCallback(() => {
    if (recognitionRef.current) {
      recognitionRef.current.stop();
    }
  }, []);

  const resetTranscript = useCallback(() => {
    setTranscript('');
  }, []);

  const sleep = useCallback(() => {
    setIsAwake(false);
    setTranscript('');
    if (recognitionRef.current) {
      recognitionRef.current.stop();
    }
    // 重新开始监听唤醒词
    setTimeout(startWakeWordListening, 1000);
  }, [startWakeWordListening]);

  useEffect(() => {
    startWakeWordListening();

    return () => {
      if (wakeWordRecognitionRef.current) {
        wakeWordRecognitionRef.current.stop();
      }
      if (recognitionRef.current) {
        recognitionRef.current.stop();
      }
    };
  }, []);

  useEffect(() => {
    if (isAwake && !isListening) {
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