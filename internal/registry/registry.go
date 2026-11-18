// Package registry manages a shared file-based registry of running xpo board
// instances so peers can discover each other.
//
// All instances are peers — there is no master. Each instance writes its
// {name, port, pid, root_dir, started_at} entry on startup and removes it on
// clean shutdown. Stale entries (process no longer alive) are pruned whenever
// the registry is read.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// Entry represents a running xpo board instance.
type Entry struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	PID       int       `json:"pid"`
	RootDir   string    `json:"root_dir"`
	StartedAt time.Time `json:"started_at"`
}

const (
	lockTimeout    = 5 * time.Second
	lockRetryDelay = 5 * time.Millisecond
)

// registryDir is the directory holding instances.json and its lock. It
// defaults to the OS temp dir so the file lives in a per-user, ephemeral
// location across reboots. Tests swap it with SetDirForTesting.
var (
	dirMu       sync.Mutex
	registryDir = filepath.Join(os.TempDir(), "xpo")
)

func dir() string {
	dirMu.Lock()
	defer dirMu.Unlock()
	return registryDir
}

// SetDirForTesting points the registry at dir and returns a function that
// restores the previous directory. Tests use it so they never read or write
// the developer's real registry.
func SetDirForTesting(d string) (restore func()) {
	dirMu.Lock()
	prev := registryDir
	registryDir = d
	dirMu.Unlock()
	return func() {
		dirMu.Lock()
		registryDir = prev
		dirMu.Unlock()
	}
}

// withLock runs fn while holding an exclusive lock on instances.lock, so
// load-modify-save sequences from different boards never interleave. flock
// locks belong to the open file, so goroutines in one process exclude each
// other too. The lock file is never deleted: removing it would let two
// processes lock different files.
func withLock(fn func(d string) error) error {
	d := dir()
	if err := os.MkdirAll(d, 0755); err != nil {
		return err
	}
	fl := flock.New(filepath.Join(d, "instances.lock"))

	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	ok, err := fl.TryLockContext(ctx, lockRetryDelay)
	if err != nil {
		return fmt.Errorf("failed to acquire registry lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("timed out waiting for the registry lock")
	}
	defer func() { _ = fl.Unlock() }()

	return fn(d)
}

// load reads the registry in d. A file that does not parse (empty, or
// garbled by an older binary that wrote without the lock) loads as empty
// with corrupt set, so the caller rewrites it instead of failing forever.
func load(d string) (entries []Entry, corrupt bool, err error) {
	data, err := os.ReadFile(filepath.Join(d, "instances.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, true, nil
	}
	return entries, false, nil
}

// save writes entries to d through a uniquely named tmp file and renames it
// over instances.json, so a reader never sees a partial file.
func save(d string, entries []Entry) error {
	if entries == nil {
		entries = []Entry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(d, "instances-*.json.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmp, 0644)
	}
	if werr == nil {
		werr = os.Rename(tmp, filepath.Join(d, "instances.json"))
	}
	if werr != nil {
		_ = os.Remove(tmp)
	}
	return werr
}

// List returns all live registered instances, pruning stale entries (and
// rewriting a corrupt file) as a side effect.
func List() ([]Entry, error) {
	var live []Entry
	err := withLock(func(d string) error {
		entries, corrupt, err := load(d)
		if err != nil {
			return err
		}
		live = make([]Entry, 0, len(entries))
		for _, e := range entries {
			if isAlive(e.PID) {
				live = append(live, e)
			}
		}
		if corrupt || len(live) != len(entries) {
			_ = save(d, live)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return live, nil
}

// Register adds the current process to the registry. Stale entries and any
// existing entry for this PID are removed first.
func Register(name string, port int, rootDir string) error {
	return withLock(func(d string) error {
		entries, _, _ := load(d)
		out := make([]Entry, 0, len(entries)+1)
		for _, e := range entries {
			if e.PID == os.Getpid() {
				continue
			}
			if isAlive(e.PID) {
				out = append(out, e)
			}
		}
		out = append(out, Entry{
			Name:      name,
			Port:      port,
			PID:       os.Getpid(),
			RootDir:   rootDir,
			StartedAt: time.Now(),
		})
		return save(d, out)
	})
}

// UnregisterByPID removes the entry for the given PID, if any.
func UnregisterByPID(pid int) error {
	return withLock(func(d string) error {
		entries, _, err := load(d)
		if err != nil {
			return err
		}
		out := make([]Entry, 0, len(entries))
		for _, e := range entries {
			if e.PID != pid {
				out = append(out, e)
			}
		}
		return save(d, out)
	})
}

// FindFreePort tries to bind a TCP listener on host starting at startPort,
// incrementing up to maxAttempts-1 times. Returns the listener and the port
// actually bound. An empty host listens on all interfaces.
func FindFreePort(host string, startPort, maxAttempts int) (net.Listener, int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			return l, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no free port found in range %d-%d", startPort, startPort+maxAttempts-1)
}
