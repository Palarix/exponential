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
	os.WriteFile(script, []byte("#!/bin/sh\ncat > /dev/null\n"+body+"\n"), 0o755)

	store := chat.NewStore(filepath.Join(dir, "chats"))
	runner := chat.NewRunner(chat.RunnerConfig{Store: store, Binary: script, Dir: dir})
	t.Cleanup(runner.Shutdown)
	s.Chat = &ChatService{
		Store:  store,
		Runner: runner,
		Status: chat.AgentStatus{Available: true, Agent: "claude"},
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
	s.Chat.Status = chat.AgentStatus{Available: false, Reason: "no claude"}
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
