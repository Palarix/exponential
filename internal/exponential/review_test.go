package exponential

import (
	"os"
	"path/filepath"
	"strings"
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

	runGit(t, dir, "checkout", "-b", "issue-abc123/feature")
	got = CurrentBranch()
	if got != "issue-abc123/feature" {
		t.Errorf("CurrentBranch() = %q, want %q", got, "issue-abc123/feature")
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
	runGit(t, dir, "checkout", "-b", "issue-abc123/feature")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Should find via current branch
	got := findLocalBranch("issue-abc123")
	if got != "issue-abc123/feature" {
		t.Errorf("findLocalBranch on current branch = %q, want %q", got, "issue-abc123/feature")
	}

	// Switch away — should still find via branch scan
	runGit(t, dir, "checkout", "main")
	got = findLocalBranch("issue-abc123")
	if got != "issue-abc123/feature" {
		t.Errorf("findLocalBranch from main = %q, want %q", got, "issue-abc123/feature")
	}

	// Non-matching
	got = findLocalBranch("issue-zzz999")
	if got != "" {
		t.Errorf("findLocalBranch for missing = %q, want empty", got)
	}
}

func TestGetWorkingTreeFullDiffText(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "committed.txt"), []byte("line1\nline2\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Modify a tracked file (uncommitted)
	os.WriteFile(filepath.Join(dir, "committed.txt"), []byte("line1\nline2\nline3\n"), 0644)
	// Create an untracked file
	os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("hello\nworld\n"), 0644)
	// Create an .xpo/ file that should be filtered
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo", "issues.db"), []byte("data"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	result := GetWorkingTreeFullDiffText(dir)

	if !strings.Contains(result, "committed.txt") {
		t.Error("diff should contain tracked modified file committed.txt")
	}
	if !strings.Contains(result, "+line3") {
		t.Error("diff should contain the added line")
	}
	if !strings.Contains(result, "untracked.txt") {
		t.Error("diff should contain untracked file untracked.txt")
	}
	if !strings.Contains(result, "+hello") {
		t.Error("diff should contain untracked file content as additions")
	}
	if !strings.Contains(result, "+world") {
		t.Error("diff should contain all lines of untracked file")
	}
	if !strings.Contains(result, "new file mode") {
		t.Error("untracked file diff should have new file mode header")
	}
	if !strings.Contains(result, "--- /dev/null") {
		t.Error("untracked file diff should have /dev/null as old file")
	}
	if strings.Contains(result, ".xpo/") {
		t.Error("diff should not contain .xpo/ paths")
	}
}

func TestGetWorkingTreeFullDiffText_BinarySkipped(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("x"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Create a binary untracked file (contains null bytes)
	binaryContent := []byte("hello\x00world")
	os.WriteFile(filepath.Join(dir, "image.bin"), binaryContent, 0644)

	result := GetWorkingTreeFullDiffText(dir)
	if strings.Contains(result, "+hello") {
		t.Error("binary file content should not appear in diff")
	}
}

func TestGetWorkingTreeFullDiffText_EmptyWorktree(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("x"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	result := GetWorkingTreeFullDiffText(dir)
	if result != "" {
		t.Errorf("expected empty diff for clean worktree, got %q", result)
	}
}

func TestListWorkingTreeAllFilesChanged(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("line1\n"), 0644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	// Modify tracked file
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("line1\nline2\n"), 0644)
	// Add untracked file
	os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("a\nb\nc\n"), 0644)
	// Add .xpo/ file that should be filtered
	os.MkdirAll(filepath.Join(dir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(dir, ".xpo", "data.db"), []byte("x"), 0644)

	files := ListWorkingTreeAllFilesChanged(dir)

	fileMap := make(map[string]FileStat)
	for _, f := range files {
		fileMap[f.Path] = f
	}

	if _, ok := fileMap["tracked.txt"]; !ok {
		t.Error("missing tracked modified file")
	}

	if f, ok := fileMap["newfile.txt"]; !ok {
		t.Error("missing untracked file newfile.txt")
	} else {
		if f.Status != "A" {
			t.Errorf("newfile.txt status = %q, want %q", f.Status, "A")
		}
		if f.Insertions != 3 {
			t.Errorf("newfile.txt insertions = %d, want 3", f.Insertions)
		}
	}

	if _, ok := fileMap[".xpo/data.db"]; ok {
		t.Error("should not include .xpo/ files")
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
