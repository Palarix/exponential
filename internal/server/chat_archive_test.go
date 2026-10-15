package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/palarix/exponential/internal/chat"
)

func TestChat_ArchiveAndUnarchive(t *testing.T) {
	_, h := setupChatServer(t, "")
	m := createChat(t, h)

	w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/archive", "")
	var got chat.Meta
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || got.ArchivedAt == nil {
		t.Fatalf("archive: %d %s", w.Code, w.Body.String())
	}

	// An archived chat is read only.
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hi"}`); w.Code != http.StatusConflict {
		t.Fatalf("send to archived chat: %d %s", w.Code, w.Body.String())
	}

	// The list still has it, flagged, so the history page can show it.
	w = doJSON(t, h, "GET", "/api/local/chats", "")
	var list []chat.Meta
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0].ArchivedAt == nil {
		t.Fatalf("list = %s", w.Body.String())
	}

	w = doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/unarchive", "")
	got = chat.Meta{}
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || got.ArchivedAt != nil {
		t.Fatalf("unarchive: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hi"}`); w.Code != http.StatusAccepted {
		t.Fatalf("send after unarchive: %d %s", w.Code, w.Body.String())
	}
}

func TestChat_ArchiveRunningChatConflicts(t *testing.T) {
	s, h := setupChatServer(t, "sleep 2")
	m := createChat(t, h)
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", `{"text":"hi"}`); w.Code != http.StatusAccepted {
		t.Fatalf("send: %d", w.Code)
	}
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/archive", ""); w.Code != http.StatusConflict {
		t.Fatalf("archive running chat: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Cancel(m.ID)
}

func TestChat_ArchiveWorkModeChatConflicts(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)
	meta, _ := s.Chat.Store.Meta(m.ID)
	meta.Mode = chat.ModeWork
	meta.Work = &chat.WorkBinding{IssueID: "xpo-1", Worktree: t.TempDir()}
	s.Chat.Store.SaveMeta(meta)

	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/archive", ""); w.Code != http.StatusConflict {
		t.Fatalf("archive chat in work mode: %d %s", w.Code, w.Body.String())
	}
}

func TestChat_ArchiveUnknownChat(t *testing.T) {
	_, h := setupChatServer(t, "")
	for _, action := range []string{"archive", "unarchive"} {
		if w := doJSON(t, h, "POST", "/api/local/chats/00000000-0000-4000-8000-000000000000/"+action, ""); w.Code != http.StatusNotFound {
			t.Errorf("%s unknown chat: %d", action, w.Code)
		}
	}
}

// "Enter work mode" on an archived chat refuses before the issue starts.
func TestChat_WorkChatRefusesArchivedChat(t *testing.T) {
	_, h, starts := setupWorkServer(t, "")
	id := seedIssue(t, "Fix login")
	m := createChat(t, h)
	doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/archive", "")

	w := doJSON(t, h, "POST", "/api/local/issues/"+id+"/work-chat", `{"chat_id":"`+m.ID+`"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("work-chat on archived chat: %d %s", w.Code, w.Body.String())
	}
	if len(*starts) != 0 {
		t.Fatalf("starts = %v, want none", *starts)
	}
}
