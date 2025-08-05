package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zboya/nala-coder/internal/agent"
	"github.com/zboya/nala-coder/pkg/log"
	"github.com/zboya/nala-coder/pkg/types"
	"github.com/zboya/nala-coder/pkg/utils"
)

func runChat(cmd *cobra.Command, args []string) error {
	// 交互式对话模式
	return handleInteractiveChat()
}

// handleInteractiveChat 处理交互式对话
func handleInteractiveChat() error {
	// 初始化配置和Agent
	agent, _, err := initializeAgent()
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

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

		ctx := context.Background()
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
