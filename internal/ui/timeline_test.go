package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestDescribeEvent_Create(t *testing.T) {
	evt := model.Event{Type: model.EventTypeCreate}
	if got := DescribeEvent(evt); got != "created this issue" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_Delete(t *testing.T) {
	evt := model.Event{Type: model.EventTypeDelete}
	if got := DescribeEvent(evt); got != "deleted this issue" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_Merge(t *testing.T) {
	evt := model.Event{
		Type:    model.EventTypeMerge,
		Payload: model.MergePayload{Branch: "feature-x", Strategy: "fast-forward"},
	}
	got := DescribeEvent(evt)
	if got != "merged feature-x via fast-forward" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_Artifact(t *testing.T) {
	evt := model.Event{
		Type:    model.EventTypeArtifact,
		Payload: model.ArtifactPayload{Filename: "spec.md", Action: "created"},
	}
	got := DescribeEvent(evt)
	if got != "created spec.md" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_UpdateStatus(t *testing.T) {
	status := "DOING"
	evt := model.Event{
		Type:    model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: &status},
	}
	got := DescribeEvent(evt)
	if got != "changed status to DOING" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_UpdateMultipleFields(t *testing.T) {
	status := "DOING"
	est := 3
	evt := model.Event{
		Type:    model.EventTypeUpdate,
		Payload: model.UpdatePayload{Status: &status, Estimate: &est},
	}
	got := DescribeEvent(evt)
	if got != "changed status to DOING and set estimate to 3" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_UpdateNoFields(t *testing.T) {
	evt := model.Event{
		Type:    model.EventTypeUpdate,
		Payload: model.UpdatePayload{},
	}
	got := DescribeEvent(evt)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestDescribeEvent_UpdateAssignee(t *testing.T) {
	assignee := "Alice <alice@example.com>"
	evt := model.Event{
		Type:    model.EventTypeUpdate,
		Payload: model.UpdatePayload{Assignee: &assignee},
	}
	got := DescribeEvent(evt)
	if got != "assigned to Alice" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_UpdateLabels(t *testing.T) {
	evt := model.Event{
		Type:    model.EventTypeUpdate,
		Payload: model.UpdatePayload{Labels: []string{"bug", "CLI"}},
	}
	got := DescribeEvent(evt)
	if got != "updated labels to [bug, CLI]" {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvent_Comment(t *testing.T) {
	evt := model.Event{Type: model.EventTypeComment}
	got := DescribeEvent(evt)
	if got != "" {
		t.Errorf("comment should return empty for DescribeEvent, got %q", got)
	}
}

func TestFormatActor_Simple(t *testing.T) {
	got := FormatActor("Alice <alice@example.com>", "")
	if got != "Alice" {
		t.Errorf("got %q", got)
	}
}

func TestFormatActor_OnBehalfOf(t *testing.T) {
	got := FormatActor("Claude Code <agent@macbook.local>", "Alice <alice@example.com>")
	if got != "Claude Code (via Alice)" {
		t.Errorf("got %q", got)
	}
}

func renderToString(entries []model.TimelineEntry, termWidth int) string {
	var buf bytes.Buffer
	RenderTimeline(&buf, entries, termWidth)
	return buf.String()
}

func issueEntry(id, eventType, createdBy string, ts time.Time, payload interface{}) model.TimelineEntry {
	return model.TimelineEntry{
		Kind:      "issue_event",
		Timestamp: ts,
		IssueID:   id,
		EventType: eventType,
		Payload:   payload,
		CreatedBy: createdBy,
	}
}

func TestRenderTimeline_Empty(t *testing.T) {
	got := renderToString(nil, 80)
	if !strings.Contains(got, "No activity recorded.") {
		t.Errorf("expected 'No activity recorded.', got %q", got)
	}
}

func TestRenderTimeline_IssueEvent(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("xpo-abc123", "CREATE", "Alice <a@b.com>", now, nil),
	}
	got := renderToString(entries, 80)
	if !strings.Contains(got, "xpo-abc123") {
		t.Error("should show issue ID")
	}
	if !strings.Contains(got, "Alice") {
		t.Error("should show actor name")
	}
	if !strings.Contains(got, "created this issue") {
		t.Error("should show event description")
	}
}

func TestRenderTimeline_Comment(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("xpo-abc123", "COMMENT", "Bob <b@c.com>", now, model.CommentPayload{Text: "Looks good to me."}),
	}
	got := renderToString(entries, 80)
	if !strings.Contains(got, "commented:") {
		t.Error("should show 'commented:'")
	}
	if !strings.Contains(got, "│") {
		t.Error("comment text should have │ border")
	}
	if !strings.Contains(got, "Looks good to me.") {
		t.Error("should show comment text")
	}
}

func TestRenderTimeline_OnBehalfOf(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{{
		Kind:       "issue_event",
		Timestamp:  now,
		IssueID:    "xpo-abc123",
		EventType:  "CREATE",
		CreatedBy:  "Claude Code <agent@macbook.local>",
		OnBehalfOf: "Nicolas <nic@example.com>",
	}}
	got := renderToString(entries, 80)
	if !strings.Contains(got, "(via Nicolas)") {
		t.Errorf("should show on_behalf_of, got %q", got)
	}
}

func TestRenderTimeline_SkipsEmptyUpdate(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("x", "UPDATE", "A <a@b.com>", now, model.UpdatePayload{}),
	}
	got := renderToString(entries, 80)
	if strings.Contains(got, "A") {
		t.Error("should skip update events with no recognized fields")
	}
}

func TestRenderTimeline_CommitEntry(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{{
		Kind:      "commit",
		Timestamp: now,
		SHA:       "abc123def456",
		Message:   "fix: handle nil payload",
		Author:    "Alice",
		IssueID:   "xpo-abc123",
	}}
	got := renderToString(entries, 100)
	if !strings.Contains(got, "abc123def456") {
		t.Error("should show commit SHA")
	}
	if !strings.Contains(got, "fix: handle nil payload") {
		t.Error("should show commit message")
	}
	if !strings.Contains(got, "Alice") {
		t.Error("should show commit author")
	}
	if !strings.Contains(got, "xpo-abc123") {
		t.Error("should show linked issue ID")
	}
}

func TestRenderTimeline_MixedEntries(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("xpo-abc123", "CREATE", "Alice <a@b.com>", now, nil),
		{
			Kind:      "commit",
			Timestamp: now.Add(time.Minute),
			SHA:       "abc123",
			Message:   "initial commit",
			Author:    "Alice",
		},
	}
	got := renderToString(entries, 100)
	if !strings.Contains(got, "created this issue") {
		t.Error("should render issue event")
	}
	if !strings.Contains(got, "initial commit") {
		t.Error("should render commit")
	}
}

func TestRenderTimelineJSON(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("xpo-abc123", "CREATE", "Alice <a@b.com>", now, nil),
		{Kind: "commit", Timestamp: now, SHA: "abc123", Message: "fix", Author: "Bob"},
	}

	var buf bytes.Buffer
	RenderTimelineJSON(entries, &buf)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		if _, ok := m["kind"]; !ok {
			t.Errorf("line %d missing 'kind' field", i)
		}
	}
}

func TestRenderTimeline_CommentMarkdown(t *testing.T) {
	now := time.Now()
	entries := []model.TimelineEntry{
		issueEntry("xpo-abc123", "COMMENT", "Bob <b@c.com>", now, model.CommentPayload{Text: "This has **bold** and `code`."}),
	}
	got := renderToString(entries, 80)
	if !strings.Contains(got, "│") {
		t.Error("comment should have │ border")
	}
	if !strings.Contains(got, "bold") {
		t.Error("should contain rendered markdown text")
	}
}
