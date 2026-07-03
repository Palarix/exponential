package exponential

import (
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestClient_WriteAndReadSpec(t *testing.T) {
	c := setupClient(t)
	issue, _ := c.AddIssue(model.CreatePayload{Title: "client spec"})

	err := c.WriteSpec(issue.ID, "client spec content")
	if err != nil {
		t.Fatal(err)
	}

	content, err := c.ReadSpec(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "client spec content" {
		t.Errorf("content = %q", content)
	}
}

func TestClient_ListArtifacts(t *testing.T) {
	c := setupClient(t)
	issue, _ := c.AddIssue(model.CreatePayload{Title: "client list"})
	c.WriteSpec(issue.ID, "spec")
	c.WriteWalkthrough(issue.ID, "walkthrough")

	artifacts, err := c.ListArtifacts(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}
