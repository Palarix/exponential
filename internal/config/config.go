package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Name              string            `mapstructure:"name" yaml:"name"`
	Prefix            string            `mapstructure:"prefix" yaml:"prefix"`
	User              string            `mapstructure:"user" yaml:"user"`
	DefaultLabels     []string          `mapstructure:"default_labels" yaml:"default_labels"`
	HideDefaultLabels bool              `mapstructure:"hide_default_labels" yaml:"hide_default_labels"`
	Editor            string            `mapstructure:"editor" yaml:"editor"`
	AutoCommit        bool              `mapstructure:"auto_commit" yaml:"auto_commit"`
	Style             Style             `mapstructure:"style" yaml:"style"`
	Version           int               `mapstructure:"version" yaml:"version"`
	EstimationSystem  string            `mapstructure:"estimation_system" yaml:"estimation_system"`
	CountUnestimated  bool              `mapstructure:"count_unestimated" yaml:"count_unestimated"`
	Automations       Automations       `mapstructure:"automations" yaml:"automations"`
	Labels            map[string]string `mapstructure:"labels" yaml:"labels"`
	Cycles            CycleConfig       `mapstructure:"cycles" yaml:"cycles"`
	Contributors      []string          `mapstructure:"contributors" yaml:"contributors"`
	Remote            RemoteConfig      `mapstructure:"remote" yaml:"remote"`
	Permissions       PermissionsConfig `mapstructure:"permissions" yaml:"permissions,omitempty"`
	Drive             DriveConfig       `mapstructure:"drive" yaml:"drive,omitempty"`
}

type DriveConfig struct {
	Supervisor string `mapstructure:"supervisor" yaml:"supervisor"`
	Coder      string `mapstructure:"coder" yaml:"coder"`
	MaxRetries int    `mapstructure:"max_retries" yaml:"max_retries"`
	TestCmd    string `mapstructure:"test_cmd" yaml:"test_cmd"`
	Timeout    string `mapstructure:"timeout" yaml:"timeout"`
}

type PermissionsConfig struct {
	Roles map[string][]string `mapstructure:"roles" yaml:"roles,omitempty"`
	Users map[string]string   `mapstructure:"users" yaml:"users,omitempty"`
}

// RoleFor returns the role name for the given email. Returns "" if no
// permissions are configured or the user has no explicit mapping.
func (p PermissionsConfig) RoleFor(email string) string {
	if len(p.Users) == 0 {
		return ""
	}
	return p.Users[strings.ToLower(email)]
}

// Capabilities returns the capability list for a role name, expanding
// the wildcard "*" into all known capabilities.
func (p PermissionsConfig) Capabilities(role string) []string {
	caps, ok := p.Roles[role]
	if !ok {
		return nil
	}
	for _, c := range caps {
		if c == "*" {
			return AllCapabilities
		}
	}
	return caps
}

// HasCapability reports whether the given email is allowed to perform
// the named capability. Returns true when no permissions are configured
// (open-access default) or when the user's role includes the capability
// or the wildcard.
func (p PermissionsConfig) HasCapability(email, capability string) bool {
	if len(p.Roles) == 0 || len(p.Users) == 0 {
		return true
	}
	role := p.RoleFor(email)
	if role == "" {
		return false
	}
	for _, c := range p.Capabilities(role) {
		if c == capability || c == "*" {
			return true
		}
	}
	return false
}

// Enabled reports whether a permissions section is configured.
func (p PermissionsConfig) Enabled() bool {
	return len(p.Roles) > 0 && len(p.Users) > 0
}

// AllCapabilities is the canonical list of capabilities.
var AllCapabilities = []string{
	"issue.read",
	"issue.create",
	"issue.update",
	"issue.comment",
	"issue.start",
	"issue.merge",
	"issue.delete",
	"issue.link",
}

type RemoteConfig struct {
	URL   string `mapstructure:"url" yaml:"url"`
	Token string `mapstructure:"token" yaml:"token"`
}

type CycleConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Duration   string `mapstructure:"duration" yaml:"duration"`
	StartDay   string `mapstructure:"start_day" yaml:"start_day"`
	AnchorDate string `mapstructure:"anchor_date" yaml:"anchor_date"`
}

type Style struct {
	Theme string `mapstructure:"theme" yaml:"theme"`
}

type Automations struct {
	FirstStart    bool `mapstructure:"first_start" yaml:"first_start"`
	LastCompleted bool `mapstructure:"last_completed" yaml:"last_completed"`
}

// BuiltinLabels are the default labels seeded into new projects.
// Config labels take precedence over these when both exist.
var BuiltinLabels = map[string]string{
	"bug":         "#eb5757",
	"feature":     "#b36cd9",
	"epic":        "#5e6ad2",
	"improvement": "#4da6e8",
}

// BuiltinLabelOrder defines the canonical display order for built-in labels.
var BuiltinLabelOrder = []string{"bug", "feature", "epic", "improvement"}

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

// AddLabel writes a label and its color to the project config file,
// preserving existing formatting via yaml.Node manipulation.
func AddLabel(name, color string) error {
	configPath := filepath.Join(".xpo", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		data = []byte{}
	}

	var doc yaml.Node
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}

	root := doc.Content[0]

	var labelsNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "labels" {
			labelsNode = root.Content[i+1]
			break
		}
	}

	if labelsNode == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "labels"},
			&yaml.Node{Kind: yaml.MappingNode},
		)
		labelsNode = root.Content[len(root.Content)-1]
	}

	found := false
	for i := 0; i < len(labelsNode.Content)-1; i += 2 {
		if labelsNode.Content[i].Value == name {
			labelsNode.Content[i+1].Value = color
			found = true
			break
		}
	}

	if !found {
		labelsNode.Content = append(labelsNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: name},
			&yaml.Node{Kind: yaml.ScalarNode, Value: color, Style: yaml.DoubleQuotedStyle},
		)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

// DeleteLabel removes a label from the project config file.
func DeleteLabel(name string) error {
	configPath := filepath.Join(".xpo", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if doc.Kind == 0 || len(doc.Content) == 0 {
		return nil
	}

	root := doc.Content[0]
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "labels" {
			labelsNode := root.Content[i+1]
			for j := 0; j < len(labelsNode.Content)-1; j += 2 {
				if labelsNode.Content[j].Value == name {
					labelsNode.Content = append(labelsNode.Content[:j], labelsNode.Content[j+2:]...)
					break
				}
			}
			break
		}
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

// UpdateLabel renames a label and/or changes its color in the project config file.
func UpdateLabel(oldName, newName, color string) error {
	configPath := filepath.Join(".xpo", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if doc.Kind == 0 || len(doc.Content) == 0 {
		return fmt.Errorf("label %q not found", oldName)
	}

	root := doc.Content[0]
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "labels" {
			labelsNode := root.Content[i+1]
			for j := 0; j < len(labelsNode.Content)-1; j += 2 {
				if labelsNode.Content[j].Value == oldName {
					labelsNode.Content[j].Value = newName
					labelsNode.Content[j+1].Value = color
					break
				}
			}
			break
		}
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Default values
	v.SetDefault("prefix", "issue-")
	v.SetDefault("user", "")
	v.SetDefault("editor", os.Getenv("EDITOR"))
	if v.GetString("editor") == "" {
		v.SetDefault("editor", "vim")
	}
	v.SetDefault("auto_commit", false)
	v.SetDefault("style.theme", "default")
	v.SetDefault("estimation_system", "fibonacci")
	v.SetDefault("count_unestimated", true)
	v.SetDefault("automations.first_start", true)
	v.SetDefault("automations.last_completed", true)
	v.SetDefault("drive.supervisor", "claude")
	v.SetDefault("drive.coder", "claude")
	v.SetDefault("drive.max_retries", 3)
	v.SetDefault("drive.timeout", "30m")

	// Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".xpo") // Project level

	// User level config (~/.config/xpo)
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "xpo"))
	}

	// Environment variables
	v.SetEnvPrefix("XPO")
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

	// Validate contributors format
	if len(cfg.Contributors) > 0 {
		re := regexp.MustCompile(`^.+\s+<[^<>]+@[^<>]+>$`)
		for i, c := range cfg.Contributors {
			if !re.MatchString(c) {
				return nil, fmt.Errorf("config: invalid contributors[%d] format: expected 'Name <email>', got %q", i, c)
			}
		}
	}

	// Validate estimation system
	if _, ok := EstimationSystems[cfg.EstimationSystem]; !ok {
		return nil, fmt.Errorf("config: invalid estimation_system: %q (allowed: fibonacci, exponential, linear, shirt)", cfg.EstimationSystem)
	}

	// Validate cycle config
	if cfg.Cycles.Enabled {
		if err := cfg.Cycles.Validate(); err != nil {
			return nil, fmt.Errorf("config: cycles: %w", err)
		}
	}

	// Re-read labels from YAML directly to preserve key casing (Viper lowercases all keys)
	if configFile := v.ConfigFileUsed(); configFile != "" {
		if raw, err := os.ReadFile(configFile); err == nil {
			var rawCfg struct {
				Labels map[string]string `yaml:"labels"`
			}
			if err := yaml.Unmarshal(raw, &rawCfg); err == nil && rawCfg.Labels != nil {
				cfg.Labels = rawCfg.Labels
			}
		}
	}

	if len(cfg.DefaultLabels) == 0 {
		cfg.DefaultLabels = BuiltinLabelOrder
	}

	// Resolve remote token from ~/.config/xpo/user.yaml
	cfg.Remote = ResolveRemote(cfg.Remote)

	return &cfg, nil
}
