package beats

import (
	"strings"
	"testing"

	"github.com/palarix/beats/internal/model"
)

func TestGenerateUpdateTemplate_Basic(t *testing.T) {
	issue := &model.Issue{
		Title:       "My Issue",
		Status:      model.StatusDoing,
		ParentID:    "parent-1",
		Estimate:    5,
		Assignee:    "Alice <a@b.com>",
		Labels:      []string{"bug", "feature"},
		Description: "some description",
	}
	got := GenerateUpdateTemplate(issue)

	if !strings.Contains(got, "Title: My Issue") {
		t.Error("missing title")
	}
	if !strings.Contains(got, "Status: DOING") {
		t.Error("missing status")
	}
	if !strings.Contains(got, "Parent: parent-1") {
		t.Error("missing parent")
	}
	if !strings.Contains(got, "Estimate: 5") {
		t.Error("missing estimate")
	}
	if !strings.Contains(got, "Assignee: Alice <a@b.com>") {
		t.Error("missing assignee")
	}
	if !strings.Contains(got, "Labels: bug, feature") {
		t.Error("missing labels")
	}
	if !strings.Contains(got, "some description") {
		t.Error("missing description")
	}
}

func TestGenerateUpdateTemplate_NoLabels(t *testing.T) {
	issue := &model.Issue{Title: "No labels", Status: model.StatusBacklog}
	got := GenerateUpdateTemplate(issue)
	if strings.Contains(got, "Labels:") {
		t.Error("should not include Labels line when none set")
	}
}

func TestParseUpdateContent_ChangedFields(t *testing.T) {
	original := &model.Issue{
		Title:       "Original",
		Status:      model.StatusBacklog,
		ParentID:    "",
		Estimate:    3,
		Assignee:    "",
		Labels:      []string{"bug"},
		Description: "old desc",
	}

	content := "Title: Renamed\nStatus: DOING\nParent: epic-1\nEstimate: 5\nAssignee: Bob <b@c.com>\nLabels: feature, improvement\n\nnew description"

	payload, err := ParseUpdateContent(content, original)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Title == nil || *payload.Title != "Renamed" {
		t.Errorf("Title = %v", payload.Title)
	}
	if payload.Status == nil || *payload.Status != "DOING" {
		t.Errorf("Status = %v", payload.Status)
	}
	if payload.ParentID == nil || *payload.ParentID != "epic-1" {
		t.Errorf("ParentID = %v", payload.ParentID)
	}
	if payload.Estimate == nil || *payload.Estimate != 5 {
		t.Errorf("Estimate = %v", payload.Estimate)
	}
	if payload.Assignee == nil || *payload.Assignee != "Bob <b@c.com>" {
		t.Errorf("Assignee = %v", payload.Assignee)
	}
	if len(payload.Labels) != 2 || payload.Labels[0] != "feature" || payload.Labels[1] != "improvement" {
		t.Errorf("Labels = %v", payload.Labels)
	}
	if payload.Description == nil || *payload.Description != "new description" {
		t.Errorf("Description = %v", payload.Description)
	}
}

func TestParseUpdateContent_UnchangedFieldsNil(t *testing.T) {
	original := &model.Issue{
		Title:       "Same",
		Status:      model.StatusDoing,
		Estimate:    5,
		Description: "same desc",
	}

	content := "Title: Same\nStatus: DOING\nParent: \nEstimate: 5\nAssignee: \n\nsame desc"

	payload, err := ParseUpdateContent(content, original)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Title != nil {
		t.Error("unchanged title should be nil")
	}
	if payload.Status != nil {
		t.Error("unchanged status should be nil")
	}
	if payload.Estimate != nil {
		t.Error("unchanged estimate should be nil")
	}
	if payload.Description != nil {
		t.Error("unchanged description should be nil")
	}
}

func TestParseUpdateContent_NoDescription(t *testing.T) {
	original := &model.Issue{Title: "X", Status: model.StatusBacklog}
	content := "Title: Y\nStatus: BACKLOG\nParent: \nEstimate: 0\nAssignee: "

	payload, err := ParseUpdateContent(content, original)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Title == nil || *payload.Title != "Y" {
		t.Errorf("Title = %v", payload.Title)
	}
}

func TestParseUpdateContent_RoundTrip(t *testing.T) {
	original := &model.Issue{
		Title:       "RT",
		Status:      model.StatusPlanned,
		ParentID:    "p1",
		Estimate:    3,
		Assignee:    "A <a@b.com>",
		Labels:      []string{"bug"},
		Description: "desc here",
	}

	template := GenerateUpdateTemplate(original)
	payload, err := ParseUpdateContent(template, original)
	if err != nil {
		t.Fatal(err)
	}

	if payload.Title != nil || payload.Status != nil || payload.ParentID != nil ||
		payload.Estimate != nil || payload.Assignee != nil || payload.Labels != nil ||
		payload.Description != nil {
		t.Errorf("round-trip should produce no changes, got %+v", payload)
	}
}
