package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/palarix/exponential/internal/jsonio"
)

func strp(s string) *string { return &s }

func milestoneOp(t *testing.T, ts *toolset, in jsonio.MilestoneToolInput) jsonio.MilestoneOutput {
	t.Helper()
	_, out, err := ts.milestone(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("milestone %s: %v", in.Operation, err)
	}
	return out
}

func TestMilestoneTool_Lifecycle(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	list := milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "list"})
	if len(list.Milestones) != 2 || !list.Milestones[0].Placeholder || list.Milestones[0].Slot != "current" || list.Milestones[1].Slot != "next" || !list.Milestones[1].Locked {
		t.Fatalf("empty board should list placeholder Current and Next: %+v", list.Milestones)
	}

	created := milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "create", Label: strp("1.5"), Goal: strp("ship")})
	if created.Milestone == nil || created.Milestone.Ref != "#1" || created.Milestone.Name != "1.5" {
		t.Fatalf("created = %+v", created.Milestone)
	}

	_, add, err := ts.add(ctx, nil, jsonio.AddInput{Title: "a", Milestone: "@current"})
	if err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: add.ID})
	if show.MilestoneID != created.Milestone.ID || show.Milestone != "1.5" {
		t.Errorf("show milestone = %q / %q", show.MilestoneID, show.Milestone)
	}

	// Moving to @next saves the Next placeholder.
	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: add.ID, UpdateInput: jsonio.UpdateInput{Milestone: strp("@next")}}); err != nil {
		t.Fatal(err)
	}
	_, listed, _ := ts.list(ctx, nil, jsonio.ListToolInput{Milestone: "#2"})
	if len(listed.Issues) != 1 || listed.Issues[0].Milestone != "#2" {
		t.Errorf("list milestone R2 = %+v", listed.Issues)
	}

	upd := milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "update", ID: "#2", Label: strp("1.6")})
	if upd.Milestone.Name != "1.6" {
		t.Errorf("updated = %+v", upd.Milestone)
	}
	milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "reorder", ID: "1.6", Before: "1.5"})
	list = milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "list"})
	if list.Milestones[0].Name != "1.6" {
		t.Errorf("reorder: current = %+v", list.Milestones[0])
	}

	done := milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "complete", ID: "@current"})
	if done.Milestone.State != "COMPLETED" || len(done.Messages) == 0 {
		t.Errorf("complete = %+v", done)
	}
	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: add.ID, UpdateInput: jsonio.UpdateInput{Milestone: strp("1.6")}}); err == nil {
		t.Error("assigning to a COMPLETED milestone must fail")
	}
	milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "delete", ID: "1.5", Issues: "remove"})
	_, listed, _ = ts.list(ctx, nil, jsonio.ListToolInput{Milestone: "none", IncludeDone: true})
	if len(listed.Issues) != 1 {
		t.Errorf("after delete with remove the issue is in no milestone: %+v", listed.Issues)
	}
}

func TestMilestoneTool_Errors(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	for _, in := range []jsonio.MilestoneToolInput{
		{Operation: "bogus"},
		{Operation: "update", ID: "@current"},
		{Operation: "complete", ID: "@current"},
		{Operation: "delete", ID: "@next"},
		{Operation: "complete"},
	} {
		if _, _, err := ts.milestone(context.Background(), nil, in); err == nil {
			t.Errorf("%+v should fail", in)
		}
	}
}

func TestUpdateTool_MilestoneNullClears(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()
	_, add, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "a", Milestone: "@current"})

	raw, _ := json.Marshal(map[string]interface{}{"id": add.ID, "milestone": nil})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: raw}}
	if _, _, err := ts.update(ctx, req, jsonio.UpdateToolInput{ID: add.ID}); err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: add.ID})
	if show.MilestoneID != "" {
		t.Errorf("milestone null should clear, got %q", show.MilestoneID)
	}
}

func TestUpdateTool_WithChildren(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()
	_, epic, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "epic", Milestone: "@current"})
	_, child, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "child", Parent: epic.ID})

	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: epic.ID, UpdateInput: jsonio.UpdateInput{Milestone: strp("@next"), WithChildren: true}}); err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: child.ID})
	if show.Milestone != "#2" {
		t.Errorf("open child should follow with_children: %q", show.Milestone)
	}
}

func TestMilestoneTool_CompleteMovesTo(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()
	for _, label := range []string{"1.5", "1.6", "1.7"} {
		milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "create", Label: strp(label)})
	}
	_, add, err := ts.add(ctx, nil, jsonio.AddInput{Title: "a", Milestone: "1.5"})
	if err != nil {
		t.Fatal(err)
	}
	milestoneOp(t, ts, jsonio.MilestoneToolInput{Operation: "complete", ID: "1.5", To: "1.7"})
	if _, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: add.ID}); show.Milestone != "1.7" {
		t.Errorf("complete with to: issue in %q, want 1.7", show.Milestone)
	}
	if _, _, err := ts.milestone(ctx, nil, jsonio.MilestoneToolInput{Operation: "delete", ID: "1.7", Issues: "remove", To: "1.6"}); err == nil {
		t.Error("to with issues remove should fail")
	}
}
