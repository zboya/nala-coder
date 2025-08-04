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

// File browser elements
const fileBrowserPanel = document.getElementById('fileBrowserPanel');
const fileBrowserToggle = document.getElementById('fileBrowserToggle');
const fileTree = document.getElementById('fileTree');
const currentPath = document.getElementById('currentPath');
const refreshTreeBtn = document.getElementById('refreshTreeBtn');
const fileViewer = document.getElementById('fileViewer');
const fileName = document.getElementById('fileName');
const fileMeta = document.getElementById('fileMeta');
const fileCode = document.getElementById('fileCode');
const closeFileBtn = document.getElementById('closeFileBtn');
const chatSection = document.getElementById('chatSection');


// State
let sessionId = '';
let isStreaming = true;
let currentStreamingMessage = null;
let sessions = [];
let tools = [];
let fileTreeData = null;
let currentWorkingPath = '';

// Voice manager instance
let voiceManager = null;




// Initialize app
document.addEventListener('DOMContentLoaded', function() {
    loadTools();
    loadFileTree();

    setupEventListeners();
    // Initialize voice manager
    if (window.VoiceManager) {
        voiceManager = new window.VoiceManager();
        voiceManager.init({
            messageInput: messageInput,
            sendMessageCallback: sendMessage,
            onVoiceStart: restoreChatLayout,
            onVoiceEnd: restorePreviousLayout
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
    
    // File browser event listeners
    fileBrowserToggle.addEventListener('click', toggleFileBrowser);
    refreshTreeBtn.addEventListener('click', loadFileTree);
    closeFileBtn.addEventListener('click', closeFileViewer);
    
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

// File Browser Functions

// Load file tree from server
async function loadFileTree() {
    try {
        refreshTreeBtn.disabled = true;
        refreshTreeBtn.textContent = '🔄';
        fileTree.innerHTML = '<div class="loading">加载目录结构中...</div>';
        
        const response = await fetch('/api/files/tree');
        const data = await response.json();
        
        if (response.ok && data.tree) {
            fileTreeData = data.tree;
            currentWorkingPath = data.path;
            currentPath.textContent = data.path;
            renderFileTree();
        } else {
            fileTree.innerHTML = '<div class="error">加载失败</div>';
        }
    } catch (error) {
        console.error('Failed to load file tree:', error);
        fileTree.innerHTML = '<div class="error">网络错误</div>';
    } finally {
        refreshTreeBtn.disabled = false;
        refreshTreeBtn.textContent = '🔄';
    }
}

// Render file tree in the UI
function renderFileTree() {
    if (!fileTreeData) {
        fileTree.innerHTML = '<div class="error">无数据</div>';
        return;
    }
    
    fileTree.innerHTML = '';
    const treeElement = createFileTreeElement(fileTreeData, 0);
    fileTree.appendChild(treeElement);
}

// Create file tree DOM element recursively
function createFileTreeElement(node, depth) {
    const div = document.createElement('div');
    div.className = `file-node ${node.type}-node`;
    div.style.paddingLeft = `${depth * 12}px`;
    
    const content = document.createElement('div');
    content.className = 'file-node-content';
    
    if (node.type === 'directory') {
        const icon = document.createElement('span');
        icon.className = 'folder-icon';
        icon.textContent = node.children && node.children.length > 0 ? '📁' : '📂';
        
        const name = document.createElement('span');
        name.className = 'file-name';
        name.textContent = node.name;
        
        content.appendChild(icon);
        content.appendChild(name);
        
        // 添加点击事件来折叠/展开目录
        content.addEventListener('click', () => {
            const childrenDiv = div.querySelector('.file-children');
            if (childrenDiv) {
                const isVisible = childrenDiv.style.display !== 'none';
                childrenDiv.style.display = isVisible ? 'none' : 'block';
                icon.textContent = isVisible ? '📂' : '📁';
            }
        });
        
        div.appendChild(content);
        
        // 添加子节点
        if (node.children && node.children.length > 0) {
            const childrenDiv = document.createElement('div');
            childrenDiv.className = 'file-children';
            
            node.children.forEach(child => {
                const childElement = createFileTreeElement(child, depth + 1);
                childrenDiv.appendChild(childElement);
            });
            
            div.appendChild(childrenDiv);
        }
    } else {
        const icon = document.createElement('span');
        icon.className = 'file-icon';
        icon.textContent = getFileIcon(node.name);
        
        const name = document.createElement('span');
        name.className = 'file-name';
        name.textContent = node.name;
        
        content.appendChild(icon);
        content.appendChild(name);
        
        // 添加点击事件来查看文件
        content.addEventListener('click', () => {
            loadFileContent(node.path);
        });
        
        div.appendChild(content);
    }
    
    return div;
}

// Get file icon based on file extension
function getFileIcon(fileName) {
    const ext = fileName.split('.').pop()?.toLowerCase();
    
    const iconMap = {
        'js': '📄',
        'ts': '📘',
        'jsx': '⚛️',
        'tsx': '⚛️',
        'go': '🐹',
        'py': '🐍',
        'java': '☕',
        'html': '🌐',
        'css': '🎨',
        'scss': '🎨',
        'sass': '🎨',
        'json': '📋',
        'xml': '📄',
        'yaml': '📄',
        'yml': '📄',
        'md': '📝',
        'txt': '📄',
        'sh': '⚡',
        'bat': '⚡',
        'sql': '🗃️',
        'dockerfile': '🐳',
        'makefile': '🔧',
    };
    
    return iconMap[ext] || '📄';
}

// Load and display file content
async function loadFileContent(filePath) {
    try {
        const response = await fetch(`/api/files/content?path=${encodeURIComponent(filePath)}`);
        const data = await response.json();
        
        if (response.ok) {
            displayFileContent(data);
        } else {
            console.error('Failed to load file content:', data.error);
            alert(`无法加载文件: ${data.error}`);
        }
    } catch (error) {
        console.error('Failed to load file content:', error);
        alert('网络错误，无法加载文件内容');
    }
}

// Display file content in the viewer
function displayFileContent(fileData) {
    fileName.textContent = fileData.path.split('/').pop();
    fileMeta.textContent = `${formatFileSize(fileData.size)} | ${fileData.mod_time}`;
    
    // 设置代码内容和语言高亮
    fileCode.textContent = fileData.content;
    fileCode.className = `language-${fileData.language}`;
    
    // 如果有 Prism.js 或其他语法高亮库，可以在这里调用
    if (window.Prism) {
        window.Prism.highlightElement(fileCode);
    }
    
    // 显示文件查看器并缩小聊天区域
    fileViewer.style.display = 'block';
    adjustLayoutForFileViewer(true);
}

// Close file viewer
function closeFileViewer() {
    fileViewer.style.display = 'none';
    adjustLayoutForFileViewer(false);
}

// Toggle file browser panel
function toggleFileBrowser() {
    const isCollapsed = fileBrowserPanel.classList.contains('collapsed');
    fileBrowserPanel.classList.toggle('collapsed', !isCollapsed);
    fileBrowserToggle.textContent = isCollapsed ? '📌' : '📁';
}

// Adjust layout when file viewer is shown/hidden
function adjustLayoutForFileViewer(showFileViewer) {
    if (showFileViewer) {
        chatSection.classList.add('with-file-viewer');
        fileViewer.style.display = 'block';
    } else {
        chatSection.classList.remove('with-file-viewer');
        fileViewer.style.display = 'none';
    }
}

// Restore chat layout (called when voice is activated)
function restoreChatLayout() {
    // 如果正在查看文件，暂时隐藏文件查看器
    const wasFileViewerVisible = fileViewer.style.display !== 'none';
    if (wasFileViewerVisible) {
        adjustLayoutForFileViewer(false);
        // 存储状态，以便稍后恢复
        fileViewer.dataset.wasVisible = 'true';
    }
}

// Restore previous layout (called when voice interaction ends)
function restorePreviousLayout() {
    // 如果之前有打开的文件查看器，恢复显示
    if (fileViewer.dataset.wasVisible === 'true') {
        adjustLayoutForFileViewer(true);
        delete fileViewer.dataset.wasVisible;
    }
}

// Legacy function for compatibility
function adjustChatContainerHeight() {
    adjustLayoutForFileViewer(fileViewer.style.display !== 'none');
}

// Format file size for display
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

