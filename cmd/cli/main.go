package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zboya/nala-coder/internal/agent"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

var (
	configFile string
	verbose    bool
	sessionID  string
)

// rootCmd CLI根命令
var rootCmd = &cobra.Command{
	Use:   "nala-coder",
	Short: "NaLa Coder - AI-powered coding assistant",
	Long: `NaLa Coder is an intelligent programming assistant powered by large language models.
It supports multiple LLM providers, rich tool ecosystem, and smart context management.`,
}

// chatCmd 聊天命令
var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Start a chat conversation with the AI agent",
	Long: `Start an interactive chat conversation with the AI agent.
You can provide a message directly or start an interactive session.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChat,
}

// serverCmd 服务器命令
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the HTTP API server",
	Long:  "Start the HTTP API server for web-based interactions",
	RunE:  runServer,
}

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./configs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// 聊天命令标志
	chatCmd.Flags().StringVar(&sessionID, "session", "", "session ID for conversation continuity")

	// 添加子命令
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(serverCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runChat 运行聊天命令
func runChat(cmd *cobra.Command, args []string) error {
	// 初始化配置和Agent
	agentInstance, logger, err := initializeAgent()
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	ctx := context.Background()

	if len(args) > 0 {
		// 单次对话模式
		message := args[0]
		return handleSingleChat(ctx, agentInstance, logger, message)
	} else {
		// 交互式对话模式
		return handleInteractiveChat(ctx, agentInstance, logger)
	}
}

// runServer 运行服务器命令
func runServer(cmd *cobra.Command, args []string) error {
	// 这里会启动HTTP服务器，后面实现
	fmt.Println("Starting HTTP API server...")
	return fmt.Errorf("server command not implemented yet")
}

// initializeAgent 初始化Agent
func initializeAgent() (*agent.Agent, log.Logger, error) {
	// 初始化配置
	if err := initConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to init config: %w", err)
	}

	// 创建logger
	logger, err := log.NewFromViperWithVerbose(verbose)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// 解析配置
	var config agent.AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	logger.Infof("config: %+v", config)

	// 创建Agent构建器
	builder := agent.NewBuilder(&config, logger)

	// 构建Agent
	agentInstance, err := builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build agent: %w", err)
	}

	logger.Info("Agent initialized successfully")
	return agentInstance, logger, nil
}

// initConfig 初始化配置
func initConfig() error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// 查找配置文件
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		viper.AddConfigPath(filepath.Join(cwd, "configs"))
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 环境变量支持
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}

// handleSingleChat 处理单次对话
func handleSingleChat(ctx context.Context, agent *agent.Agent, logger log.Logger, message string) error {
	query := fmt.Sprintf("<user_query>\n%s\n</user_query>", message)
	request := types.ChatRequest{
		Message:   query,
		SessionID: sessionID,
		Stream:    false,
	}

	response, err := agent.Chat(ctx, request)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	fmt.Printf("AI: %s\n", response.Response)

	if verbose {
		fmt.Printf("\nSession ID: %s\n", response.SessionID)
		fmt.Printf("Usage: %d tokens\n", response.Usage.TotalTokens)
	}

	return nil
}

// handleInteractiveChat 处理交互式对话
func handleInteractiveChat(ctx context.Context, agent *agent.Agent, logger log.Logger) error {
	fmt.Println("NaLa Coder - Interactive Chat Mode")
	fmt.Println("Type 'exit' or 'quit' to end the conversation")
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	currentSessionID := sessionID
	if currentSessionID == "" {
		currentSessionID = utils.GenerateID()
		fmt.Printf("Started new session: %s\n\n", currentSessionID)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return nil
		}
		input = strings.TrimSpace(input)

		// 处理特殊命令
		switch input {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil
		case "help":
			printHelp()
			continue
		case "session":
			fmt.Printf("Current session: %s\n", currentSessionID)
			continue
		case "new":
			currentSessionID = utils.GenerateID()
			fmt.Printf("Started new session: %s\n", currentSessionID)
			continue
		}

		if input == "" {
			continue
		}

		// 发送消息给Agent
		query := fmt.Sprintf("<user_query>\n%s\n</user_query>", input)
		request := types.ChatRequest{
			Message:   query,
			SessionID: currentSessionID,
			Stream:    true,
		}

		stream, err := agent.ChatStream(ctx, request)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Print("AI: ")
		for response := range stream {
			if response.Response != "" {
				fmt.Print(response.Response)
			}

			if response.Finished {
				fmt.Println()
				if verbose && response.Usage.TotalTokens > 0 {
					fmt.Printf("(Used %d tokens)\n", response.Usage.TotalTokens)
				}
				break
			}
		}
		fmt.Println()
	}
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help     - Show this help message")
	fmt.Println("  session  - Show current session ID")
	fmt.Println("  new      - Start a new session")
	fmt.Println("  exit     - Exit the chat")
	fmt.Println("  quit     - Exit the chat")
	fmt.Println()
}
