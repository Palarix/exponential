package auth

import (
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHKey represents a discovered SSH key available for authentication.
type SSHKey struct {
	Path    string     // file path, empty for agent keys
	Comment string     // key comment
	Signer  ssh.Signer // ready-to-use signer
}

// Label returns a human-readable description of the key.
func (k SSHKey) Label() string {
	algo := k.Signer.PublicKey().Type()
	if k.Comment != "" {
		return fmt.Sprintf("%s (%s)", k.Comment, algo)
	}
	if k.Path != "" {
		return fmt.Sprintf("%s (%s)", k.Path, algo)
	}
	return algo
}

// DiscoverSSHKeys finds available SSH keys from standard file locations
// and the ssh-agent.
func DiscoverSSHKeys() ([]SSHKey, error) {
	var keys []SSHKey

	home, err := os.UserHomeDir()
	if err == nil {
		candidates := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
		}
		for _, path := range candidates {
			key, err := loadKeyFromFile(path)
			if err == nil {
				keys = append(keys, *key)
			}
		}
	}

	agentKeys, _ := discoverAgentKeys()
	keys = append(keys, agentKeys...)

	return keys, nil
}

func loadKeyFromFile(path string) (*SSHKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	comment := filepath.Base(path)
	pub := signer.PublicKey()
	switch pub.Type() {
	case "ssh-ed25519":
		comment = fmt.Sprintf("%s (ed25519)", filepath.Base(path))
	case "ssh-rsa":
		if cryptoPub, ok := pub.(ssh.CryptoPublicKey); ok {
			if rsaPub, ok := cryptoPub.CryptoPublicKey().(*rsa.PublicKey); ok {
				comment = fmt.Sprintf("%s (rsa-%d)", filepath.Base(path), rsaPub.N.BitLen())
			}
		}
	}

	return &SSHKey{
		Path:    path,
		Comment: comment,
		Signer:  signer,
	}, nil
}

func discoverAgentKeys() ([]SSHKey, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ag := agent.NewClient(conn)
	signers, err := ag.Signers()
	if err != nil {
		return nil, err
	}

	agentKeyList, err := ag.List()
	if err != nil {
		agentKeyList = nil
	}

	commentMap := make(map[string]string)
	for _, ak := range agentKeyList {
		commentMap[string(ak.Marshal())] = ak.Comment
	}

	var keys []SSHKey
	for _, s := range signers {
		comment := commentMap[string(s.PublicKey().Marshal())]
		if comment == "" {
			comment = "ssh-agent key"
		}
		keys = append(keys, SSHKey{
			Comment: comment,
			Signer:  s,
		})
	}

	return keys, nil
}

// FindKeyByType returns the first key matching the given SSH key type
// (e.g. "ssh-ed25519", "ssh-rsa").
func FindKeyByType(keys []SSHKey, keyType string) *SSHKey {
	for i, k := range keys {
		if k.Signer.PublicKey().Type() == keyType {
			return &keys[i]
		}
	}
	return nil
}

// PreferEd25519 returns the ed25519 key if available, otherwise the first key.
func PreferEd25519(keys []SSHKey) *SSHKey {
	if k := FindKeyByType(keys, "ssh-ed25519"); k != nil {
		return k
	}
	if len(keys) > 0 {
		return &keys[0]
	}
	return nil
}
