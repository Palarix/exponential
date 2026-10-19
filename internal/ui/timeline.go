package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/identity"
	"github.com/palarix/exponential/internal/model"
)

// FormatActor returns a principal-first display name: when an agent acted
// on someone's behalf, the principal leads and the agent follows as "(via …)".
func FormatActor(createdBy, onBehalfOf string) string {
	return identity.Actor(createdBy, onBehalfOf)
}

// DescribeEvent returns a human-readable description for an event.
// Returns empty string for unrecognized or empty updates (caller should skip).
func DescribeEvent(evt model.Event) string {
	return DescribeEventNamed(evt, ReleaseNames{})
}

// ReleaseNames carries the display names an event's payload refers to by
// ID only (see exponential.ReleaseNotes).
type ReleaseNames struct {
	Release string // the release an UPDATE sets, or a release event's own release
	MovedTo string // release events: where the issues moved
	Cause   string // caused events: e.g. "when 1.5 was released"
}

// DescribeEventNamed is DescribeEvent with release names filled in.
func DescribeEventNamed(evt model.Event, names ReleaseNames) string {
	desc := describeEvent(evt, names)
	if desc != "" && names.Cause != "" {
		desc += " " + names.Cause
	}
	return desc
}

func describeEvent(evt model.Event, names ReleaseNames) string {
	switch evt.Type {
	case model.EventTypeCreate:
		return "created this issue"
	case model.EventTypeDelete:
		return "deleted this issue"
	case model.EventTypeComment:
		return ""
	case model.EventTypeMerge:
		return describeMerge(evt.Payload)
	case model.EventTypeArtifact:
		return describeArtifact(evt.Payload)
	case model.EventTypeUpdate:
		return describeUpdate(evt.Payload, names.Release)
	case model.EventTypeReleaseComplete:
		p := coercePayload[model.ReleaseCompletePayload](evt.Payload)
		return "released " + names.Release + describeMoved(p.Moved, "open issue", names.MovedTo, p.MovedTo)
	case model.EventTypeReleaseDelete:
		p := coercePayload[model.ReleaseDeletePayload](evt.Payload)
		return "deleted release " + names.Release + describeMoved(p.Moved, "issue", names.MovedTo, p.MovedTo)
	default:
		return ""
	}
}

func describeMerge(payload interface{}) string {
	p := coercePayload[model.MergePayload](payload)
	var parts []string
	if p.Branch != "" {
		parts = append(parts, fmt.Sprintf("merged %s", p.Branch))
	} else {
		parts = append(parts, "merged")
	}
	if p.Strategy != "" {
		parts = append(parts, fmt.Sprintf("via %s", p.Strategy))
	}
	return strings.Join(parts, " ")
}

func describeArtifact(payload interface{}) string {
	p := coercePayload[model.ArtifactPayload](payload)
	action := p.Action
	if action == "" {
		action = "updated"
	}
	filename := p.Filename
	if filename == "" {
		filename = "artifact"
	}
	return fmt.Sprintf("%s %s", action, filename)
}

func describeUpdate(payload interface{}, release string) string {
	p := coercePayload[model.UpdatePayload](payload)
	var fragments []string

	if p.Status != nil {
		fragments = append(fragments, fmt.Sprintf("changed status to %s", *p.Status))
	}
	if p.Estimate != nil {
		if *p.Estimate == 0 {
			fragments = append(fragments, "cleared the estimate")
		} else {
			fragments = append(fragments, fmt.Sprintf("set estimate to %d", *p.Estimate))
		}
	}
	if p.Title != nil {
		fragments = append(fragments, "updated the title")
	}
	if p.Description != nil {
		fragments = append(fragments, "updated the description")
	}
	if p.Assignee != nil {
		if *p.Assignee == "" {
			fragments = append(fragments, "removed the assignee")
		} else {
			fragments = append(fragments, fmt.Sprintf("assigned to %s", ExtractName(*p.Assignee)))
		}
	}
	if p.Labels != nil && len(p.Labels) == 0 {
		fragments = append(fragments, "removed all labels")
	} else if len(p.Labels) > 0 {
		fragments = append(fragments, fmt.Sprintf("updated labels to [%s]", strings.Join(p.Labels, ", ")))
	}
	if p.Priority != nil {
		if *p.Priority == 0 {
			fragments = append(fragments, "cleared the priority")
		} else {
			fragments = append(fragments, fmt.Sprintf("set priority to %d", *p.Priority))
		}
	}
	if p.ParentID != nil {
		if *p.ParentID == "" {
			fragments = append(fragments, "removed the parent issue")
		} else {
			fragments = append(fragments, fmt.Sprintf("set parent to %s", *p.ParentID))
		}
	}
	if p.CycleID != nil {
		if *p.CycleID == "" {
			fragments = append(fragments, "removed this issue from its cycle")
		} else {
			fragments = append(fragments, fmt.Sprintf("added this issue to cycle %s", *p.CycleID))
		}
	}

	switch {
	case p.Children != nil:
		kids := fmt.Sprintf("%d open %s", p.Children.Count, plural(p.Children.Count, "child", "children"))
		subject := kids
		if p.ReleaseID != nil {
			subject = "this issue and " + kids
		}
		if p.Children.ReleaseID == "" {
			fragments = append(fragments, fmt.Sprintf("removed %s from the release", subject))
		} else {
			fragments = append(fragments, fmt.Sprintf("moved %s to %s", subject, releaseRef(release)))
		}
	case p.ReleaseID != nil && *p.ReleaseID == "":
		fragments = append(fragments, "removed this issue from its release")
	case p.ReleaseID != nil:
		fragments = append(fragments, fmt.Sprintf("moved this issue to %s", releaseRef(release)))
	}

	if len(fragments) == 0 {
		return ""
	}
	return strings.Join(fragments, " and ")
}

func releaseRef(name string) string {
	if name == "" {
		return "a release"
	}
	return "release " + name
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// describeMoved is the " · moved 12 open issues to #3" tail of a release
// complete or delete entry; movedToID "" means the issues were removed.
func describeMoved(n int, noun, movedTo, movedToID string) string {
	if n == 0 {
		return ""
	}
	what := fmt.Sprintf("%d %s", n, plural(n, noun, noun+"s"))
	if movedToID == "" {
		return fmt.Sprintf(" · removed %s from the release", what)
	}
	if movedTo == "" {
		movedTo = "another release"
	}
	return fmt.Sprintf(" · moved %s to %s", what, movedTo)
}

// coercePayload converts an event payload (which may be map[string]interface{}
// from JSON deserialization) into a concrete typed struct.
func coercePayload[T any](payload interface{}) T {
	var result T
	if payload == nil {
		return result
	}
	if typed, ok := payload.(T); ok {
		return typed
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return result
	}
	json.Unmarshal(b, &result)
	return result
}

var (
	commitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5c07b"))
	lineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3e4451"))
	dotEvent    = lipgloss.NewStyle().Foreground(lipgloss.Color("#61afef")).Render("●")
	dotCommit   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5c07b")).Render("○")
	dotMerge    = lipgloss.NewStyle().Foreground(DoingColor).Render("◆")
	dotArtifact = lipgloss.NewStyle().Foreground(AccentColor).Render("☰")
	dotComment  = lipgloss.NewStyle().Foreground(lipgloss.Color("#56b6c2")).Render("●")
)

func timelineDot(entry model.TimelineEntry) string {
	if entry.Kind == "commit" {
		return dotCommit
	}
	switch model.EventType(entry.EventType) {
	case model.EventTypeMerge:
		return dotMerge
	case model.EventTypeArtifact:
		return dotArtifact
	case model.EventTypeComment:
		return dotComment
	default:
		return dotEvent
	}
}

func rail(s string) string {
	return lineStyle.Render(s)
}

// RenderTimeline renders a human-readable activity timeline to w.
// Accepts TimelineEntry slices that may contain both issue events and commits.
func RenderTimeline(w io.Writer, entries []model.TimelineEntry, termWidth int) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No activity recorded.")
		return
	}

	rendered := 0
	for i, entry := range entries {
		isLast := i == len(entries)-1

		if entry.Kind == "commit" {
			renderCommitEntry(w, entry, isLast)
			rendered++
		} else {
			if renderIssueEventEntry(w, entry, termWidth, isLast) {
				rendered++
			}
		}
	}
	if rendered == 0 {
		fmt.Fprintln(w, "No activity recorded.")
	}
}

func renderCommitEntry(w io.Writer, entry model.TimelineEntry, isLast bool) {
	timeStr := MutedStyle.Render(humanize.Time(entry.Timestamp))
	sha := commitStyle.Render(entry.SHA)
	author := WhiteStyle.Render(entry.Author)
	msg := entry.Message

	var issueRef string
	if entry.IssueID != "" {
		issueRef = fmt.Sprintf(" %s %s",
			MutedStyle.Render("·"),
			AccentStyle.Render(entry.IssueID))
	}

	fmt.Fprintf(w, "  %s %s %s %s %s%s\n",
		timelineDot(entry), timeStr, sha, author, msg, issueRef)

	if !isLast {
		fmt.Fprintf(w, "  %s\n", rail("│"))
	}
}

func renderIssueEventEntry(w io.Writer, entry model.TimelineEntry, termWidth int, isLast bool) bool {
	timeStr := MutedStyle.Render(humanize.Time(entry.Timestamp))
	actor := WhiteStyle.Render(FormatActor(entry.CreatedBy, entry.OnBehalfOf))

	dot := timelineDot(entry)

	var idPart string
	if model.IsReleaseEvent(model.EventType(entry.EventType)) {
		idPart = "" // the description names the release
	} else if entry.IssueID != "" {
		idPart = fmt.Sprintf("%s %s ",
			AccentStyle.Render(entry.IssueID),
			MutedStyle.Render("·"))
	}

	if entry.EventType == string(model.EventTypeComment) {
		p := coercePayload[model.CommentPayload](entry.Payload)
		fmt.Fprintf(w, "  %s %s %s%s commented:\n", dot, timeStr, idPart, actor)
		renderCommentBlock(w, p.Text, termWidth)
		if !isLast {
			fmt.Fprintf(w, "  %s\n", rail("│"))
		}
		return true
	}

	evt := model.Event{
		Type:    model.EventType(entry.EventType),
		Payload: entry.Payload,
	}
	desc := DescribeEventNamed(evt, ReleaseNames{Release: entry.Release, MovedTo: entry.MovedTo, Cause: entry.CauseNote})
	if desc == "" {
		return false
	}
	fmt.Fprintf(w, "  %s %s %s%s %s\n", dot, timeStr, idPart, actor, desc)
	if !isLast {
		fmt.Fprintf(w, "  %s\n", rail("│"))
	}
	return true
}

func renderCommentBlock(w io.Writer, text string, termWidth int) {
	wrapWidth := termWidth - 6
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
	)

	var rendered string
	if err == nil {
		rendered, err = renderer.Render(text)
	}
	if err != nil {
		rendered = text + "\n"
	}

	rendered = strings.TrimRight(rendered, "\n")
	connector := rail("│")
	for _, line := range strings.Split(rendered, "\n") {
		fmt.Fprintf(w, "  %s   %s\n", connector, line)
	}
}

// RenderTimelineJSON writes timeline entries as JSONL (one JSON object per line).
func RenderTimelineJSON(entries []model.TimelineEntry, w io.Writer) {
	for _, entry := range entries {
		b, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "%s\n", b)
	}
}
