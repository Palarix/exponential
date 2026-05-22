package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/palarix/beats/internal/model"
)

func committedLineCount() int {
	out, err := exec.Command("git", "show", "HEAD:.beats/issues.db").Output()
	if err != nil {
		return 0
	}
	s := strings.TrimRight(string(out), "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func AppendEventCollapsed(event model.Event) error {
	path := filepath.Join(".beats", "issues.db")

	committed := committedLineCount()

	events, err := ReadEvents()
	if err != nil {
		return err
	}

	uncommitted := events[committed:]

	merged := false
	for i, existing := range uncommitted {
		if existing.ID != event.ID || existing.Type != event.Type {
			continue
		}

		switch event.Type {
		case model.EventTypeUpdate:
			merged = mergeUpdatePayloads(&uncommitted[i], event)
		case model.EventTypeComment:
			merged = mergeCommentPayloads(&uncommitted[i], event)
		}
		if merged {
			break
		}
	}

	if !merged {
		uncommitted = append(uncommitted, event)
	}

	return rewriteFile(path, events[:committed], uncommitted)
}

func mergeUpdatePayloads(existing *model.Event, incoming model.Event) bool {
	oldBytes, _ := json.Marshal(existing.Payload)
	newBytes, _ := json.Marshal(incoming.Payload)

	var oldP, newP model.UpdatePayload
	json.Unmarshal(oldBytes, &oldP)
	json.Unmarshal(newBytes, &newP)

	if newP.Title != nil {
		oldP.Title = newP.Title
	}
	if newP.Description != nil {
		oldP.Description = newP.Description
	}
	if newP.Status != nil {
		oldP.Status = newP.Status
	}
	if newP.ParentID != nil {
		oldP.ParentID = newP.ParentID
	}
	if newP.Estimate != nil {
		oldP.Estimate = newP.Estimate
	}
	if newP.Priority != nil {
		oldP.Priority = newP.Priority
	}
	if newP.SortOrder != nil {
		oldP.SortOrder = newP.SortOrder
	}
	if newP.Assignee != nil {
		oldP.Assignee = newP.Assignee
	}
	if newP.Labels != nil {
		oldP.Labels = newP.Labels
	}
	if newP.Dependencies != nil {
		oldP.Dependencies = newP.Dependencies
	}

	existing.Payload = oldP
	existing.CreatedAt = incoming.CreatedAt
	return true
}

func mergeCommentPayloads(existing *model.Event, incoming model.Event) bool {
	oldBytes, _ := json.Marshal(existing.Payload)
	newBytes, _ := json.Marshal(incoming.Payload)

	var oldP, newP model.CommentPayload
	json.Unmarshal(oldBytes, &oldP)
	json.Unmarshal(newBytes, &newP)

	if oldP.ID != newP.ID {
		return false
	}

	existing.Payload = newP
	existing.CreatedAt = incoming.CreatedAt
	return true
}

func rewriteFile(path string, committed []model.Event, uncommitted []model.Event) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)

	for _, evt := range committed {
		bytes, err := json.Marshal(evt)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		w.Write(bytes)
		w.WriteString("\n")
	}

	for _, evt := range uncommitted {
		bytes, err := json.Marshal(evt)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		w.Write(bytes)
		w.WriteString("\n")
	}

	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	return os.Rename(tmp, path)
}
