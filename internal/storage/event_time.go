package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// stampAfter returns at, or prev+1ns when at is not after prev. Writers use
// it so created_at strictly increases in file order whatever time a caller
// stamped an event with: a clock step back, an event built before an
// earlier write, or several events sharing one time.Now().
func stampAfter(prev, at time.Time) time.Time {
	if prev.IsZero() || at.After(prev) {
		return at
	}
	return prev.Add(time.Nanosecond)
}

// stampInOrder applies stampAfter along events, starting after prev.
func stampInOrder(prev time.Time, events []model.Event) {
	for i := range events {
		events[i].CreatedAt = stampAfter(prev, events[i].CreatedAt)
		prev = events[i].CreatedAt
	}
}

// lastEventTime returns the created_at of the last complete line in the
// file at path, reading backwards from the end so an append doesn't scan
// the whole log. A torn final line (no newline) is skipped, as ReadEvents
// does. A missing or empty file returns the zero time.
func lastEventTime(path string) (time.Time, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return time.Time{}, err
	}
	size := info.Size()

	for chunk := int64(8 << 10); ; chunk *= 2 {
		start := max(size-chunk, 0)
		buf := make([]byte, size-start)
		if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
			return time.Time{}, err
		}
		// Drop a torn final line, then the partial first line unless the
		// chunk starts at the beginning of the file.
		end := bytes.LastIndexByte(buf, '\n')
		if end < 0 {
			if start == 0 {
				return time.Time{}, nil
			}
			continue
		}
		buf = buf[:end]
		if start > 0 {
			i := bytes.IndexByte(buf, '\n')
			if i < 0 {
				continue
			}
			buf = buf[i+1:]
		}

		for len(buf) > 0 {
			line := buf
			if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
				line, buf = buf[i+1:], buf[:i]
			} else {
				buf = nil
			}
			var evt struct {
				CreatedAt time.Time `json:"created_at"`
			}
			if json.Unmarshal(line, &evt) == nil {
				return evt.CreatedAt, nil
			}
		}
		if start == 0 {
			return time.Time{}, nil
		}
	}
}
