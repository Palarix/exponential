// Package registry manages a shared file-based registry of running xpo board
// instances so peers can discover each other.
//
// All instances are peers — there is no master. Each instance writes its
// {name, port, pid, root_dir, started_at} entry on startup and removes it on
// clean shutdown. Stale entries (process no longer alive) are pruned whenever
// the registry is read.
package registry

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Entry represents a running xpo board instance.
type Entry struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	PID       int       `json:"pid"`
	RootDir   string    `json:"root_dir"`
	StartedAt time.Time `json:"started_at"`
}

// registryPath returns the shared registry file path. Uses the OS temp dir
// so the file lives in a per-user, ephemeral location across reboots.
func registryPath() (string, error) {
	dir := filepath.Join(os.TempDir(), "xpo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "instances.json"), nil
}

func load() ([]Entry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func save(entries []Entry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// isAlive checks whether a process is still running by sending it signal 0.
// Reliable on Unix; on Windows os.FindProcess always succeeds so this may
// produce false positives.
func isAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

// List returns all live registered instances, pruning any stale entries from
// the file as a side effect.
func List() ([]Entry, error) {
	entries, err := load()
	if err != nil {
		return nil, err
	}
	live := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if isAlive(e.PID) {
			live = append(live, e)
		}
	}
	if len(live) != len(entries) {
		_ = save(live)
	}
	return live, nil
}

// Register adds the current process to the registry. Stale entries and any
// existing entry for this PID are removed first.
func Register(name string, port int, rootDir string) error {
	entries, _ := load()
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
	return save(out)
}

// UnregisterByPID removes the entry for the given PID, if any.
func UnregisterByPID(pid int) error {
	entries, err := load()
	if err != nil {
		return err
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.PID != pid {
			out = append(out, e)
		}
	}
	return save(out)
}

// FindFreePort tries to bind a TCP listener starting at startPort, incrementing
// up to maxAttempts-1 times. Returns the listener and the port actually bound.
func FindFreePort(startPort, maxAttempts int) (net.Listener, int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return l, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no free port found in range %d-%d", startPort, startPort+maxAttempts-1)
}
