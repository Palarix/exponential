package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Prefix     string `mapstructure:"prefix"` // Issue ID prefix, e.g. "myproject-"
	User       string `mapstructure:"user"`   // Override git user, format: "Name <email>"
	Editor     string `mapstructure:"editor"`
	AutoCommit bool   `mapstructure:"auto_commit"`
	Style      Style  `mapstructure:"style"`
	Version    int    `mapstructure:"version"` // Database version
}

type Style struct {
	Theme string `mapstructure:"theme"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Default values
	v.SetDefault("prefix", "beats-") // Default prefix for issue IDs
	v.SetDefault("user", "")         // BEATS_USER env will override
	v.SetDefault("editor", os.Getenv("EDITOR"))
	if v.GetString("editor") == "" {
		v.SetDefault("editor", "vim") // Fallback
	}
	v.SetDefault("auto_commit", false)
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

	// Validate user format if provided
	if cfg.User != "" {
		// Pattern: "One or more chars" followed by space(s), then "<email@domain>"
		re := regexp.MustCompile(`^.+\s+<[^<>]+@[^<>]+>$`)
		if !re.MatchString(cfg.User) {
			return nil, fmt.Errorf("config: invalid user format: expected 'Name <email>', got %q", cfg.User)
		}
	}

	return &cfg, nil
}
