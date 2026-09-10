package refstore

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func initTestRepo(t *testing.T) (string, *Store) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %s\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("init", "-b", "main")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")

	store := New(dir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init: %v", err)
	}
	return dir, store
}

func TestInit(t *testing.T) {
	_, store := initTestRepo(t)
	if !store.RefExists() {
		t.Fatal("ref should exist after Init")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	_, store := initTestRepo(t)

	err := store.WriteFile("issues.db", `{"id":"xpo-1"}`+"\n", "add event")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	content, err := store.ReadFile("issues.db")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if content != `{"id":"xpo-1"}`+"\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestReadNonexistent(t *testing.T) {
	_, store := initTestRepo(t)

	content, err := store.ReadFile("does-not-exist.txt")
	if err != nil {
		t.Fatalf("ReadFile should not error for missing file: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty string, got %q", content)
	}
}

func TestAppendFile(t *testing.T) {
	_, store := initTestRepo(t)

	store.AppendFile("issues.db", `{"id":"xpo-1"}`, "event 1")
	store.AppendFile("issues.db", `{"id":"xpo-2"}`, "event 2")
	store.AppendFile("issues.db", `{"id":"xpo-3"}`, "event 3")

	content, _ := store.ReadFile("issues.db")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	for i, line := range lines {
		var evt struct{ ID string }
		json.Unmarshal([]byte(line), &evt)
		if evt.ID != fmt.Sprintf("xpo-%d", i+1) {
			t.Errorf("line %d: expected xpo-%d, got %s", i, i+1, evt.ID)
		}
	}
}

func TestNestedPath(t *testing.T) {
	_, store := initTestRepo(t)

	err := store.WriteFile("artifacts/xpo-abc/spec.md", "# Spec\nDo the thing.", "add spec")
	if err != nil {
		t.Fatalf("WriteFile nested: %v", err)
	}

	content, err := store.ReadFile("artifacts/xpo-abc/spec.md")
	if err != nil {
		t.Fatalf("ReadFile nested: %v", err)
	}
	if content != "# Spec\nDo the thing." {
		t.Fatalf("unexpected: %q", content)
	}
}

func TestListDir(t *testing.T) {
	_, store := initTestRepo(t)

	store.WriteFile("artifacts/xpo-1/spec.md", "spec 1", "s1")
	store.WriteFile("artifacts/xpo-2/spec.md", "spec 2", "s2")
	store.WriteFile("artifacts/xpo-2/walkthrough.md", "walk 2", "w2")

	entries, err := store.ListDir("artifacts")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %v", entries)
	}

	entries, err = store.ListDir("artifacts/xpo-2")
	if err != nil {
		t.Fatalf("ListDir sub: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %v", entries)
	}
}

func TestDeleteFile(t *testing.T) {
	_, store := initTestRepo(t)

	store.WriteFile("artifacts/xpo-1/spec.md", "spec", "add")
	store.DeleteFile("artifacts/xpo-1/spec.md", "delete spec")

	content, err := store.ReadFile("artifacts/xpo-1/spec.md")
	if err != nil {
		t.Fatalf("ReadFile after delete: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty after delete, got %q", content)
	}
}

func TestMultipleFilesCoexist(t *testing.T) {
	_, store := initTestRepo(t)

	store.WriteFile("issues.db", "event1\n", "e1")
	store.WriteFile("artifacts/xpo-1/spec.md", "spec content", "spec")
	store.AppendFile("issues.db", "event2", "e2")

	content, _ := store.ReadFile("issues.db")
	if !strings.Contains(content, "event1") || !strings.Contains(content, "event2") {
		t.Fatalf("issues.db missing content: %q", content)
	}

	spec, _ := store.ReadFile("artifacts/xpo-1/spec.md")
	if spec != "spec content" {
		t.Fatalf("spec should be unchanged: %q", spec)
	}
}

func TestRefDoesNotPolluteBranches(t *testing.T) {
	dir, store := initTestRepo(t)

	store.WriteFile("issues.db", "event\n", "test")

	// Verify refs/xpo/data doesn't show up in git branch
	out, _ := exec.Command("git", "-C", dir, "branch", "-a").CombinedOutput()
	if strings.Contains(string(out), "xpo") {
		t.Fatalf("ref should not appear in git branch: %s", out)
	}

	// Verify the working tree is clean (no .xpo/ files)
	out, _ = exec.Command("git", "-C", dir, "status", "--porcelain").CombinedOutput()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("working tree should be clean: %s", out)
	}
}

func TestCommitHistory(t *testing.T) {
	dir, store := initTestRepo(t)

	store.AppendFile("issues.db", "event1", "first event")
	store.AppendFile("issues.db", "event2", "second event")
	store.WriteFile("artifacts/xpo-1/spec.md", "spec", "add spec")

	out, err := exec.Command("git", "-C", dir, "log", "--oneline", store.ref).CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	// init + 3 operations = 4 commits
	if len(lines) != 4 {
		t.Fatalf("expected 4 commits, got %d:\n%s", len(lines), out)
	}
}

func TestCASAppend(t *testing.T) {
	_, store := initTestRepo(t)

	ok, err := store.AppendFileCAS("issues.db", `{"id":"1"}`, "event 1")
	if err != nil {
		t.Fatalf("CAS append: %v", err)
	}
	if !ok {
		t.Fatal("first CAS append should succeed")
	}

	ok, err = store.AppendFileCAS("issues.db", `{"id":"2"}`, "event 2")
	if err != nil {
		t.Fatalf("CAS append 2: %v", err)
	}
	if !ok {
		t.Fatal("sequential CAS should succeed")
	}

	content, _ := store.ReadFile("issues.db")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestConcurrentAppends(t *testing.T) {
	_, store := initTestRepo(t)

	const n = 20
	var wg sync.WaitGroup
	var successes, retries int64
	var mu sync.Mutex

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			line := fmt.Sprintf(`{"id":"xpo-%d"}`, idx)
			for attempt := 0; attempt < 50; attempt++ {
				ok, err := store.AppendFileCAS("issues.db", line, fmt.Sprintf("event %d", idx))
				if err != nil {
					t.Errorf("goroutine %d: %v", idx, err)
					return
				}
				if ok {
					mu.Lock()
					successes++
					mu.Unlock()
					return
				}
				mu.Lock()
				retries++
				mu.Unlock()
				time.Sleep(time.Duration(attempt) * time.Millisecond)
			}
			t.Errorf("goroutine %d: exhausted retries", idx)
		}(i)
	}
	wg.Wait()

	content, _ := store.ReadFile("issues.db")
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != n {
		t.Fatalf("expected %d lines, got %d (successes=%d, retries=%d)",
			n, len(lines), successes, retries)
	}

	t.Logf("concurrent appends: %d successes, %d retries", successes, retries)
}

func TestFsnotifyPicksUpRefChange(t *testing.T) {
	dir, store := initTestRepo(t)

	refFile := filepath.Join(dir, ".git", "refs", "xpo", "data")
	if _, err := os.Stat(refFile); os.IsNotExist(err) {
		t.Skip("ref stored in packed-refs, not as loose file")
	}

	infoBefore, _ := os.Stat(refFile)

	store.AppendFile("issues.db", "new-event", "trigger change")

	infoAfter, _ := os.Stat(refFile)
	if infoBefore.ModTime() == infoAfter.ModTime() && infoBefore.Size() == infoAfter.Size() {
		t.Fatal("ref file should have changed on disk after append")
	}
}

func TestReadPerformance(t *testing.T) {
	_, store := initTestRepo(t)

	// Build a ~3000-line issues.db (similar to real repo)
	var content strings.Builder
	for i := 0; i < 3000; i++ {
		content.WriteString(fmt.Sprintf(`{"id":"xpo-%06d","type":"CREATE","payload":{"title":"Issue %d"}}`, i, i))
		content.WriteByte('\n')
	}
	store.WriteFile("issues.db", content.String(), "bulk load")

	// Benchmark read
	const iterations = 100
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := store.ReadFile("issues.db")
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	perRead := elapsed / iterations

	t.Logf("read 3000-line issues.db: %v avg over %d iterations", perRead, iterations)
	if perRead > 50*time.Millisecond {
		t.Errorf("read too slow: %v (threshold 50ms)", perRead)
	}
}
