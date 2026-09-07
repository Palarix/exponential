package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/model"
)

// FormatActor returns a display name, appending "(via Principal)" when
// on_behalf_of is set.
func FormatActor(createdBy, onBehalfOf string) string {
	name := ExtractName(createdBy)
	if onBehalfOf != "" {
		return fmt.Sprintf("%s (via %s)", name, ExtractName(onBehalfOf))
	}
	return name
}

// DescribeEvent returns a human-readable description for an event.
// Returns empty string for unrecognized or empty updates (caller should skip).
func DescribeEvent(evt model.Event) string {
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
		return describeUpdate(evt.Payload)
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

func describeUpdate(payload interface{}) string {
	p := coercePayload[model.UpdatePayload](payload)
	var fragments []string

	if p.Status != nil {
		fragments = append(fragments, fmt.Sprintf("changed status to %s", *p.Status))
	}
	if p.Estimate != nil {
		fragments = append(fragments, fmt.Sprintf("set estimate to %d", *p.Estimate))
	}
	if p.Title != nil {
		fragments = append(fragments, "updated the title")
	}
	if p.Description != nil {
		fragments = append(fragments, "updated the description")
	}
	if p.Assignee != nil {
		fragments = append(fragments, fmt.Sprintf("assigned to %s", ExtractName(*p.Assignee)))
	}
	if len(p.Labels) > 0 {
		fragments = append(fragments, fmt.Sprintf("updated labels to [%s]", strings.Join(p.Labels, ", ")))
	}
	if p.Priority != nil {
		fragments = append(fragments, fmt.Sprintf("set priority to %d", *p.Priority))
	}
	if p.ParentID != nil {
		if *p.ParentID == "" {
			fragments = append(fragments, "removed parent")
		} else {
			fragments = append(fragments, fmt.Sprintf("set parent to %s", *p.ParentID))
		}
	}
	if p.CycleID != nil {
		if *p.CycleID == "" {
			fragments = append(fragments, "removed from cycle")
		} else {
			fragments = append(fragments, fmt.Sprintf("added to cycle %s", *p.CycleID))
		}
	}

	if len(fragments) == 0 {
		return ""
	}
	return strings.Join(fragments, " and ")
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
	if entry.IssueID != "" {
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
	desc := DescribeEvent(evt)
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
