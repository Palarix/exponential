package chat

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ErrNotFound is returned for a chat ID with no meta file.
var ErrNotFound = errors.New("chat not found")

// Store keeps chats as <id>.json (meta) and <id>.jsonl (settled events) in
// Dir. Chats are personal scratch, so files are private to the user.
type Store struct {
	Dir string
	// OnCreateDir runs once, after the store creates Dir. The board uses it
	// to gitignore .xpo/chats/.
	OnCreateDir func()

	mu      sync.Mutex
	lastSeq map[string]int
}

// NewStore returns a store rooted at dir. Dir is created on the first write.
func NewStore(dir string) *Store {
	return &Store{Dir: dir, lastSeq: map[string]int{}}
}

func (s *Store) ensureDir() error {
	if _, err := os.Stat(s.Dir); err == nil {
		return nil
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	if s.OnCreateDir != nil {
		s.OnCreateDir()
	}
	return nil
}

// validID accepts only UUIDs, so an ID can never escape Dir.
func validID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func (s *Store) metaPath(id string) string   { return filepath.Join(s.Dir, id+".json") }
func (s *Store) eventsPath(id string) string { return filepath.Join(s.Dir, id+".jsonl") }

// Create makes a new, empty chat.
func (s *Store) Create() (Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDir(); err != nil {
		return Meta{}, err
	}
	now := time.Now().UTC()
	m := Meta{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now, Agent: "claude", State: StateIdle}
	return m, s.writeMeta(m)
}

// SaveMeta overwrites a chat's meta.
func (s *Store) SaveMeta(m Meta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeMeta(m)
}

func (s *Store) writeMeta(m Meta) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.metaPath(m.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.metaPath(m.ID))
}

// Meta returns a chat's meta.
func (s *Store) Meta(id string) (Meta, error) {
	if !validID(id) {
		return Meta{}, ErrNotFound
	}
	data, err := os.ReadFile(s.metaPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return Meta{}, ErrNotFound
	}
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	return m, json.Unmarshal(data, &m)
}

// Get returns a chat's meta and settled events, oldest first.
func (s *Store) Get(id string) (Meta, []Event, error) {
	m, err := s.Meta(id)
	if err != nil {
		return Meta{}, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	events, err := s.readEvents(id)
	return m, events, err
}

func (s *Store) readEvents(id string) ([]Event, error) {
	f, err := os.Open(s.eventsPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	events := []Event{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			events = append(events, e)
		}
	}
	return events, sc.Err()
}

// Append stamps events with the next sequence numbers and the current time,
// appends them to the chat's log, and returns them as stored.
func (s *Store) Append(id string, events ...Event) ([]Event, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	seq, ok := s.lastSeq[id]
	if !ok {
		existing, err := s.readEvents(id)
		if err != nil {
			return nil, err
		}
		if n := len(existing); n > 0 {
			seq = existing[n-1].Seq
		}
	}

	f, err := os.OpenFile(s.eventsPath(id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	now := time.Now().UTC()
	out := make([]Event, len(events))
	for i, e := range events {
		seq++
		e.Seq = seq
		if e.Time.IsZero() {
			e.Time = now
		}
		line, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return nil, err
		}
		out[i] = e
	}
	s.lastSeq[id] = seq
	return out, nil
}

// List returns every chat's meta, most recently updated first.
func (s *Store) List() ([]Meta, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Meta{}, nil
	}
	if err != nil {
		return nil, err
	}
	list := []Meta{}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !validID(id) {
			continue
		}
		if m, err := s.Meta(id); err == nil {
			list = append(list, m)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt.After(list[j].UpdatedAt) })
	return list, nil
}

// MarkInterrupted flags chats left running by a previous board process.
func (s *Store) MarkInterrupted() error {
	list, err := s.List()
	if err != nil {
		return err
	}
	for _, m := range list {
		if m.State == StateRunning {
			m.State = StateInterrupted
			if err := s.SaveMeta(m); err != nil {
				return err
			}
		}
	}
	return nil
}

const maxTitleRunes = 60

// TitleFrom derives a chat title from the first user message: its first
// non-empty line, cut to 60 characters.
func TitleFrom(text string) string {
	line := strings.TrimSpace(text)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if utf8.RuneCountInString(line) <= maxTitleRunes {
		return line
	}
	return string([]rune(line)[:maxTitleRunes]) + "…"
}
