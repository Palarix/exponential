package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// NonceStore is a thread-safe in-memory store for authentication nonces.
// Nonces are single-use and expire after a configurable TTL.
type NonceStore struct {
	mu     sync.Mutex
	nonces map[string]time.Time
	ttl    time.Duration
}

// NewNonceStore creates a new nonce store with the given TTL.
func NewNonceStore(ttl time.Duration) *NonceStore {
	return &NonceStore{
		nonces: make(map[string]time.Time),
		ttl:    ttl,
	}
}

// Generate creates a new nonce and returns it as a base64-encoded string.
func (ns *NonceStore) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	nonce := base64.StdEncoding.EncodeToString(b)

	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.cleanup()
	ns.nonces[nonce] = time.Now().Add(ns.ttl)
	return nonce, nil
}

// Validate checks whether the nonce is valid (exists and not expired).
// A valid nonce is consumed (deleted) to prevent replay.
func (ns *NonceStore) Validate(nonce string) bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	expiry, ok := ns.nonces[nonce]
	if !ok {
		return false
	}
	delete(ns.nonces, nonce)
	return time.Now().Before(expiry)
}

func (ns *NonceStore) cleanup() {
	now := time.Now()
	for k, exp := range ns.nonces {
		if now.After(exp) {
			delete(ns.nonces, k)
		}
	}
}
