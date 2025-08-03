// ============ Voice Manager Module ============

class VoiceManager {
    constructor() {
        // DOM elements
        this.voiceButton = null;
        this.voiceButtonContainer = null;
        this.voiceStatus = null;
        this.voiceIndicator = null;
        this.voiceText = null;
        
        // Voice related state
        this.isVoiceSupported = false;
        this.wakeWordRecognition = null;
        this.voiceInputRecognition = null;
        this.isWakeListening = false;
        this.isVoiceInputActive = false;
        this.voiceTimeoutId = null;
        
        // Voice configuration (will be loaded from server)
        this.WAKE_WORDS = ['nala', 'nala coder', '小助手', '助手']; // 默认值
        this.WAKE_TIMEOUT = 30000; // 30 seconds
        this.VOICE_LANG = 'zh-CN';
        
        // 预编译的唤醒词变体，提高匹配速度
        this.COMPILED_WAKE_PATTERNS = [];
        
        // Global variable to track notification timeout
        this.notificationTimeoutId = null;
        
        // External dependencies
        this.messageInput = null;
        this.sendMessageCallback = null;
        this.onVoiceStart = null;
        this.onVoiceEnd = null;
    }
    
    // Initialize voice manager
    init(options = {}) {
        // Get DOM elements
        this.voiceButton = document.getElementById('voiceButton');
        this.voiceButtonContainer = document.querySelector('.voice-button-container');
        this.voiceStatus = document.getElementById('voiceStatus');
        this.voiceIndicator = document.getElementById('voiceIndicator');
        this.voiceText = document.getElementById('voiceText');
        
        // Set external dependencies
        this.messageInput = options.messageInput || document.getElementById('messageInput');
        this.sendMessageCallback = options.sendMessageCallback;
        this.onVoiceStart = options.onVoiceStart;
        this.onVoiceEnd = options.onVoiceEnd;
        
        // Setup event listeners
        if (this.voiceButton) {
            this.voiceButton.addEventListener('click', () => this.toggleVoiceInput());
        }
        
        // Load configuration and initialize
        this.loadVoiceConfig().then(() => {
            this.compileWakePatterns();
            this.initializeVoice();
        });
    }
    
    // 编译唤醒词模式
    compileWakePatterns() {
        this.COMPILED_WAKE_PATTERNS = [];
        
        for (const word of this.WAKE_WORDS) {
            const variations = [];
            const normalized = word.toLowerCase().trim();
            
            // 基本形式
            variations.push(normalized);
            variations.push(normalized.replace(/\s+/g, '')); // 去空格版本
            
            // 特殊发音变体
            if (normalized === 'nala') {
                variations.push('纳拉', '娜拉', 'na la', 'na-la', 'nara', '那拉');
            } else if (normalized === 'nala coder') {
                variations.push('纳拉coder', '娜拉coder', '纳拉 coder', '娜拉 coder', 
                              'nala码农', 'nala编程', 'nalacoder', 'nara coder');
            } else if (normalized === '小助手') {
                variations.push('小助理', '助手', '小帮手');
            }
            
            this.COMPILED_WAKE_PATTERNS.push({
                original: word,
                variations: variations
            });
        }
        
        console.log('Compiled wake patterns:', this.COMPILED_WAKE_PATTERNS);
    }
    
    // Load voice configuration from server
    async loadVoiceConfig() {
        try {
            const response = await fetch('/api/speech/config');
            const data = await response.json();
            
            if (response.ok) {
                // Update voice configuration
                if (data.wake_words && data.wake_words.length > 0) {
                    this.WAKE_WORDS = data.wake_words;
                }
                if (data.wake_timeout) {
                    this.WAKE_TIMEOUT = data.wake_timeout * 1000; // Convert seconds to milliseconds
                }
                if (data.language) {
                    this.VOICE_LANG = data.language;
                }
                
                console.log('Voice configuration loaded:', {
                    wakeWords: this.WAKE_WORDS,
                    wakeTimeout: this.WAKE_TIMEOUT,
                    language: this.VOICE_LANG
                });
            }
        } catch (error) {
            console.warn('Failed to load voice config, using defaults:', error);
        }
    }
    
    // Initialize voice recognition
    initializeVoice() {
        // Check if Web Speech API is supported
        if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
            console.warn('Web Speech API not supported');
            if (this.voiceButton) {
                this.voiceButton.disabled = true;
                this.voiceButton.title = '浏览器不支持语音识别';
            }
            return;
        }
        
        this.isVoiceSupported = true;
        
        // First request microphone permission explicitly
        this.requestMicrophonePermission();
    }
    
    // Request microphone permission before initializing speech recognition
    async requestMicrophonePermission() {
        try {
            console.log('Requesting microphone permission...');
            
            // Check if getUserMedia is supported
            if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
                console.warn('getUserMedia not supported, trying direct initialization...');
                console.warn('提示：如果您不是通过 localhost 访问，请使用 https:// 协议或改为 http://localhost:8888 访问');
                this.initializeVoiceRecognition();
                return;
            }
            
            // Request microphone access
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            
            // Permission granted, stop the stream immediately
            stream.getTracks().forEach(track => track.stop());
            
            console.log('Microphone permission granted');
            this.showNotification('麦克风权限已获取，语音功能已启用', 'success');
            
            // Now initialize voice recognition
            this.initializeVoiceRecognition();
            
        } catch (error) {
            console.error('Microphone permission denied:', error);
            
            // Handle permission denial
            if (this.voiceButton) {
                this.voiceButton.disabled = true;
                this.voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
            }
            this.updateVoiceStatus('🚫', '麦克风权限被拒绝');
            this.showNotification('语音功能需要麦克风权限，请在浏览器设置中允许麦克风访问后刷新页面', 'warning');
        }
    }
    
    // Initialize voice recognition after permission is granted
    initializeVoiceRecognition() {
        // Initialize wake word recognition
        this.initializeWakeWordRecognition();
        
        // Start wake word listening automatically
        this.startWakeListening();
        
        console.log('Voice recognition initialized');
    }
    
    // Initialize wake word recognition
    initializeWakeWordRecognition() {
        const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
        
        this.wakeWordRecognition = new SpeechRecognition();
        this.wakeWordRecognition.continuous = true;
        this.wakeWordRecognition.interimResults = true;
        this.wakeWordRecognition.lang = this.VOICE_LANG;
        // 优化参数以提高响应速度
        this.wakeWordRecognition.maxAlternatives = 1; // 减少备选结果，提高速度
        this.wakeWordRecognition.serviceURI = null; // 使用本地识别（如果支持）
        
        // 设置更快的超时时间（如果浏览器支持）
        if ('speechTimeout' in this.wakeWordRecognition) {
            this.wakeWordRecognition.speechTimeout = 1000; // 1秒无语音则触发结果
        }
        if ('speechMaxLength' in this.wakeWordRecognition) {
            this.wakeWordRecognition.speechMaxLength = 5000; // 最长识别5秒
        }
        
        this.wakeWordRecognition.onstart = () => {
            console.log('Wake word recognition started');
            this.updateVoiceStatus('🔊', '等待唤醒词...');
            if (this.voiceButton) this.voiceButton.classList.add('wake-listening');
            if (this.voiceButtonContainer) this.voiceButtonContainer.classList.add('wake-listening');
        };
        
        this.wakeWordRecognition.onresult = (event) => {
            // 立即检查最新的结果，无论是interim还是final
            const latestResult = event.results[event.results.length - 1];
            if (!latestResult) return;
            
            const transcript = latestResult[0].transcript.toLowerCase().trim();
            if (!transcript) return;
            
            // 使用预编译的模式进行快速匹配
            for (const pattern of this.COMPILED_WAKE_PATTERNS) {
                for (const variation of pattern.variations) {
                    if (transcript.includes(variation)) {
                        console.log('Wake word detected immediately:', transcript, 'matched:', variation, 'from:', pattern.original);
                        this.onWakeWordDetected();
                        return; // 找到匹配就立即返回
                    }
                }
            }
            
            // 调试：显示当前识别的内容（仅在开发模式下）
            if (transcript.length > 0 && window.location.hostname === 'localhost') {
                console.debug('Listening:', transcript);
            }
        };
        
        this.wakeWordRecognition.onerror = (event) => {
            console.error('Wake word recognition error:', event.error);
            
            if (event.error === 'not-allowed') {
                // User denied microphone permission
                console.warn('Microphone permission denied by user');
                this.isWakeListening = false;
                if (this.voiceButton) this.voiceButton.classList.remove('wake-listening');
                if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('wake-listening');
                if (this.voiceButton) {
                    this.voiceButton.disabled = true;
                    this.voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
                }
                this.updateVoiceStatus('🚫', '麦克风权限被拒绝');
                
                // Show user-friendly message
                this.showNotification('语音功能需要麦克风权限，请刷新页面并允许麦克风访问', 'warning');
                
            } else if (event.error === 'network') {
                // Try to restart after network error (快速重试)
                setTimeout(() => {
                    if (this.isWakeListening) {
                        this.startWakeListening();
                    }
                }, 500);
            } else if (event.error === 'aborted') {
                // Recognition was aborted, this is normal
                console.log('Wake word recognition was aborted');
            } else {
                // Other errors
                console.error('Unhandled wake word recognition error:', event.error);
                // Don't automatically restart for unknown errors
                this.isWakeListening = false;
                if (this.voiceButton) this.voiceButton.classList.remove('wake-listening');
                if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('wake-listening');
            }
        };
        
        this.wakeWordRecognition.onend = () => {
            console.log('Wake word recognition ended');
            if (this.isWakeListening && !this.isVoiceInputActive && this.isVoiceSupported && 
                this.voiceButton && !this.voiceButton.disabled) {
                // 快速重启以减少延迟 (从1000ms降低到100ms)
                setTimeout(() => {
                    if (this.isWakeListening && this.voiceButton && !this.voiceButton.disabled) {
                        this.startWakeListening();
                    }
                }, 100);
            } else {
                // Clear the wake listening state if we shouldn't restart
                if (this.voiceButton) this.voiceButton.classList.remove('wake-listening');
                if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('wake-listening');
                this.hideVoiceStatus();
            }
        };
    }
    
    // Show notification to user
    showNotification(message, type = 'info') {
        // Clear any existing timeout
        if (this.notificationTimeoutId) {
            clearTimeout(this.notificationTimeoutId);
            this.notificationTimeoutId = null;
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
        this.notificationTimeoutId = setTimeout(() => {
            if (notification) {
                notification.style.transform = 'translateX(100%)';
                // Clean up after animation completes
                setTimeout(() => {
                    if (notification && notification.parentNode) {
                        notification.parentNode.removeChild(notification);
                    }
                }, 300); // Wait for transition to complete
            }
            this.notificationTimeoutId = null;
        }, 3000);
    }
    
    // Start wake word listening
    startWakeListening() {
        if (!this.isVoiceSupported || this.isVoiceInputActive || 
            (this.voiceButton && this.voiceButton.disabled)) return;
        
        try {
            this.isWakeListening = true;
            this.wakeWordRecognition.start();
            console.log('Started wake word listening');
        } catch (error) {
            console.error('Failed to start wake word listening:', error);
            this.isWakeListening = false;
        }
    }
    
    // Stop wake word listening
    stopWakeListening() {
        if (!this.isWakeListening) return;
        
        this.isWakeListening = false;
        try {
            this.wakeWordRecognition.stop();
            if (this.voiceButton) this.voiceButton.classList.remove('wake-listening');
            if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('wake-listening');
            console.log('Stopped wake word listening');
        } catch (error) {
            console.error('Failed to stop wake word listening:', error);
        }
    }
    
    // Handle wake word detection
    onWakeWordDetected() {
        this.stopWakeListening();
        this.startVoiceInput();
    }
    
    // Toggle voice input manually
    toggleVoiceInput() {
        // If button is disabled due to permission issues, try to re-initialize
        if (this.voiceButton && this.voiceButton.disabled) {
            this.retryVoicePermission();
            return;
        }
        
        if (this.isVoiceInputActive) {
            this.stopVoiceInput();
        } else {
            this.stopWakeListening();
            this.startVoiceInput();
        }
    }
    
    // Retry voice permission
    retryVoicePermission() {
        console.log('Retrying voice permission...');
        
        // Reset button state
        if (this.voiceButton) {
            this.voiceButton.disabled = false;
            this.voiceButton.title = '语音输入';
            this.voiceButton.classList.remove('wake-listening');
        }
        if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('wake-listening');
        this.hideVoiceStatus();
        
        // Try to re-request microphone permission
        if (this.isVoiceSupported) {
            this.showNotification('正在重新请求麦克风权限...', 'info');
            // Use the same permission request logic
            this.requestMicrophonePermission();
        }
    }
    
    // Start voice input
    startVoiceInput() {
        if (!this.isVoiceSupported || this.isVoiceInputActive || 
            (this.voiceButton && this.voiceButton.disabled)) return;
        
        const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
        
        this.voiceInputRecognition = new SpeechRecognition();
        this.voiceInputRecognition.continuous = false;
        this.voiceInputRecognition.interimResults = true;
        this.voiceInputRecognition.lang = this.VOICE_LANG;
        // 优化语音输入参数以提高响应速度
        this.voiceInputRecognition.maxAlternatives = 3; // 语音输入需要更多备选以提高准确性
        
        // 设置更快的超时时间（如果浏览器支持）
        if ('speechTimeout' in this.voiceInputRecognition) {
            this.voiceInputRecognition.speechTimeout = 1500; // 1.5秒无语音则触发结果
        }
        if ('speechMaxLength' in this.voiceInputRecognition) {
            this.voiceInputRecognition.speechMaxLength = 10000; // 最长识别10秒
        }
        
        this.voiceInputRecognition.onstart = () => {
            this.isVoiceInputActive = true;
            if (this.voiceButton) {
                this.voiceButton.classList.add('active');
                this.voiceButton.classList.remove('wake-listening');
            }
            if (this.voiceButtonContainer) {
                this.voiceButtonContainer.classList.add('active');
                this.voiceButtonContainer.classList.remove('wake-listening');
            }
            this.updateVoiceStatus('🎤', '正在录音...');
            this.showVoiceStatus();
            
            // 语音激活时恢复聊天布局
            if (this.onVoiceStart && typeof this.onVoiceStart === 'function') {
                this.onVoiceStart();
            }
            
            // Set timeout for voice input
            this.voiceTimeoutId = setTimeout(() => {
                this.stopVoiceInput();
            }, this.WAKE_TIMEOUT);
            
            console.log('Voice input started');
        };
        
        this.voiceInputRecognition.onresult = (event) => {
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
                this.updateVoiceStatus('🎤', `识别中: ${interimTranscript}`);
            }
            
            // Handle final result
            if (finalTranscript.trim()) {
                console.log('Voice input result:', finalTranscript);
                if (this.messageInput) {
                    this.messageInput.value = finalTranscript.trim();
                    
                    // Auto-resize textarea
                    this.messageInput.style.height = 'auto';
                    this.messageInput.style.height = Math.min(this.messageInput.scrollHeight, 120) + 'px';
                }
                this.updateVoiceStatus('✅', `识别完成: ${finalTranscript.trim()}`);
                
                // Auto-send the message after a short delay
                setTimeout(() => {
                    this.hideVoiceStatus();
                    if (finalTranscript.trim() && this.sendMessageCallback) {
                        this.sendMessageCallback();
                    }
                    this.stopVoiceInput();
                }, 1000);
            }
        };
        
        this.voiceInputRecognition.onerror = (event) => {
            console.error('Voice input recognition error:', event.error);
            
            if (event.error === 'not-allowed') {
                // User denied microphone permission
                this.updateVoiceStatus('🚫', '麦克风权限被拒绝');
                if (this.voiceButton) {
                    this.voiceButton.disabled = true;
                    this.voiceButton.title = '麦克风权限被拒绝，请在浏览器设置中允许麦克风访问';
                }
                this.showNotification('语音功能需要麦克风权限，请刷新页面并允许麦克风访问', 'warning');
            } else {
                this.updateVoiceStatus('❌', `识别错误: ${event.error}`);
            }
            
            setTimeout(() => {
                this.stopVoiceInput();
            }, 2000);
        };
        
        this.voiceInputRecognition.onend = () => {
            console.log('Voice input recognition ended');
            this.stopVoiceInput();
        };
        
        try {
            this.voiceInputRecognition.start();
        } catch (error) {
            console.error('Failed to start voice input:', error);
            this.updateVoiceStatus('❌', '启动语音输入失败');
            this.stopVoiceInput();
        }
    }
    
    // Stop voice input
    stopVoiceInput() {
        if (!this.isVoiceInputActive) return;
        
        this.isVoiceInputActive = false;
        if (this.voiceButton) this.voiceButton.classList.remove('active');
        if (this.voiceButtonContainer) this.voiceButtonContainer.classList.remove('active');
        
        if (this.voiceTimeoutId) {
            clearTimeout(this.voiceTimeoutId);
            this.voiceTimeoutId = null;
        }
        
        try {
            if (this.voiceInputRecognition) {
                this.voiceInputRecognition.stop();
            }
        } catch (error) {
            console.error('Failed to stop voice input:', error);
        }
        
        setTimeout(() => {
            this.hideVoiceStatus();
            this.startWakeListening();
            
            // 语音结束时恢复之前的布局
            if (this.onVoiceEnd && typeof this.onVoiceEnd === 'function') {
                this.onVoiceEnd();
            }
        }, 300); // 快速重新开始唤醒监听
        
        console.log('Voice input stopped');
    }
    
    // Update voice status display
    updateVoiceStatus(indicator, text) {
        if (this.voiceIndicator) this.voiceIndicator.textContent = indicator;
        if (this.voiceText) this.voiceText.textContent = text;
    }
    
    // Show voice status
    showVoiceStatus() {
        if (this.voiceStatus) this.voiceStatus.style.display = 'flex';
    }
    
    // Hide voice status
    hideVoiceStatus() {
        if (this.voiceStatus) this.voiceStatus.style.display = 'none';
    }
    
    // Set voice input enabled/disabled
    setVoiceInputEnabled(enabled) {
        if (enabled) {
            if (this.voiceButton) {
                this.voiceButton.disabled = false;
                this.voiceButton.title = '语音输入';
            }
            // 如果没有语音输入活动且没有正在监听唤醒词，重新启动唤醒监听
            if (!this.isVoiceInputActive && !this.isWakeListening && this.isVoiceSupported) {
                console.log('Re-starting wake listening after input enabled');
                this.startWakeListening();
            }
        } else {
            if (this.voiceButton) {
                this.voiceButton.disabled = true;
                this.voiceButton.title = '输入功能已禁用';
            }
        }
        
        if (!enabled && this.isVoiceInputActive) {
            this.stopVoiceInput();
        }
        if (!enabled && this.isWakeListening) {
            this.stopWakeListening();
        }
    }
}

// Create and export voice manager instance
window.VoiceManager = VoiceManager;
