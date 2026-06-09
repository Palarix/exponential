package beats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentBranch(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := CurrentBranch()
	if got != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", got, "main")
	}

	runGit(t, dir, "checkout", "-b", "beats-abc123/feature")
	got = CurrentBranch()
	if got != "beats-abc123/feature" {
		t.Errorf("CurrentBranch() = %q, want %q", got, "beats-abc123/feature")
	}
}

func TestListBranchCommits(t *testing.T) {
	remoteDir := t.TempDir()
	runGit(t, remoteDir, "init", "--bare")
	runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

	pusherDir := t.TempDir()
	runGit(t, pusherDir, "init", "-b", "main")
	runGit(t, pusherDir, "config", "user.email", "test@test.com")
	runGit(t, pusherDir, "config", "user.name", "Test")
	runGit(t, pusherDir, "remote", "add", "origin", remoteDir)

	os.WriteFile(filepath.Join(pusherDir, "f.txt"), []byte("base"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "init")
	runGit(t, pusherDir, "push", "origin", "HEAD:main")

	runGit(t, pusherDir, "checkout", "-b", "feature")
	os.WriteFile(filepath.Join(pusherDir, "a.txt"), []byte("a\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "first change")
	os.WriteFile(filepath.Join(pusherDir, "b.txt"), []byte("b\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "second change")
	runGit(t, pusherDir, "push", "origin", "feature")

	localDir := t.TempDir()
	runGit(t, localDir, "clone", remoteDir, ".")

	origDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(origDir)

	commits := ListBranchCommits("origin/feature", "main")
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	if commits[0].Message != "second change" {
		t.Errorf("first commit message = %q, want %q", commits[0].Message, "second change")
	}
	if commits[1].Message != "first change" {
		t.Errorf("second commit message = %q, want %q", commits[1].Message, "first change")
	}
}

func TestFindLocalBranch(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	runGit(t, dir, "checkout", "-b", "beats-abc123/feature")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Should find via current branch
	got := findLocalBranch("beats-abc123")
	if got != "beats-abc123/feature" {
		t.Errorf("findLocalBranch on current branch = %q, want %q", got, "beats-abc123/feature")
	}

	// Switch away — should still find via branch scan
	runGit(t, dir, "checkout", "main")
	got = findLocalBranch("beats-abc123")
	if got != "beats-abc123/feature" {
		t.Errorf("findLocalBranch from main = %q, want %q", got, "beats-abc123/feature")
	}

	// Non-matching
	got = findLocalBranch("beats-zzz999")
	if got != "" {
		t.Errorf("findLocalBranch for missing = %q, want empty", got)
	}
}

func TestListFilesChanged(t *testing.T) {
	remoteDir := t.TempDir()
	runGit(t, remoteDir, "init", "--bare")
	runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

	pusherDir := t.TempDir()
	runGit(t, pusherDir, "init", "-b", "main")
	runGit(t, pusherDir, "config", "user.email", "test@test.com")
	runGit(t, pusherDir, "config", "user.name", "Test")
	runGit(t, pusherDir, "remote", "add", "origin", remoteDir)

	os.WriteFile(filepath.Join(pusherDir, "existing.txt"), []byte("line1\nline2\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "init")
	runGit(t, pusherDir, "push", "origin", "HEAD:main")

	runGit(t, pusherDir, "checkout", "-b", "feature")
	os.WriteFile(filepath.Join(pusherDir, "new.txt"), []byte("new\n"), 0644)
	os.WriteFile(filepath.Join(pusherDir, "existing.txt"), []byte("line1\nline2\nline3\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "changes")
	runGit(t, pusherDir, "push", "origin", "feature")

	localDir := t.TempDir()
	runGit(t, localDir, "clone", remoteDir, ".")

	origDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(origDir)

	files := ListFilesChanged("origin/feature", "main")
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}

	fileMap := make(map[string]FileStat)
	for _, f := range files {
		fileMap[f.Path] = f
	}

	if f, ok := fileMap["new.txt"]; !ok {
		t.Error("missing new.txt")
	} else if f.Status != "A" {
		t.Errorf("new.txt status = %q, want %q", f.Status, "A")
	}

	if f, ok := fileMap["existing.txt"]; !ok {
		t.Error("missing existing.txt")
	} else if f.Insertions != 1 {
		t.Errorf("existing.txt insertions = %d, want 1", f.Insertions)
	}
}
