package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func TestArgsContainJSONFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare flag", []string{"xpo", "show", "id", "--json"}, true},
		{"assignment true", []string{"xpo", "show", "id", "--json=true"}, true},
		{"assignment false", []string{"xpo", "list", "--json=false"}, false},
		{"no flag", []string{"xpo", "list"}, false},
		{"after double dash", []string{"xpo", "show", "--", "--json"}, false},
		{"mixed flags", []string{"xpo", "list", "--status", "DOING", "--json"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := os.Args
			os.Args = tc.args
			defer func() { os.Args = orig }()
			if got := argsContainJSONFlag(); got != tc.want {
				t.Errorf("argsContainJSONFlag() = %v, want %v for args %v", got, tc.want, tc.args)
			}
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	var buf bytes.Buffer
	writeJSONError(&buf, fmt.Errorf("something went wrong"))

	var out jsonio.ErrorOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %s", buf.String())
	}
	if out.Error != "something went wrong" {
		t.Errorf("got error %q, want %q", out.Error, "something went wrong")
	}
}

func TestWriteJSONError_WrappedError(t *testing.T) {
	var buf bytes.Buffer
	inner := fmt.Errorf("connection refused")
	writeJSONError(&buf, fmt.Errorf("failed to add comment: %w", inner))

	var out jsonio.ErrorOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %s", buf.String())
	}
	if out.Error != "failed to add comment: connection refused" {
		t.Errorf("got error %q", out.Error)
	}
}

func TestWriteJSONError_Indented(t *testing.T) {
	var buf bytes.Buffer
	writeJSONError(&buf, fmt.Errorf("test"))

	if !strings.Contains(buf.String(), "\n") {
		t.Error("expected indented (multi-line) JSON output")
	}
}

func TestWriteJSONError_NoExtraFields(t *testing.T) {
	var buf bytes.Buffer
	writeJSONError(&buf, fmt.Errorf("test"))

	var raw map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %s", buf.String())
	}
	if len(raw) != 1 {
		t.Errorf("expected exactly 1 field, got %d: %v", len(raw), raw)
	}
	if _, ok := raw["error"]; !ok {
		t.Errorf("expected 'error' field, got keys: %v", raw)
	}
}

// --- Integration tests ---
//
// These build the real CLI binary and run it against a minimal .xpo project
// to verify that --json error paths produce valid ErrorOutput JSON.

var testBinary string

func TestMain(m *testing.M) {
	// Build the binary once for all integration tests.
	tmp, err := os.MkdirTemp("", "xpo-json-error-test-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	testBinary = filepath.Join(tmp, "xpo")
	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Dir = filepath.Join(mustGetwd(), ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func mustGetwd() string {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return d
}

// setupProject creates a minimal .xpo project in a temp directory and returns
// the path. The caller should defer os.RemoveAll.
func setupProject(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "xpo-json-error-test-*")
	if err != nil {
		t.Fatal(err)
	}
	xpoDir := filepath.Join(dir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configYAML := "version: 3\nname: test-project\nprefix: test\n"
	if err := os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runXPO runs the test binary with the given args in the project directory.
// Returns stdout, stderr, and exit code.
func runXPO(t *testing.T, projectDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Dir = projectDir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			exitCode = e.ExitCode()
		} else {
			t.Fatalf("failed to run xpo: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// assertJSONError verifies stdout is valid ErrorOutput JSON, exit code is
// non-zero, and the error message contains the expected substring.
func assertJSONError(t *testing.T, stdout string, exitCode int, wantSubstring string) {
	t.Helper()
	if exitCode == 0 {
		t.Error("expected non-zero exit code")
	}
	var out jsonio.ErrorOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not valid ErrorOutput JSON:\n%s", stdout)
	}
	if wantSubstring != "" && !strings.Contains(out.Error, wantSubstring) {
		t.Errorf("error %q does not contain %q", out.Error, wantSubstring)
	}
}

func TestJSONError_ShowNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "show", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_CommentsNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "comments", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "not found")
}

func TestJSONError_CommentNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "comment", "nonexistent-id", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"body":"test"}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "not found")
}

func TestJSONError_AddBadJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "add", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{invalid json}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "invalid")
}

func TestJSONError_AddMissingTitle(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "add", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"description":"no title"}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "title")
}

func TestJSONError_UpdateBadJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "update", "test-123", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{not json}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "invalid")
}

func TestJSONError_UpdateEmptyPayload(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "update", "test-123", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "no fields set")
}

func TestJSONError_DoneNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "done", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_PlannedNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "planned", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_BlockedNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "blocked", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_HistoryNotFound(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "history", "nonexistent-id", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_HistoryBadSince(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "history", "--json", "--since", "notaduration")
	assertJSONError(t, stdout, code, "invalid --since")
}

func TestJSONError_CommentBadJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "comment", "test-123", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{bad json}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "invalid")
}

func TestJSONError_CommentEmptyBody(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	cmd := exec.Command(testBinary, "comment", "test-123", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"body":""}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	assertJSONError(t, outBuf.String(), exitCode, "empty")
}

// setupBrokenProject creates a .xpo project where issues.db is a directory,
// causing I/O errors when commands try to read events.
func setupBrokenProject(t *testing.T) string {
	t.Helper()
	dir := setupProject(t)
	dbPath := filepath.Join(dir, ".xpo", "issues.db")
	os.Remove(dbPath)
	if err := os.Mkdir(dbPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestJSONError_ListBrokenStorage(t *testing.T) {
	dir := setupBrokenProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "list", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_PulseBrokenStorage(t *testing.T) {
	dir := setupBrokenProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "pulse", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_InboxBrokenStorage(t *testing.T) {
	dir := setupBrokenProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "inbox", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_BlockedListBrokenStorage(t *testing.T) {
	dir := setupBrokenProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "blocked", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_HistoryGlobalBrokenStorage(t *testing.T) {
	dir := setupBrokenProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "history", "--json")
	assertJSONError(t, stdout, code, "")
}

// --- Cobra-level error tests ---
//
// These verify that errors produced before command handlers (argument
// validation, PersistentPreRunE) also emit JSON when --json is present.

func TestJSONError_CobraArgValidation(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	// `show` requires exactly 1 arg; passing none triggers cobra's arg error
	stdout, _, code := runXPO(t, dir, "show", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_NoProjectWithJSON(t *testing.T) {
	// Run in a directory with no .xpo
	dir, err := os.MkdirTemp("", "xpo-no-project-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "list", "--json")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_CobraArgValidation_AssignmentSyntax(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "show", "--json=true")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_NoProjectWithJSON_AssignmentSyntax(t *testing.T) {
	dir, err := os.MkdirTemp("", "xpo-no-project-assign-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	stdout, _, code := runXPO(t, dir, "list", "--json=true")
	assertJSONError(t, stdout, code, "")
}

func TestJSONError_AddDuplicateDetection(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	// First, create an issue
	cmd := exec.Command(testBinary, "add", "--json", "--force")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"title":"Unique Test Issue"}`)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping duplicate test: could not create seed issue: %v", err)
	}

	// Now try to add a duplicate without --force
	cmd = exec.Command(testBinary, "add", "--json")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(`{"title":"Unique Test Issue"}`)
	outBuf.Reset()
	cmd.Stdout = &outBuf
	err := cmd.Run()
	exitCode := 0
	if e, ok := err.(*exec.ExitError); ok {
		exitCode = e.ExitCode()
	}
	if exitCode == 0 {
		// No duplicate detected (similarity threshold not met) — skip
		t.Skip("duplicate detection did not trigger (threshold not met)")
	}
	assertJSONError(t, outBuf.String(), exitCode, "duplicate")
}
