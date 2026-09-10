package server

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchGitRefs watches .git/refs/heads/ (recursively) and .git/packed-refs
// for changes. When git refs change (new commit, new/deleted branch, gc),
// the cached issues are invalidated and an SSE update is broadcast so the
// GUI refreshes branch stats immediately.
func (s *Server) WatchGitRefs() {
	if s.SSEHub == nil {
		return
	}

	gitDir, err := filepath.Abs(".git")
	if err != nil {
		return
	}
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return
	}

	refsHeads := filepath.Join(gitDir, "refs", "heads")
	if _, err := os.Stat(refsHeads); os.IsNotExist(err) {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("watch-git: failed to create watcher: %v", err)
		return
	}

	// Watch all directories under refs/heads/ (handles hierarchical branch names).
	filepath.WalkDir(refsHeads, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	// Watch refs/xpo/ for ref-based storage changes.
	refsXpo := filepath.Join(gitDir, "refs", "xpo")
	if err := os.MkdirAll(refsXpo, 0755); err == nil {
		watcher.Add(refsXpo)
	}

	// Watch packed-refs if it exists, and the .git dir itself
	// so we detect packed-refs creation (e.g. after git gc).
	packedRefs := filepath.Join(gitDir, "packed-refs")
	if _, err := os.Stat(packedRefs); err == nil {
		watcher.Add(packedRefs)
	}
	watcher.Add(gitDir)

	go func() {
		defer watcher.Close()
		var debounce *time.Timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Chmod {
					continue
				}

				isUnderRefsHeads := strings.HasPrefix(event.Name, refsHeads)
				isUnderRefsXpo := strings.HasPrefix(event.Name, refsXpo)
				isPackedRefs := event.Name == packedRefs
				if !isUnderRefsHeads && !isUnderRefsXpo && !isPackedRefs {
					continue
				}

				// Watch newly created subdirectories (hierarchical branch namespaces).
				if isUnderRefsHeads && event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						watcher.Add(event.Name)
					}
				}

				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(150*time.Millisecond, func() {
					s.mu.Lock()
					s.cachedIssues = nil
					s.mu.Unlock()
					s.SSEHub.Broadcast(SSEEvent{Type: "UPDATE", IssueID: ""})
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watch-git: error: %v", err)
			}
		}
	}()
}
