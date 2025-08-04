# 语音助手多关键词唤醒功能

## 功能说明

语音助手现在支持多个关键词唤醒，用户可以使用任意一个配置的唤醒词来激活语音助手。

## 默认唤醒词

系统默认支持以下唤醒词：
- `小娜`
- `小助手` 

## 使用方法

### 1. 基本使用（使用默认唤醒词）

```typescript
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition';

const MyComponent = () => {
  const { isAwake, transcript, isWakeWordListening } = useSpeechRecognition();
  
  return (
    <div>
      <p>状态: {isWakeWordListening ? '等待唤醒' : isAwake ? '已唤醒' : '休眠'}</p>
      <p>识别内容: {transcript}</p>
    </div>
  );
};
```

### 2. 自定义唤醒词

```typescript
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition';

const MyComponent = () => {
  // 使用自定义唤醒词数组
  const customWakeWords = ['小爱', '小度', 'hello assistant', '语音助手'];
  
  const { isAwake, transcript, isWakeWordListening } = useSpeechRecognition({
    wakeWords: customWakeWords
  });
  
  return (
    <div>
      <p>等待唤醒词: {customWakeWords.join(', ')}</p>
      <p>状态: {isWakeWordListening ? '等待唤醒' : isAwake ? '已唤醒' : '休眠'}</p>
    </div>
  );
};
```

### 3. 单个唤醒词（向后兼容）

```typescript
import { useSpeechRecognition } from '@/hooks/useSpeechRecognition';

const MyComponent = () => {
  // 仍然支持单个字符串作为唤醒词
  const { isAwake, transcript } = useSpeechRecognition({
    wakeWords: '小助手'
  });
  
  return (
    <div>
      <p>状态: {isAwake ? '已唤醒' : '等待唤醒'}</p>
    </div>
  );
};
```

## 控制台日志

启用了详细的控制台日志，方便调试：

- `🎤 [Wake Word]` - 唤醒词相关日志
- `🎤 [Speech]` - 语音识别相关日志  
- `🎉 [Assistant]` - 助手状态变化日志
- `🎬 [Init]` - 初始化日志
- `🧹 [Cleanup]` - 清理日志

## 工作流程

1. **初始化**: 系统启动时开始监听唤醒词
2. **唤醒检测**: 检测到任意一个配置的唤醒词时激活助手
3. **语音识别**: 助手激活后开始正式的语音识别
4. **处理完成**: 识别完成后助手进入休眠状态，重新开始监听唤醒词

## 注意事项

- 唤醒词检测使用包含匹配（`text.includes(wakeWord)`）
- 支持中英文混合唤醒词
- 系统会自动处理重复启动和错误恢复
- 所有语音识别功能都有详细的日志输出，便于调试