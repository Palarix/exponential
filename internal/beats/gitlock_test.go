package beats

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWithGitLock_Serializes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WithGitLock(func() error {
				cur := concurrent.Add(1)
				for {
					old := maxConcurrent.Load()
					if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
						break
					}
				}
				// Simulate work
				for j := 0; j < 1000; j++ {
					_ = j
				}
				concurrent.Add(-1)
				return nil
			})
		}()
	}

	wg.Wait()

	if maxConcurrent.Load() > 1 {
		t.Errorf("expected max concurrency 1, got %d", maxConcurrent.Load())
	}
}

func TestWithGitLock_PropagatesError(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beats"), 0755)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	err := WithGitLock(func() error {
		return os.ErrNotExist
	})
	if err != os.ErrNotExist {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}
