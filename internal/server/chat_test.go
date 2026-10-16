package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/chat"
	"github.com/palarix/exponential/internal/live"
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
	s.Live = NewLiveHub()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.jsonl")
	os.WriteFile(fixture, []byte(chatFixture), 0o644)
	if body == "" {
		body = `cat "` + fixture + `"`
	}
	script := filepath.Join(dir, "claude")
	// Like Claude Code, the fake lives across turns: it runs body for each
	// user message.
	os.WriteFile(script, []byte("#!/bin/sh\nturn() {\n"+body+"\n:\n}\nwhile IFS= read -r line; do\ncase \"$line\" in *'\"type\":\"user\"'*) turn ;; esac\ndone\n"), 0o755)

	store := chat.NewStore(filepath.Join(dir, "chats"))
	// Both providers run the fake claude; the driver does not matter here.
	fake := &chat.ClaudeDriver{Binary: script}
	runner := chat.NewRunner(chat.RunnerConfig{
		Store:          store,
		Drivers:        map[string]chat.Driver{chat.ProviderClaude: fake, chat.ProviderCodex: fake},
		Dir:            dir,
		InterruptGrace: 200 * time.Millisecond,
		OnEvent:        ChatEventsTo(s.Live),
	})
	t.Cleanup(runner.Shutdown)
	s.Chat = &ChatService{
		Store:   store,
		Runner:  runner,
		Catalog: chat.NewCatalog("", func(name string) (string, error) { return "/usr/bin/" + name, nil }),
	}
	return s, s.SetupRoutes()
}

// newChatServer is setupChatServer behind an HTTP server.
func newChatServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	s, h := setupChatServer(t, "")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return s, srv
}

// waitApproval returns the ID of the chat's first approval request.
func waitApproval(t *testing.T, s *Server, id string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, events, _ := s.Chat.Store.Get(id)
		for _, e := range events {
			if e.Type == chat.EventApprovalRequested {
				return e.ApprovalID
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no approval_requested event")
	return ""
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

func TestChat_StreamReplaysAfterAfter(t *testing.T) {
	s, srv := newChatServer(t)
	h := srv.Config.Handler
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hello"}`)
	s.Chat.Runner.Wait()

	c := openEvents(t, srv.URL)
	c.topics(`{"add":[{"topic":"chat:` + m.ID + `","after":1}]}`)
	var seqs, types []string
	for len(types) < 2 {
		f := c.next()
		data, _ := f.data["data"].(map[string]any)
		seqs = append(seqs, fmt.Sprint(data["seq"]))
		types = append(types, fmt.Sprint(data["type"]))
	}
	if strings.Join(seqs, ",") != "2,3" || strings.Join(types, ",") != "assistant_text,turn_done" {
		t.Fatalf("seqs = %v, types = %v", seqs, types)
	}
}

// A client that stops reading (a frozen background tab) falls behind. Its
// stream stays open, and once it reads again it gets the whole turn: the
// settled events, replayed from the store, once and in order.
func TestChat_StreamCatchesUpAfterTheClientFellBehind(t *testing.T) {
	// 3000 deltas of 1 KiB: more than the socket buffers and the topic's
	// queue hold, while the stored text stays under the store's line limit.
	const deltas = 3000
	body := `awk 'BEGIN { s = sprintf("%1024s", ""); gsub(/ /, "x", s);
for (i = 0; i < ` + fmt.Sprint(deltas) + `; i++) printf "{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"%s\"}}}\n", s;
print "{\"type\":\"result\",\"result\":\"done\",\"session_id\":\"sess-9\",\"duration_ms\":5}" }'`
	s, h := setupChatServer(t, body)
	s.Live.QueueCap = 8
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	m := createChat(t, h)

	c := openEvents(t, srv.URL)
	c.topics(`{"add":[{"topic":"chat:` + m.ID + `"}]}`)
	// Do not read while the turn runs.
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hello"}`)
	s.Chat.Runner.Wait()

	_, stored, _ := s.Chat.Store.Get(m.ID)
	var got []int
	deltasGot := 0
	for {
		f := c.next()
		if f.event != live.EventLog {
			t.Fatalf("event = %+v, want only log frames", f)
		}
		data, _ := f.data["data"].(map[string]any)
		if seq, ok := data["seq"].(float64); ok {
			got = append(got, int(seq))
		} else {
			deltasGot++
		}
		if data["type"] == chat.EventTurnDone {
			break
		}
	}
	if deltasGot >= deltas {
		t.Fatalf("got all %d deltas: the client never fell behind, so this tests nothing", deltasGot)
	}
	if len(got) != len(stored) {
		t.Fatalf("settled seqs = %v, want all %d stored events once", got, len(stored))
	}
	for i, seq := range got {
		if seq != stored[i].Seq {
			t.Fatalf("settled seqs = %v, want them in order", got)
		}
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
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"merge it"}`); w.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", w.Code, w.Body.String())
	}

	approvalID := waitApproval(t, s, m.ID)
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

const chatQuestionRequest = `{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Which color?","header":"Color","options":[{"label":"Red","description":"Warm"}],"multiSelect":false}]},"tool_use_id":"toolu_1"}}`

func TestChat_QuestionAnswer(t *testing.T) {
	s, h := setupChatServer(t, `printf '%s\n' '`+chatQuestionRequest+`'; IFS= read -r answer; cat "$(dirname "$0")/fixture.jsonl"`)
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"ask me"}`)

	approvalID := waitApproval(t, s, m.ID)
	if got := waitingChats(t, h); len(got) != 1 {
		t.Errorf("waiting chats = %v, want the chat with the open question", got)
	}
	path := "/api/local/chats/" + m.ID + "/approvals/" + approvalID
	if w := doJSON(t, h, "POST", path, `{"decision":"answer","answers":{}}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty answers: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", path, `{"decision":"answer","answers":{"Which color?":"Red"}}`); w.Code != http.StatusOK {
		t.Fatalf("answer: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Wait()

	_, events, _ := s.Chat.Store.Get(m.ID)
	for _, e := range events {
		if e.Type == chat.EventApprovalResolved && (e.Decision != chat.DecisionAnswer || e.Answers["Which color?"] != "Red") {
			t.Errorf("approval_resolved = %+v", e)
		}
	}
}

func TestChat_ApprovalUnknownChat(t *testing.T) {
	_, h := setupChatServer(t, "")
	w := doJSON(t, h, "POST", "/api/local/chats/00000000-0000-0000-0000-000000000000/approvals/x", `{"decision":"allow"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

func TestChat_BackgroundTasksAndStop(t *testing.T) {
	s, h := setupChatServer(t, `printf '%s\n' '{"type":"system","subtype":"init","session_id":"sess-9"}'
printf '%s\n' '{"type":"system","subtype":"background_tasks_changed","tasks":[{"task_id":"bg-1","task_type":"local_bash","description":"Serve the board"}]}'
printf '%s\n' '{"type":"result","result":"ok","session_id":"sess-9","duration_ms":5}'`)
	m := createChat(t, h)
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/background/stop", ""); w.Code != http.StatusConflict {
		t.Fatalf("stop without an agent: %d", w.Code)
	}
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"serve it"}`)
	s.Chat.Runner.Wait()

	var got struct {
		Running    bool                  `json:"running"`
		Background []chat.BackgroundTask `json:"background"`
	}
	json.Unmarshal(doJSON(t, h, "GET", "/api/local/chats/"+m.ID, "").Body.Bytes(), &got)
	if got.Running || len(got.Background) != 1 || got.Background[0].Kind != chat.TaskMonitor || got.Background[0].Description != "Serve the board" {
		t.Fatalf("chat = %+v, want a finished turn and one monitor task", got)
	}

	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/background/stop", ""); w.Code != http.StatusOK {
		t.Fatalf("stop: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(doJSON(t, h, "GET", "/api/local/chats/"+m.ID, "").Body.Bytes(), &got)
	if got.Background == nil || len(got.Background) != 0 {
		t.Errorf("background after stop = %v, want []", got.Background)
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/missing/background/stop", ""); w.Code != http.StatusNotFound {
		t.Errorf("unknown chat: %d", w.Code)
	}
}
