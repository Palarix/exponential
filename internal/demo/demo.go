package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
)

type persona struct {
	Name  string
	Email string
	Role  string
}

func (p persona) Identity() string {
	return fmt.Sprintf("%s <%s>", p.Name, p.Email)
}

var personas = []persona{
	{Name: "Sarah Chen", Email: "sarah@upurreats.com", Role: "Founder / CTO"},
	{Name: "Jake Morrison", Email: "jake@upurreats.com", Role: "Full-Stack Engineer"},
	{Name: "Luna Garcia", Email: "luna@upurreats.com", Role: "Designer / Product"},
	{Name: "Claude", Email: "agent@claude.ai", Role: "AI Agent"},
	{Name: "Devin", Email: "agent@devin.ai", Role: "AI Agent"},
	{Name: "Whiskers", Email: "agent@upurreats.local", Role: "AI Agent"},
}

const (
	pSarah    = 0
	pJake     = 1
	pLuna     = 2
	pClaude   = 3
	pDevin    = 4
	pWhiskers = 5
)

type seedIssue struct {
	Ref      string
	Title    string
	Desc     string
	Labels   []string
	Parent   string
	Estimate int
	Actor    int
	Offset   int // hours from project start
	Status   string
}

type eventKind int

const (
	evStatus eventKind = iota
	evComment
	evLink
)

type seedEvent struct {
	Ref       string
	Offset    int
	Actor     int
	Kind      eventKind
	Status    string
	Text      string
	LinkType  string
	TargetRef string
}

func Generate(outputDir string) error {
	xpoDir := filepath.Join(outputDir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .xpo directory: %w", err)
	}

	now := time.Now().UTC()
	anchor := now.Add(-60 * 24 * time.Hour) // project started 60 days ago

	issues := allIssues()
	events := allEvents()

	// Assign real IDs to issue refs
	idMap := make(map[string]string)
	counter := 0
	for _, iss := range issues {
		counter++
		idMap[iss.Ref] = fmt.Sprintf("upe-%06x", counter)
	}

	// Build model events
	var modelEvents []model.Event

	// CREATE events from issues
	for _, iss := range issues {
		issueID := idMap[iss.Ref]
		parentID := ""
		if iss.Parent != "" {
			parentID = idMap[iss.Parent]
		}
		status := iss.Status
		if status == "" {
			status = "BACKLOG"
		}
		assignee := ""
		if iss.Actor >= 0 && iss.Actor < len(personas) {
			assignee = personas[iss.Actor].Identity()
		}

		payload := model.CreatePayload{
			Title:       iss.Title,
			Description: iss.Desc,
			Status:      status,
			ParentID:    parentID,
			Estimate:    iss.Estimate,
			Labels:      iss.Labels,
			Assignee:    assignee,
		}

		ts := anchor.Add(time.Duration(iss.Offset) * time.Hour)
		actor := personas[iss.Actor]

		modelEvents = append(modelEvents, model.Event{
			ID:        issueID,
			Type:      model.EventTypeCreate,
			Payload:   payload,
			CreatedAt: ts,
			CreatedBy: actor.Identity(),
		})
	}

	// Timeline events
	commentCounter := 0
	for _, ev := range events {
		issueID, ok := idMap[ev.Ref]
		if !ok {
			continue
		}
		ts := anchor.Add(time.Duration(ev.Offset) * time.Hour)
		actor := personas[ev.Actor]

		switch ev.Kind {
		case evStatus:
			status := ev.Status
			modelEvents = append(modelEvents, model.Event{
				ID:   issueID,
				Type: model.EventTypeUpdate,
				Payload: model.UpdatePayload{
					Status: &status,
				},
				CreatedAt: ts,
				CreatedBy: actor.Identity(),
			})

		case evComment:
			commentCounter++
			modelEvents = append(modelEvents, model.Event{
				ID:   issueID,
				Type: model.EventTypeComment,
				Payload: model.CommentPayload{
					ID:   fmt.Sprintf("cmt-%06d", commentCounter),
					Text: ev.Text,
				},
				CreatedAt: ts,
				CreatedBy: actor.Identity(),
			})

		case evLink:
			targetID, ok := idMap[ev.TargetRef]
			if !ok {
				continue
			}
			kind := model.NormalizeDependencyKind(ev.LinkType)
			if kind == "" {
				continue
			}
			modelEvents = append(modelEvents, model.Event{
				ID:   issueID,
				Type: model.EventTypeUpdate,
				Payload: model.UpdatePayload{
					Dependencies: []model.Dependency{
						{SourceID: issueID, TargetID: targetID, Kind: model.DependencyKind(kind)},
					},
				},
				CreatedAt: ts,
				CreatedBy: actor.Identity(),
			})
		}
	}

	// Snap to workweek: no weekend work, Mon=planning, Tue-Thu=implementation
	modelEvents = snapToWorkWeek(modelEvents)

	// Write issues.db
	dbPath := filepath.Join(xpoDir, "issues.db")
	f, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create issues.db: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	for _, evt := range modelEvents {
		if err := encoder.Encode(evt); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	// Write config.yaml
	if err := writeConfig(xpoDir); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func isDoneEvent(evt model.Event) bool {
	if evt.Type != model.EventTypeUpdate {
		return false
	}
	up, ok := evt.Payload.(model.UpdatePayload)
	return ok && up.Status != nil && *up.Status == "DONE"
}

func isStatusEvent(evt model.Event, status string) bool {
	if evt.Type != model.EventTypeUpdate {
		return false
	}
	up, ok := evt.Payload.(model.UpdatePayload)
	return ok && up.Status != nil && *up.Status == status
}

// snapToWorkWeek adjusts event timestamps to respect a realistic work cadence:
//   - Weekend events snap to Monday (DONE events snap to Tuesday)
//   - CREATE and PLANNED events prefer Monday (planning day)
//   - DONE events prefer Tuesday-Thursday (implementation days)
//   - Friday keeps whatever lands there (testing, bug fixes)
//
// Per-issue chronological ordering is preserved.
func snapToWorkWeek(events []model.Event) []model.Event {
	day := 24 * time.Hour

	for i := range events {
		t := events[i].CreatedAt
		dow := t.Weekday()

		// 1. Snap weekends forward
		if dow == time.Saturday {
			if isDoneEvent(events[i]) {
				t = t.Add(3 * day) // Sat → Tue
			} else {
				t = t.Add(2 * day) // Sat → Mon
			}
		} else if dow == time.Sunday {
			if isDoneEvent(events[i]) {
				t = t.Add(2 * day) // Sun → Tue
			} else {
				t = t.Add(1 * day) // Sun → Mon
			}
		}

		// 2. DONE events on Monday → Tuesday
		if isDoneEvent(events[i]) && t.Weekday() == time.Monday {
			t = t.Add(1 * day)
		}

		// 3. PLANNED transitions on Wed-Thu → pull to Monday
		//    (planning happens early in the week; Tue stays for ad-hoc re-prioritization)
		if isStatusEvent(events[i], "PLANNED") {
			switch t.Weekday() {
			case time.Wednesday:
				t = t.Add(-2 * day)
			case time.Thursday:
				t = t.Add(-3 * day)
			}
		}

		events[i].CreatedAt = t
	}

	// Sort globally
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	// Fix per-issue ordering: each event for an issue must be after the previous one
	lastTime := make(map[string]time.Time)
	for i := range events {
		id := events[i].ID
		if prev, ok := lastTime[id]; ok {
			if !events[i].CreatedAt.After(prev) {
				events[i].CreatedAt = prev.Add(time.Minute)
			}
		}
		lastTime[id] = events[i].CreatedAt
	}

	// Final sort
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	return events
}

func writeConfig(xpoDir string) error {
	cfg := `name: Upurr Eats
prefix: upe-
version: 3
estimation_system: fibonacci
count_unestimated: true
automations:
    auto_complete_parent: true
    auto_close_sub_issues: true
    auto_progress_sub_issues: true
    auto_progress_parent: true
labels:
    bug: "#eb5757"
    feature: "#b36cd9"
    epic: "#5e6ad2"
    improvement: "#4da6e8"
    backend: "#e07058"
    frontend: "#26b5b0"
    mobile: "#e06091"
    design: "#d4a030"
    infra: "#6b7280"
    hotfix: "#f59e0b"
cycles:
    enabled: true
    duration: 2w
    start_day: monday
contributors:
`
	for _, p := range personas {
		cfg += fmt.Sprintf("    - \"%s <%s>\"\n", p.Name, p.Email)
	}

	return os.WriteFile(filepath.Join(xpoDir, "config.yaml"), []byte(cfg), 0644)
}
