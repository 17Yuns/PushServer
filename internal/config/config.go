package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server     ServerConfig               `mapstructure:"server"`
	Log        LogConfig                  `mapstructure:"log"`
	Recipients map[string]RecipientConfig `mapstructure:"recipients"`
	Queue      QueueConfig                `mapstructure:"queue"`
	Task       TaskConfig                 `mapstructure:"task"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level         string `mapstructure:"level"`
	Format        string `mapstructure:"format"`
	Output        string `mapstructure:"output"`
	FilePath      string `mapstructure:"file_path"`
	ErrorFilePath string `mapstructure:"error_file_path"`
	InfoFilePath  string `mapstructure:"info_file_path"`
	DebugFilePath string `mapstructure:"debug_file_path"`
}

// RecipientConfig 接收者配置
type RecipientConfig struct {
	Name      string                    `mapstructure:"name"`
	Platforms map[string]PlatformConfig `mapstructure:"platforms"`
}

// PlatformConfig 推送平台配置
type PlatformConfig struct {
	Enabled  bool            `mapstructure:"enabled"`
	Webhooks []WebhookConfig `mapstructure:"webhooks"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	URL    string `mapstructure:"url"`
	Secret string `mapstructure:"secret"`
	Name   string `mapstructure:"name"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	WorkerCount              int `mapstructure:"worker_count"`                // 工作协程数量
	BufferSize               int `mapstructure:"buffer_size"`                 // 队列缓冲区大小
	Timeout                  int `mapstructure:"timeout"`                     // 推送超时时间(秒)
	MaxConcurrentPerPlatform int `mapstructure:"max_concurrent_per_platform"` // 每个平台最大并发数
	BatchSize                int `mapstructure:"batch_size"`                  // 批处理大小
}

// TaskConfig 任务状态配置
type TaskConfig struct {
	CleanupInterval int `mapstructure:"cleanup_interval"` // 清理间隔(秒)
	MaxAge          int `mapstructure:"max_age"`          // 任务最大保存时间(秒)
}

var AppConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置环境变量前缀
	viper.SetEnvPrefix("PUSH_SERVER")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败 [%s]: %w", configPath, err)
	}

	// 解析配置到结构体
	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// GetServerAddr 获取服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetRecipient 获取接收者配置
func (c *Config) GetRecipient(alias string) (RecipientConfig, bool) {
	recipient, exists := c.Recipients[alias]
	return recipient, exists
}

// IsPlatformEnabled 检查接收者的平台是否启用
func (c *Config) IsPlatformEnabled(recipientAlias, platform string) bool {
	if recipient, exists := c.Recipients[recipientAlias]; exists {
		if platformConfig, platformExists := recipient.Platforms[platform]; platformExists {
			return platformConfig.Enabled
		}
	}
	return false
}
