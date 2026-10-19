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

// AppendEvents runs build on the current log and appends the events it
// returns, all under one hold of the events lock. Writes that span several
// events (a release complete moving many issues) land whole or not at all,
// and no other writer can change the log between the read and the write.
// With collapse, the batch folds into the uncommitted tail as
// AppendEventCollapsed does; without it, the lines go out in one write.
func AppendEvents(collapse bool, build func(current []model.Event) ([]model.Event, error)) error {
	return withEventsLock(func() error {
		current, err := ReadEvents()
		if err != nil {
			return err
		}
		batch, err := build(current)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		if collapse {
			return appendEventsCollapsed(batch)
		}

		path := filepath.Join(XpoDir(), "issues.db")
		last, err := lastEventTime(path)
		if err != nil {
			return err
		}
		stampInOrder(last, batch)
		var buf []byte
		for _, evt := range batch {
			line, err := json.Marshal(evt)
			if err != nil {
				return err
			}
			buf = append(buf, line...)
			buf = append(buf, '\n')
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		if _, err := f.Write(buf); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}
