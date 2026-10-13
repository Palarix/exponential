package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/chat"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// setupWorkServer returns a board server with Work Mode enabled. Starting
// an issue makes a directory under the temp dir instead of a git worktree,
// and FindWorktree finds those directories; starts records the issues
// started. An issue titled "fails" cannot start.
func setupWorkServer(t *testing.T, body string) (*Server, http.Handler, *[]string) {
	t.Helper()
	s, _ := setupChatServer(t, body)
	dir := t.TempDir()
	old := s.Chat.Runner
	t.Cleanup(old.Shutdown)
	script := filepath.Join(dir, "claude")
	if body == "" {
		body = "sleep 0"
	}
	os.WriteFile(script, []byte("#!/bin/sh\nIFS= read -r prompt\n"+body+"\n"), 0o755)
	fake := &chat.ClaudeDriver{Binary: script}
	s.Chat.Runner = chat.NewRunner(chat.RunnerConfig{
		Store:   s.Chat.Store,
		Drivers: map[string]chat.Driver{chat.ProviderClaude: fake, chat.ProviderCodex: fake},
		Dir:     dir,
		CanWork: true,
	})
	t.Cleanup(s.Chat.Runner.Shutdown)
	var starts []string
	s.Chat.StartWork = func(id string) (string, error) {
		if issues, _ := s.GetProjectedIssues(); issues[id] != nil && issues[id].Title == "fails" {
			return "", errors.New("issue " + id + " is BLOCKED")
		}
		starts = append(starts, id)
		wt := filepath.Join(dir, "worktrees", id)
		return wt, os.MkdirAll(wt, 0o755)
	}
	s.Chat.FindWorktree = func(id string) (string, bool) {
		wt := filepath.Join(dir, "worktrees", id)
		_, err := os.Stat(wt)
		return wt, err == nil
	}
	return s, s.SetupRoutes(), &starts
}

// seedDoingIssue seeds an issue and moves it to DOING.
func seedDoingIssue(t *testing.T, title string) string {
	t.Helper()
	id := seedIssue(t, title)
	status := "DOING"
	if err := storage.AppendEvent(model.Event{
		ID:        id,
		Type:      model.EventTypeUpdate,
		Payload:   model.UpdatePayload{Status: &status},
		CreatedBy: "Test User <test@test.com>",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

type workChatResponse struct {
	Chat    chat.Meta `json:"chat"`
	Created bool      `json:"created"`
	Resumed bool      `json:"resumed"`
}

func postWorkChat(t *testing.T, h http.Handler, issue string) (int, workChatResponse, string) {
	t.Helper()
	w := doJSON(t, h, "POST", "/api/local/issues/"+issue+"/work-chat", "")
	var got workChatResponse
	json.Unmarshal(w.Body.Bytes(), &got)
	return w.Code, got, w.Body.String()
}

func TestChat_WorkChatStartsIssueAndBindsNewChat(t *testing.T) {
	_, h, starts := setupWorkServer(t, "")
	id := seedIssue(t, "Fix login")

	code, got, body := postWorkChat(t, h, id)
	if code != http.StatusCreated || !got.Created {
		t.Fatalf("work-chat: %d %s", code, body)
	}
	m := got.Chat
	if m.Mode != chat.ModeWork || m.Work == nil || m.Work.IssueID != id || !strings.HasSuffix(m.Work.Worktree, id) {
		t.Errorf("chat = %+v, want Work Mode on %s", m, id)
	}
	// The title makes the chat easy to find in recent chats.
	if m.Title != id+" Fix login" {
		t.Errorf("title = %q", m.Title)
	}
	if len(*starts) != 1 || (*starts)[0] != id {
		t.Errorf("starts = %v", *starts)
	}

	// The chat has no message yet: nothing runs until the user sends one.
	w := doJSON(t, h, "GET", "/api/local/chats/"+m.ID, "")
	if strings.Contains(w.Body.String(), `"user_message"`) || strings.Contains(w.Body.String(), `"running":true`) {
		t.Errorf("chat = %s, want no turn", w.Body.String())
	}

	// A second Start Work opens the same chat and starts nothing.
	code, again, body := postWorkChat(t, h, id)
	if code != http.StatusOK || again.Created || again.Chat.ID != m.ID {
		t.Fatalf("second work-chat: %d %s", code, body)
	}
	if len(*starts) != 1 {
		t.Errorf("starts = %v, want one", *starts)
	}
}

// Continue Work: a DOING issue gets a chat bound to its worktree, and is not
// started again.
func TestChat_WorkChatContinuesDoingIssue(t *testing.T) {
	s, h, starts := setupWorkServer(t, "")
	id := seedDoingIssue(t, "Half done")
	wt, _ := s.Chat.StartWork(id) // its worktree from an earlier start
	*starts = nil

	code, got, body := postWorkChat(t, h, id)
	if code != http.StatusCreated || got.Chat.Work == nil || got.Chat.Work.Worktree != wt {
		t.Fatalf("work-chat: %d %s", code, body)
	}
	if len(*starts) != 0 {
		t.Errorf("starts = %v, want none for a DOING issue", *starts)
	}
}

func TestChat_WorkChatDoingIssueWithoutWorktree(t *testing.T) {
	s, h, starts := setupWorkServer(t, "")
	id := seedDoingIssue(t, "Elsewhere")
	code, _, body := postWorkChat(t, h, id)
	if code != http.StatusConflict || !strings.Contains(body, "no worktree") || len(*starts) != 0 {
		t.Fatalf("got %d %s, starts %v", code, body, *starts)
	}
	if list, _ := s.Chat.Store.List(); len(list) != 0 {
		t.Errorf("chats = %+v, want none", list)
	}
}

func TestChat_WorkChatUnknownIssue(t *testing.T) {
	_, h, _ := setupWorkServer(t, "")
	if code, _, body := postWorkChat(t, h, "test-ffffff"); code != http.StatusNotFound {
		t.Fatalf("got %d %s", code, body)
	}
}

func TestChat_WorkChatStartFails(t *testing.T) {
	s, h, _ := setupWorkServer(t, "")
	code, _, body := postWorkChat(t, h, seedIssue(t, "fails"))
	if code != http.StatusConflict || !strings.Contains(body, "BLOCKED") {
		t.Fatalf("got %d %s", code, body)
	}
	if list, _ := s.Chat.Store.List(); len(list) != 0 {
		t.Errorf("chats = %+v, want none", list)
	}
}

func TestChat_WorkChatUnavailableWithoutWork(t *testing.T) {
	_, h := setupChatServer(t, "") // CanWork is false, as in remote mode
	code, _, body := postWorkChat(t, h, seedIssue(t, "x"))
	if code != http.StatusNotImplemented {
		t.Fatalf("got %d %s", code, body)
	}
}

func TestChat_LeaveWork(t *testing.T) {
	_, h, _ := setupWorkServer(t, "")
	id := seedIssue(t, "x")
	_, got, _ := postWorkChat(t, h, id)

	w := doJSON(t, h, "POST", "/api/local/chats/"+got.Chat.ID+"/leave-work", "")
	var m chat.Meta
	json.Unmarshal(w.Body.Bytes(), &m)
	if w.Code != http.StatusOK || m.Mode != chat.ModeChat || m.Work != nil {
		t.Fatalf("leave: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+got.Chat.ID+"/leave-work", ""); w.Code != http.StatusConflict {
		t.Errorf("leave again: %d %s", w.Code, w.Body.String())
	}

	// The issue is free again: Start Work makes a new chat.
	code, next, body := postWorkChat(t, h, id)
	if code != http.StatusCreated || next.Chat.ID == got.Chat.ID {
		t.Errorf("work-chat after leave: %d %s", code, body)
	}
}

func TestChat_LeaveWorkWhileRunningIsConflict(t *testing.T) {
	s, h, _ := setupWorkServer(t, "sleep 5")
	_, got, _ := postWorkChat(t, h, seedIssue(t, "x"))
	if err := s.Chat.Runner.Send(got.Chat.ID, chat.SendRequest{Text: "go"}); err != nil {
		t.Fatal(err)
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+got.Chat.ID+"/leave-work", ""); w.Code != http.StatusConflict {
		t.Errorf("leave while running: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.CancelAndWait(got.Chat.ID)
}

func TestChat_LeaveWorkUnknownChat(t *testing.T) {
	_, h, _ := setupWorkServer(t, "")
	if w := doJSON(t, h, "POST", "/api/local/chats/00000000-0000-4000-8000-000000000000/leave-work", ""); w.Code != http.StatusNotFound {
		t.Errorf("got %d", w.Code)
	}
}

func postWorkChatWith(t *testing.T, h http.Handler, issue, chatID string) (int, workChatResponse, string) {
	t.Helper()
	w := doJSON(t, h, "POST", "/api/local/issues/"+issue+"/work-chat", `{"chat_id":"`+chatID+`"}`)
	var got workChatResponse
	json.Unmarshal(w.Body.Bytes(), &got)
	return w.Code, got, w.Body.String()
}

// pastWorkChat returns a chat that worked on issue and left Work Mode.
func pastWorkChat(t *testing.T, h http.Handler, issue string) chat.Meta {
	t.Helper()
	_, got, body := postWorkChat(t, h, issue)
	if got.Chat.ID == "" {
		t.Fatalf("work-chat: %s", body)
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+got.Chat.ID+"/leave-work", ""); w.Code != http.StatusOK {
		t.Fatalf("leave: %d %s", w.Code, w.Body.String())
	}
	return got.Chat
}

// "Enter work mode" on a past chat binds that chat again, with its title
// and history, instead of creating one.
func TestChat_WorkChatResumesGivenChat(t *testing.T) {
	s, h, starts := setupWorkServer(t, "")
	id := seedDoingIssue(t, "Fix login")
	s.Chat.StartWork(id) // its worktree from an earlier start
	past := pastWorkChat(t, h, id)
	*starts = nil

	code, got, body := postWorkChatWith(t, h, id, past.ID)
	if code != http.StatusOK || got.Created || !got.Resumed || got.Chat.ID != past.ID {
		t.Fatalf("work-chat: %d %s", code, body)
	}
	if got.Chat.Mode != chat.ModeWork || got.Chat.Work == nil || got.Chat.Work.IssueID != id || got.Chat.Title != past.Title {
		t.Errorf("chat = %+v", got.Chat)
	}
	// The issue is DOING; it is not started again.
	if len(*starts) != 0 {
		t.Errorf("starts = %v", *starts)
	}
	if list, _ := s.Chat.Store.List(); len(list) != 1 {
		t.Errorf("chats = %d, want 1", len(list))
	}
}

func TestChat_WorkChatResumeStartsPlannedIssue(t *testing.T) {
	s, h, starts := setupWorkServer(t, "")
	m, _ := s.Chat.Store.Create()
	id := seedIssue(t, "Reopened")
	code, got, body := postWorkChatWith(t, h, id, m.ID)
	if code != http.StatusOK || !got.Resumed || got.Chat.ID != m.ID || len(*starts) != 1 {
		t.Fatalf("work-chat: %d %s, starts %v", code, body, *starts)
	}
}

func TestChat_WorkChatResumeRefused(t *testing.T) {
	s, h, starts := setupWorkServer(t, "sleep 5")
	held := seedIssue(t, "Held")
	_, holder, _ := postWorkChat(t, h, held)
	other, _ := s.Chat.Store.Create()
	elsewhere := seedIssue(t, "Elsewhere")
	running, _ := s.Chat.Store.Create()
	if err := s.Chat.Runner.Send(running.ID, chat.SendRequest{Text: "go"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Chat.Runner.CancelAndWait(running.ID) })
	free := seedIssue(t, "Free")
	*starts = nil

	cases := []struct {
		name, issue, chatID string
		code                int
	}{
		{"another chat holds the issue", held, other.ID, http.StatusConflict},
		{"chat in Work Mode on another issue", elsewhere, holder.Chat.ID, http.StatusConflict},
		{"running chat", free, running.ID, http.StatusConflict},
		{"unknown chat", free, "00000000-0000-4000-8000-000000000000", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if code, _, body := postWorkChatWith(t, h, c.issue, c.chatID); code != c.code {
				t.Errorf("got %d %s, want %d", code, body, c.code)
			}
		})
	}
	// A refused request starts nothing.
	if len(*starts) != 0 {
		t.Errorf("starts = %v, want none", *starts)
	}
}

// The chat that holds the issue is returned as it is.
func TestChat_WorkChatResumeHolder(t *testing.T) {
	_, h, _ := setupWorkServer(t, "")
	id := seedIssue(t, "x")
	_, first, _ := postWorkChat(t, h, id)
	code, got, body := postWorkChatWith(t, h, id, first.Chat.ID)
	if code != http.StatusOK || got.Created || got.Chat.ID != first.Chat.ID {
		t.Fatalf("got %d %s", code, body)
	}
}
