package auth

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type keyEntry struct {
	publicKey ssh.PublicKey
	identity  UserIdentity
}

// AuthorizedKeys holds parsed SSH public keys and their associated identities.
type AuthorizedKeys struct {
	entries []keyEntry
}

// LoadOrCreateAuthorizedKeys loads the authorized_keys file, creating it
// with a comment header if it doesn't exist.
func LoadOrCreateAuthorizedKeys(path string) (*AuthorizedKeys, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		seed := "# xpo authorized_keys — add SSH public keys here, one per line.\n" +
			"# Format: ssh-ed25519 AAAA... user@example.com\n"
		if err := os.WriteFile(path, []byte(seed), 0644); err != nil {
			return nil, fmt.Errorf("create authorized_keys: %w", err)
		}
	}
	return LoadAuthorizedKeys(path)
}

// LoadAuthorizedKeys parses an OpenSSH authorized_keys file.
// Each line's comment field is parsed as the user identity.
func LoadAuthorizedKeys(path string) (*AuthorizedKeys, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read authorized_keys: %w", err)
	}

	ak := &AuthorizedKeys{}
	rest := data
	for len(rest) > 0 {
		pubKey, comment, _, remaining, err := ssh.ParseAuthorizedKey(rest)
		if err != nil {
			// Skip unparseable lines (comments, blank lines)
			idx := bytes.IndexByte(rest, '\n')
			if idx == -1 {
				break
			}
			rest = remaining
			if len(rest) == 0 {
				rest = data[len(data)-len(remaining):]
				idx2 := bytes.IndexByte(rest, '\n')
				if idx2 == -1 {
					break
				}
				rest = rest[idx2+1:]
			}
			continue
		}
		rest = remaining
		ak.entries = append(ak.entries, keyEntry{
			publicKey: pubKey,
			identity:  ParseIdentity(comment),
		})
	}

	return ak, nil
}

// Lookup finds the identity associated with the given public key.
func (ak *AuthorizedKeys) Lookup(pubKey ssh.PublicKey) (UserIdentity, bool) {
	needle := pubKey.Marshal()
	for _, e := range ak.entries {
		if bytes.Equal(e.publicKey.Marshal(), needle) {
			return e.identity, true
		}
	}
	return UserIdentity{}, false
}

// Len returns the number of authorized keys.
func (ak *AuthorizedKeys) Len() int {
	return len(ak.entries)
}
