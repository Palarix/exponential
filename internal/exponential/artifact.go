package exponential

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
)

var reservedArtifactNames = map[string]bool{
	"spec.md":        true,
	"walkthrough.md": true,
}

func validateArtifactFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename must not be empty")
	}
	cleaned := filepath.Clean(filename)
	if cleaned != filename || strings.Contains(filename, "/") || strings.Contains(filename, "\\") || filepath.IsAbs(filename) {
		return fmt.Errorf("invalid artifact filename: %q", filename)
	}
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("invalid artifact filename: %q", filename)
	}
	if cleaned == "." {
		return fmt.Errorf("invalid artifact filename: %q", filename)
	}
	return nil
}

func artifactDir(issueID string) string {
	return filepath.Join(".xpo", "artifacts", issueID)
}

// AddArtifact writes a generic artifact for an issue. Reserved filenames
// (spec.md, walkthrough.md) are rejected — use WriteSpec/WriteWalkthrough
// instead.
func (t *LocalTransport) AddArtifact(issueID, artifactType, filename, content string) error {
	if reservedArtifactNames[filename] {
		return fmt.Errorf("filename %q is reserved; use WriteSpec or WriteWalkthrough instead", filename)
	}
	return t.writeArtifact(issueID, artifactType, filename, content)
}

// writeArtifact is the shared internal implementation used by AddArtifact
// and the first-class WriteSpec/WriteWalkthrough convenience methods. It
// does not enforce the reserved-filename check.
func (t *LocalTransport) writeArtifact(issueID, artifactType, filename, content string) error {
	if err := validateArtifactFilename(filename); err != nil {
		return err
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	dir := artifactDir(issueID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write artifact: %w", err)
	}

	action := "created"
	for _, a := range issue.Artifacts {
		if a.Filename == filename {
			action = "updated"
			break
		}
	}

	event := model.Event{
		ID:   issueID,
		Type: model.EventTypeArtifact,
		Payload: model.ArtifactPayload{
			ArtifactType: artifactType,
			Filename:     filename,
			Action:       action,
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: t.GetUser(),
	}

	if err := t.appendEvent(event); err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	return nil
}

// ReadArtifact returns the contents of an artifact for an issue.
func (t *LocalTransport) ReadArtifact(issueID, filename string) (string, error) {
	if err := validateArtifactFilename(filename); err != nil {
		return "", err
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return "", err
	}
	issueID = issue.ID

	path := filepath.Join(artifactDir(issueID), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("artifact %q not found on issue %s", filename, issueID)
		}
		return "", fmt.Errorf("failed to read artifact: %w", err)
	}
	return string(data), nil
}

// DeleteArtifact removes an artifact from disk and records a "deleted"
// ARTIFACT event so the projection drops it from Issue.Artifacts.
func (t *LocalTransport) DeleteArtifact(issueID, filename string) error {
	if err := validateArtifactFilename(filename); err != nil {
		return err
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	path := filepath.Join(artifactDir(issueID), filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("artifact %q not found on issue %s", filename, issueID)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	// Determine artifact type from projection so the deletion event carries
	// the same type the artifact was created/updated with.
	artifactType := "generic"
	for _, a := range issue.Artifacts {
		if a.Filename == filename {
			artifactType = a.ArtifactType
			break
		}
	}

	event := model.Event{
		ID:   issueID,
		Type: model.EventTypeArtifact,
		Payload: model.ArtifactPayload{
			ArtifactType: artifactType,
			Filename:     filename,
			Action:       "deleted",
		},
		CreatedAt: time.Now().UTC(),
		CreatedBy: t.GetUser(),
	}

	if err := t.appendEvent(event); err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	return nil
}

// ListArtifacts returns the current artifacts attached to an issue, as
// derived from the projection.
func (t *LocalTransport) ListArtifacts(issueID string) ([]model.ArtifactSummary, error) {
	issue, err := t.GetIssue(issueID)
	if err != nil {
		return nil, err
	}

	if issue.Artifacts == nil {
		return []model.ArtifactSummary{}, nil
	}
	return issue.Artifacts, nil
}

func (t *LocalTransport) WriteSpec(issueID, content string) error {
	return t.writeArtifact(issueID, "spec", "spec.md", content)
}

func (t *LocalTransport) ReadSpec(issueID string) (string, error) {
	return t.ReadArtifact(issueID, "spec.md")
}

func (t *LocalTransport) DeleteSpec(issueID string) error {
	return t.DeleteArtifact(issueID, "spec.md")
}

func (t *LocalTransport) WriteWalkthrough(issueID, content string) error {
	return t.writeArtifact(issueID, "walkthrough", "walkthrough.md", content)
}

func (t *LocalTransport) ReadWalkthrough(issueID string) (string, error) {
	return t.ReadArtifact(issueID, "walkthrough.md")
}

func (t *LocalTransport) DeleteWalkthrough(issueID string) error {
	return t.DeleteArtifact(issueID, "walkthrough.md")
}
