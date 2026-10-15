package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/chat"
)

var testPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func attachmentJSON(name, mime string, data []byte) string {
	return fmt.Sprintf(`{"name":%q,"mime":%q,"data":%q}`, name, mime, base64.StdEncoding.EncodeToString(data))
}

func TestChat_SendWithAttachments(t *testing.T) {
	s, h := setupChatServer(t, "")
	m := createChat(t, h)

	body := `{"text":"","attachments":[` + attachmentJSON("shot.png", "image/png", testPNG) + `,` + attachmentJSON("page.html", "text/html", []byte("<script>alert(1)</script>")) + `]}`
	if w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", body); w.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", w.Code, w.Body.String())
	}
	s.Chat.Runner.Wait()

	_, events, _ := s.Chat.Store.Get(m.ID)
	atts := events[0].Attachments
	if events[0].Type != chat.EventUserMessage || len(atts) != 2 {
		t.Fatalf("first event = %+v", events[0])
	}

	// The image is served as an image.
	base := "/api/local/chats/" + m.ID + "/attachments/"
	w := doJSON(t, h, "GET", base+atts[0].ID, "")
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || w.Body.String() != string(testPNG) {
		t.Errorf("image: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("image: no nosniff header")
	}

	// Anything else is a download, never rendered in the board's origin.
	w = doJSON(t, h, "GET", base+atts[1].ID, "")
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("html: %d %q %q", w.Code, w.Header().Get("Content-Type"), w.Header().Get("Content-Disposition"))
	}

	for _, id := range []string{"..%2F..%2Fx", "nope", "00000000-0000-0000-0000-000000000000"} {
		if w := doJSON(t, h, "GET", base+id, ""); w.Code != http.StatusNotFound {
			t.Errorf("GET %s: %d, want 404", id, w.Code)
		}
	}
}

func TestChat_SendAttachmentValidation(t *testing.T) {
	_, h := setupChatServer(t, "")
	m := createChat(t, h)
	many := make([]string, chat.MaxAttachments+1)
	for i := range many {
		many[i] = attachmentJSON("a.txt", "text/plain", []byte("x"))
	}
	for name, body := range map[string]string{
		"no text, no attachments": `{"text":"","attachments":[]}`,
		"too many":                `{"text":"x","attachments":[` + strings.Join(many, ",") + `]}`,
		"empty file":              `{"text":"x","attachments":[{"name":"a.txt","data":""}]}`,
		"bad base64":              `{"text":"x","attachments":[{"name":"a.txt","data":"!!"}]}`,
	} {
		w := doJSON(t, h, "POST", "/api/local/chats/"+m.ID+"/messages", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d %s", name, w.Code, w.Body.String())
		}
	}
}
