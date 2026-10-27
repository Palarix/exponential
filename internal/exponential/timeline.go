package exponential

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func IsMeaningfulActivityEvent(evt model.Event) bool {
	switch evt.Type {
	case model.EventTypeCreate, model.EventTypeComment, model.EventTypeMerge, model.EventTypeArtifact:
		return true
	case model.EventTypeUpdate:
		payload, ok := evt.Payload.(map[string]interface{})
		if !ok {
			return true
		}
		if len(payload) == 1 {
			if _, hasOnlySortOrder := payload["sort_order"]; hasOnlySortOrder {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// IsFeedEvent reports whether an event gets its own entry in the activity
// feed and the project timeline. A bulk operation shows once: a milestone
// complete or delete is an entry, and the issue moves it caused are not.
func IsFeedEvent(evt model.Event) bool {
	if evt.Cause != "" {
		return false
	}
	if isMilestoneBulkEvent(evt.Type) {
		return true
	}
	return IsMeaningfulActivityEvent(evt)
}

func isMilestoneBulkEvent(t model.EventType) bool {
	return t == model.EventTypeMilestoneComplete || t == model.EventTypeMilestoneDelete
}

// MilestoneNotes names the milestones an event refers to, by their current
// display names.
type MilestoneNotes struct {
	Milestone   string // the milestone an UPDATE sets (or moves children to), or a milestone event's own milestone
	MovedTo   string // milestone events: where the issues moved
	CauseNote string // caused events: e.g. "when 1.5 was completed"
}

// MilestoneNoter returns a function that gives the MilestoneNotes of an event.
// The milestones are projected on first use.
func MilestoneNoter(events []model.Event) func(model.Event) MilestoneNotes {
	var byID map[string]*model.Milestone
	find := func(id string) *model.Milestone {
		if id == "" {
			return nil
		}
		if byID == nil {
			byID = make(map[string]*model.Milestone)
			for _, r := range ProjectMilestones(events) {
				byID[r.ID] = r
			}
		}
		return byID[id]
	}
	name := func(id string) string {
		if r := find(id); r != nil {
			return r.Display()
		}
		return ""
	}
	return func(evt model.Event) MilestoneNotes {
		var n MilestoneNotes
		switch evt.Type {
		case model.EventTypeUpdate:
			var p model.UpdatePayload
			decodePayload(evt.Payload, &p)
			if p.MilestoneID != nil {
				n.Milestone = name(*p.MilestoneID)
			} else if p.Children != nil {
				n.Milestone = name(p.Children.MilestoneID)
			}
		case model.EventTypeMilestoneComplete:
			var p model.MilestoneCompletePayload
			decodePayload(evt.Payload, &p)
			n.Milestone, n.MovedTo = name(evt.ID), name(p.MovedTo)
		case model.EventTypeMilestoneDelete:
			var p model.MilestoneDeletePayload
			decodePayload(evt.Payload, &p)
			n.Milestone, n.MovedTo = name(evt.ID), name(p.MovedTo)
		}
		switch {
		case strings.HasPrefix(evt.Cause, model.MilestoneIDPrefix):
			if r := find(evt.Cause); r != nil && r.Deleted {
				n.CauseNote = fmt.Sprintf("when %s was deleted", r.Display())
			} else if r != nil {
				n.CauseNote = fmt.Sprintf("when %s was completed", r.Display())
			}
		case evt.Cause != "":
			n.CauseNote = fmt.Sprintf("with %s", evt.Cause)
		}
		return n
	}
}

func listRecentCommits(branch string, maxCount int) []DetailedCommit {
	out, err := exec.Command("git", "log", "--format=%H%n%s%n%an%n%aI",
		"-n", fmt.Sprintf("%d", maxCount), branch).Output()
	if err != nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var commits []DetailedCommit
	for i := 0; i+3 < len(lines); i += 4 {
		sha := lines[i]
		if len(sha) > 12 {
			sha = sha[:12]
		}
		commits = append(commits, DetailedCommit{
			SHA:     sha,
			Message: lines[i+1],
			Author:  lines[i+2],
			Date:    lines[i+3],
		})
	}
	return commits
}

type CommitDetail struct {
	SHA     string       `json:"sha"`
	Subject string       `json:"subject"`
	Body    string       `json:"body,omitempty"`
	Author  string       `json:"author"`
	Date    string       `json:"date"`
	Files   []CommitFile `json:"files"`
}

type CommitFile struct {
	Path       string `json:"path"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
}

func GetCommitDetail(sha string) *CommitDetail {
	out, err := exec.Command("git", "log", "-1", "--format=%H%n%s%n%an%n%aI%n%b", sha).Output()
	if err != nil {
		return nil
	}
	lines := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 5)
	if len(lines) < 4 {
		return nil
	}
	detail := &CommitDetail{
		SHA:     lines[0][:12],
		Subject: lines[1],
		Author:  lines[2],
		Date:    lines[3],
	}
	if len(lines) == 5 {
		detail.Body = strings.TrimSpace(lines[4])
	}

	stat, err := exec.Command("git", "diff-tree", "--no-commit-id", "--numstat", "-r", sha).Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(stat)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			add, del := 0, 0
			fmt.Sscanf(parts[0], "%d", &add)
			fmt.Sscanf(parts[1], "%d", &del)
			detail.Files = append(detail.Files, CommitFile{
				Path:      parts[2],
				Additions: add,
				Deletions: del,
			})
		}
	}
	return detail
}

func BuildTimeline(events []model.Event, issues map[string]*model.Issue, limit int, kindFilter string) []model.TimelineEntry {
	titles := make(map[string]string, len(issues))
	for id, issue := range issues {
		titles[id] = issue.Title
	}

	var entries []model.TimelineEntry

	if kindFilter == "" || kindFilter == "issue_event" {
		notes := MilestoneNoter(events)
		for i := len(events) - 1; i >= 0; i-- {
			evt := events[i]
			if !IsFeedEvent(evt) {
				continue
			}
			n := notes(evt)
			title := titles[evt.ID]
			if model.IsMilestoneEvent(evt.Type) {
				title = n.Milestone
			}
			entries = append(entries, model.TimelineEntry{
				Milestone:    n.Milestone,
				MovedTo:    n.MovedTo,
				Kind:       "issue_event",
				Timestamp:  evt.CreatedAt,
				IssueID:    evt.ID,
				IssueTitle: title,
				EventType:  string(evt.Type),
				Payload:    evt.Payload,
				CreatedBy:  evt.CreatedBy,
				OnBehalfOf: evt.OnBehalfOf,
				Source:     evt.Source,
			})
		}
	}

	if kindFilter == "" || kindFilter == "commit" {
		base := DefaultBranch()
		commits := listRecentCommits(base, 500)
		for _, c := range commits {
			ts, _ := time.Parse(time.RFC3339, c.Date)
			entry := model.TimelineEntry{
				Kind:      "commit",
				Timestamp: ts,
				SHA:       c.SHA,
				Message:   c.Message,
				Author:    c.Author,
				Branch:    base,
			}
			for id := range titles {
				if strings.Contains(c.Message, id) {
					entry.IssueID = id
					entry.IssueTitle = titles[id]
					break
				}
			}
			entries = append(entries, entry)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}
