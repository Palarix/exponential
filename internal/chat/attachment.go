package chat

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Attachment limits. Images are capped at Anthropic's per-image limit.
const (
	MaxAttachments = 10
	MaxImageBytes  = 5 << 20
	MaxFileBytes   = 10 << 20
)

// ErrAttachment is returned for attachments over a limit.
var ErrAttachment = errors.New("invalid attachment")

// imageTypes are the image types both agents take as image input. Other
// images, such as SVG or HEIC, go in as files.
var imageTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

// AttachmentData is an attachment as the user sent it.
type AttachmentData struct {
	Name string
	MIME string // as the browser reported it
	Data []byte
}

// Attachment is a saved attachment, as the user_message event records it.
type Attachment struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}

// Image reports whether the agent gets the attachment as an image.
func (a Attachment) Image() bool { return imageTypes[a.MIME] }

// sniffImage returns data's image type when it is one the agents take, by
// its content rather than its claimed type.
func sniffImage(data []byte) string {
	if t := http.DetectContentType(data); imageTypes[t] {
		return t
	}
	return ""
}

// ValidateAttachments checks the count and the size limits.
func ValidateAttachments(list []AttachmentData) error {
	if len(list) > MaxAttachments {
		return fmt.Errorf("%w: at most %d attachments per message", ErrAttachment, MaxAttachments)
	}
	for _, a := range list {
		limit, kind := MaxFileBytes, "files"
		if sniffImage(a.Data) != "" {
			limit, kind = MaxImageBytes, "images"
		}
		switch {
		case len(a.Data) == 0:
			return fmt.Errorf("%w: %s is empty", ErrAttachment, a.Name)
		case len(a.Data) > limit:
			return fmt.Errorf("%w: %s is over %d MB, the limit for %s", ErrAttachment, a.Name, limit>>20, kind)
		}
	}
	return nil
}

var safeExt = regexp.MustCompile(`^\.[a-z0-9]{1,10}$`)

func (s *Store) attachmentsDir(id string) string { return filepath.Join(s.Dir, id, "attachments") }

// SaveAttachments writes a chat's attachments to
// <Dir>/<chat-id>/attachments/<attachment-id><ext>. The file name is a new
// UUID plus the name's extension when it is a plain one, so the agent sees
// what kind of file it is and the browser's name never reaches the file
// system.
func (s *Store) SaveAttachments(id string, list []AttachmentData) ([]Attachment, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	if len(list) == 0 {
		return nil, nil
	}
	if err := s.ensureDir(); err != nil {
		return nil, err
	}
	dir := s.attachmentsDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(list))
	for _, d := range list {
		name := filepath.Base(strings.ReplaceAll(d.Name, `\`, "/"))
		if name == "." || name == "/" {
			name = ""
		}
		a := Attachment{ID: uuid.NewString(), Name: name, MIME: d.MIME, Size: int64(len(d.Data))}
		if t := sniffImage(d.Data); t != "" {
			a.MIME = t
		} else if imageTypes[a.MIME] {
			a.MIME = "application/octet-stream" // claims to be an image, is not
		}
		if a.MIME == "" {
			a.MIME = "application/octet-stream"
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !safeExt.MatchString(ext) {
			ext = ""
		}
		if err := os.WriteFile(filepath.Join(dir, a.ID+ext), d.Data, 0o600); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// Attachment returns the path of a chat's saved attachment, with its type
// sniffed from its content: a known image type, else
// application/octet-stream. Name is not stored with the file and is empty.
func (s *Store) Attachment(id, attachmentID string) (string, Attachment, error) {
	if !validID(id) || !validID(attachmentID) {
		return "", Attachment{}, ErrNotFound
	}
	matches, err := filepath.Glob(filepath.Join(s.attachmentsDir(id), attachmentID+"*"))
	if err != nil || len(matches) == 0 {
		return "", Attachment{}, ErrNotFound
	}
	path := matches[0]
	f, err := os.Open(path)
	if err != nil {
		return "", Attachment{}, ErrNotFound
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", Attachment{}, err
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", Attachment{}, err
	}
	a := Attachment{ID: attachmentID, MIME: sniffImage(head[:n]), Size: info.Size()}
	if a.MIME == "" {
		a.MIME = "application/octet-stream"
	}
	return path, a, nil
}

// attachmentPaths returns the absolute paths of saved attachments.
func (s *Store) attachmentPaths(id string, list []Attachment) ([]string, error) {
	paths := make([]string, len(list))
	for i, a := range list {
		p, _, err := s.Attachment(id, a.ID)
		if err != nil {
			return nil, err
		}
		if paths[i], err = filepath.Abs(p); err != nil {
			return nil, err
		}
	}
	return paths, nil
}
