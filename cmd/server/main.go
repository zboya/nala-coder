package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zboya/nala-coder/internal/agent"
	"github.com/zboya/nala-coder/internal/interfaces"
	"github.com/zboya/nala-coder/pkg/log"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 初始化配置
	if err := initConfig(); err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	// 创建logger
	logger, err := log.NewFromViper()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// 解析配置
	var config agent.AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 创建Agent构建器
	builder := agent.NewBuilder(&config, logger)

	// 构建Agent
	agentInstance, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// 设置Gin模式
	if config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建HTTP服务器
	server := interfaces.NewHTTPServer(agentInstance, logger, config.Speech)
	router := server.SetupRoutes()

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 启动服务器
	go func() {
		logger.Infof("Starting HTTP server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start server: %v\n", err)
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	fmt.Printf("Access the web interface at: http://%s\n", addr)
	logger.Infof("Access the web interface at: http://%s", addr)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
	return nil
}

// initConfig 初始化配置
func initConfig() error {
	// 查找配置文件
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	viper.AddConfigPath(filepath.Join(cwd, "configs"))
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 环境变量支持
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认配置
			setDefaultConfig()
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return nil
}

// setDefaultConfig 设置默认配置
func setDefaultConfig() {
	viper.SetDefault("server.port", "8888")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("llm.default_provider", "openai")
	viper.SetDefault("agent.max_loops", 10)
	viper.SetDefault("agent.context_window", 32000)
	viper.SetDefault("agent.compression_threshold", 0.9)
	viper.SetDefault("tools.max_concurrency", 10)
	viper.SetDefault("context.history_limit", 6)
	viper.SetDefault("context.storage_path", "./storage")
	viper.SetDefault("context.persistence_file", "CODE_AGENT.md")
	viper.SetDefault("prompts.directory", "./prompts")
	viper.SetDefault("prompts.hot_reload", true)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("speech.enabled", false)
	viper.SetDefault("speech.provider", "baidu")
	viper.SetDefault("speech.timeout", 30)
}
