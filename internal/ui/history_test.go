package ui

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRenderHistory_Empty(t *testing.T) {
	got := captureStdout(t, func() { RenderHistory(nil, 80) })
	if !strings.Contains(got, "No events found") {
		t.Errorf("empty events should say no events, got %q", got)
	}
}

func TestRenderHistory_AllTypes(t *testing.T) {
	now := time.Now()
	events := []model.Event{
		{ID: "x", Type: model.EventTypeCreate, CreatedBy: "Alice <a@b.com>", CreatedAt: now},
		{ID: "x", Type: model.EventTypeUpdate, CreatedBy: "Bob <b@c.com>", CreatedAt: now, Payload: model.UpdatePayload{}},
		{ID: "x", Type: model.EventTypeComment, CreatedBy: "Carol <c@d.com>", CreatedAt: now, Payload: model.CommentPayload{Text: "hi"}},
		{ID: "x", Type: model.EventTypeDelete, CreatedBy: "Dave <d@e.com>", CreatedAt: now},
	}
	got := captureStdout(t, func() { RenderHistory(events, 100) })
	if !strings.Contains(got, "CREATE") {
		t.Error("should show CREATE event type")
	}
	if !strings.Contains(got, "UPDATE") {
		t.Error("should show UPDATE event type")
	}
	if !strings.Contains(got, "COMMENT") {
		t.Error("should show COMMENT event type")
	}
	if !strings.Contains(got, "DELETE") {
		t.Error("should show DELETE event type")
	}
}
