package beats

import (
	"testing"
	"time"

	"github.com/palarix/beats/internal/model"
)

const (
	me    = "Alice Dev <alice@example.com>"
	bob   = "Bob Ops <bob@example.com>"
	carol = "Carol QA <carol@example.com>"
)

func ts(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// buildFixture returns an event log and the issues it projects to.
func buildFixture() ([]model.Event, map[string]*model.Issue) {
	events := []model.Event{
		// Issue A: created by me, later commented on by bob.
		{ID: "a", Type: model.EventTypeCreate, CreatedBy: me, CreatedAt: ts("2026-06-15T09:00:00Z"),
			Payload: model.CreatePayload{Title: "A", Status: "BACKLOG"}},
		{ID: "a", Type: model.EventTypeComment, CreatedBy: bob, CreatedAt: ts("2026-06-15T11:00:00Z"),
			Payload: model.CommentPayload{ID: "c1", Text: "Looks good, one nit about naming"}},

		// Issue B: created by bob, assigned to me by carol.
		{ID: "b", Type: model.EventTypeCreate, CreatedBy: bob, CreatedAt: ts("2026-06-15T09:30:00Z"),
			Payload: model.CreatePayload{Title: "B", Status: "BACKLOG"}},
		{ID: "b", Type: model.EventTypeUpdate, CreatedBy: carol, CreatedAt: ts("2026-06-15T11:30:00Z"),
			Payload: model.UpdatePayload{Assignee: strptr(me)}},

		// Issue C: created by bob, no involvement from me — but it gets merged.
		{ID: "c", Type: model.EventTypeCreate, CreatedBy: bob, CreatedAt: ts("2026-06-15T09:45:00Z"),
			Payload: model.CreatePayload{Title: "C", Status: "BACKLOG"}},
		{ID: "c", Type: model.EventTypeMerge, CreatedBy: bob, CreatedAt: ts("2026-06-15T12:00:00Z"),
			Payload: model.MergePayload{Branch: "feat/c", Strategy: "squash"}},

		// Issue D: created by bob, no involvement, never merged — should never appear.
		{ID: "d", Type: model.EventTypeCreate, CreatedBy: bob, CreatedAt: ts("2026-06-15T09:50:00Z"),
			Payload: model.CreatePayload{Title: "D", Status: "BACKLOG"}},
		{ID: "d", Type: model.EventTypeUpdate, CreatedBy: carol, CreatedAt: ts("2026-06-15T12:30:00Z"),
			Payload: model.UpdatePayload{Status: strptr("DOING")}},
	}
	return events, ProjectIssues(events)
}

func strptr(s string) *string { return &s }

func TestBuildInbox_RelevanceAndMerge(t *testing.T) {
	events, issues := buildFixture()

	items := BuildInbox(events, issues, me, time.Time{})

	// Relevant issues: A (I created), B (assigned to me), C (merged, team-wide).
	// All non-self events on those issues appear:
	//   - A: bob's comment (my own CREATE is excluded)
	//   - B: bob's CREATE and carol's assignment (both by others, B is mine now)
	//   - C: bob's merge
	// Issue D never appears (no involvement, never merged).
	byIssue := map[string][]model.EventType{}
	for _, it := range items {
		byIssue[it.IssueID] = append(byIssue[it.IssueID], it.Type)
		if it.IssueID == "d" {
			t.Errorf("issue D should not appear (no involvement, no merge)")
		}
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d: %+v", len(items), items)
	}
	if got := byIssue["a"]; len(got) != 1 || got[0] != model.EventTypeComment {
		t.Errorf("issue A: expected [COMMENT], got %v", got)
	}
	if got := byIssue["c"]; len(got) != 1 || got[0] != model.EventTypeMerge {
		t.Errorf("issue C: expected [MERGE], got %v", got)
	}
	if got := len(byIssue["b"]); got != 2 {
		t.Errorf("issue B: expected 2 events (create + assign), got %d", got)
	}
}

func TestBuildInbox_NewestFirst(t *testing.T) {
	events, issues := buildFixture()
	items := BuildInbox(events, issues, me, time.Time{})

	for i := 1; i < len(items); i++ {
		if items[i-1].CreatedAt.Before(items[i].CreatedAt) {
			t.Errorf("items not sorted newest-first: %v before %v", items[i-1].CreatedAt, items[i].CreatedAt)
		}
	}
}

func TestBuildInbox_SinceCutoff(t *testing.T) {
	events, issues := buildFixture()

	// Cursor just after the comment on A and assignment on B, before the merge.
	since := ts("2026-06-15T11:45:00Z")
	items := BuildInbox(events, issues, me, since)

	if len(items) != 1 {
		t.Fatalf("expected 1 item after cutoff, got %d: %+v", len(items), items)
	}
	if items[0].IssueID != "c" || items[0].Type != model.EventTypeMerge {
		t.Errorf("expected only the merge of C, got %+v", items[0])
	}
}

func TestBuildInbox_ExcludesOwnActions(t *testing.T) {
	events, issues := buildFixture()

	// From bob's perspective: he created B/C/D and commented on A. He should
	// see carol's status change on D (he created it) and the assignment on B,
	// but never his own creations or his own comment.
	items := BuildInbox(events, issues, bob, time.Time{})
	for _, it := range items {
		if identityMatches(it.CreatedBy, bob) {
			t.Errorf("inbox should exclude own actions, got %+v", it)
		}
	}
}

func TestBuildInbox_SkipsSortOrderNoise(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, CreatedBy: me, CreatedAt: ts("2026-06-15T09:00:00Z"),
			Payload: model.CreatePayload{Title: "A"}},
		// Sort-order-only update by another actor — should be filtered out.
		{ID: "a", Type: model.EventTypeUpdate, CreatedBy: bob, CreatedAt: ts("2026-06-15T10:00:00Z"),
			Payload: model.UpdatePayload{SortOrder: strptr("a1")}},
		// A real status change on the same issue — should appear.
		{ID: "a", Type: model.EventTypeUpdate, CreatedBy: bob, CreatedAt: ts("2026-06-15T10:30:00Z"),
			Payload: model.UpdatePayload{Status: strptr("DOING")}},
	}
	issues := ProjectIssues(events)

	items := BuildInbox(events, issues, me, time.Time{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item (sort-order update filtered), got %d: %+v", len(items), items)
	}
	if items[0].Type != model.EventTypeUpdate || items[0].Payload == nil {
		t.Errorf("unexpected item: %+v", items[0])
	}
	// Confirm the surviving event is the status change, not the sort-order one.
	if got := FormatInboxItem(items[0], me); got != "Bob Ops changed a status to DOING" {
		t.Errorf("expected status-change item, got %q", got)
	}
}

// mapEvent ensures the sort-order filter also works on map[string]interface{}
// payloads (the on-disk / over-the-wire representation).
func TestBuildInbox_SkipsSortOrderNoise_MapPayload(t *testing.T) {
	events := []model.Event{
		{ID: "a", Type: model.EventTypeCreate, CreatedBy: me, CreatedAt: ts("2026-06-15T09:00:00Z"),
			Payload: map[string]interface{}{"title": "A"}},
		{ID: "a", Type: model.EventTypeUpdate, CreatedBy: bob, CreatedAt: ts("2026-06-15T10:00:00Z"),
			Payload: map[string]interface{}{"sort_order": "a1"}},
	}
	issues := ProjectIssues(events)

	items := BuildInbox(events, issues, me, time.Time{})
	if len(items) != 0 {
		t.Fatalf("expected 0 items (only create-by-me and sort-order noise), got %d: %+v", len(items), items)
	}
}

func TestBuildInbox_UpdateCountsAsParticipation(t *testing.T) {
	// Issue created by bob; alice only made an UPDATE (status transition).
	// Alice should still see subsequent events by others on that issue.
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, CreatedBy: bob, CreatedAt: ts("2026-06-15T09:00:00Z"),
			Payload: model.CreatePayload{Title: "X", Status: "BACKLOG"}},
		{ID: "x", Type: model.EventTypeUpdate, CreatedBy: me, CreatedAt: ts("2026-06-15T10:00:00Z"),
			Payload: model.UpdatePayload{Status: strptr("PLANNED")}},
		{ID: "x", Type: model.EventTypeUpdate, CreatedBy: bob, CreatedAt: ts("2026-06-15T11:00:00Z"),
			Payload: model.UpdatePayload{Status: strptr("DONE")}},
	}
	issues := ProjectIssues(events)

	items := BuildInbox(events, issues, me, time.Time{})

	// Alice should see bob's CREATE and bob's DONE transition (her own
	// PLANNED transition is excluded as a self-action).
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(items), items)
	}
	for _, it := range items {
		if identityMatches(it.CreatedBy, me) {
			t.Errorf("should not include own actions: %+v", it)
		}
	}
}

func TestFormatInboxItem(t *testing.T) {
	cases := []struct {
		name string
		item InboxItem
		want string
	}{
		{
			name: "assigned to me",
			item: InboxItem{IssueID: "beats-b", Type: model.EventTypeUpdate, CreatedBy: carol,
				Payload: model.UpdatePayload{Assignee: strptr(me)}},
			want: "Carol QA assigned beats-b to you",
		},
		{
			name: "assigned to other",
			item: InboxItem{IssueID: "beats-b", Type: model.EventTypeUpdate, CreatedBy: carol,
				Payload: model.UpdatePayload{Assignee: strptr(bob)}},
			want: "Carol QA assigned beats-b to Bob Ops",
		},
		{
			name: "status change",
			item: InboxItem{IssueID: "beats-a", Type: model.EventTypeUpdate, CreatedBy: bob,
				Payload: model.UpdatePayload{Status: strptr("BLOCKED")}},
			want: "Bob Ops changed beats-a status to BLOCKED",
		},
		{
			name: "comment",
			item: InboxItem{IssueID: "beats-a", Type: model.EventTypeComment, CreatedBy: bob,
				Payload: model.CommentPayload{Text: "Looks good"}},
			want: `Bob Ops commented on beats-a "Looks good"`,
		},
		{
			name: "merge",
			item: InboxItem{IssueID: "beats-c", Type: model.EventTypeMerge, CreatedBy: bob,
				Payload: model.MergePayload{Strategy: "squash"}},
			want: "Bob Ops merged beats-c via squash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatInboxItem(tc.item, me)
			if got != tc.want {
				t.Errorf("FormatInboxItem = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatInboxItem_MapPayload(t *testing.T) {
	// Payloads arrive as map[string]interface{} after JSON round-tripping
	// (both local read and remote decode). Ensure the formatter handles that.
	item := InboxItem{
		IssueID:   "beats-c",
		Type:      model.EventTypeMerge,
		CreatedBy: bob,
		Payload:   map[string]interface{}{"branch": "feat/c", "strategy": "squash"},
	}
	if got, want := FormatInboxItem(item, me), "Bob Ops merged beats-c via squash"; got != want {
		t.Errorf("FormatInboxItem = %q, want %q", got, want)
	}
}

func TestIdentityMatches(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{me, me, true},
		{me, "alice@example.com", true},                       // email vs full
		{"Alice <alice@example.com>", "Al <alice@example.com>", true}, // same email, diff name
		{me, bob, false},
		{"", me, false},
		{"agent", "agent", true}, // no email, exact match
		{"agent", "other", false},
	}
	for _, tc := range cases {
		if got := identityMatches(tc.a, tc.b); got != tc.want {
			t.Errorf("identityMatches(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
