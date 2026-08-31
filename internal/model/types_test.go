package model

import "testing"

func TestIsTerminal(t *testing.T) {
	terminal := []IssueStatus{StatusDone, StatusCanceled, StatusDuplicate}
	for _, s := range terminal {
		if !IsTerminal(s) {
			t.Errorf("IsTerminal(%q) = false, want true", s)
		}
	}

	active := []IssueStatus{StatusBacklog, StatusPlanned, StatusDoing, StatusBlocked}
	for _, s := range active {
		if IsTerminal(s) {
			t.Errorf("IsTerminal(%q) = true, want false", s)
		}
	}
}

func TestIsCompleted(t *testing.T) {
	if !IsCompleted(StatusDone) {
		t.Error("IsCompleted(DONE) = false, want true")
	}
	for _, s := range []IssueStatus{StatusCanceled, StatusDuplicate, StatusBacklog, StatusPlanned, StatusDoing, StatusBlocked} {
		if IsCompleted(s) {
			t.Errorf("IsCompleted(%q) = true, want false", s)
		}
	}
}
