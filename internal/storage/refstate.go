package storage

import (
	"bufio"
	"encoding/json"
	"strings"
	"sync"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage/refstore"
)

var (
	refStore     *refstore.Store
	refStoreOnce sync.Once
)

// RefStore returns the lazily initialized ref-based store.
func RefStore() *refstore.Store {
	refStoreOnce.Do(func() {
		refStore = refstore.New(HubRoot())
	})
	return refStore
}

// ResetRefStore clears the cached store for tests.
func ResetRefStore() {
	refStoreOnce = sync.Once{}
	refStore = nil
}

// InitRefStore creates the refs/xpo/data ref in the repo.
func InitRefStore() error {
	return RefStore().Init()
}

// RefStoreReady returns true if the ref has been initialized.
func RefStoreReady() bool {
	return RefStore().RefExists()
}

// --- Events ---

func ReadEventsRef() ([]model.Event, error) {
	content, err := RefStore().ReadFile("issues.db")
	if err != nil {
		return nil, err
	}
	if content == "" {
		return []model.Event{}, nil
	}
	return parseEventsFromString(content)
}

func AppendEventRef(event model.Event) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := "xpo: " + string(event.Type) + " " + event.ID
	return RefStore().AppendFile("issues.db", string(bytes), msg)
}

func AppendEventRefCAS(event model.Event) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := "xpo: " + string(event.Type) + " " + event.ID
	for attempt := 0; attempt < 50; attempt++ {
		ok, err := RefStore().AppendFileCAS("issues.db", string(bytes), msg)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return nil
}

func ValidateEventsRef() (int, error) {
	content, err := RefStore().ReadFile("issues.db")
	if err != nil {
		return 0, err
	}
	if content == "" {
		return 0, nil
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	line := 0
	for scanner.Scan() {
		line++
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return line, err
		}
	}
	return line, scanner.Err()
}

// --- Artifacts ---

func ReadArtifact(issueID, filename string) (string, error) {
	return RefStore().ReadFile("artifacts/" + issueID + "/" + filename)
}

func WriteArtifact(issueID, filename, content string) error {
	msg := "xpo: artifact " + issueID + "/" + filename
	return RefStore().WriteFile("artifacts/"+issueID+"/"+filename, content, msg)
}

func DeleteArtifact(issueID, filename string) error {
	msg := "xpo: delete artifact " + issueID + "/" + filename
	return RefStore().DeleteFile("artifacts/"+issueID+"/"+filename, msg)
}

func ListArtifactFiles(issueID string) ([]string, error) {
	return RefStore().ListDir("artifacts/" + issueID)
}

// --- Archive ---

func ReadArchivedEventsRef() ([]model.Event, error) {
	content, err := RefStore().ReadFile("archive.db")
	if err != nil {
		return nil, err
	}
	if content == "" {
		return []model.Event{}, nil
	}
	return parseEventsFromString(content)
}

// --- Helpers ---

func parseEventsFromString(content string) ([]model.Event, error) {
	var events []model.Event
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, scanner.Err()
}
