package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// UserConfig stores all per-user state. Server credentials and the
// distributed-mode inbox cursor live under Servers (keyed by host); the
// local-mode inbox cursor lives under Projects (keyed by absolute .xpo
// path). It is the single file for per-user concerns.
type UserConfig struct {
	Servers  map[string]ServerCredential `yaml:"servers"`
	Projects map[string]ProjectState     `yaml:"projects,omitempty"`
}

// ServerCredential holds auth info and per-user state for a single server.
type ServerCredential struct {
	URL      string `yaml:"url"`
	Token    string `yaml:"token"`
	LastRead string `yaml:"last_read,omitempty"`
}

// ProjectState holds per-user state for a local project, keyed by the
// absolute path to its .xpo directory. Read state is always per-user and
// never shared in .xpo/, even in local mode.
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
		return UserConfig{Servers: make(map[string]ServerCredential)}
	}
	yaml.Unmarshal(data, &c)
	if c.Servers == nil {
		c.Servers = make(map[string]ServerCredential)
	}
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

// SaveServerCredential stores a token for a server URL, preserving any
// existing per-user state (e.g. last_read) for that server.
func SaveServerCredential(serverURL, token string) error {
	c := LoadUserConfig()
	host := hostFromURL(serverURL)
	cred := c.Servers[host]
	cred.URL = serverURL
	cred.Token = token
	c.Servers[host] = cred
	return SaveUserConfig(c)
}

// GetServerCredential retrieves the token for a server URL.
func GetServerCredential(serverURL string) (ServerCredential, bool) {
	c := LoadUserConfig()
	host := hostFromURL(serverURL)
	cred, ok := c.Servers[host]
	return cred, ok
}

// RemoveServerCredential removes a stored server credential.
func RemoveServerCredential(serverURL string) error {
	c := LoadUserConfig()
	host := hostFromURL(serverURL)
	delete(c.Servers, host)
	return SaveUserConfig(c)
}

// ResolveRemote fills in the token for a RemoteConfig by looking up
// the server URL in the user's credential store.
func ResolveRemote(rc RemoteConfig) RemoteConfig {
	if rc.URL == "" {
		return rc
	}
	if rc.Token != "" {
		return rc
	}
	if cred, ok := GetServerCredential(rc.URL); ok {
		rc.Token = cred.Token
	}
	return rc
}

// ReadRemoteURL returns the remote.url from .xpo/config.yaml, if any.
func ReadRemoteURL() string {
	data, err := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if err != nil {
		return ""
	}
	var raw struct {
		Remote struct {
			URL string `yaml:"url"`
		} `yaml:"remote"`
	}
	if yaml.Unmarshal(data, &raw) != nil {
		return ""
	}
	return raw.Remote.URL
}

// IsLocalProjectConfig returns true if .xpo/config.yaml looks like a
// server-side project config (has a prefix field) rather than a minimal
// remote-only config.
func IsLocalProjectConfig() bool {
	data, err := os.ReadFile(filepath.Join(".xpo", "config.yaml"))
	if err != nil {
		return false
	}
	var raw struct {
		Prefix string `yaml:"prefix"`
	}
	if yaml.Unmarshal(data, &raw) != nil {
		return false
	}
	return raw.Prefix != ""
}

// SetRemoteURL writes the remote.url field into .xpo/config.yaml,
// creating the file if it doesn't exist. Preserves existing content.
func SetRemoteURL(serverURL string) error {
	configPath := filepath.Join(".xpo", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config: %w", err)
		}
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

	// Find or create the "remote" mapping node
	var remoteNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "remote" {
			remoteNode = root.Content[i+1]
			break
		}
	}

	if remoteNode == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "remote"},
			&yaml.Node{Kind: yaml.MappingNode},
		)
		remoteNode = root.Content[len(root.Content)-1]
	}

	// Find or create the "url" key inside remote
	found := false
	for i := 0; i < len(remoteNode.Content)-1; i += 2 {
		if remoteNode.Content[i].Value == "url" {
			remoteNode.Content[i+1].Value = serverURL
			found = true
			break
		}
	}

	if !found {
		remoteNode.Content = append(remoteNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "url"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: serverURL},
		)
	}

	// Ensure version field exists (required for data model compatibility checks)
	hasVersion := false
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "version" {
			hasVersion = true
			break
		}
	}
	if !hasVersion {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "version"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "2", Tag: "!!int"},
		)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create .xpo directory: %w", err)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

// localProjectKey returns the absolute path to the local .xpo directory,
// used as the per-user inbox cursor key in local mode.
func localProjectKey() string {
	abs, err := filepath.Abs(".xpo")
	if err != nil {
		return ".xpo"
	}
	return abs
}

// GetInboxLastRead returns the per-user inbox read cursor. In distributed
// mode (remoteURL set) it is stored under the server host; in local mode it
// is stored under the absolute .xpo path. A zero time means never read.
func GetInboxLastRead(remoteURL string) time.Time {
	c := LoadUserConfig()
	var raw string
	if remoteURL != "" {
		raw = c.Servers[hostFromURL(remoteURL)].LastRead
	} else {
		raw = c.Projects[localProjectKey()].LastRead
	}
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SetInboxLastRead persists the per-user inbox read cursor, routing to the
// server host (distributed) or absolute .xpo path (local) as appropriate.
func SetInboxLastRead(remoteURL string, t time.Time) error {
	c := LoadUserConfig()
	stamp := t.UTC().Format(time.RFC3339)
	if remoteURL != "" {
		host := hostFromURL(remoteURL)
		cred := c.Servers[host]
		cred.LastRead = stamp
		c.Servers[host] = cred
	} else {
		if c.Projects == nil {
			c.Projects = make(map[string]ProjectState)
		}
		key := localProjectKey()
		ps := c.Projects[key]
		ps.LastRead = stamp
		c.Projects[key] = ps
	}
	return SaveUserConfig(c)
}

func hostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}
