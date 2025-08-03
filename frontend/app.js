// 应用主类
class NaLaCoderApp {
    constructor() {
        this.currentSessionId = null;
        this.isListening = false;
        this.isInputMode = false;
        this.wakeupRecognition = null;
        this.inputRecognition = null;
        this.inputTimeout = null;
        this.synthesis = null;
        this.currentFile = null;
        this.fileTree = null;
        
        this.init();
    }

    init() {
        this.initSpeechRecognition();
        this.initSpeechSynthesis();
        this.bindEvents();
        this.loadFileTree();
        this.loadSpeechConfig();
        
        // 请求语音权限并启动唤醒监听
        this.requestVoicePermission();
    }

    // 初始化语音识别
    initSpeechRecognition() {
        if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
            const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
            
            // 创建唤醒监听器
            this.wakeupRecognition = new SpeechRecognition();
            this.wakeupRecognition.continuous = true;
            this.wakeupRecognition.interimResults = true;
            this.wakeupRecognition.lang = 'zh-CN';
            
            // 创建语音输入监听器
            this.inputRecognition = new SpeechRecognition();
            this.inputRecognition.continuous = true;
            this.inputRecognition.interimResults = true;
            this.inputRecognition.lang = 'zh-CN';
            
            // 唤醒监听器事件
            this.wakeupRecognition.onstart = () => {
                console.log('开始监听语音唤醒...');
                this.updateWakeupStatus('正在监听唤醒词...');
            };
            
            this.wakeupRecognition.onresult = (event) => {
                const transcript = Array.from(event.results)
                    .map(result => result[0])
                    .map(result => result.transcript)
                    .join('');
                
                // 检查是否包含唤醒词
                if (transcript.toLowerCase().includes('小助手')) {
                    console.log('检测到唤醒词:', transcript);
                    this.handleWakeup();
                }
            };
            
            this.wakeupRecognition.onerror = (event) => {
                console.error('唤醒监听错误:', event.error);
                if (event.error !== 'no-speech') {
                    this.updateWakeupStatus('唤醒监听出错，正在重试...');
                    setTimeout(() => this.startWakeupListening(), 1000);
                }
            };
            
            this.wakeupRecognition.onend = () => {
                console.log('唤醒监听结束，重新开始...');
                if (!this.isInputMode) {
                    setTimeout(() => this.startWakeupListening(), 100);
                }
            };
            
            // 语音输入监听器事件
            this.inputRecognition.onstart = () => {
                this.isInputMode = true;
                this.isListening = true;
                this.showVoiceStatus();
                this.updateVoiceButton();
                this.updateWakeupStatus('请说出您的问题...');
                this.updateVoiceStatus('请说出您的问题...');
            };
            
            this.inputRecognition.onresult = (event) => {
                const transcript = Array.from(event.results)
                    .map(result => result[0])
                    .map(result => result.transcript)
                    .join('');
                
                // 显示实时识别结果
                this.updateVoiceStatus(`正在听: ${transcript}`);
                
                // 检查是否有足够的语音内容
                if (event.results[0].isFinal && transcript.trim().length > 0) {
                    this.handleVoiceInput(transcript);
                }
            };
            
            this.inputRecognition.onerror = (event) => {
                console.error('语音输入错误:', event.error);
                this.hideVoiceStatus();
                this.updateVoiceButton();
                this.endInputMode();
            };
            
            this.inputRecognition.onend = () => {
                this.isListening = false;
                this.hideVoiceStatus();
                this.updateVoiceButton();
                
                // 延迟结束输入模式，给用户更多时间
                setTimeout(() => {
                    this.endInputMode();
                }, 2000);
            };
        } else {
            console.warn('浏览器不支持语音识别');
        }
    }

    // 初始化语音合成
    initSpeechSynthesis() {
        if ('speechSynthesis' in window) {
            this.synthesis = window.speechSynthesis;
        }
    }

    // 绑定事件
    bindEvents() {
        // 发送按钮
        const sendBtn = document.getElementById('sendBtn');
        const messageInput = document.getElementById('messageInput');
        
        sendBtn.addEventListener('click', () => this.sendMessage());
        messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });

        // 语音按钮
        const voiceBtn = document.getElementById('voiceBtn');
        voiceBtn.addEventListener('click', () => this.toggleVoiceRecognition());


    }

    // 发送消息
    async sendMessage() {
        const messageInput = document.getElementById('messageInput');
        const message = messageInput.value.trim();
        
        if (!message) return;

        // 添加用户消息到聊天界面
        this.addMessage(message, 'user');
        messageInput.value = '';

        // 显示正在输入状态
        this.addTypingMessage();

        try {
            const response = await this.chatWithAgent(message);
            this.removeTypingMessage();
            this.addMessage(response, 'assistant');
        } catch (error) {
            console.error('聊天错误:', error);
            this.removeTypingMessage();
            this.addMessage('抱歉，发生了错误，请稍后重试。', 'assistant');
        }
    }

    // 与后端Agent通信
    async chatWithAgent(message) {
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                message: message,
                session_id: this.currentSessionId,
                stream: false
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        this.currentSessionId = data.session_id;
        return data.response;
    }

    // 流式聊天
    async chatWithAgentStream(message) {
        const response = await fetch('/api/chat/stream', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                message: message,
                session_id: this.currentSessionId,
                stream: true
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop();

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.slice(6);
                    if (data === '[DONE]') return;
                    
                    try {
                        const parsed = JSON.parse(data);
                        this.updateStreamMessage(parsed.response);
                    } catch (e) {
                        console.error('解析流数据错误:', e);
                    }
                }
            }
        }
    }

    // 添加消息到聊天界面
    addMessage(content, type) {
        const chatMessages = document.getElementById('chatMessages');
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${type}`;
        messageDiv.textContent = content;
        chatMessages.appendChild(messageDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    // 添加正在输入消息
    addTypingMessage() {
        const chatMessages = document.getElementById('chatMessages');
        const typingDiv = document.createElement('div');
        typingDiv.className = 'message assistant typing';
        typingDiv.id = 'typingMessage';
        typingDiv.textContent = '正在思考...';
        chatMessages.appendChild(typingDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    // 移除正在输入消息
    removeTypingMessage() {
        const typingMessage = document.getElementById('typingMessage');
        if (typingMessage) {
            typingMessage.remove();
        }
    }

    // 更新流式消息
    updateStreamMessage(content) {
        let messageDiv = document.querySelector('.message.assistant:last-child');
        if (!messageDiv || messageDiv.classList.contains('typing')) {
            this.removeTypingMessage();
            messageDiv = document.createElement('div');
            messageDiv.className = 'message assistant';
            document.getElementById('chatMessages').appendChild(messageDiv);
        }
        messageDiv.textContent = content;
    }

    // 处理语音输入
    handleVoiceInput(transcript) {
        console.log('处理语音输入:', transcript);
        
        // 清除超时定时器
        if (this.inputTimeout) {
            clearTimeout(this.inputTimeout);
            this.inputTimeout = null;
        }
        
        // 停止语音输入
        this.inputRecognition.stop();
        
        // 将语音转换为文字并发送
        const messageInput = document.getElementById('messageInput');
        messageInput.value = transcript;
        this.sendMessage();
    }

    // 开始唤醒监听
    startWakeupListening() {
        if (!this.wakeupRecognition) {
            console.warn('唤醒监听器未初始化');
            return;
        }
        
        try {
            this.wakeupRecognition.start();
        } catch (error) {
            console.error('启动唤醒监听失败:', error);
        }
    }
    
    // 停止唤醒监听
    stopWakeupListening() {
        if (this.wakeupRecognition) {
            this.wakeupRecognition.stop();
        }
    }
    
    // 处理唤醒
    handleWakeup() {
        console.log('语音唤醒成功！');
        this.stopWakeupListening();
        this.updateWakeupStatus('已唤醒，请说出您的问题...');
        
        // 播放唤醒提示音（可选）
        this.playWakeupSound();
        
        // 延迟一下再开始语音输入
        setTimeout(() => {
            this.startVoiceInput();
        }, 500);
    }
    
    // 开始语音输入
    startVoiceInput() {
        if (!this.inputRecognition) {
            console.warn('语音输入监听器未初始化');
            return;
        }
        
        try {
            this.inputRecognition.start();
            
            // 设置超时机制，如果30秒内没有语音输入，自动结束
            this.inputTimeout = setTimeout(() => {
                console.log('语音输入超时，自动结束');
                this.inputRecognition.stop();
            }, 30000);
            
        } catch (error) {
            console.error('启动语音输入失败:', error);
        }
    }
    
    // 结束输入模式
    endInputMode() {
        this.isInputMode = false;
        this.updateWakeupStatus('正在重新启动唤醒监听...');
        
        // 延迟后重新开始唤醒监听
        setTimeout(() => {
            this.startWakeupListening();
        }, 1000);
    }
    
    // 播放唤醒提示音
    playWakeupSound() {
        // 创建音频上下文播放提示音
        try {
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const oscillator = audioContext.createOscillator();
            const gainNode = audioContext.createGain();
            
            oscillator.connect(gainNode);
            gainNode.connect(audioContext.destination);
            
            oscillator.frequency.setValueAtTime(800, audioContext.currentTime);
            oscillator.frequency.setValueAtTime(1200, audioContext.currentTime + 0.1);
            oscillator.frequency.setValueAtTime(800, audioContext.currentTime + 0.2);
            
            gainNode.gain.setValueAtTime(0.1, audioContext.currentTime);
            gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3);
            
            oscillator.start(audioContext.currentTime);
            oscillator.stop(audioContext.currentTime + 0.3);
        } catch (error) {
            console.log('无法播放提示音:', error);
        }
    }
    
    // 更新唤醒状态显示
    updateWakeupStatus(message) {
        const statusElement = document.getElementById('wakeupStatus');
        if (statusElement) {
            statusElement.textContent = message;
        }
    }
    
    // 切换语音识别（保留原有功能）
    toggleVoiceRecognition() {
        if (!this.inputRecognition) {
            alert('您的浏览器不支持语音识别功能');
            return;
        }

        if (this.isListening) {
            this.inputRecognition.stop();
        } else {
            this.startVoiceInput();
        }
    }

    // 显示语音状态
    showVoiceStatus() {
        const voiceStatus = document.getElementById('voiceStatus');
        voiceStatus.classList.add('active');
    }

    // 隐藏语音状态
    hideVoiceStatus() {
        const voiceStatus = document.getElementById('voiceStatus');
        voiceStatus.classList.remove('active');
    }
    
    // 更新语音状态显示
    updateVoiceStatus(message) {
        const voiceStatus = document.getElementById('voiceStatus');
        const voiceIndicator = voiceStatus.querySelector('.voice-indicator span');
        if (voiceIndicator) {
            voiceIndicator.textContent = message;
        }
    }

    // 更新语音按钮状态
    updateVoiceButton() {
        const voiceBtn = document.getElementById('voiceBtn');
        const icon = voiceBtn.querySelector('i');
        
        if (this.isListening) {
            icon.className = 'fas fa-microphone-slash';
            voiceBtn.style.color = '#f85149';
        } else {
            icon.className = 'fas fa-microphone';
            voiceBtn.style.color = '#8b949e';
        }
    }

    // 加载文件树
    async loadFileTree() {
        try {
            const response = await fetch('/api/files/tree');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            this.fileTree = data.tree;
            this.renderFileTree(this.fileTree);
        } catch (error) {
            console.error('加载文件树错误:', error);
        }
    }

    // 渲染文件树
    renderFileTree(node, level = 0) {
        const fileTree = document.getElementById('fileTree');
        
        if (level === 0) {
            fileTree.innerHTML = '';
        }

        const item = document.createElement('div');
        item.className = 'file-item';
        item.style.paddingLeft = `${level * 20}px`;
        
        const icon = document.createElement('i');
        if (node.type === 'directory') {
            icon.className = 'fas fa-folder';
        } else {
            icon.className = this.getFileIcon(node.name);
        }
        
        const name = document.createElement('span');
        name.textContent = node.name;
        
        item.appendChild(icon);
        item.appendChild(name);
        
        if (node.type === 'file') {
            const count = document.createElement('span');
            count.className = 'file-count';
            count.textContent = `+${Math.floor(Math.random() * 200) + 1}`;
            item.appendChild(count);
        }
        
        item.addEventListener('click', () => {
            if (node.type === 'file') {
                this.loadFileContent(node.path);
            }
        });
        
        fileTree.appendChild(item);
        
        // 递归渲染子节点
        if (node.children && node.children.length > 0) {
            node.children.forEach(child => {
                this.renderFileTree(child, level + 1);
            });
        }
    }

    // 获取文件图标
    getFileIcon(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        const iconMap = {
            'js': 'fab fa-js-square',
            'ts': 'fas fa-code',
            'tsx': 'fas fa-code',
            'jsx': 'fab fa-react',
            'html': 'fab fa-html5',
            'css': 'fab fa-css3-alt',
            'py': 'fab fa-python',
            'go': 'fas fa-code',
            'java': 'fab fa-java',
            'cpp': 'fas fa-code',
            'c': 'fas fa-code',
            'php': 'fab fa-php',
            'rb': 'fas fa-gem',
            'rs': 'fas fa-code',
            'swift': 'fab fa-swift',
            'kt': 'fas fa-code',
            'scala': 'fas fa-code',
            'sql': 'fas fa-database',
            'json': 'fas fa-code',
            'xml': 'fas fa-code',
            'yaml': 'fas fa-code',
            'yml': 'fas fa-code',
            'md': 'fas fa-markdown',
            'txt': 'fas fa-file-alt',
            'log': 'fas fa-file-alt',
            'sh': 'fas fa-terminal',
            'bat': 'fas fa-terminal',
            'ps1': 'fas fa-terminal',
            'dockerfile': 'fab fa-docker',
            'gitignore': 'fab fa-git-alt',
            'readme': 'fas fa-book',
            'license': 'fas fa-gavel'
        };
        
        return iconMap[ext] || 'fas fa-file';
    }

    // 加载文件内容
    async loadFileContent(filePath) {
        try {
            const response = await fetch(`/api/files/content?path=${encodeURIComponent(filePath)}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            this.currentFile = data;
            this.renderFileContent(data);
        } catch (error) {
            console.error('加载文件内容错误:', error);
        }
    }

    // 渲染文件内容
    renderFileContent(fileData) {
        // 更新面包屑
        this.updateBreadcrumb(fileData.path);
        
        // 更新代码内容
        const codeContent = document.getElementById('codeContent');
        codeContent.textContent = fileData.content;
        codeContent.className = `language-${this.getLanguageClass(fileData.language)}`;
        
        // 重新高亮代码
        if (window.Prism) {
            Prism.highlightElement(codeContent);
        }
        
        // 更新活动文件项
        this.updateActiveFileItem(fileData.path);
    }

    // 更新面包屑
    updateBreadcrumb(filePath) {
        const breadcrumb = document.getElementById('breadcrumb');
        const parts = filePath.split('/');
        
        breadcrumb.innerHTML = '';
        
        parts.forEach((part, index) => {
            if (index > 0) {
                const separator = document.createElement('span');
                separator.className = 'breadcrumb-separator';
                separator.textContent = ' > ';
                breadcrumb.appendChild(separator);
            }
            
            const item = document.createElement('span');
            item.className = 'breadcrumb-item';
            item.textContent = part;
            breadcrumb.appendChild(item);
        });
    }

    // 获取语言CSS类
    getLanguageClass(language) {
        const languageMap = {
            'javascript': 'javascript',
            'typescript': 'typescript',
            'python': 'python',
            'go': 'go',
            'java': 'java',
            'cpp': 'cpp',
            'c': 'c',
            'php': 'php',
            'ruby': 'ruby',
            'rust': 'rust',
            'swift': 'swift',
            'kotlin': 'kotlin',
            'scala': 'scala',
            'sql': 'sql',
            'json': 'json',
            'xml': 'xml',
            'yaml': 'yaml',
            'markdown': 'markdown',
            'html': 'html',
            'css': 'css',
            'shell': 'bash',
            'dockerfile': 'dockerfile'
        };
        
        return languageMap[language.toLowerCase()] || 'text';
    }

    // 更新活动文件项
    updateActiveFileItem(filePath) {
        document.querySelectorAll('.file-item').forEach(item => {
            item.classList.remove('active');
        });
        
        // 找到对应的文件项并激活
        const fileItems = document.querySelectorAll('.file-item');
        for (const item of fileItems) {
            const name = item.querySelector('span').textContent;
            if (filePath.endsWith(name)) {
                item.classList.add('active');
                break;
            }
        }
    }

    // 加载语音配置
    async loadSpeechConfig() {
        try {
            const response = await fetch('/api/speech/config');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const config = await response.json();
            console.log('语音配置:', config);
        } catch (error) {
            console.error('加载语音配置错误:', error);
        }
    }

    // 请求语音权限
    async requestVoicePermission() {
        try {
            // 检查是否支持语音识别
            if (!('webkitSpeechRecognition' in window || 'SpeechRecognition' in window)) {
                console.warn('浏览器不支持语音识别');
                return;
            }
            
            // 显示权限请求提示
            this.updateWakeupStatus('正在请求麦克风权限...');
            
            // 尝试启动一次语音识别来触发权限请求
            const testRecognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
            testRecognition.lang = 'zh-CN';
            testRecognition.continuous = false;
            testRecognition.interimResults = false;
            
            testRecognition.onstart = () => {
                console.log('权限请求成功');
                testRecognition.stop();
                this.updateWakeupStatus('权限获取成功，开始监听唤醒词...');
                this.startWakeupListening();
            };
            
            testRecognition.onerror = (event) => {
                console.error('权限请求失败:', event.error);
                this.updateWakeupStatus('麦克风权限被拒绝，请允许权限后刷新页面');
            };
            
            testRecognition.onend = () => {
                // 权限测试结束，如果成功则启动唤醒监听
                if (this.wakeupRecognition) {
                    setTimeout(() => this.startWakeupListening(), 500);
                }
            };
            
            testRecognition.start();
            
        } catch (error) {
            console.error('请求语音权限失败:', error);
            this.updateWakeupStatus('语音权限请求失败');
        }
    }
    
    // 语音合成
    speak(text) {
        if (this.synthesis) {
            const utterance = new SpeechSynthesisUtterance(text);
            utterance.lang = 'zh-CN';
            utterance.rate = 0.9;
            utterance.pitch = 1.0;
            this.synthesis.speak(utterance);
        }
    }
}

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    new NaLaCoderApp();
});

// 全局错误处理
window.addEventListener('error', (event) => {
    console.error('全局错误:', event.error);
});

window.addEventListener('unhandledrejection', (event) => {
    console.error('未处理的Promise拒绝:', event.reason);
}); 