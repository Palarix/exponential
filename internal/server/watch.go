package server

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchDB watches issues.db for external modifications and broadcasts
// an SSE event so the GUI refreshes immediately.
func (s *Server) WatchDB() {
	if s.SSEHub == nil {
		return
	}

	dbPath := filepath.Join(".xpo", "issues.db")
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		log.Printf("watch: failed to resolve path: %v", err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("watch: failed to create watcher: %v", err)
		return
	}

	// Watch the file directly for WRITE events (works on macOS kqueue).
	// Also watch the parent directory so we can re-add the file watch
	// if the file is recreated.
	absDir := filepath.Dir(absPath)
	if err := watcher.Add(absDir); err != nil {
		log.Printf("watch: failed to watch directory %s: %v", absDir, err)
		watcher.Close()
		return
	}

	fileWatched := false
	if _, err := os.Stat(absPath); err == nil {
		if err := watcher.Add(absPath); err != nil {
			log.Printf("watch: failed to watch file %s: %v", absPath, err)
		} else {
			fileWatched = true
		}
	}

	go func() {
		defer watcher.Close()
		var debounce *time.Timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Base(event.Name) != "issues.db" {
					continue
				}

				// Ignore CHMOD — the server's own reads trigger these.
				if event.Op == fsnotify.Chmod {
					continue
				}

				// If the file was created/recreated, start watching it directly.
				if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
					if !fileWatched {
						if err := watcher.Add(absPath); err == nil {
							fileWatched = true
						}
					}
				}

				// If the file was removed, stop watching it so we can
				// re-add when it reappears.
				if event.Op&fsnotify.Remove != 0 {
					fileWatched = false
					continue
				}

				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(150*time.Millisecond, func() {
					s.SSEHub.Broadcast(SSEEvent{Type: "UPDATE", IssueID: ""})
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watch: error: %v", err)
			}
		}
	}()
}
