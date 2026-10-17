package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// UserConfig stores all per-user state. The inbox cursor lives under
// Projects (keyed by absolute .xpo path). It is the single file for
// per-user concerns.
type UserConfig struct {
	Projects map[string]ProjectState `yaml:"projects,omitempty"`
}

// ProjectState holds per-user state for a local project, keyed by the
// absolute path to its .xpo directory. Read state is always per-user and
// never shared in .xpo/.
type ProjectState struct {
	LastRead string `yaml:"last_read,omitempty"`
}

func userConfigPath() string {
	if dir := os.Getenv("XPO_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "user.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "xpo", "user.yaml")
	}
	return filepath.Join(home, ".config", "xpo", "user.yaml")
}

// LoadUserConfig reads all stored per-user state.
func LoadUserConfig() UserConfig {
	var c UserConfig
	data, err := os.ReadFile(userConfigPath())
	if err != nil {
		return UserConfig{}
	}
	yaml.Unmarshal(data, &c)
	return c
}

// SaveUserConfig writes all per-user state.
func SaveUserConfig(c UserConfig) error {
	path := userConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// localProjectKey returns the absolute path to the local .xpo directory,
// used as the per-user inbox cursor key.
func localProjectKey() string {
	abs, err := filepath.Abs(".xpo")
	if err != nil {
		return ".xpo"
	}
	return abs
}

// GetInboxLastRead returns the per-user inbox read cursor, stored under the
// absolute .xpo path. A zero time means never read.
func GetInboxLastRead() time.Time {
	raw := LoadUserConfig().Projects[localProjectKey()].LastRead
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SetInboxLastRead persists the per-user inbox read cursor under the
// absolute .xpo path.
func SetInboxLastRead(t time.Time) error {
	c := LoadUserConfig()
	if c.Projects == nil {
		c.Projects = make(map[string]ProjectState)
	}
	key := localProjectKey()
	ps := c.Projects[key]
	ps.LastRead = t.UTC().Format(time.RFC3339)
	c.Projects[key] = ps
	return SaveUserConfig(c)
}
