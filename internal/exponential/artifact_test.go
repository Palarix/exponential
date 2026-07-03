package exponential

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestAddArtifact_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "artifact test"})

	err := tr.AddArtifact(issue.ID, "generic", "notes.md", "# Notes\n\nSome content.")
	if err != nil {
		t.Fatal(err)
	}

	// Verify file on disk
	path := filepath.Join(".xpo", "artifacts", issue.ID, "notes.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artifact file not found: %v", err)
	}
	if string(data) != "# Notes\n\nSome content." {
		t.Errorf("content = %q", string(data))
	}

	// Verify event was emitted and projected
	issues := readAllIssues(t)
	if len(issues[issue.ID].Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(issues[issue.ID].Artifacts))
	}
	a := issues[issue.ID].Artifacts[0]
	if a.ArtifactType != "generic" || a.Filename != "notes.md" {
		t.Errorf("artifact = %+v", a)
	}
}

func TestAddArtifact_Upsert(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "upsert test"})

	tr.AddArtifact(issue.ID, "generic", "notes.md", "version 1")
	tr.AddArtifact(issue.ID, "generic", "notes.md", "version 2")

	path := filepath.Join(".xpo", "artifacts", issue.ID, "notes.md")
	data, _ := os.ReadFile(path)
	if string(data) != "version 2" {
		t.Errorf("expected upsert, got %q", string(data))
	}

	issues := readAllIssues(t)
	if len(issues[issue.ID].Artifacts) != 1 {
		t.Fatalf("expected 1 artifact after upsert, got %d", len(issues[issue.ID].Artifacts))
	}
}

func TestAddArtifact_IssueNotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	err := tr.AddArtifact("nonexistent", "generic", "notes.md", "content")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestAddArtifact_RejectsReservedFilenames(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "reserved test"})

	for _, name := range []string{"spec.md", "walkthrough.md"} {
		err := tr.AddArtifact(issue.ID, "generic", name, "content")
		if err == nil {
			t.Errorf("expected error for reserved filename %q", name)
		}
	}
}

func TestAddArtifact_RejectsPathTraversal(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "traversal test"})

	for _, name := range []string{"../evil.md", "sub/../../evil.md", "/etc/passwd", "."} {
		err := tr.AddArtifact(issue.ID, "generic", name, "content")
		if err == nil {
			t.Errorf("expected error for path traversal %q", name)
		}
	}
}

func TestReadArtifact_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "read test"})
	tr.AddArtifact(issue.ID, "generic", "notes.md", "hello world")

	content, err := tr.ReadArtifact(issue.ID, "notes.md")
	if err != nil {
		t.Fatal(err)
	}
	if content != "hello world" {
		t.Errorf("content = %q", content)
	}
}

func TestReadArtifact_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "read missing"})

	_, err := tr.ReadArtifact(issue.ID, "nonexistent.md")
	if err == nil {
		t.Error("expected error for missing artifact")
	}
}

func TestDeleteArtifact_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "delete test"})
	tr.AddArtifact(issue.ID, "generic", "notes.md", "content")

	err := tr.DeleteArtifact(issue.ID, "notes.md")
	if err != nil {
		t.Fatal(err)
	}

	// File should be removed
	path := filepath.Join(".xpo", "artifacts", issue.ID, "notes.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("artifact file still exists after delete")
	}

	// Projection should show no artifacts
	issues := readAllIssues(t)
	if len(issues[issue.ID].Artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(issues[issue.ID].Artifacts))
	}
}

func TestDeleteArtifact_NotFound(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "delete missing"})

	err := tr.DeleteArtifact(issue.ID, "nonexistent.md")
	if err == nil {
		t.Error("expected error for missing artifact")
	}
}

func TestListArtifacts_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "list test"})
	tr.writeArtifact(issue.ID, "spec", "spec.md", "the spec")
	tr.AddArtifact(issue.ID, "generic", "notes.md", "notes")

	artifacts, err := tr.ListArtifacts(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestListArtifacts_Empty(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "empty list"})

	artifacts, err := tr.ListArtifacts(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(artifacts))
	}
}

func TestWriteSpec_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "spec test"})

	err := tr.WriteSpec(issue.ID, "# Spec\n\nThe spec content.")
	if err != nil {
		t.Fatal(err)
	}

	content, err := tr.ReadSpec(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Spec\n\nThe spec content." {
		t.Errorf("spec content = %q", content)
	}

	issues := readAllIssues(t)
	a := issues[issue.ID].Artifacts[0]
	if a.ArtifactType != "spec" || a.Filename != "spec.md" {
		t.Errorf("artifact = %+v", a)
	}
}

func TestDeleteSpec_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "spec delete"})
	tr.WriteSpec(issue.ID, "content")

	err := tr.DeleteSpec(issue.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tr.ReadSpec(issue.ID)
	if err == nil {
		t.Error("expected error reading deleted spec")
	}
}

func TestWriteWalkthrough_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "walkthrough test"})

	err := tr.WriteWalkthrough(issue.ID, "## Walkthrough\n\nDetails.")
	if err != nil {
		t.Fatal(err)
	}

	content, err := tr.ReadWalkthrough(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "## Walkthrough\n\nDetails." {
		t.Errorf("walkthrough content = %q", content)
	}

	issues := readAllIssues(t)
	a := issues[issue.ID].Artifacts[0]
	if a.ArtifactType != "walkthrough" || a.Filename != "walkthrough.md" {
		t.Errorf("artifact = %+v", a)
	}
}

func TestDeleteWalkthrough_HappyPath(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "walkthrough delete"})
	tr.WriteWalkthrough(issue.ID, "content")

	err := tr.DeleteWalkthrough(issue.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tr.ReadWalkthrough(issue.ID)
	if err == nil {
		t.Error("expected error reading deleted walkthrough")
	}
}

func TestWriteSpec_ShortID(t *testing.T) {
	tr := setupLocalTransport(t)
	issue, _ := tr.AddIssue(model.CreatePayload{Title: "short id spec"})

	shortID := issue.ID[len("test-"):]
	err := tr.WriteSpec(shortID, "spec via short id")
	if err != nil {
		t.Fatalf("write spec via short ID failed: %v", err)
	}

	content, err := tr.ReadSpec(shortID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "spec via short id" {
		t.Errorf("content = %q", content)
	}
}
