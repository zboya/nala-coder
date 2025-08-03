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
const voiceButton = document.getElementById('voiceButton');
const voiceStatus = document.getElementById('voiceStatus');
const voiceIndicator = document.getElementById('voiceIndicator');
const voiceText = document.getElementById('voiceText');

// State
let sessionId = '';
let isStreaming = true;
let currentStreamingMessage = null;
let sessions = [];
let tools = [];

// Voice related state
let isVoiceSupported = false;
let wakeWordRecognition = null;
let voiceInputRecognition = null;
let isWakeListening = false;
let isVoiceInputActive = false;
let voiceTimeoutId = null;

// Voice configuration (will be loaded from server)
let WAKE_WORDS = ['小助手', '助手', 'hello', 'hey']; // 默认值
let WAKE_TIMEOUT = 30000; // 30 seconds
let VOICE_LANG = 'zh-CN';

// Initialize app
document.addEventListener('DOMContentLoaded', function() {
    loadTools();
    loadVoiceConfig();
    setupEventListeners();
    initializeVoice();
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
    
    // Voice button event listener
    voiceButton.addEventListener('click', toggleVoiceInput);
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
    setVoiceInputEnabled(enabled);
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

// Load voice configuration from server
async function loadVoiceConfig() {
    try {
        const response = await fetch('/api/speech/config');
        const data = await response.json();
        
        if (response.ok) {
            // Update voice configuration
            if (data.wake_words && data.wake_words.length > 0) {
                WAKE_WORDS = data.wake_words;
            }
            if (data.wake_timeout) {
                WAKE_TIMEOUT = data.wake_timeout * 1000; // Convert seconds to milliseconds
            }
            if (data.language) {
                VOICE_LANG = data.language;
            }
            
            console.log('Voice configuration loaded:', {
                wakeWords: WAKE_WORDS,
                wakeTimeout: WAKE_TIMEOUT,
                language: VOICE_LANG
            });
        }
    } catch (error) {
        console.warn('Failed to load voice config, using defaults:', error);
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

// ============ Voice Functions ============

// Initialize voice recognition
function initializeVoice() {
    // Check if Web Speech API is supported
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
        console.warn('Web Speech API not supported');
        voiceButton.disabled = true;
        voiceButton.title = '浏览器不支持语音识别';
        return;
    }
    
    isVoiceSupported = true;
    
    // First request microphone permission explicitly
    requestMicrophonePermission();
}

// Request microphone permission before initializing speech recognition
async function requestMicrophonePermission() {
    try {
        console.log('Requesting microphone permission...');
        
        // Check if getUserMedia is supported
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
            console.warn('getUserMedia not supported, trying direct initialization...');
            console.warn('提示：如果您不是通过 localhost 访问，请使用 https:// 协议或改为 http://localhost:8888 访问');
            initializeVoiceRecognition();
            return;
        }
        
        // Request microphone access
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        
        // Permission granted, stop the stream immediately
        stream.getTracks().forEach(track => track.stop());
        
        console.log('Microphone permission granted');
        showNotification('麦克风权限已获取，语音功能已启用', 'success');
        
        // Now initialize voice recognition
        initializeVoiceRecognition();
        
    } catch (error) {
        console.error('Microphone permission denied:', error);
        
        // Handle permission denial
        voiceButton.disabled = true;
        voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
        updateVoiceStatus('🚫', '麦克风权限被拒绝');
        showNotification('语音功能需要麦克风权限，请在浏览器设置中允许麦克风访问后刷新页面', 'warning');
    }
}

// Initialize voice recognition after permission is granted
function initializeVoiceRecognition() {
    // Initialize wake word recognition
    initializeWakeWordRecognition();
    
    // Start wake word listening automatically
    startWakeListening();
    
    console.log('Voice recognition initialized');
}

// Initialize wake word recognition
function initializeWakeWordRecognition() {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    
    wakeWordRecognition = new SpeechRecognition();
    wakeWordRecognition.continuous = true;
    wakeWordRecognition.interimResults = true;
    wakeWordRecognition.lang = VOICE_LANG;
    
    wakeWordRecognition.onstart = function() {
        console.log('Wake word recognition started');
        updateVoiceStatus('🔊', '等待唤醒词...');
        voiceButton.classList.add('wake-listening');
    };
    
    wakeWordRecognition.onresult = function(event) {
        let interimTranscript = '';
        let finalTranscript = '';
        
        for (let i = event.resultIndex; i < event.results.length; i++) {
            const transcript = event.results[i][0].transcript.toLowerCase().trim();
            
            if (event.results[i].isFinal) {
                finalTranscript += transcript;
            } else {
                interimTranscript += transcript;
            }
        }
        
        const fullTranscript = (finalTranscript + interimTranscript).toLowerCase();
        
        // Check for wake words
        if (WAKE_WORDS.some(word => fullTranscript.includes(word.toLowerCase()))) {
            console.log('Wake word detected:', fullTranscript);
            onWakeWordDetected();
        }
    };
    
    wakeWordRecognition.onerror = function(event) {
        console.error('Wake word recognition error:', event.error);
        
        if (event.error === 'not-allowed') {
            // User denied microphone permission
            console.warn('Microphone permission denied by user');
            isWakeListening = false;
            voiceButton.classList.remove('wake-listening');
            voiceButton.disabled = true;
            voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
            updateVoiceStatus('🚫', '麦克风权限被拒绝');
            
            // Show user-friendly message
            showNotification('语音功能需要麦克风权限，请刷新页面并允许麦克风访问', 'warning');
            
        } else if (event.error === 'network') {
            // Try to restart after network error
            setTimeout(() => {
                if (isWakeListening) {
                    startWakeListening();
                }
            }, 2000);
        } else if (event.error === 'aborted') {
            // Recognition was aborted, this is normal
            console.log('Wake word recognition was aborted');
        } else {
            // Other errors
            console.error('Unhandled wake word recognition error:', event.error);
            // Don't automatically restart for unknown errors
            isWakeListening = false;
            voiceButton.classList.remove('wake-listening');
        }
    };
    
    wakeWordRecognition.onend = function() {
        console.log('Wake word recognition ended');
        if (isWakeListening && !isVoiceInputActive && isVoiceSupported && !voiceButton.disabled) {
            // Only restart if we should be listening and have permissions
            setTimeout(() => {
                if (isWakeListening && !voiceButton.disabled) {
                    startWakeListening();
                }
            }, 1000);
        } else {
            // Clear the wake listening state if we shouldn't restart
            voiceButton.classList.remove('wake-listening');
            hideVoiceStatus();
        }
    };
}

// Global variable to track notification timeout
let notificationTimeoutId = null;

// Show notification to user
function showNotification(message, type = 'info') {
    // Clear any existing timeout
    if (notificationTimeoutId) {
        clearTimeout(notificationTimeoutId);
        notificationTimeoutId = null;
    }
    
    // Create notification element if it doesn't exist
    let notification = document.getElementById('voice-notification');
    if (!notification) {
        notification = document.createElement('div');
        notification.id = 'voice-notification';
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            max-width: 300px;
            padding: 12px 16px;
            border-radius: 6px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            color: white;
            font-size: 14px;
            z-index: 10000;
            transform: translateX(100%);
            transition: transform 0.3s ease-in-out;
        `;
        document.body.appendChild(notification);
    }
    
    // Set background color based on type
    const colors = {
        info: '#2196F3',
        warning: '#FF9800',
        error: '#F44336',
        success: '#4CAF50'
    };
    notification.style.backgroundColor = colors[type] || colors.info;
    
    // Set message and show
    notification.textContent = message;
    notification.style.transform = 'translateX(0)';
    
    // Auto hide after 3 seconds (reduced from 5 seconds)
    notificationTimeoutId = setTimeout(() => {
        if (notification) {
            notification.style.transform = 'translateX(100%)';
            // Clean up after animation completes
            setTimeout(() => {
                if (notification && notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300); // Wait for transition to complete
        }
        notificationTimeoutId = null;
    }, 3000);
}

// Start wake word listening
function startWakeListening() {
    if (!isVoiceSupported || isVoiceInputActive || voiceButton.disabled) return;
    
    try {
        isWakeListening = true;
        wakeWordRecognition.start();
        console.log('Started wake word listening');
    } catch (error) {
        console.error('Failed to start wake word listening:', error);
        isWakeListening = false;
    }
}

// Stop wake word listening
function stopWakeListening() {
    if (!isWakeListening) return;
    
    isWakeListening = false;
    try {
        wakeWordRecognition.stop();
        voiceButton.classList.remove('wake-listening');
        console.log('Stopped wake word listening');
    } catch (error) {
        console.error('Failed to stop wake word listening:', error);
    }
}

// Handle wake word detection
function onWakeWordDetected() {
    stopWakeListening();
    startVoiceInput();
}

// Toggle voice input manually
function toggleVoiceInput() {
    // If button is disabled due to permission issues, try to re-initialize
    if (voiceButton.disabled) {
        retryVoicePermission();
        return;
    }
    
    if (isVoiceInputActive) {
        stopVoiceInput();
    } else {
        stopWakeListening();
        startVoiceInput();
    }
}

// Retry voice permission
function retryVoicePermission() {
    console.log('Retrying voice permission...');
    
    // Reset button state
    voiceButton.disabled = false;
    voiceButton.title = '语音输入';
    voiceButton.classList.remove('wake-listening');
    hideVoiceStatus();
    
    // Try to re-request microphone permission
    if (isVoiceSupported) {
        showNotification('正在重新请求麦克风权限...', 'info');
        // Use the same permission request logic
        requestMicrophonePermission();
    }
}

// Start voice input
function startVoiceInput() {
    if (!isVoiceSupported || isVoiceInputActive || voiceButton.disabled) return;
    
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    
    voiceInputRecognition = new SpeechRecognition();
    voiceInputRecognition.continuous = false;
    voiceInputRecognition.interimResults = true;
    voiceInputRecognition.lang = VOICE_LANG;
    
    voiceInputRecognition.onstart = function() {
        isVoiceInputActive = true;
        voiceButton.classList.add('active');
        voiceButton.classList.remove('wake-listening');
        updateVoiceStatus('🎤', '正在录音...');
        showVoiceStatus();
        
        // Set timeout for voice input
        voiceTimeoutId = setTimeout(() => {
            stopVoiceInput();
        }, WAKE_TIMEOUT);
        
        console.log('Voice input started');
    };
    
    voiceInputRecognition.onresult = function(event) {
        let interimTranscript = '';
        let finalTranscript = '';
        
        for (let i = event.resultIndex; i < event.results.length; i++) {
            const transcript = event.results[i][0].transcript;
            
            if (event.results[i].isFinal) {
                finalTranscript += transcript;
            } else {
                interimTranscript += transcript;
            }
        }
        
        // Update display with interim results
        if (interimTranscript) {
            updateVoiceStatus('🎤', `识别中: ${interimTranscript}`);
        }
        
        // Handle final result
        if (finalTranscript.trim()) {
            console.log('Voice input result:', finalTranscript);
            messageInput.value = finalTranscript.trim();
            updateVoiceStatus('✅', `识别完成: ${finalTranscript.trim()}`);
            
            // Auto-resize textarea
            messageInput.style.height = 'auto';
            messageInput.style.height = Math.min(messageInput.scrollHeight, 120) + 'px';
            
            // Auto-send the message after a short delay
            setTimeout(() => {
                hideVoiceStatus();
                if (finalTranscript.trim()) {
                    sendMessage();
                }
                stopVoiceInput();
            }, 1000);
        }
    };
    
    voiceInputRecognition.onerror = function(event) {
        console.error('Voice input recognition error:', event.error);
        
        if (event.error === 'not-allowed') {
            // User denied microphone permission
            updateVoiceStatus('🚫', '麦克风权限被拒绝');
            voiceButton.disabled = true;
            voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
            showNotification('语音功能需要麦克风权限，请刷新页面并允许麦克风访问', 'warning');
        } else {
            updateVoiceStatus('❌', `识别错误: ${event.error}`);
        }
        
        setTimeout(() => {
            stopVoiceInput();
        }, 2000);
    };
    
    voiceInputRecognition.onend = function() {
        console.log('Voice input recognition ended');
        stopVoiceInput();
    };
    
    try {
        voiceInputRecognition.start();
    } catch (error) {
        console.error('Failed to start voice input:', error);
        updateVoiceStatus('❌', '启动语音输入失败');
        stopVoiceInput();
    }
}

// Stop voice input
function stopVoiceInput() {
    if (!isVoiceInputActive) return;
    
    isVoiceInputActive = false;
    voiceButton.classList.remove('active');
    
    if (voiceTimeoutId) {
        clearTimeout(voiceTimeoutId);
        voiceTimeoutId = null;
    }
    
    try {
        if (voiceInputRecognition) {
            voiceInputRecognition.stop();
        }
    } catch (error) {
        console.error('Failed to stop voice input:', error);
    }
    
    setTimeout(() => {
        hideVoiceStatus();
        startWakeListening();
    }, 2000);
    
    console.log('Voice input stopped');
}

// Update voice status display
function updateVoiceStatus(indicator, text) {
    voiceIndicator.textContent = indicator;
    voiceText.textContent = text;
}

// Show voice status
function showVoiceStatus() {
    voiceStatus.style.display = 'flex';
}

// Hide voice status
function hideVoiceStatus() {
    voiceStatus.style.display = 'none';
}

// Set voice input enabled/disabled
function setVoiceInputEnabled(enabled) {
    if (enabled) {
        voiceButton.disabled = false;
        voiceButton.title = '语音输入';
        // 如果没有语音输入活动且没有正在监听唤醒词，重新启动唤醒监听
        if (!isVoiceInputActive && !isWakeListening && isVoiceSupported) {
            console.log('Re-starting wake listening after input enabled');
            startWakeListening();
        }
    } else {
        voiceButton.disabled = true;
        voiceButton.title = '输入功能已禁用';
    }
    
    if (!enabled && isVoiceInputActive) {
        stopVoiceInput();
    }
    if (!enabled && isWakeListening) {
        stopWakeListening();
    }
}