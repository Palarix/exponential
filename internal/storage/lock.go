package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	eventsLockTimeout    = 10 * time.Second
	eventsLockRetryDelay = 5 * time.Millisecond
)

// withEventsLock runs fn while holding an exclusive lock on .xpo/events.lock.
// Every write to issues.db goes through it, so appends, collapses and
// archives from different processes (MCP servers, the CLI, the web server)
// never interleave. flock locks belong to the open file, so goroutines in
// one process exclude each other too.
//
// Lock order: when both are needed, take git.lock (exponential.WithGitLock)
// first. Nothing in storage takes git.lock, so holding events.lock never
// waits on it. The lock is not reentrant: fn must not call another write
// function in this package.
func withEventsLock(fn func() error) error {
	fl := flock.New(filepath.Join(XpoDir(), "events.lock"))

	ctx, cancel := context.WithTimeout(context.Background(), eventsLockTimeout)
	defer cancel()

	ok, err := fl.TryLockContext(ctx, eventsLockRetryDelay)
	if err != nil {
		return fmt.Errorf("failed to acquire events lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("another xpo process is writing issues.db — try again in a moment")
	}
	defer fl.Unlock()

	return fn()
}
