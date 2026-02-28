package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	MikroTik MikroTikConfig `mapstructure:"mikrotik"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	Log      LogConfig      `mapstructure:"log"`
}

type MikroTikConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	Username          string        `mapstructure:"username"`
	Password          string        `mapstructure:"password"`
	UseTLS            bool          `mapstructure:"use_tls"`
	ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

type MCPConfig struct {
	Transport string `mapstructure:"transport"`
	Port      int    `mapstructure:"port"`
	ReadOnly  bool   `mapstructure:"read_only"`
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
	v.SetDefault("mikrotik.use_tls", false)
	v.SetDefault("mikrotik.reconnect_interval", 5*time.Second)
	v.SetDefault("mikrotik.timeout", 10*time.Second)
	v.SetDefault("mcp.transport", "stdio")
	v.SetDefault("mcp.port", 8080)
	v.SetDefault("mcp.read_only", false)
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
