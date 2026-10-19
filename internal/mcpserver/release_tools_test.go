package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/palarix/exponential/internal/jsonio"
)

func strp(s string) *string { return &s }

func releaseOp(t *testing.T, ts *toolset, in jsonio.ReleaseToolInput) jsonio.ReleaseOutput {
	t.Helper()
	_, out, err := ts.release(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("release %s: %v", in.Operation, err)
	}
	return out
}

func TestReleaseTool_Lifecycle(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()

	list := releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "list"})
	if len(list.Releases) != 2 || !list.Releases[0].Placeholder || list.Releases[0].Slot != "current" || list.Releases[1].Slot != "next" || !list.Releases[1].Locked {
		t.Fatalf("empty board should list placeholder Current and Next: %+v", list.Releases)
	}

	created := releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "create", Label: strp("1.5"), Goal: strp("ship")})
	if created.Release == nil || created.Release.Ref != "#1" || created.Release.Name != "1.5" {
		t.Fatalf("created = %+v", created.Release)
	}

	_, add, err := ts.add(ctx, nil, jsonio.AddInput{Title: "a", Release: "@current"})
	if err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: add.ID})
	if show.ReleaseID != created.Release.ID || show.Release != "1.5" {
		t.Errorf("show release = %q / %q", show.ReleaseID, show.Release)
	}

	// Moving to @next saves the Next placeholder.
	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: add.ID, UpdateInput: jsonio.UpdateInput{Release: strp("@next")}}); err != nil {
		t.Fatal(err)
	}
	_, listed, _ := ts.list(ctx, nil, jsonio.ListToolInput{Release: "#2"})
	if len(listed.Issues) != 1 || listed.Issues[0].Release != "#2" {
		t.Errorf("list release R2 = %+v", listed.Issues)
	}

	upd := releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "update", ID: "#2", Label: strp("1.6")})
	if upd.Release.Name != "1.6" {
		t.Errorf("updated = %+v", upd.Release)
	}
	releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "reorder", ID: "1.6", Before: "1.5"})
	list = releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "list"})
	if list.Releases[0].Name != "1.6" {
		t.Errorf("reorder: current = %+v", list.Releases[0])
	}

	done := releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "complete", ID: "@current"})
	if done.Release.State != "RELEASED" || len(done.Messages) == 0 {
		t.Errorf("complete = %+v", done)
	}
	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: add.ID, UpdateInput: jsonio.UpdateInput{Release: strp("1.6")}}); err == nil {
		t.Error("assigning to a RELEASED release must fail")
	}
	releaseOp(t, ts, jsonio.ReleaseToolInput{Operation: "delete", ID: "1.5", Issues: "remove"})
	_, listed, _ = ts.list(ctx, nil, jsonio.ListToolInput{Release: "none", IncludeDone: true})
	if len(listed.Issues) != 1 {
		t.Errorf("after delete with remove the issue is in no release: %+v", listed.Issues)
	}
}

func TestReleaseTool_Errors(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	for _, in := range []jsonio.ReleaseToolInput{
		{Operation: "bogus"},
		{Operation: "update", ID: "@current"},
		{Operation: "complete", ID: "@current"},
		{Operation: "delete", ID: "@next"},
		{Operation: "complete"},
	} {
		if _, _, err := ts.release(context.Background(), nil, in); err == nil {
			t.Errorf("%+v should fail", in)
		}
	}
}

func TestUpdateTool_ReleaseNullClears(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()
	_, add, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "a", Release: "@current"})

	raw, _ := json.Marshal(map[string]interface{}{"id": add.ID, "release": nil})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: raw}}
	if _, _, err := ts.update(ctx, req, jsonio.UpdateToolInput{ID: add.ID}); err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: add.ID})
	if show.ReleaseID != "" {
		t.Errorf("release null should clear, got %q", show.ReleaseID)
	}
}

func TestUpdateTool_WithChildren(t *testing.T) {
	ts, cleanup := setup(t)
	defer cleanup()
	ctx := context.Background()
	_, epic, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "epic", Release: "@current"})
	_, child, _ := ts.add(ctx, nil, jsonio.AddInput{Title: "child", Parent: epic.ID})

	if _, _, err := ts.update(ctx, nil, jsonio.UpdateToolInput{ID: epic.ID, UpdateInput: jsonio.UpdateInput{Release: strp("@next"), WithChildren: true}}); err != nil {
		t.Fatal(err)
	}
	_, show, _ := ts.show(ctx, nil, jsonio.ShowToolInput{ID: child.ID})
	if show.Release != "#2" {
		t.Errorf("open child should follow with_children: %q", show.Release)
	}
}
