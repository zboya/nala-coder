package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	verbose    bool
	sessionID  string
)

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./configs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	// 聊天命令标志
	chatCmd.Flags().StringVar(&sessionID, "session", "", "session ID for conversation continuity")
}

// rootCmd CLI根命令
var rootCmd = &cobra.Command{
	Use:   "nala-coder",
	Short: "NaLa Coder - AI-powered coding assistant",
	Long: `NaLa Coder is an intelligent programming assistant powered by large language models.
It supports multiple LLM providers, rich tool ecosystem, and smart context management.`,
	RunE: run,
}

// chatCmd 聊天命令
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat conversation with the AI agent",
	Long: `Start an interactive chat conversation with the AI agent.
You can provide a message directly or start an interactive session.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChat,
}

func init() {
	// 添加子命令
	rootCmd.AddCommand(chatCmd)
}

// initConfig 初始化配置
func initConfig() error {
	// 查找配置文件
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()

	viper.AddConfigPath(".")                                // 当前目录
	viper.AddConfigPath(filepath.Join(home, ".nala-coder")) // 用户目录下的nala-coder文件夹
	viper.AddConfigPath(filepath.Join(cwd, "configs"))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("NALA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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
	home, _ := os.UserHomeDir()
	viper.SetDefault("server.port", "8888")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("llm.default_provider", "deepseek")
	viper.SetDefault("agent.max_loops", 50)
	viper.SetDefault("agent.context_window", 32000)
	viper.SetDefault("agent.compression_threshold", 0.9)
	viper.SetDefault("tools.max_concurrency", 10)
	viper.SetDefault("context.history_limit", 6)
	viper.SetDefault("context.storage_path", filepath.Join(home, ".nala-coder", "storage"))
	viper.SetDefault("context.persistence_file", "CODE_AGENT.md")
	viper.SetDefault("prompts.directory", filepath.Join(home, ".nala-coder", "prompts"))
	viper.SetDefault("prompts.hot_reload", true)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("speech.enabled", true)
	viper.SetDefault("speech.timeout", 30)
}

// run 运行命令
func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := runServer()
		if err != nil {
			return fmt.Errorf("failed to run server: %w", err)
		}
		return nil
	}
	return errors.New("unknown command: " + args[0])
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
