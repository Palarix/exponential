package refstore

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultRef = "refs/xpo/data"

type Store struct {
	repoDir string
	ref     string
}

func New(repoDir string) *Store {
	return &Store{repoDir: repoDir, ref: DefaultRef}
}

// Init creates the initial ref with an empty tree commit.
func (s *Store) Init() error {
	tree, err := s.gitStdin("", "mktree", "--missing")
	if err != nil {
		return fmt.Errorf("mktree: %w", err)
	}
	commit, err := s.git("commit-tree", "-m", "xpo: init", strings.TrimSpace(tree))
	if err != nil {
		return fmt.Errorf("commit-tree: %w", err)
	}
	_, err = s.git("update-ref", s.ref, strings.TrimSpace(commit))
	return err
}

// ReadFile reads a file from the ref tree. Returns ("", nil) if not found.
func (s *Store) ReadFile(path string) (string, error) {
	out, err := s.git("show", s.ref+":"+path)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "not exist in") ||
			strings.Contains(err.Error(), "fatal: path") {
			return "", nil
		}
		return "", err
	}
	return out, nil
}

// WriteFile writes or overwrites a file in the ref tree, creating a new commit.
func (s *Store) WriteFile(path, content, message string) error {
	// Hash the new content as a blob
	blobSHA, err := s.gitStdin(content, "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("hash-object: %w", err)
	}
	blobSHA = strings.TrimSpace(blobSHA)

	return s.updateTree(path, blobSHA, "100644", message)
}

// AppendFile appends content to an existing file (or creates it).
func (s *Store) AppendFile(path, line, message string) error {
	existing, err := s.ReadFile(path)
	if err != nil {
		return err
	}
	newContent := existing + line + "\n"

	return s.WriteFile(path, newContent, message)
}

// DeleteFile removes a file from the ref tree.
func (s *Store) DeleteFile(path, message string) error {
	return s.updateTree(path, "", "", message)
}

// ListDir lists entries in a directory within the ref tree.
func (s *Store) ListDir(path string) ([]string, error) {
	target := s.ref + ":" + path
	if path == "" || path == "." {
		target = s.ref
	}
	out, err := s.git("ls-tree", "--name-only", target)
	if err != nil {
		if strings.Contains(err.Error(), "not a tree") ||
			strings.Contains(err.Error(), "does not exist") {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}

// LogEntry represents one commit in the ref's history.
type LogEntry struct {
	SHA     string
	Message string
	Author  string
	Date    string
}

// Log returns the commit history of the ref.
func (s *Store) Log(maxCount int) ([]LogEntry, error) {
	args := []string{"log", "--format=%H\t%s\t%an <%ae>\t%ci", s.ref}
	if maxCount > 0 {
		args = []string{"log", fmt.Sprintf("-n%d", maxCount), "--format=%H\t%s\t%an <%ae>\t%ci", s.ref}
	}
	out, err := s.git(args...)
	if err != nil {
		return nil, err
	}
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil, nil
	}
	var entries []LogEntry
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, LogEntry{
			SHA:     parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return entries, nil
}

// ReadFileAt reads a file from the ref at a specific commit SHA.
func (s *Store) ReadFileAt(sha, path string) (string, error) {
	out, err := s.git("show", sha+":"+path)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ResetTo moves the ref to point at the given commit SHA.
func (s *Store) ResetTo(sha string) error {
	_, err := s.git("update-ref", s.ref, sha)
	return err
}

// Diff returns the diff between two commits.
func (s *Store) Diff(oldSHA, newSHA string) (string, error) {
	return s.git("diff", oldSHA, newSHA)
}

// CurrentSHA returns the SHA the ref currently points to.
func (s *Store) CurrentSHA() (string, error) {
	out, err := s.git("rev-parse", s.ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RefExists checks whether the ref has been initialized.
func (s *Store) RefExists() bool {
	_, err := s.git("rev-parse", "--verify", s.ref)
	return err == nil
}

// AppendFileCAS atomically appends to a file using compare-and-swap on the ref.
// Returns false if the ref moved (caller should retry).
func (s *Store) AppendFileCAS(path, line, message string) (bool, error) {
	oldSHA, err := s.git("rev-parse", s.ref)
	if err != nil {
		return false, fmt.Errorf("rev-parse: %w", err)
	}
	oldSHA = strings.TrimSpace(oldSHA)

	existing, err := s.ReadFile(path)
	if err != nil {
		return false, err
	}
	newContent := existing + line + "\n"

	blobSHA, err := s.gitStdin(newContent, "hash-object", "-w", "--stdin")
	if err != nil {
		return false, fmt.Errorf("hash-object: %w", err)
	}
	blobSHA = strings.TrimSpace(blobSHA)

	treeSHA, err := s.buildUpdatedTree(path, blobSHA, "100644")
	if err != nil {
		return false, err
	}

	commitSHA, err := s.git("commit-tree", "-m", message, "-p", oldSHA, treeSHA)
	if err != nil {
		return false, fmt.Errorf("commit-tree: %w", err)
	}
	commitSHA = strings.TrimSpace(commitSHA)

	// CAS: update only succeeds if ref still points to oldSHA
	stdin := fmt.Sprintf("update %s %s %s\n", s.ref, commitSHA, oldSHA)
	_, err = s.gitStdin(stdin, "update-ref", "--stdin")
	if err != nil {
		return false, nil // ref moved — caller should retry
	}
	return true, nil
}

func (s *Store) updateTree(path, blobSHA, mode, message string) error {
	treeSHA, err := s.buildUpdatedTree(path, blobSHA, mode)
	if err != nil {
		return err
	}

	parentSHA, err := s.git("rev-parse", s.ref)
	if err != nil {
		return fmt.Errorf("rev-parse: %w", err)
	}
	parentSHA = strings.TrimSpace(parentSHA)

	commitSHA, err := s.git("commit-tree", "-m", message, "-p", parentSHA, treeSHA)
	if err != nil {
		return fmt.Errorf("commit-tree: %w", err)
	}
	_, err = s.git("update-ref", s.ref, strings.TrimSpace(commitSHA))
	return err
}

func (s *Store) buildUpdatedTree(path, blobSHA, mode string) (string, error) {
	// Get the current root tree
	rootTree, err := s.git("rev-parse", s.ref+"^{tree}")
	if err != nil {
		return "", fmt.Errorf("rev-parse tree: %w", err)
	}
	rootTree = strings.TrimSpace(rootTree)

	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		return s.spliceEntry(rootTree, parts[0], blobSHA, mode)
	}

	// Walk down, collecting tree SHAs for each directory level
	treeSHAs := make([]string, len(parts))
	currentTree := rootTree
	for i := 0; i < len(parts)-1; i++ {
		subtree, err := s.getSubtreeSHA(currentTree, parts[i])
		if err != nil {
			return "", err
		}
		if subtree == "" {
			subtree, err = s.gitStdin("", "mktree", "--missing")
			if err != nil {
				return "", err
			}
			subtree = strings.TrimSpace(subtree)
		}
		treeSHAs[i] = currentTree
		currentTree = subtree
	}

	// Splice the blob into the deepest directory
	newTree, err := s.spliceEntry(currentTree, parts[len(parts)-1], blobSHA, mode)
	if err != nil {
		return "", err
	}

	// Walk back up, re-splicing each parent
	for i := len(parts) - 2; i >= 0; i-- {
		newTree, err = s.spliceEntry(treeSHAs[i], parts[i], newTree, "040000")
		if err != nil {
			return "", err
		}
	}

	return newTree, nil
}

func (s *Store) spliceEntry(treeSHA, name, entrySHA, mode string) (string, error) {
	// Read existing tree entries
	out, err := s.git("ls-tree", treeSHA)
	if err != nil {
		return "", fmt.Errorf("ls-tree: %w", err)
	}

	var lines []string
	found := false
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) == 2 && fields[1] == name {
			found = true
			if entrySHA != "" {
				entryType := "blob"
				if mode == "040000" {
					entryType = "tree"
				}
				lines = append(lines, fmt.Sprintf("%s %s %s\t%s", mode, entryType, entrySHA, name))
			}
			// else: deleting — skip this entry
		} else {
			lines = append(lines, line)
		}
	}
	if !found && entrySHA != "" {
		entryType := "blob"
		if mode == "040000" {
			entryType = "tree"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s\t%s", mode, entryType, entrySHA, name))
	}

	mktreeInput := strings.Join(lines, "\n")
	if mktreeInput != "" {
		mktreeInput += "\n"
	}
	newTree, err := s.gitStdin(mktreeInput, "mktree")
	if err != nil {
		return "", fmt.Errorf("mktree: %w", err)
	}
	return strings.TrimSpace(newTree), nil
}

func (s *Store) getSubtreeSHA(treeSHA, name string) (string, error) {
	out, err := s.git("ls-tree", treeSHA, name)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", nil
	}
	// Format: <mode> <type> <sha>\t<name>
	fields := strings.Fields(out)
	if len(fields) < 3 {
		return "", nil
	}
	return fields[2], nil
}

func (s *Store) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", s.repoDir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (s *Store) gitStdin(input string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", s.repoDir}, args...)...)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
