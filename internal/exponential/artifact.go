package exponential

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
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
	return filepath.Join(storage.XpoDir(), "artifacts", issueID)
}

func (t *LocalTransport) AddArtifact(issueID, artifactType, filename, content string) error {
	if reservedArtifactNames[filename] {
		return fmt.Errorf("filename %q is reserved; use WriteSpec or WriteWalkthrough instead", filename)
	}
	return t.writeArtifact(issueID, artifactType, filename, content)
}

const MaxArtifactContentLen = 1024 * 1024

func (t *LocalTransport) writeArtifact(issueID, artifactType, filename, content string) error {
	if err := validateArtifactFilename(filename); err != nil {
		return err
	}
	if len(content) > MaxArtifactContentLen {
		return fmt.Errorf("artifact content exceeds maximum size of %d bytes", MaxArtifactContentLen)
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	if err := storage.WriteArtifact(issueID, filename, content); err != nil {
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

func (t *LocalTransport) ReadArtifact(issueID, filename string) (string, error) {
	if err := validateArtifactFilename(filename); err != nil {
		return "", err
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return "", err
	}
	issueID = issue.ID

	content, err := storage.ReadArtifact(issueID, filename)
	if err != nil {
		return "", fmt.Errorf("failed to read artifact: %w", err)
	}
	if content == "" {
		return "", fmt.Errorf("artifact %q not found on issue %s", filename, issueID)
	}
	return content, nil
}

func (t *LocalTransport) DeleteArtifact(issueID, filename string) error {
	if err := validateArtifactFilename(filename); err != nil {
		return err
	}

	issue, err := t.GetIssue(issueID)
	if err != nil {
		return err
	}
	issueID = issue.ID

	existing, _ := storage.ReadArtifact(issueID, filename)
	if existing == "" {
		return fmt.Errorf("artifact %q not found on issue %s", filename, issueID)
	}
	if err := storage.DeleteArtifact(issueID, filename); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

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
