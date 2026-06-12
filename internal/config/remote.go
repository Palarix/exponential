package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Credentials stores tokens indexed by server host.
type Credentials struct {
	Servers map[string]ServerCredential `yaml:"servers"`
}

// ServerCredential holds auth info for a single server.
type ServerCredential struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

func credentialsPath() string {
	if dir := os.Getenv("BEATS_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "credentials.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "beats", "credentials.yaml")
	}
	return filepath.Join(home, ".config", "beats", "credentials.yaml")
}

// LoadCredentials reads all stored server credentials.
func LoadCredentials() Credentials {
	var c Credentials
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		return Credentials{Servers: make(map[string]ServerCredential)}
	}
	yaml.Unmarshal(data, &c)
	if c.Servers == nil {
		c.Servers = make(map[string]ServerCredential)
	}
	return c
}

// SaveCredentials writes all server credentials.
func SaveCredentials(c Credentials) error {
	path := credentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// SaveServerCredential stores a token for a server URL.
func SaveServerCredential(serverURL, token string) error {
	c := LoadCredentials()
	host := hostFromURL(serverURL)
	c.Servers[host] = ServerCredential{URL: serverURL, Token: token}
	return SaveCredentials(c)
}

// GetServerCredential retrieves the token for a server URL.
func GetServerCredential(serverURL string) (ServerCredential, bool) {
	c := LoadCredentials()
	host := hostFromURL(serverURL)
	cred, ok := c.Servers[host]
	return cred, ok
}

// RemoveServerCredential removes a stored server credential.
func RemoveServerCredential(serverURL string) error {
	c := LoadCredentials()
	host := hostFromURL(serverURL)
	delete(c.Servers, host)
	return SaveCredentials(c)
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

// ReadRemoteURL returns the remote.url from .beats/config.yaml, if any.
func ReadRemoteURL() string {
	data, err := os.ReadFile(filepath.Join(".beats", "config.yaml"))
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

// IsLocalProjectConfig returns true if .beats/config.yaml looks like a
// server-side project config (has a prefix field) rather than a minimal
// remote-only config.
func IsLocalProjectConfig() bool {
	data, err := os.ReadFile(filepath.Join(".beats", "config.yaml"))
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

// SetRemoteURL writes the remote.url field into .beats/config.yaml,
// creating the file if it doesn't exist. Preserves existing content.
func SetRemoteURL(serverURL string) error {
	configPath := filepath.Join(".beats", "config.yaml")

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
		return fmt.Errorf("failed to create .beats directory: %w", err)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

func hostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}
