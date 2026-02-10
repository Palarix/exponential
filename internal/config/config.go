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
	Prefix           string      `mapstructure:"prefix"`
	User             string      `mapstructure:"user"`
	Editor           string      `mapstructure:"editor"`
	AutoCommit       bool        `mapstructure:"auto_commit"`
	Style            Style       `mapstructure:"style"`
	Version          int         `mapstructure:"version"`
	EstimationSystem string      `mapstructure:"estimation_system"`
	CountUnestimated bool        `mapstructure:"count_unestimated"`
	Automations      Automations `mapstructure:"automations"`
}

type Style struct {
	Theme string `mapstructure:"theme"`
}

type Automations struct {
	AutoCompleteParent    bool `mapstructure:"auto_complete_parent"`
	AutoCloseSubIssues    bool `mapstructure:"auto_close_sub_issues"`
	AutoProgressSubIssues bool `mapstructure:"auto_progress_sub_issues"`
	AutoProgressParent    bool `mapstructure:"auto_progress_parent"`
}

// Estimation system allowed values
var EstimationSystems = map[string][]int{
	"fibonacci":   {1, 2, 3, 5, 8},
	"exponential": {1, 2, 4, 8, 16},
	"linear":      {1, 2, 3, 4, 5},
	"shirt":       {1, 2, 3, 5, 8},
}

// ShirtLabels maps shirt-size estimate values to display labels
var ShirtLabels = map[int]string{
	1: "XS",
	2: "S",
	3: "M",
	5: "L",
	8: "XL",
}

// ShirtValues maps shirt-size display labels to estimate values
var ShirtValues = map[string]int{
	"XS": 1,
	"S":  2,
	"M":  3,
	"L":  5,
	"XL": 8,
}

// ValidateEstimate checks if a value is valid for the given estimation system.
func ValidateEstimate(system string, value int) error {
	if value == 0 {
		return fmt.Errorf("zero estimates are not allowed")
	}
	allowed, ok := EstimationSystems[system]
	if !ok {
		return fmt.Errorf("unknown estimation system: %s", system)
	}
	for _, v := range allowed {
		if v == value {
			return nil
		}
	}
	return fmt.Errorf("estimate %d is not valid for %s system (allowed: %v)", value, system, allowed)
}

// EstimateDisplayValue returns the display string for an estimate value.
// For shirt system, returns the shirt label; for others, returns the number as string.
func EstimateDisplayValue(system string, value int) string {
	if system == "shirt" {
		if label, ok := ShirtLabels[value]; ok {
			return label
		}
	}
	return fmt.Sprintf("%d", value)
}

// ParseEstimateInput parses user input into an int estimate value.
// For shirt system, accepts both numeric and shirt-size labels (XS, S, M, L, XL).
func ParseEstimateInput(system string, input string) (int, error) {
	// Try shirt label first
	if system == "shirt" {
		upper := strings.ToUpper(strings.TrimSpace(input))
		if val, ok := ShirtValues[upper]; ok {
			return val, nil
		}
	}
	// Try as integer
	var val int
	if _, err := fmt.Sscanf(input, "%d", &val); err != nil {
		return 0, fmt.Errorf("invalid estimate value: %s", input)
	}
	return val, nil
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Default values
	v.SetDefault("prefix", "beats-")
	v.SetDefault("user", "")
	v.SetDefault("editor", os.Getenv("EDITOR"))
	if v.GetString("editor") == "" {
		v.SetDefault("editor", "vim")
	}
	v.SetDefault("auto_commit", false)
	v.SetDefault("style.theme", "default")
	v.SetDefault("estimation_system", "fibonacci")
	v.SetDefault("count_unestimated", true)
	v.SetDefault("automations.auto_complete_parent", false)
	v.SetDefault("automations.auto_close_sub_issues", false)
	v.SetDefault("automations.auto_progress_sub_issues", false)
	v.SetDefault("automations.auto_progress_parent", false)

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
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate user format if provided
	if cfg.User != "" {
		re := regexp.MustCompile(`^.+\s+<[^<>]+@[^<>]+>$`)
		if !re.MatchString(cfg.User) {
			return nil, fmt.Errorf("config: invalid user format: expected 'Name <email>', got %q", cfg.User)
		}
	}

	// Validate estimation system
	if _, ok := EstimationSystems[cfg.EstimationSystem]; !ok {
		return nil, fmt.Errorf("config: invalid estimation_system: %q (allowed: fibonacci, exponential, linear, shirt)", cfg.EstimationSystem)
	}

	return &cfg, nil
}
