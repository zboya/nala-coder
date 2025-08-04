// API服务层
export interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

export interface ChatStreamResponse {
  session_id: string;
  response: string;
  finished: boolean;
  usage?: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
  metadata?: Record<string, any>;
}

export interface FileNode {
  id: string;
  name: string;
  type: 'file' | 'folder';
  children?: FileNode[];
  path: string;
  extension?: string;
  size?: number;
  mod_time?: string;
}

export interface FileTreeResponse {
  tree: {
    name: string;
    path: string;
    type: 'directory' | 'file';
    size: number;
    mod_time: string;
    children?: any[];
  };
  path: string;
}

export interface FileContentResponse {
  path: string;
  content: string;
  language: string;
  size: number;
  mod_time: string;
}

export interface SpeechConfig {
  enabled: boolean;
  wake_words: string[];
  wake_timeout: number;
  language: string;
}

// 流式聊天API
export async function streamChat(
  message: string,
  sessionId?: string,
  onChunk?: (chunk: ChatStreamResponse) => void
): Promise<string> {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message,
      session_id: sessionId,
      stream: true,
    }),
  });

  if (!response.ok) {
    throw new Error(`Chat API error: ${response.status}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }

  let fullResponse = '';
  const decoder = new TextDecoder();

  try {
    let buffer = '';
    
    while (true) {
      const { done, value } = await reader.read();
      
      if (done) break;
      
      const text = decoder.decode(value, { stream: true });
      buffer += text;
      
      // 按双换行符分割事件
      const events = buffer.split('\n\n');
      // 保留最后一个可能不完整的事件
      buffer = events.pop() || '';
      
      for (const event of events) {
        if (!event.trim()) continue;
        
        const lines = event.split('\n');
        let eventType = '';
        let eventData = '';
        
        for (const line of lines) {
          if (line.startsWith('event:')) {
            eventType = line.substring(6).trim();
          } else if (line.startsWith('data:')) {
            eventData = line.substring(5).trim();
          }
        }
        
        // 处理消息事件
        if (eventType === 'message' && eventData) {
          try {
            const data: ChatStreamResponse = JSON.parse(eventData);
            fullResponse += data.response;
            
            if (onChunk) {
              onChunk(data);
            }
            
            if (data.finished) {
              return fullResponse;
            }
          } catch (e) {
            console.warn('Failed to parse SSE chunk:', e, 'Data:', eventData);
          }
        }
        // 处理结束事件
        else if (eventType === 'end') {
          return fullResponse;
        }
      }
    }
    
    // 处理缓冲区中剩余的数据
    if (buffer.trim()) {
      const lines = buffer.split('\n');
      let eventType = '';
      let eventData = '';
      
      for (const line of lines) {
        if (line.startsWith('event:')) {
          eventType = line.substring(6).trim();
        } else if (line.startsWith('data:')) {
          eventData = line.substring(5).trim();
        }
      }
      
      if (eventType === 'message' && eventData) {
        try {
          const data: ChatStreamResponse = JSON.parse(eventData);
          fullResponse += data.response;
          
          if (onChunk) {
            onChunk(data);
          }
        } catch (e) {
          console.warn('Failed to parse final SSE chunk:', e);
        }
      }
    }
  } finally {
    reader.releaseLock();
  }

  return fullResponse;
}

// 获取文件树
export async function getFileTree(path?: string, depth?: number): Promise<FileNode[]> {
  const params = new URLSearchParams();
  if (path) params.append('path', path);
  if (depth) params.append('depth', depth.toString());

  const response = await fetch(`/api/files/tree?${params}`);
  
  if (!response.ok) {
    throw new Error(`File tree API error: ${response.status}`);
  }

  const data: FileTreeResponse = await response.json();
  
  // 转换API响应格式为组件所需格式
  const convertToFileNode = (item: any, parentPath = ''): FileNode => {
    const fullPath = item.path || `${parentPath}/${item.name}`;
    const extension = item.type === 'file' ? item.name.split('.').pop() : undefined;
    
    return {
      id: fullPath, // 使用路径作为ID
      name: item.name,
      type: item.type === 'directory' ? 'folder' : 'file',
      path: fullPath,
      extension,
      size: item.size,
      children: item.children ? item.children.map((child: any) => convertToFileNode(child, fullPath)) : undefined,
    };
  };

  if (data.tree.children) {
    return data.tree.children.map(item => convertToFileNode(item));
  } else {
    return [convertToFileNode(data.tree)];
  }
}

// 获取文件内容
export async function getFileContent(path: string): Promise<FileContentResponse> {
  const params = new URLSearchParams();
  params.append('path', path);

  const response = await fetch(`/api/files/content?${params}`);
  
  if (!response.ok) {
    throw new Error(`File content API error: ${response.status}`);
  }

  return await response.json();
}

// 获取语音配置
export async function getSpeechConfig(): Promise<SpeechConfig> {
  const response = await fetch('/api/speech/config');
  
  if (!response.ok) {
    throw new Error(`Speech config API error: ${response.status}`);
  }

  return await response.json();
}

// 获取会话详情
export async function getSession(sessionId: string) {
  const response = await fetch(`/api/session/${sessionId}`);
  
  if (!response.ok) {
    throw new Error(`Session API error: ${response.status}`);
  }

  return await response.json();
}

// 获取会话列表
export async function getSessions() {
  const response = await fetch('/api/sessions');
  
  if (!response.ok) {
    throw new Error(`Sessions API error: ${response.status}`);
  }

  return await response.json();
}