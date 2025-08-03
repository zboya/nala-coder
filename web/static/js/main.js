// DOM elements
const chatContainer = document.getElementById('chatContainer');
const messageInput = document.getElementById('messageInput');
const sendButton = document.getElementById('sendButton');
const status = document.getElementById('status');
const sessionInfo = document.getElementById('sessionInfo');
const streamToggle = document.getElementById('streamToggle');
const sidebarToggle = document.getElementById('sidebarToggle');
const sidebar = document.getElementById('sidebar');
const newSessionBtn = document.getElementById('newSessionBtn');
const sessionsList = document.getElementById('sessionsList');
const toolsList = document.getElementById('toolsList');


// State
let sessionId = '';
let isStreaming = true;
let currentStreamingMessage = null;
let sessions = [];
let tools = [];

// Voice manager instance
let voiceManager = null;




// Initialize app
document.addEventListener('DOMContentLoaded', function() {
    loadTools();

    setupEventListeners();
    // Initialize voice manager
    if (window.VoiceManager) {
        voiceManager = new window.VoiceManager();
        voiceManager.init({
            messageInput: messageInput,
            sendMessageCallback: sendMessage
        });
    }
    messageInput.focus();
    
    // Auto-resize textarea
    messageInput.addEventListener('input', function() {
        this.style.height = 'auto';
        this.style.height = Math.min(this.scrollHeight, 120) + 'px';
    });
});

// Setup event listeners
function setupEventListeners() {
    sendButton.addEventListener('click', sendMessage);
    
    messageInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    streamToggle.addEventListener('click', () => {
        isStreaming = !isStreaming;
        streamToggle.classList.toggle('active', isStreaming);
        streamToggle.textContent = isStreaming ? '流式响应' : '普通响应';
    });

    sidebarToggle.addEventListener('click', () => {
        const isVisible = sidebar.style.display !== 'none';
        sidebar.style.display = isVisible ? 'none' : 'block';
        sidebarToggle.classList.toggle('active', !isVisible);
        sidebarToggle.textContent = isVisible ? '显示侧边栏' : '隐藏侧边栏';
    });

    newSessionBtn.addEventListener('click', startNewSession);
    
    // Voice button event listener is handled by voice manager
}

// Send message function
async function sendMessage() {
    const message = messageInput.value.trim();
    if (!message) return;

    // Disable input
    setInputEnabled(false);
    status.textContent = '发送中...';

    // Add user message to chat
    addMessage('user', message);
    messageInput.value = '';
    messageInput.style.height = 'auto';

    try {
        if (isStreaming) {
            await sendStreamMessage(message);
        } else {
            await sendNormalMessage(message);
        }
    } catch (error) {
        addMessage('ai', `网络错误: ${error.message}`);
        status.textContent = '网络错误';
    }

    // Re-enable input
    setInputEnabled(true);
    messageInput.focus();
}

// Send normal message
async function sendNormalMessage(message) {
    const response = await fetch('/api/chat', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            message: message,
            session_id: sessionId,
            stream: false
        })
    });

    const data = await response.json();

    if (response.ok) {
        sessionId = data.session_id;
        updateSessionInfo();
        addMessage('ai', data.response, data.usage);
        status.textContent = `就绪 (使用 ${data.usage.total_tokens} tokens)`;
    } else {
        addMessage('ai', `错误: ${data.error}`);
        status.textContent = '发生错误';
    }
}

// Send stream message
async function sendStreamMessage(message) {
    const response = await fetch('/api/chat/stream', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            message: message,
            session_id: sessionId,
            stream: true
        })
    });

    if (!response.ok) {
        const data = await response.json();
        addMessage('ai', `错误: ${data.error}`);
        status.textContent = '发生错误';
        return;
    }

    // Create streaming message element
    currentStreamingMessage = addMessage('ai', '', null, true);
    const contentDiv = currentStreamingMessage.querySelector('.message-content');
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let fullResponse = ''; // 累积完整响应

    status.textContent = '接收响应中...';

    while (true) {
        const { value, done } = await reader.read();
        
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop(); // Keep incomplete line in buffer

        for (const line of lines) {
            // 处理标准SSE格式: event: 和 data: 行
            if (line.startsWith('data:')) {
                const data = line.slice(5).trim();
                
                if (data === '[DONE]' || data === '<nil>') {
                    finishStreaming();
                    return;
                }
                
                try {
                    const eventData = JSON.parse(data);
                    
                    // 检查是否有响应内容（包括空字符串）
                    if (eventData.hasOwnProperty('response')) {
                        // 累积响应内容
                        fullResponse += eventData.response;
                        contentDiv.innerHTML = '<strong>AI:</strong> ' + escapeHtml(fullResponse);
                        chatContainer.scrollTop = chatContainer.scrollHeight;
                    }
                    
                    // 检查是否完成
                    if (eventData.finished) {
                        sessionId = eventData.session_id;
                        updateSessionInfo();
                        finishStreaming(eventData.usage);
                        return;
                    }
                } catch (e) {
                    console.warn('Failed to parse SSE data:', data, e);
                }
            }
            // 处理event行
            else if (line.startsWith('event: ')) {
                const eventType = line.slice(7).trim();
                if (eventType === 'end') {
                    finishStreaming();
                    return;
                }
            }
        }
    }
}

// Finish streaming
function finishStreaming(usage = null) {
    if (currentStreamingMessage) {
        currentStreamingMessage.classList.remove('streaming');
        if (usage) {
            const metaDiv = currentStreamingMessage.querySelector('.message-meta');
            metaDiv.textContent = `完成 - 使用 ${usage.total_tokens} tokens`;
            status.textContent = `就绪 (使用 ${usage.total_tokens} tokens)`;
        } else {
            status.textContent = '就绪';
        }
        currentStreamingMessage = null;
    }
}

// Add message to chat
function addMessage(sender, content, usage = null, streaming = false) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${sender}-message${streaming ? ' streaming' : ''}`;
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    contentDiv.innerHTML = `<strong>${sender === 'user' ? '你' : 'AI'}:</strong> ${escapeHtml(content)}`;
    
    const metaDiv = document.createElement('div');
    metaDiv.className = 'message-meta';
    
    if (streaming) {
        metaDiv.textContent = '输入中...';
    } else if (sender === 'user') {
        metaDiv.textContent = new Date().toLocaleTimeString();
    } else if (usage) {
        metaDiv.textContent = `完成 - 使用 ${usage.total_tokens} tokens`;
    } else {
        metaDiv.textContent = '完成';
    }
    
    messageDiv.appendChild(contentDiv);
    messageDiv.appendChild(metaDiv);
    chatContainer.appendChild(messageDiv);
    chatContainer.scrollTop = chatContainer.scrollHeight;
    
    return messageDiv;
}

// Set input enabled/disabled
function setInputEnabled(enabled) {
    messageInput.disabled = !enabled;
    sendButton.disabled = !enabled;
    if (voiceManager) {
        voiceManager.setVoiceInputEnabled(enabled);
    }
}

// Start new session
function startNewSession() {
    sessionId = '';
    sessionInfo.textContent = '新会话';
    chatContainer.innerHTML = `<div class="message system-message">
    <div class="message-content">
        <strong>🤖 NaLa Coder</strong>
        <br>你好！我是你的AI编程助手。我可以帮助你：
        <br>• 🛠️ 各种开发任务　• 🔍 搜索代码　• 💻 执行系统命令
        <br>今天我可以为你做些什么？
    </div>
    <div class="message-meta">系统消息</div>
</div>`;
    messageInput.focus();
}

// Update session info
function updateSessionInfo() {
    if (sessionId) {
        sessionInfo.textContent = `会话: ${sessionId.substring(0, 8)}...`;
    }
}

// Load available tools
async function loadTools() {
    try {
        const response = await fetch('/api/tools');
        const data = await response.json();
        
        if (response.ok && data.tools) {
            tools = data.tools;
            renderTools();
        }
    } catch (error) {
        console.warn('Failed to load tools:', error);
        toolsList.innerHTML = '<div class="tool-item">加载工具失败</div>';
    }
}



// Render tools list
function renderTools() {
    if (tools.length === 0) {
        toolsList.innerHTML = '<div class="tool-item">暂无可用工具</div>';
        return;
    }

    toolsList.innerHTML = tools.map(tool => `
        <div class="tool-item">
            <div class="tool-name">${tool.name}</div>
            <div class="tool-desc">${tool.description}</div>
        </div>
    `).join('');
}

// Utility function to escape HTML
function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, function(m) { return map[m]; });
}

// Health check
async function checkHealth() {
    try {
        const response = await fetch('/api/health');
        const data = await response.json();
        console.log('Health check:', data);
    } catch (error) {
        console.warn('Health check failed:', error);
    }
}

// Check health on load
checkHealth();

