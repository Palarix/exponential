package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/chat"
)

const chatFixture = `{"type":"system","subtype":"init","session_id":"sess-9"}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi there."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"result","result":"Hi there.","session_id":"sess-9","duration_ms":5}
`

// setupChatServer returns a board server whose chat runs a fake claude that
// prints chatFixture, or runs body when given.
func setupChatServer(t *testing.T, body string) (*Server, http.Handler) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	s := setupTestServer(t)
	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.jsonl")
	os.WriteFile(fixture, []byte(chatFixture), 0o644)
	if body == "" {
		body = `cat "` + fixture + `"`
	}
	script := filepath.Join(dir, "claude")
	// The driver keeps stdin open for approval answers: read the prompt line only.
	os.WriteFile(script, []byte("#!/bin/sh\nIFS= read -r prompt\n"+body+"\n"), 0o755)

	store := chat.NewStore(filepath.Join(dir, "chats"))
	// Both providers run the fake claude; the driver does not matter here.
	fake := &chat.ClaudeDriver{Binary: script}
	runner := chat.NewRunner(chat.RunnerConfig{
		Store:   store,
		Drivers: map[string]chat.Driver{chat.ProviderClaude: fake, chat.ProviderCodex: fake},
		Dir:     dir,
	})
	t.Cleanup(runner.Shutdown)
	s.Chat = &ChatService{
		Store:   store,
		Runner:  runner,
		Catalog: chat.NewCatalog("", func(name string) (string, error) { return "/usr/bin/" + name, nil }),
	}
	return s, s.SetupRoutes()
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func createChat(t *testing.T, h http.Handler) chat.Meta {
	t.Helper()
	w := doJSON(t, h, "POST", "/api/local/chats", "")
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var m chat.Meta
	json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func TestChat_AgentStatus(t *testing.T) {
	_, h := setupChatServer(t, "")
	w := doJSON(t, h, "GET", "/api/local/agent", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"available":true`) {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestChat_AgentStatusWithoutChatService(t *testing.T) {
	s := setupTestServer(t)
	w := doJSON(t, s.SetupRoutes(), "GET", "/api/local/agent", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

func TestChat_CreateListGet(t *testing.T) {
	_, h := setupChatServer(t, "")
	m := createChat(t, h)

	w := doJSON(t, h, "GET", "/api/local/chats", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), m.ID) {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, h, "GET", "/api/local/chats/"+m.ID, "")
	var got struct {
		Meta    chat.Meta    `json:"meta"`
		Events  []chat.Event `json:"events"`
		Running bool         `json:"running"`
	}
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || got.Meta.ID != m.ID || got.Events == nil || got.Running {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
}

func TestChat_GetUnknown(t *testing.T) {
	_, h := setupChatServer(t, "")
	for _, path := range []string{"/api/local/chats/00000000-0000-4000-8000-000000000000", "/api/local/chats/nope/stream"} {
		if w := doJSON(t, h, "GET", path, ""); w.Code != http.StatusNotFound {
			t.Errorf("%s: got %d", path, w.Code)
		}
	}
}

func TestChat_SendRunsTurn(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)

	w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages",
		`{"text":"hello","context":{"view":"board","issue_id":"test-1"},"model":"sonnet","effort":"high"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Wait()

	meta, events, _ := s.Chat.Store.Get(m.ID)
	if len(events) != 3 || events[1].Text != "Hi there." || events[2].Type != chat.EventTurnDone {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Context == nil || events[0].Context.IssueID != "test-1" {
		t.Errorf("context not stored: %+v", events[0])
	}
	if meta.Model != "sonnet" || meta.Effort != "high" || meta.Title != "hello" {
		t.Errorf("meta = %+v", meta)
	}
}

func TestChat_SendValidation(t *testing.T) {
	_, h := setupChatServer(t, "")
	m := createChat(t, h)
	for _, body := range []string{
		`{"text":"   "}`,
		`{"text":"x","model":"gpt-5"}`,
		`{"text":"x","effort":"ultra"}`,
		`{"text":"x","provider":"gemini"}`,
		`{"text":"x","provider":"codex","model":"sonnet"}`,
		`not json`,
	} {
		if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d", body, w.Code)
		}
	}
}

func TestChat_SendWhileRunningIsConflict(t *testing.T) {
	s, h := setupChatServer(t, "sleep 5")
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"one"}`)
	w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"two"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}

	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/cancel", ""); w.Code != http.StatusOK {
		t.Fatalf("cancel: %d", w.Code)
	}
	s.Chat.Runner.Wait()
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/cancel", ""); w.Code != http.StatusConflict {
		t.Fatalf("cancel when idle: %d", w.Code)
	}
}

func TestChat_SendUnavailable(t *testing.T) {
	s, h := setupChatServer(t, "")
	s.Chat.Catalog = chat.NewCatalog("", func(string) (string, error) { return "", os.ErrNotExist })
	m := createChat(t, h)
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"x"}`); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d", w.Code)
	}
}

func TestChat_StreamReplaysAfterLastEventID(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hello"}`)
	s.Chat.Runner.Wait()

	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/local/chats/"+m.ID+"/stream", nil)
	req.Header.Set("Last-Event-ID", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}

	var ids []string
	var types []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() && len(types) < 2 {
		line := sc.Text()
		if id, ok := strings.CutPrefix(line, "id: "); ok {
			ids = append(ids, id)
		}
		if data, ok := strings.CutPrefix(line, "data: "); ok {
			var e chat.Event
			json.Unmarshal([]byte(data), &e)
			types = append(types, e.Type)
		}
	}
	if strings.Join(ids, ",") != "2,3" || strings.Join(types, ",") != "assistant_text,turn_done" {
		t.Fatalf("ids = %v, types = %v", ids, types)
	}
}

func TestChat_RoutesServedLocallyInProxyMode(t *testing.T) {
	var proxied bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer backend.Close()

	s, _ := setupChatServer(t, "")
	s.ProxyURL = backend.URL
	h := s.SetupRoutes()
	if w := doJSON(t, h, "POST", "/api/local/chats", ""); w.Code != http.StatusCreated || proxied {
		t.Fatalf("got %d, proxied=%v", w.Code, proxied)
	}
}

func TestChat_NotRegisteredWhenHeadless(t *testing.T) {
	s, _ := setupChatServer(t, "")
	s.Headless = true
	if w := doJSON(t, s.SetupRoutes(), "GET", "/api/local/chats", ""); w.Code == http.StatusOK {
		t.Fatalf("headless server must not serve chats: %d", w.Code)
	}
}

func TestChat_Delete(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)
	if w := doJSON(t, h, "DELETE", "/api/local/chats/"+m.ID, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	if _, err := s.Chat.Store.Meta(m.ID); err != chat.ErrNotFound {
		t.Fatalf("chat still stored: %v", err)
	}
	if w := doJSON(t, h, "DELETE", "/api/local/chats/"+m.ID, ""); w.Code != http.StatusNotFound {
		t.Fatalf("second delete: %d", w.Code)
	}
}

func TestChat_DeleteStopsRunningTurn(t *testing.T) {
	s, h := setupChatServer(t, "sleep 30")
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"x"}`)

	done := make(chan *httptest.ResponseRecorder)
	go func() { done <- doJSON(t, h, "DELETE", "/api/local/chats/"+m.ID, "") }()
	select {
	case w := <-done:
		if w.Code != http.StatusNoContent {
			t.Fatalf("delete: %d %s", w.Code, w.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("delete of a running chat did not return")
	}
	if s.Chat.Runner.Running(m.ID) {
		t.Fatal("turn still running")
	}
	if _, err := s.Chat.Store.Meta(m.ID); err != chat.ErrNotFound {
		t.Fatalf("the finished turn recreated the chat: %v", err)
	}
}

func TestChat_Rename(t *testing.T) {
	_, h := setupChatServer(t, "")
	m := createChat(t, h)

	w := doJSON(t, h, "PATCH", "/api/local/chats/"+m.ID, `{"title":"Triage login bugs"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"title":"Triage login bugs"`) {
		t.Fatalf("rename: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "PATCH", "/api/local/chats/"+m.ID, `{"title":"  "}`); w.Code != http.StatusBadRequest {
		t.Fatalf("empty title: %d", w.Code)
	}
	if w := doJSON(t, h, "PATCH", "/api/local/chats/00000000-0000-4000-8000-000000000000", `{"title":"x"}`); w.Code != http.StatusNotFound {
		t.Fatalf("unknown chat: %d", w.Code)
	}
}

func TestChat_AgentStatusListsProviders(t *testing.T) {
	s, h := setupChatServer(t, "")
	s.Chat.Catalog.Learn("claude", "", "claude-opus-5-5[1m]", 1000000)
	w := doJSON(t, h, "GET", "/api/local/agent", "")
	var st chat.AgentStatus
	json.Unmarshal(w.Body.Bytes(), &st)
	if !st.Available || st.DefaultProvider != "claude" || len(st.Providers) != 2 || st.Providers[1].ID != "codex" || len(st.Providers[0].Models) != 5 {
		t.Fatalf("status = %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"label":"Default model","name":"Claude Opus 5.5","context_window":1000000`) {
		t.Errorf("status = %s, want the learned name and context window", w.Body.String())
	}
}

func TestChat_SendWithProvider(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)
	w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hi","provider":"codex"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Wait()
	meta, _ := s.Chat.Store.Meta(m.ID)
	if meta.Agent != "codex" {
		t.Fatalf("agent = %q", meta.Agent)
	}
}

func TestChat_SendOtherProviderIsConflict(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hi","provider":"claude"}`)
	s.Chat.Runner.Wait()
	w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"again","provider":"codex"}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "new chat") {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

const chatMergeRequest = `{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"mcp__xpo__merge","input":{"id":"xpo-1"},"tool_use_id":"toolu_1"}}`

// waitingChats returns the IDs of chats the list marks as waiting.
func waitingChats(t *testing.T, h http.Handler) []string {
	t.Helper()
	var list []chat.Meta
	json.Unmarshal(doJSON(t, h, "GET", "/api/local/chats", "").Body.Bytes(), &list)
	var ids []string
	for _, m := range list {
		if m.Waiting {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

func TestChat_Approval(t *testing.T) {
	s, h := setupChatServer(t, `printf '%s\n' '`+chatMergeRequest+`'; IFS= read -r answer; cat "$(dirname "$0")/fixture.jsonl"`)
	m := createChat(t, h)
	ch, unsubscribe := s.Chat.Runner.Subscribe(m.ID)
	defer unsubscribe()
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"merge it"}`); w.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", w.Code, w.Body.String())
	}

	var approvalID string
	timeout := time.After(5 * time.Second)
	for approvalID == "" {
		select {
		case e := <-ch:
			if e.Type == chat.EventApprovalRequested {
				approvalID = e.ApprovalID
			}
		case <-timeout:
			t.Fatal("no approval_requested event")
		}
	}
	if got := waitingChats(t, h); len(got) != 1 || got[0] != m.ID {
		t.Errorf("waiting chats = %v, want [%s]", got, m.ID)
	}

	path := "/api/local/chats/" + m.ID + "/approvals/" + approvalID
	if w := doJSON(t, h, "POST", path, `{"decision":"maybe"}`); w.Code != http.StatusBadRequest {
		t.Errorf("bad decision: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/approvals/nope", `{"decision":"allow"}`); w.Code != http.StatusNotFound {
		t.Errorf("unknown approval: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", path, `{"decision":"allow"}`); w.Code != http.StatusOK {
		t.Fatalf("decide: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Wait()

	if w := doJSON(t, h, "POST", path, `{"decision":"allow"}`); w.Code != http.StatusNotFound {
		t.Errorf("answered approval: %d %s", w.Code, w.Body.String())
	}
	if got := waitingChats(t, h); len(got) != 0 {
		t.Errorf("waiting chats after the decision = %v", got)
	}
	_, events, _ := s.Chat.Store.Get(m.ID)
	var resolved string
	for _, e := range events {
		if e.Type == chat.EventApprovalResolved {
			resolved = e.Decision
		}
	}
	if resolved != chat.DecisionAllow {
		t.Errorf("resolved decision = %q", resolved)
	}
}

func TestChat_ApprovalUnknownChat(t *testing.T) {
	_, h := setupChatServer(t, "")
	w := doJSON(t, h, "POST", "/api/local/chats/00000000-0000-0000-0000-000000000000/approvals/x", `{"decision":"allow"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}
