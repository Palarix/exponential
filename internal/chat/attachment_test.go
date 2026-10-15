package chat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pngBytes is the start of a PNG file: enough for content sniffing.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestValidateAttachments(t *testing.T) {
	ok := AttachmentData{Name: "a.txt", MIME: "text/plain", Data: []byte("hi")}
	if err := ValidateAttachments([]AttachmentData{ok}); err != nil {
		t.Errorf("one small file: %v", err)
	}

	many := make([]AttachmentData, MaxAttachments+1)
	for i := range many {
		many[i] = ok
	}
	if err := ValidateAttachments(many); !errors.Is(err, ErrAttachment) {
		t.Errorf("too many: %v, want ErrAttachment", err)
	}

	bigImage := AttachmentData{Name: "big.png", MIME: "image/png", Data: append(bytes.Clone(pngBytes), make([]byte, MaxImageBytes)...)}
	if err := ValidateAttachments([]AttachmentData{bigImage}); !errors.Is(err, ErrAttachment) || !strings.Contains(err.Error(), "big.png") {
		t.Errorf("big image: %v, want ErrAttachment naming the file", err)
	}

	// A file under the file limit but over the image limit is fine when it
	// is not an image.
	bigText := AttachmentData{Name: "log.txt", MIME: "text/plain", Data: make([]byte, MaxImageBytes+1)}
	if err := ValidateAttachments([]AttachmentData{bigText}); err != nil {
		t.Errorf("big text file: %v", err)
	}
	bigFile := AttachmentData{Name: "dump.bin", Data: make([]byte, MaxFileBytes+1)}
	if err := ValidateAttachments([]AttachmentData{bigFile}); !errors.Is(err, ErrAttachment) {
		t.Errorf("big file: %v, want ErrAttachment", err)
	}

	empty := AttachmentData{Name: "empty.txt"}
	if err := ValidateAttachments([]AttachmentData{empty}); !errors.Is(err, ErrAttachment) {
		t.Errorf("empty file: %v, want ErrAttachment", err)
	}
}

func TestStore_SaveAttachments(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "chats"))
	m, _ := store.Create()

	saved, err := store.SaveAttachments(m.ID, []AttachmentData{
		{Name: "shot.png", MIME: "image/png", Data: pngBytes},
		{Name: "../../evil name.LOG", MIME: "text/plain", Data: []byte("boom")},
		{Name: "fake.png", MIME: "image/png", Data: []byte("not a png")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 3 {
		t.Fatalf("saved %d, want 3", len(saved))
	}

	img := saved[0]
	if !validID(img.ID) || img.Name != "shot.png" || img.MIME != "image/png" || img.Size != int64(len(pngBytes)) || !img.Image() {
		t.Errorf("image = %+v", img)
	}
	path, a, err := store.Attachment(m.ID, img.ID)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Join(store.Dir, m.ID, "attachments") || filepath.Ext(path) != ".png" {
		t.Errorf("path = %s", path)
	}
	if !a.Image() || a.MIME != "image/png" {
		t.Errorf("looked up = %+v", a)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("file mode: %v %v", info, err)
	}

	// The browser's name never reaches the file system; a sane extension
	// does, so the agent sees what kind of file it is.
	logPath, logA, err := store.Attachment(m.ID, saved[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(logPath) != saved[1].ID+".log" || logA.Image() {
		t.Errorf("log path = %s, attachment = %+v", logPath, logA)
	}
	if saved[1].Name != "evil name.LOG" {
		t.Errorf("name = %q, want the base name", saved[1].Name)
	}

	// A file that claims to be an image but is not goes in as a file.
	if saved[2].Image() {
		t.Errorf("fake png = %+v, want not an image", saved[2])
	}
}

func TestStore_AttachmentRejectsBadIDs(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "chats"))
	m, _ := store.Create()
	for _, id := range []string{"", "..", "../x", "*", "not-a-uuid"} {
		if _, _, err := store.Attachment(m.ID, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Attachment(%q) = %v, want ErrNotFound", id, err)
		}
	}
	if _, _, err := store.Attachment("../x", "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("bad chat ID: %v", err)
	}
}

func TestStore_DeleteRemovesAttachments(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "chats"))
	m, _ := store.Create()
	if _, err := store.SaveAttachments(m.ID, []AttachmentData{{Name: "a.txt", Data: []byte("x")}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, m.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("attachments dir still there: %v", err)
	}
}

func TestStore_ListIgnoresAttachmentDirs(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "chats"))
	m, _ := store.Create()
	store.SaveAttachments(m.ID, []AttachmentData{{Name: "a.txt", Data: []byte("x")}})
	list, err := store.List()
	if err != nil || len(list) != 1 {
		t.Errorf("list = %v, %v; want the one chat", list, err)
	}
}

func TestBuildPrompt_Attachments(t *testing.T) {
	p := ChatProfile("/repo", true)
	files := []promptFile{{Name: "shot.png", Path: "/c/a.png", Image: true}, {Name: "notes.txt", Path: "/c/b.txt"}}
	got := buildPrompt("look", nil, p, files)
	want := "<xpo-context>\nmode: chat\n</xpo-context>\n\nlook\n\n<attachments>\n- shot.png (image, attached): /c/a.png\n- notes.txt: /c/b.txt\n</attachments>"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	got = buildPrompt("", nil, p, files[1:])
	if want := "<xpo-context>\nmode: chat\n</xpo-context>\n\n<attachments>\n- notes.txt: /c/b.txt\n</attachments>"; got != want {
		t.Errorf("no text: %q", got)
	}
}

func TestRunner_ClaudeGetsAttachments(t *testing.T) {
	r, store, dir := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()

	err := r.Send(m.ID, SendRequest{Attachments: []AttachmentData{
		{Name: "shot.png", MIME: "image/png", Data: pngBytes},
		{Name: "notes.txt", MIME: "text/plain", Data: []byte("hello")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	r.Wait()

	meta, events, _ := store.Get(m.ID)
	if meta.Title != "shot.png" {
		t.Errorf("title = %q, want the first attachment's name", meta.Title)
	}
	msgs := eventsOfType(events, EventUserMessage)
	if len(msgs) != 1 || len(msgs[0].Attachments) != 2 {
		t.Fatalf("user messages = %+v, want one with two attachments", msgs)
	}
	img, file := msgs[0].Attachments[0], msgs[0].Attachments[1]
	if img.Name != "shot.png" || img.MIME != "image/png" || file.Name != "notes.txt" {
		t.Errorf("attachments = %+v", msgs[0].Attachments)
	}
	filePath, _, _ := store.Attachment(m.ID, file.ID)
	filePath, _ = filepath.Abs(filePath)

	data, _ := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	var line struct {
		Message struct {
			Content []map[string]any `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &line); err != nil {
		t.Fatalf("stdin %s: %v", data, err)
	}
	c := line.Message.Content
	if len(c) != 2 || c[0]["type"] != "image" || c[1]["type"] != "text" {
		t.Fatalf("content = %v, want an image block, then the text", c)
	}
	src, _ := c[0]["source"].(map[string]any)
	if src["type"] != "base64" || src["media_type"] != "image/png" || src["data"] != base64.StdEncoding.EncodeToString(pngBytes) {
		t.Errorf("image source = %v", src)
	}
	if text, _ := c[1]["text"].(string); !strings.Contains(text, "- notes.txt: "+filePath) {
		t.Errorf("text = %q, want the file's path", text)
	}
}

func TestRunner_RejectsBadAttachments(t *testing.T) {
	r, store, _ := newTestRunner(t, `cat "$FIXTURE"`)
	m, _ := store.Create()
	err := r.Send(m.ID, SendRequest{Text: "hi", Attachments: []AttachmentData{{Name: "empty.txt"}}})
	if !errors.Is(err, ErrAttachment) {
		t.Errorf("err = %v, want ErrAttachment", err)
	}
	if _, events, _ := store.Get(m.ID); len(events) != 0 {
		t.Errorf("events = %v, want none", events)
	}
}

func TestCodexDriver_Attachments(t *testing.T) {
	r, store, dir := newCodexRunner(t, "")
	m, _ := store.Create()
	err := r.Send(m.ID, SendRequest{Text: "see", Provider: "codex", Attachments: []AttachmentData{
		{Name: "shot.png", MIME: "image/png", Data: pngBytes},
	}})
	if err != nil {
		t.Fatal(err)
	}
	r.Wait()

	_, events, _ := store.Get(m.ID)
	att := eventsOfType(events, EventUserMessage)[0].Attachments[0]
	path, _, _ := store.Attachment(m.ID, att.ID)
	path, _ = filepath.Abs(path)

	starts := sentRequests(t, dir)["turn/start"]
	if len(starts) != 1 {
		t.Fatalf("turn/start sent %d times", len(starts))
	}
	input, _ := params(starts[0])["input"].([]any)
	if len(input) != 2 {
		t.Fatalf("input = %v, want text and an image", input)
	}
	text, _ := input[0].(map[string]any)
	img, _ := input[1].(map[string]any)
	if text["type"] != "text" || img["type"] != "localImage" || img["path"] != path {
		t.Errorf("input = %v, want text, then localImage %s", input, path)
	}
}
