package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zboya/nala-coder/internal/agent"
	"github.com/zboya/nala-coder/internal/interfaces"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/utils"
)

func runServer() error {
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

	logger.Infof("get config: %+v", config)

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

	// 使用随机未占用端口
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:0", config.Server.Host))
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	// 获取实际分配的端口
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 启动服务器
	go func() {
		logger.Infof("Starting HTTP server on %s", addr)
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start server: %v\n", err)
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	fmt.Printf("Access the web interface at: http://%s\n", addr)
	utils.OpenURL(fmt.Sprintf("http://%s", addr))

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
