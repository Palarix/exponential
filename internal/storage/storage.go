package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/model"
)

// AppendEvent appends one event to issues.db under the events lock. The
// line and its newline go out in a single write so a reader never sees
// one without the other. created_at is moved past the last line's when it
// isn't already later, so timestamps strictly increase in file order.
func AppendEvent(event model.Event) error {
	return withEventsLock(func() error {
		path := filepath.Join(XpoDir(), "issues.db")
		last, err := lastEventTime(path)
		if err != nil {
			return err
		}
		event.CreatedAt = stampAfter(last, event.CreatedAt)

		line, err := json.Marshal(event)
		if err != nil {
			return err
		}
		line = append(line, '\n')

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		if _, err := f.Write(line); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}
