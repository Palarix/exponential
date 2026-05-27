package main

import (
	"strings"
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	var got addJSONInput
	err := decodeStrict(`{"title":"x","bogus":1}`, &got)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected error to mention unknown field name, got: %v", err)
	}
}

func TestAddJSONInputToCreatePayload(t *testing.T) {
	in := addJSONInput{
		Title:       "My Issue",
		Description: "## Body\n\nWith `code` and \"quotes\" and $vars",
		Status:      "PLANNED",
		Parent:      "beats-abc123",
		StoryPoints: 5,
		Priority:    2,
		Assignee:    "Dev <dev@example.com>",
		Labels:      []string{"feature", "CLI"},
		Links: []linkInput{
			{Target: "beats-def456", Type: "relates_to"},
			{Target: "beats-ghi789", Type: "blocks"},
		},
	}

	got, err := in.toCreatePayload()
	if err != nil {
		t.Fatalf("toCreatePayload failed: %v", err)
	}

	if got.Title != in.Title {
		t.Errorf("Title: got %q want %q", got.Title, in.Title)
	}
	if got.Description != in.Description {
		t.Errorf("Description: got %q want %q", got.Description, in.Description)
	}
	if got.Status != "PLANNED" {
		t.Errorf("Status: got %q want PLANNED", got.Status)
	}
	if got.ParentID != "beats-abc123" {
		t.Errorf("ParentID: got %q want beats-abc123", got.ParentID)
	}
	if got.Estimate != 5 {
		t.Errorf("Estimate: got %d want 5", got.Estimate)
	}
	if got.Priority != 2 {
		t.Errorf("Priority: got %d want 2", got.Priority)
	}
	if got.Assignee != in.Assignee {
		t.Errorf("Assignee mismatch")
	}
	if len(got.Labels) != 2 {
		t.Errorf("Labels: got %d want 2", len(got.Labels))
	}
	if len(got.Dependencies) != 2 {
		t.Fatalf("Dependencies: got %d want 2", len(got.Dependencies))
	}
	if got.Dependencies[0].TargetID != "beats-def456" || got.Dependencies[0].Kind != "relates_to" {
		t.Errorf("dep[0] wrong: %+v", got.Dependencies[0])
	}
	if got.Dependencies[1].Kind != model.DependencyBlocks {
		t.Errorf("dep[1].Kind: got %q want blocks", got.Dependencies[1].Kind)
	}
}

func TestAddJSONInputRequiresTitle(t *testing.T) {
	_, err := addJSONInput{}.toCreatePayload()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestAddJSONInputInvalidStatus(t *testing.T) {
	_, err := addJSONInput{Title: "x", Status: "GARBAGE"}.toCreatePayload()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestAddJSONInputInvalidLinkType(t *testing.T) {
	_, err := addJSONInput{
		Title: "x",
		Links: []linkInput{{Target: "beats-abc", Type: "wat"}},
	}.toCreatePayload()
	if err == nil {
		t.Fatal("expected error for invalid link type")
	}
}

func TestAddJSONInputLinkMissingTarget(t *testing.T) {
	_, err := addJSONInput{
		Title: "x",
		Links: []linkInput{{Type: "blocks"}},
	}.toCreatePayload()
	if err == nil {
		t.Fatal("expected error for missing link target")
	}
}

func TestUpdateJSONInputPartial(t *testing.T) {
	newTitle := "Updated"
	in := updateJSONInput{Title: &newTitle}
	got, err := in.toUpdatePayload()
	if err != nil {
		t.Fatalf("toUpdatePayload failed: %v", err)
	}
	if got.Title == nil || *got.Title != "Updated" {
		t.Errorf("Title not set: %+v", got.Title)
	}
	if got.Description != nil || got.Status != nil || got.ParentID != nil {
		t.Errorf("expected other fields nil, got: %+v", got)
	}
}

func TestUpdateJSONInputEmpty(t *testing.T) {
	got, err := updateJSONInput{}.toUpdatePayload()
	if err != nil {
		t.Fatalf("toUpdatePayload failed: %v", err)
	}
	if !updatePayloadEmpty(got) {
		t.Error("expected empty payload, got non-empty")
	}
}

func TestUpdateJSONInputInvalidStatus(t *testing.T) {
	bad := "WRONG"
	_, err := updateJSONInput{Status: &bad}.toUpdatePayload()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateStatus(t *testing.T) {
	for _, s := range []string{"BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE"} {
		if err := validateStatus(s); err != nil {
			t.Errorf("validateStatus(%q) returned error: %v", s, err)
		}
	}
	for _, s := range []string{"", "backlog", "TODO", "INVALID"} {
		if err := validateStatus(s); err == nil {
			t.Errorf("validateStatus(%q) accepted invalid status", s)
		}
	}
}

func TestUpdatePayloadEmpty(t *testing.T) {
	if !updatePayloadEmpty(model.UpdatePayload{}) {
		t.Error("zero payload should be empty")
	}
	s := "DOING"
	if updatePayloadEmpty(model.UpdatePayload{Status: &s}) {
		t.Error("payload with Status should not be empty")
	}
	if updatePayloadEmpty(model.UpdatePayload{Labels: []string{"x"}}) {
		t.Error("payload with Labels should not be empty")
	}
	if updatePayloadEmpty(model.UpdatePayload{Dependencies: []model.Dependency{{}}}) {
		t.Error("payload with Dependencies should not be empty")
	}
}

func TestCommentJSONInput(t *testing.T) {
	var got commentJSONInput
	err := decodeStrict(`{"body":"hello\n\nworld"}`, &got)
	if err != nil {
		t.Fatalf("decodeStrict failed: %v", err)
	}
	if got.Body != "hello\n\nworld" {
		t.Errorf("Body: got %q want %q", got.Body, "hello\n\nworld")
	}
}

func TestMultilineMarkdownRoundTrip(t *testing.T) {
	// The whole point of JSON mode: arbitrary markdown survives the trip
	// from JSON-encoded payload into a CreatePayload without mangling.
	body := "## Heading\n\nA paragraph with `code`, \"quotes\", $vars, and `!` bangs.\n\n- item 1\n- item 2\n"
	jsonStr := `{"title":"x","description":` + jsonQuote(body) + `}`
	var in addJSONInput
	if err := decodeStrict(jsonStr, &in); err != nil {
		t.Fatalf("decodeStrict failed: %v", err)
	}
	got, err := in.toCreatePayload()
	if err != nil {
		t.Fatalf("toCreatePayload failed: %v", err)
	}
	if got.Description != body {
		t.Errorf("description round-trip mismatch:\n got: %q\nwant: %q", got.Description, body)
	}
}

// jsonQuote returns a JSON-encoded string literal for use in test fixtures.
func jsonQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
