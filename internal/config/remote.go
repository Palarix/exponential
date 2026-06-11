package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const remoteConfigFile = ".beats/remote.yaml"

// LoadRemoteConfig reads the remote config from .beats/remote.yaml.
// Returns a zero-value RemoteConfig if the file doesn't exist.
func LoadRemoteConfig() RemoteConfig {
	var rc RemoteConfig
	data, err := os.ReadFile(remoteConfigFile)
	if err != nil {
		return rc
	}
	yaml.Unmarshal(data, &rc)
	return rc
}

// SaveRemoteConfig writes the remote config to .beats/remote.yaml
// and ensures it's listed in .beats/.gitignore.
func SaveRemoteConfig(rc RemoteConfig) error {
	data, err := yaml.Marshal(rc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(remoteConfigFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(remoteConfigFile, data, 0600); err != nil {
		return err
	}
	ensureGitignored("remote.yaml")
	ensureGitignored("server.key")
	return nil
}

// ClearRemoteConfig removes the remote config file.
func ClearRemoteConfig() error {
	err := os.Remove(remoteConfigFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func ensureGitignored(pattern string) {
	gitignorePath := ".beats/.gitignore"
	data, _ := os.ReadFile(gitignorePath)
	content := string(data)
	for _, line := range splitLines(content) {
		if line == pattern {
			return
		}
	}
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}
	content += pattern + "\n"
	os.WriteFile(gitignorePath, []byte(content), 0644)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
