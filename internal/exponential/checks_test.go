package exponential

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckGitRepo_True(t *testing.T) {
	dir := initTestRepo(t, "main")
	_ = dir

	if !CheckGitRepo() {
		t.Error("expected true for a git repo")
	}
}

func TestCheckGitRepo_False(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	if CheckGitRepo() {
		t.Error("expected false for a non-git directory")
	}
}

func TestCheckGithubWorkflows_True(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	wfDir := filepath.Join(dir, ".github", "workflows")
	os.MkdirAll(wfDir, 0755)
	os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\n"), 0644)

	if !CheckGithubWorkflows() {
		t.Error("expected true when workflow files exist")
	}
}

func TestCheckGithubWorkflows_False(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(origDir) })

	if CheckGithubWorkflows() {
		t.Error("expected false when no workflow files exist")
	}
}

func TestCheckGitHooks_None(t *testing.T) {
	dir := initTestRepo(t, "main")
	_ = dir

	// Remove all non-sample hooks (fresh git init only has .sample files)
	hooks := CheckGitHooks()
	if len(hooks) != 0 {
		t.Errorf("expected no active hooks in fresh repo, got %v", hooks)
	}
}

func TestCheckGitHooks_Active(t *testing.T) {
	dir := initTestRepo(t, "main")

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 0\n"), 0755)

	hooks := CheckGitHooks()
	found := false
	for _, h := range hooks {
		if h == "pre-commit" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected pre-commit in active hooks, got %v", hooks)
	}
}

func TestCheckGitHooks_IgnoresSamples(t *testing.T) {
	dir := initTestRepo(t, "main")

	// Ensure .sample files are not counted
	samplePath := filepath.Join(dir, ".git", "hooks", "pre-push.sample")
	os.WriteFile(samplePath, []byte("#!/bin/sh\nexit 0\n"), 0755)

	hooks := CheckGitHooks()
	for _, h := range hooks {
		if h == "pre-push.sample" {
			t.Error("sample hooks should be excluded")
		}
	}
}
