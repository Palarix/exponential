package inputs

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	var got AddInput
	err := DecodeStrict(`{"title":"x","bogus":1}`, &got)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected error to mention unknown field name, got: %v", err)
	}
}

func TestAddInputToCreatePayload(t *testing.T) {
	in := AddInput{
		Title:       "My Issue",
		Description: "## Body\n\nWith `code` and \"quotes\" and $vars",
		Status:      "PLANNED",
		Parent:      "issue-abc123",
		StoryPoints: 5,
		Priority:    2,
		Assignee:    "Dev <dev@example.com>",
		Labels:      []string{"feature", "CLI"},
		Links: []LinkInput{
			{Target: "issue-def456", Type: "relates_to"},
			{Target: "issue-ghi789", Type: "blocks"},
		},
	}

	got, err := in.ToCreatePayload()
	if err != nil {
		t.Fatalf("ToCreatePayload failed: %v", err)
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
	if got.ParentID != "issue-abc123" {
		t.Errorf("ParentID: got %q want issue-abc123", got.ParentID)
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
	if got.Dependencies[0].TargetID != "issue-def456" || got.Dependencies[0].Kind != "relates_to" {
		t.Errorf("dep[0] wrong: %+v", got.Dependencies[0])
	}
	if got.Dependencies[1].Kind != model.DependencyBlocks {
		t.Errorf("dep[1].Kind: got %q want blocks", got.Dependencies[1].Kind)
	}
}

func TestAddInputRequiresTitle(t *testing.T) {
	_, err := AddInput{}.ToCreatePayload()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestAddInputInvalidStatus(t *testing.T) {
	_, err := AddInput{Title: "x", Status: "GARBAGE"}.ToCreatePayload()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestAddInputInvalidLinkType(t *testing.T) {
	_, err := AddInput{
		Title: "x",
		Links: []LinkInput{{Target: "issue-abc", Type: "wat"}},
	}.ToCreatePayload()
	if err == nil {
		t.Fatal("expected error for invalid link type")
	}
}

func TestAddInputLinkMissingTarget(t *testing.T) {
	_, err := AddInput{
		Title: "x",
		Links: []LinkInput{{Type: "blocks"}},
	}.ToCreatePayload()
	if err == nil {
		t.Fatal("expected error for missing link target")
	}
}

func TestUpdateInputPartial(t *testing.T) {
	newTitle := "Updated"
	in := UpdateInput{Title: &newTitle}
	got, err := in.ToUpdatePayload()
	if err != nil {
		t.Fatalf("ToUpdatePayload failed: %v", err)
	}
	if got.Title == nil || *got.Title != "Updated" {
		t.Errorf("Title not set: %+v", got.Title)
	}
	if got.Description != nil || got.Status != nil || got.ParentID != nil {
		t.Errorf("expected other fields nil, got: %+v", got)
	}
}

func TestUpdateInputEmpty(t *testing.T) {
	got, err := UpdateInput{}.ToUpdatePayload()
	if err != nil {
		t.Fatalf("ToUpdatePayload failed: %v", err)
	}
	if !UpdatePayloadEmpty(got) {
		t.Error("expected empty payload, got non-empty")
	}
}

func TestUpdateInputInvalidStatus(t *testing.T) {
	bad := "WRONG"
	_, err := UpdateInput{Status: &bad}.ToUpdatePayload()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateStatus(t *testing.T) {
	for _, s := range []string{"BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE", "CANCELED", "DUPLICATE"} {
		if err := ValidateStatus(s); err != nil {
			t.Errorf("ValidateStatus(%q) returned error: %v", s, err)
		}
	}
	for _, s := range []string{"", "backlog", "TODO", "INVALID"} {
		if err := ValidateStatus(s); err == nil {
			t.Errorf("ValidateStatus(%q) accepted invalid status", s)
		}
	}
}

func TestUpdatePayloadEmpty(t *testing.T) {
	if !UpdatePayloadEmpty(model.UpdatePayload{}) {
		t.Error("zero payload should be empty")
	}
	s := "DOING"
	if UpdatePayloadEmpty(model.UpdatePayload{Status: &s}) {
		t.Error("payload with Status should not be empty")
	}
	if UpdatePayloadEmpty(model.UpdatePayload{Labels: []string{"x"}}) {
		t.Error("payload with Labels should not be empty")
	}
	if UpdatePayloadEmpty(model.UpdatePayload{Dependencies: []model.Dependency{{}}}) {
		t.Error("payload with Dependencies should not be empty")
	}
}

func TestCommentInput(t *testing.T) {
	var got CommentInput
	err := DecodeStrict(`{"body":"hello\n\nworld"}`, &got)
	if err != nil {
		t.Fatalf("DecodeStrict failed: %v", err)
	}
	if got.Body != "hello\n\nworld" {
		t.Errorf("Body: got %q want %q", got.Body, "hello\n\nworld")
	}
}

func TestMultilineMarkdownRoundTrip(t *testing.T) {
	body := "## Heading\n\nA paragraph with `code`, \"quotes\", $vars, and `!` bangs.\n\n- item 1\n- item 2\n"
	jsonStr := `{"title":"x","description":` + jsonQuote(body) + `}`
	var in AddInput
	if err := DecodeStrict(jsonStr, &in); err != nil {
		t.Fatalf("DecodeStrict failed: %v", err)
	}
	got, err := in.ToCreatePayload()
	if err != nil {
		t.Fatalf("ToCreatePayload failed: %v", err)
	}
	if got.Description != body {
		t.Errorf("description round-trip mismatch:\n got: %q\nwant: %q", got.Description, body)
	}
}

func TestNormalizeDependencyKindForms(t *testing.T) {
	cases := map[string]string{
		"blocks":        "blocks",
		"blocked_by":    "blocked_by",
		"BlockedBy":     "blocked_by",
		"depends_on":    "depends_on",
		"DEPENDS_ON":    "depends_on",
		"relates_to":    "relates_to",
		"relatesTo":     "relates_to",
		"duplicated_by": "duplicated_by",
		"unknown":       "",
		"":              "",
	}
	for in, want := range cases {
		if got := model.NormalizeDependencyKind(in); got != want {
			t.Errorf("NormalizeDependencyKind(%q) = %q, want %q", in, got, want)
		}
	}
}

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
