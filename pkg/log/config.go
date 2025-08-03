package log

// Config 日志配置
type Config struct {
	Level  string `yaml:"level" json:"level"`   // 日志级别: debug, info, warn, error, fatal, panic
	Output string `yaml:"output" json:"output"` // 输出类型: stdout, file, both
	File   string `yaml:"file" json:"file"`     // 日志文件路径
	Format string `yaml:"format" json:"format"` // 日志格式: text, json
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:  "info",
		Output: "stdout",
		File:   "./logs/app.log",
		Format: "text",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证日志级别
	if _, err := ParseLevel(c.Level); err != nil {
		return err
	}

	// 验证输出类型
	switch c.Output {
	case "stdout", "file", "both":
		// 有效的输出类型
	default:
		c.Output = "stdout" // 默认值
	}

	// 验证格式
	switch c.Format {
	case "text", "json":
		// 有效的格式
	default:
		c.Format = "text" // 默认值
	}

	return nil
}
