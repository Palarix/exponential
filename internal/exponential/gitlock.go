package exponential

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/palarix/exponential/internal/storage"
)

const (
	gitLockTimeout    = 30 * time.Second
	gitLockRetryDelay = 250 * time.Millisecond
)

// WithGitLock acquires an exclusive file lock on .xpo/git.lock
// before running fn, preventing concurrent git-mutating operations
// from corrupting the working tree. The lock is released when fn
// returns or if the process crashes.
func WithGitLock(fn func() error) error {
	lockPath := filepath.Join(storage.XpoDir(), "git.lock")
	fl := flock.New(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), gitLockTimeout)
	defer cancel()

	ok, err := fl.TryLockContext(ctx, gitLockRetryDelay)
	if err != nil {
		return fmt.Errorf("failed to acquire git lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("another xpo operation is using git — try again in a moment")
	}
	defer fl.Unlock()

	return fn()
}
