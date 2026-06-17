package exponential

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestBranchMatchesIssue(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		issueID  string
		want     bool
	}{
		{
			name:     "exact segment match",
			branches: []string{"origin/issue-3c2135"},
			issueID:  "issue-3c2135",
			want:     true,
		},
		{
			name:     "segment starts with issue ID",
			branches: []string{"origin/issue-3c2135/fix-login"},
			issueID:  "issue-3c2135",
			want:     true,
		},
		{
			name:     "segment with suffix after ID",
			branches: []string{"origin/issue-3c2135-fix-login"},
			issueID:  "issue-3c2135",
			want:     true,
		},
		{
			name:     "nested path with issue segment",
			branches: []string{"origin/feature/issue-3c2135/impl"},
			issueID:  "issue-3c2135",
			want:     true,
		},
		{
			name:     "no match - ID in middle of segment",
			branches: []string{"origin/fix-xpo-3c2135"},
			issueID:  "issue-3c2135",
			want:     false,
		},
		{
			name:     "no match - different ID",
			branches: []string{"origin/issue-aaaaaa/fix"},
			issueID:  "issue-3c2135",
			want:     false,
		},
		{
			name:     "no match - empty branches",
			branches: []string{},
			issueID:  "issue-3c2135",
			want:     false,
		},
		{
			name:     "no match - branch without remote prefix",
			branches: []string{"issue-3c2135"},
			issueID:  "issue-3c2135",
			want:     false,
		},
		{
			name:     "no match - partial ID prefix",
			branches: []string{"origin/issue-3c213"},
			issueID:  "issue-3c2135",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchMatchesIssue(tt.branches, tt.issueID)
			if got != tt.want {
				t.Errorf("branchMatchesIssue(%v, %q) = %v, want %v", tt.branches, tt.issueID, got, tt.want)
			}
		})
	}
}

func TestApplyBranchInference_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	issues := map[string]*model.Issue{
		"test-abc123": {
			ID:     "test-abc123",
			Status: model.StatusPlanned,
		},
	}

	applyBranchInference(issues)

	if issues["test-abc123"].Status != model.StatusPlanned {
		t.Errorf("expected PLANNED in non-git repo, got %s", issues["test-abc123"].Status)
	}
	if issues["test-abc123"].InferredStatus {
		t.Error("expected InferredStatus=false in non-git repo")
	}
}

func TestApplyBranchInference_WithRemoteBranch(t *testing.T) {
	// Create a "remote" bare repo
	remoteDir := t.TempDir()
	runGit(t, remoteDir, "init", "--bare")

	// Create a local repo that pushes a branch to the remote
	pusherDir := t.TempDir()
	runGit(t, pusherDir, "init")
	runGit(t, pusherDir, "config", "user.email", "test@test.com")
	runGit(t, pusherDir, "config", "user.name", "Test")
	runGit(t, pusherDir, "remote", "add", "origin", remoteDir)

	// Create initial commit on main
	os.WriteFile(filepath.Join(pusherDir, "file.txt"), []byte("hello"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "init")
	runGit(t, pusherDir, "push", "origin", "HEAD:main")

	// Create and push a feature branch named after an issue
	runGit(t, pusherDir, "checkout", "-b", "test-abc123/fix-login")
	os.WriteFile(filepath.Join(pusherDir, "fix.txt"), []byte("fix"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "fix")
	runGit(t, pusherDir, "push", "origin", "test-abc123/fix-login")

	// Create the "consumer" repo that fetches from the remote
	localDir := t.TempDir()
	runGit(t, localDir, "clone", remoteDir, ".")

	origDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(origDir)

	issues := map[string]*model.Issue{
		"test-abc123": {
			ID:     "test-abc123",
			Status: model.StatusPlanned,
		},
		"test-def456": {
			ID:     "test-def456",
			Status: model.StatusPlanned,
		},
		"test-ghi789": {
			ID:     "test-ghi789",
			Status: model.StatusDone,
		},
		"test-jkl012": {
			ID:     "test-jkl012",
			Status: model.StatusBlocked,
		},
	}

	applyBranchInference(issues)

	// Matching issue should be upgraded to DOING
	if issues["test-abc123"].Status != model.StatusDoing {
		t.Errorf("expected DOING for matched issue, got %s", issues["test-abc123"].Status)
	}
	if !issues["test-abc123"].InferredStatus {
		t.Error("expected InferredStatus=true for matched issue")
	}

	// Non-matching issue should remain PLANNED
	if issues["test-def456"].Status != model.StatusPlanned {
		t.Errorf("expected PLANNED for unmatched issue, got %s", issues["test-def456"].Status)
	}
	if issues["test-def456"].InferredStatus {
		t.Error("expected InferredStatus=false for unmatched issue")
	}

	// DONE issue should not be overridden
	if issues["test-ghi789"].Status != model.StatusDone {
		t.Errorf("expected DONE to be preserved, got %s", issues["test-ghi789"].Status)
	}

	// BLOCKED issue should not be overridden
	if issues["test-jkl012"].Status != model.StatusBlocked {
		t.Errorf("expected BLOCKED to be preserved, got %s", issues["test-jkl012"].Status)
	}
}

func TestMatchingBranch(t *testing.T) {
	tests := []struct {
		name     string
		branches []string
		issueID  string
		want     string
	}{
		{
			name:     "returns matched branch",
			branches: []string{"origin/issue-3c2135/fix-login"},
			issueID:  "issue-3c2135",
			want:     "origin/issue-3c2135/fix-login",
		},
		{
			name:     "returns empty on no match",
			branches: []string{"origin/issue-aaaaaa"},
			issueID:  "issue-3c2135",
			want:     "",
		},
		{
			name:     "returns first match",
			branches: []string{"origin/issue-3c2135", "origin/issue-3c2135/v2"},
			issueID:  "issue-3c2135",
			want:     "origin/issue-3c2135",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchingBranch(tt.branches, tt.issueID)
			if got != tt.want {
				t.Errorf("matchingBranch(%v, %q) = %q, want %q", tt.branches, tt.issueID, got, tt.want)
			}
		})
	}
}

func TestParseShortstat(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantFiles    int
		wantInsert   int
		wantDelete   int
	}{
		{
			name:       "full stat line",
			line:       "7 files changed, 142 insertions(+), 38 deletions(-)",
			wantFiles:  7,
			wantInsert: 142,
			wantDelete: 38,
		},
		{
			name:       "single file, insertions only",
			line:       "1 file changed, 5 insertions(+)",
			wantFiles:  1,
			wantInsert: 5,
			wantDelete: 0,
		},
		{
			name:       "deletions only",
			line:       "3 files changed, 10 deletions(-)",
			wantFiles:  3,
			wantInsert: 0,
			wantDelete: 10,
		},
		{
			name:       "empty line",
			line:       "",
			wantFiles:  0,
			wantInsert: 0,
			wantDelete: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &model.BranchStats{}
			parseShortstat(tt.line, stats)
			if stats.FilesChanged != tt.wantFiles {
				t.Errorf("FilesChanged = %d, want %d", stats.FilesChanged, tt.wantFiles)
			}
			if stats.Insertions != tt.wantInsert {
				t.Errorf("Insertions = %d, want %d", stats.Insertions, tt.wantInsert)
			}
			if stats.Deletions != tt.wantDelete {
				t.Errorf("Deletions = %d, want %d", stats.Deletions, tt.wantDelete)
			}
		})
	}
}

func TestBranchStatsComputed(t *testing.T) {
	remoteDir := t.TempDir()
	runGit(t, remoteDir, "init", "--bare")
	runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

	pusherDir := t.TempDir()
	runGit(t, pusherDir, "init", "-b", "main")
	runGit(t, pusherDir, "config", "user.email", "test@test.com")
	runGit(t, pusherDir, "config", "user.name", "Test")
	runGit(t, pusherDir, "remote", "add", "origin", remoteDir)

	os.WriteFile(filepath.Join(pusherDir, "file.txt"), []byte("hello"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "init")
	runGit(t, pusherDir, "push", "origin", "HEAD:main")

	runGit(t, pusherDir, "checkout", "-b", "test-abc123/feature")
	os.WriteFile(filepath.Join(pusherDir, "new.txt"), []byte("new file\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "add new file")
	os.WriteFile(filepath.Join(pusherDir, "another.txt"), []byte("another\n"), 0644)
	runGit(t, pusherDir, "add", ".")
	runGit(t, pusherDir, "commit", "-m", "add another")
	runGit(t, pusherDir, "push", "origin", "test-abc123/feature")

	localDir := t.TempDir()
	runGit(t, localDir, "clone", remoteDir, ".")

	origDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(origDir)

	issues := map[string]*model.Issue{
		"test-abc123": {
			ID:     "test-abc123",
			Status: model.StatusPlanned,
		},
	}

	applyBranchInference(issues)

	issue := issues["test-abc123"]
	if issue.BranchStats == nil {
		t.Fatal("expected BranchStats to be set")
	}
	if issue.BranchStats.Commits != 2 {
		t.Errorf("Commits = %d, want 2", issue.BranchStats.Commits)
	}
	if issue.BranchStats.FilesChanged != 2 {
		t.Errorf("FilesChanged = %d, want 2", issue.BranchStats.FilesChanged)
	}
	if issue.BranchStats.Insertions != 2 {
		t.Errorf("Insertions = %d, want 2", issue.BranchStats.Insertions)
	}
	if issue.BranchStats.Branch != "origin/test-abc123/feature" {
		t.Errorf("Branch = %q, want %q", issue.BranchStats.Branch, "origin/test-abc123/feature")
	}
	if len(issue.BranchStats.HeadSHA) != 12 {
		t.Errorf("HeadSHA length = %d, want 12 (got %q)", len(issue.BranchStats.HeadSHA), issue.BranchStats.HeadSHA)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, out)
	}
}
