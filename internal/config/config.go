package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	MikroTik  MikroTikConfig  `mapstructure:"mikrotik"`
	MCP       MCPConfig       `mapstructure:"mcp"`
	WhatsApp  WhatsAppConfig  `mapstructure:"whatsapp"`
	AI        AIConfig        `mapstructure:"ai"`
	Bot       BotConfig       `mapstructure:"bot"`
	Log       LogConfig       `mapstructure:"log"`
}

type MikroTikConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	Username          string        `mapstructure:"username"`
	Password          string        `mapstructure:"password"`
	UseTLS            bool          `mapstructure:"use_tls"`
	ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
	Timeout           time.Duration `mapstructure:"timeout"`
	PoolSize          int           `mapstructure:"pool_size"` // concurrent connections (default 3)
}

type MCPConfig struct {
	Transport string `mapstructure:"transport"`
	Port      int    `mapstructure:"port"`
	ReadOnly  bool   `mapstructure:"read_only"`
}

type WhatsAppConfig struct {
	GowaURL          string `mapstructure:"gowa_url"`
	GowaDeviceID     string `mapstructure:"gowa_device_id"`
	GowaUsername     string `mapstructure:"gowa_username"`
	GowaPassword     string `mapstructure:"gowa_password"`
	WebhookPort      int    `mapstructure:"webhook_port"`
	WebhookPath      string `mapstructure:"webhook_path"`
	WebhookSecret    string `mapstructure:"webhook_secret"` // HMAC secret untuk X-Hub-Signature-256
}

type AIConfig struct {
	APIKey        string  `mapstructure:"api_key"`
	BaseURL       string  `mapstructure:"base_url"`
	Model         string  `mapstructure:"model"`
	MaxTokens     int     `mapstructure:"max_tokens"`
	Temperature   float64 `mapstructure:"temperature"`
	SystemPrompt  string  `mapstructure:"system_prompt"`
	ThinkingMode  string  `mapstructure:"thinking_mode"` // "enabled" | "disabled" | "" (pakai default server)
}

type AuthUser struct {
	Phone  string `mapstructure:"phone"`
	Name   string `mapstructure:"name"`
	Access string `mapstructure:"access"` // "full" | "readonly"
}

type BotConfig struct {
	MCPServerURL          string        `mapstructure:"mcp_server_url"`
	MaxFunctionCallLoops  int           `mapstructure:"max_function_call_loops"`
	SessionTTL            time.Duration `mapstructure:"session_ttl"`
	MaxHistoryMessages    int           `mapstructure:"max_history_messages"`
	AuthorizedUsers       []AuthUser    `mapstructure:"authorized_users"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("mikrotik.port", 8728)
	v.SetDefault("mikrotik.pool_size", 3)
	v.SetDefault("mikrotik.use_tls", false)
	v.SetDefault("mikrotik.reconnect_interval", 5*time.Second)
	v.SetDefault("mikrotik.timeout", 10*time.Second)
	v.SetDefault("mcp.transport", "stdio")
	v.SetDefault("mcp.port", 8080)
	v.SetDefault("mcp.read_only", false)
	v.SetDefault("whatsapp.gowa_url", "http://localhost:3000")
	v.SetDefault("whatsapp.webhook_port", 8090)
	v.SetDefault("whatsapp.webhook_path", "/webhook/message")
	v.SetDefault("ai.base_url", "https://api.z.ai/api/paas/v4")
	v.SetDefault("ai.model", "glm-4-airx")
	v.SetDefault("ai.max_tokens", 1024)
	v.SetDefault("ai.temperature", 0.7)
	v.SetDefault("bot.mcp_server_url", "http://localhost:8080")
	v.SetDefault("bot.max_function_call_loops", 5)
	v.SetDefault("bot.session_ttl", 2*time.Hour)
	v.SetDefault("bot.max_history_messages", 20)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// Allow env overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("MIKROTIK")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
