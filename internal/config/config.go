package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Editor string `mapstructure:"editor"`
	Style  Style  `mapstructure:"style"`
}

type Style struct {
	Theme string `mapstructure:"theme"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Default values
	v.SetDefault("editor", os.Getenv("EDITOR"))
	if v.GetString("editor") == "" {
		v.SetDefault("editor", "vim") // Fallback
	}
	v.SetDefault("style.theme", "default")

	// Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".beats") // Project level

	// User level config (~/.config/beats)
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "beats"))
	}

	// Environment variables
	v.SetEnvPrefix("BEATS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is fine, we use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
