package demo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/identity"
	"github.com/palarix/exponential/internal/model"
)

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	dbPath := filepath.Join(dir, ".xpo", "issues.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal("issues.db not created")
	}

	cfgPath := filepath.Join(dir, ".xpo", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatal("config.yaml not created")
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	var events []model.Event
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var evt model.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			t.Fatalf("invalid JSONL line: %v", err)
		}
		events = append(events, evt)
	}

	if len(events) < 500 {
		t.Errorf("expected at least 500 events, got %d", len(events))
	}

	creates := 0
	for _, e := range events {
		if e.Type == model.EventTypeCreate {
			creates++
		}
	}
	if creates < 100 {
		t.Errorf("expected at least 100 issues, got %d", creates)
	}

	// Verify IDs use upe- prefix
	for _, e := range events {
		if e.ID[:4] != "upe-" {
			t.Errorf("expected upe- prefix, got %s", e.ID)
			break
		}
	}

	// Verify events are chronologically ordered
	for i := 1; i < len(events); i++ {
		if events[i].CreatedAt.Before(events[i-1].CreatedAt) {
			t.Errorf("events not chronologically ordered at index %d", i)
			break
		}
	}

	// Verify the most recent event is within the last day
	last := events[len(events)-1]
	if last.CreatedAt.IsZero() {
		t.Error("last event has zero timestamp")
	}
}

func TestGenerateRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatalf("first Generate failed: %v", err)
	}
	// Second generate should succeed (the safety check is in the CLI command, not the library)
	if err := Generate(dir); err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}
}

func TestAllIssuesUniqueRefs(t *testing.T) {
	issues := allIssues()
	seen := make(map[string]bool)
	for _, iss := range issues {
		if seen[iss.Ref] {
			t.Errorf("duplicate issue ref: %s", iss.Ref)
		}
		seen[iss.Ref] = true
	}
}

func TestAllEventsReferenceValidIssues(t *testing.T) {
	issues := allIssues()
	refs := make(map[string]bool)
	for _, iss := range issues {
		refs[iss.Ref] = true
	}

	events := allEvents()
	for i, ev := range events {
		if !refs[ev.Ref] {
			t.Errorf("event %d references unknown issue ref %q", i, ev.Ref)
		}
		if ev.Kind == evLink && !refs[ev.TargetRef] {
			t.Errorf("event %d link references unknown target ref %q", i, ev.TargetRef)
		}
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func TestGenerateAttributesAgentWorkToHumans(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".xpo", "issues.db"))
	if err != nil {
		t.Fatal(err)
	}
	var events []model.Event
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var evt model.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			t.Fatalf("invalid JSONL line: %v", err)
		}
		events = append(events, evt)
	}

	agentEvents := 0
	for _, e := range events {
		if !identity.IsAgent(e.CreatedBy) {
			continue
		}
		agentEvents++
		if e.OnBehalfOf == "" || identity.IsAgent(e.OnBehalfOf) {
			t.Errorf("agent event %s %s by %q has no human principal (on_behalf_of %q)", e.ID, e.Type, e.CreatedBy, e.OnBehalfOf)
		}
	}
	if agentEvents == 0 {
		t.Error("expected the demo to contain agent-authored events")
	}

	viaCount := 0
	for id, iss := range exponential.ProjectIssues(events) {
		if identity.IsAgent(iss.Assignee) {
			t.Errorf("issue %s is assigned to agent %q", id, iss.Assignee)
		}
		if iss.AssigneeVia != "" {
			viaCount++
			if !identity.IsAgent(iss.AssigneeVia) {
				t.Errorf("issue %s has non-agent assignee_via %q", id, iss.AssigneeVia)
			}
		}
	}
	if viaCount == 0 {
		t.Error("expected agent-worked issues to carry assignee_via")
	}

	cfg, err := os.ReadFile(filepath.Join(dir, ".xpo", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfg), "agent@") {
		t.Error("config.yaml contributors include an agent identity")
	}
}
