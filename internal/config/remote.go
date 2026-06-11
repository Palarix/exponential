package config

import (
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

func hostFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}
