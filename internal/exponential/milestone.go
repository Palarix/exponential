package exponential

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/sortorder"
	"github.com/palarix/exponential/internal/storage"
)

// Milestones are event-sourced like issues: MILESTONE_* events in issues.db,
// projected by ProjectMilestones. Current and Next are the first two OPEN
// milestones in the manual order; missing slots are placeholders that are
// stored only on their first write. See the specs of xpo-4c92d4 and
// xpo-f89e55.

// MilestoneInput holds the editable fields of a milestone. A nil field is left
// unchanged; an empty string clears it.
type MilestoneInput struct {
	Label      *string
	Goal       *string
	TargetDate *string
}

func (in MilestoneInput) empty() bool {
	return in.Label == nil && in.Goal == nil && in.TargetDate == nil
}

// --- Projection ---

// ProjectMilestones returns every milestone in the log, deleted ones included,
// ordered by number. Deleted milestones are kept so their numbers are never
// given out again.
func ProjectMilestones(events []model.Event) []*model.Milestone {
	byID := make(map[string]*model.Milestone)
	var all []*model.Milestone
	for _, evt := range events {
		if !model.IsMilestoneEvent(evt.Type) {
			continue
		}
		r := byID[evt.ID]
		switch evt.Type {
		case model.EventTypeMilestoneCreate:
			if r != nil {
				continue
			}
			var p model.MilestoneCreatePayload
			decodePayload(evt.Payload, &p)
			r = &model.Milestone{
				ID:         evt.ID,
				Number:     p.Number,
				Label:      p.Label,
				Goal:       p.Goal,
				TargetDate: p.TargetDate,
				SortOrder:  p.SortOrder,
				State:      model.MilestoneOpen,
				CreatedAt:  evt.CreatedAt,
				CreatedBy:  principalOf(evt),
			}
			byID[evt.ID] = r
			all = append(all, r)
		case model.EventTypeMilestoneUpdate:
			if r == nil {
				continue
			}
			var p model.MilestoneUpdatePayload
			decodePayload(evt.Payload, &p)
			if p.Label != nil {
				r.Label = *p.Label
			}
			if p.Goal != nil {
				r.Goal = *p.Goal
			}
			if p.TargetDate != nil {
				r.TargetDate = *p.TargetDate
			}
			if p.SortOrder != nil {
				r.SortOrder = *p.SortOrder
			}
			// started_at is set once: the first time the milestone became Current.
			if p.StartedAt != nil && r.StartedAt == nil {
				at := *p.StartedAt
				r.StartedAt = &at
			}
		case model.EventTypeMilestoneComplete:
			if r == nil || r.State == model.MilestoneCompleted {
				continue
			}
			var p model.MilestoneCompletePayload
			decodePayload(evt.Payload, &p)
			at := evt.CreatedAt
			r.State = model.MilestoneCompleted
			r.CompletedAt = &at
			stats := p.Stats
			r.Stats = &stats
		case model.EventTypeMilestoneDelete:
			if r != nil {
				r.Deleted = true
			}
		}
	}
	renumberMilestones(all)
	sort.Slice(all, func(i, j int) bool { return all[i].Number < all[j].Number })
	return all
}

// renumberMilestones resolves duplicate numbers, which a git merge of two
// branches that each created a milestone can produce. The earlier milestone
// keeps the number; the later one gets the next free number.
func renumberMilestones(all []*model.Milestone) {
	byAge := append([]*model.Milestone(nil), all...)
	sort.SliceStable(byAge, func(i, j int) bool {
		if !byAge[i].CreatedAt.Equal(byAge[j].CreatedAt) {
			return byAge[i].CreatedAt.Before(byAge[j].CreatedAt)
		}
		return byAge[i].ID < byAge[j].ID
	})
	maxNumber := 0
	for _, r := range all {
		maxNumber = max(maxNumber, r.Number)
	}
	used := make(map[int]bool)
	for _, r := range byAge {
		if r.Number <= 0 || used[r.Number] {
			maxNumber++
			r.Number = maxNumber
		}
		used[r.Number] = true
	}
}

// OpenMilestones returns the saved OPEN milestones in the manual order.
func OpenMilestones(milestones []*model.Milestone) []*model.Milestone {
	var open []*model.Milestone
	for _, r := range milestones {
		if !r.Deleted && !r.Placeholder && r.State == model.MilestoneOpen {
			open = append(open, r)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		if open[i].SortOrder != open[j].SortOrder {
			return open[i].SortOrder < open[j].SortOrder
		}
		if !open[i].CreatedAt.Equal(open[j].CreatedAt) {
			return open[i].CreatedAt.Before(open[j].CreatedAt)
		}
		return open[i].Number < open[j].Number
	})
	return open
}

// NextMilestoneNumber is the number the next saved milestone gets.
func NextMilestoneNumber(milestones []*model.Milestone) int {
	n := 0
	for _, r := range milestones {
		n = max(n, r.Number)
	}
	return n + 1
}

// CurrentNext returns the Current and Next slots. A slot with no saved OPEN
// milestone holds a placeholder with the next free number; a Next placeholder
// is locked while Current is a placeholder.
func CurrentNext(milestones []*model.Milestone) (current, next *model.Milestone) {
	slots := fillSlots(OpenMilestones(milestones), NextMilestoneNumber(milestones))
	return slots[0], slots[1]
}

func fillSlots(open []*model.Milestone, nextNumber int) [2]*model.Milestone {
	var slots [2]*model.Milestone
	for i := range slots {
		if i < len(open) {
			slots[i] = open[i]
			continue
		}
		slots[i] = &model.Milestone{Number: nextNumber, State: model.MilestoneOpen, Placeholder: true}
		nextNumber++
	}
	slots[1].Locked = slots[1].Placeholder && slots[0].Placeholder
	return slots
}

var milestoneNumberRef = regexp.MustCompile(`^#(\d+)$`)

// ResolveMilestone finds the milestone a ref names: "#7" (number), a ms- ID,
// "@current", "@next", or else a label (case-insensitive; an OPEN milestone
// wins over COMPLETED ones). "none" and "" return nil. A ref that names
// an unsaved Current or Next returns its placeholder.
func ResolveMilestone(milestones []*model.Milestone, ref string) (*model.Milestone, error) {
	ref = strings.TrimSpace(ref)
	lower := strings.ToLower(ref)
	switch lower {
	case "", "none":
		return nil, nil
	case "@current":
		cur, _ := CurrentNext(milestones)
		return cur, nil
	case "@next":
		_, next := CurrentNext(milestones)
		return next, nil
	}

	live := make([]*model.Milestone, 0, len(milestones))
	for _, r := range milestones {
		if !r.Deleted {
			live = append(live, r)
		}
	}

	if strings.HasPrefix(ref, model.MilestoneIDPrefix) {
		for _, r := range live {
			if r.ID == ref {
				return r, nil
			}
		}
		return nil, fmt.Errorf("milestone %s not found", ref)
	}

	if m := milestoneNumberRef.FindStringSubmatch(ref); m != nil {
		n, _ := strconv.Atoi(m[1])
		for _, r := range live {
			if r.Number == n {
				return r, nil
			}
		}
		cur, next := CurrentNext(milestones)
		for _, slot := range []*model.Milestone{cur, next} {
			if slot.Placeholder && slot.Number == n {
				return slot, nil
			}
		}
		return nil, fmt.Errorf("milestone #%d not found", n)
	}

	var completed *model.Milestone
	for _, r := range live {
		if strings.ToLower(r.Label) != lower {
			continue
		}
		if r.State == model.MilestoneOpen {
			return r, nil
		}
		if completed == nil || (r.CompletedAt != nil && completed.CompletedAt != nil && r.CompletedAt.After(*completed.CompletedAt)) {
			completed = r
		}
	}
	if completed != nil {
		return completed, nil
	}
	return nil, fmt.Errorf("milestone %q not found", ref)
}

// --- Validation ---

func validateMilestoneLabel(milestones []*model.Milestone, label, selfID string) error {
	if label == "" {
		return nil
	}
	if len(label) > MaxLabelLen {
		return fmt.Errorf("milestone label exceeds maximum length of %d characters", MaxLabelLen)
	}
	lower := strings.ToLower(label)
	if strings.HasPrefix(label, "#") || strings.HasPrefix(label, "@") || lower == "none" || strings.HasPrefix(lower, model.MilestoneIDPrefix) {
		return fmt.Errorf("invalid milestone label %q: labels may not start with # or @, start with ms-, or be none", label)
	}
	for _, r := range OpenMilestones(milestones) {
		if r.ID != selfID && strings.ToLower(r.Label) == lower {
			return fmt.Errorf("an open milestone is already labelled %q (%s)", r.Label, r.Ref())
		}
	}
	return nil
}

func validateTargetDate(date string) error {
	if date == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid target date %q: expected YYYY-MM-DD", date)
	}
	return nil
}

func normalizeMilestoneMode(mode string) (string, error) {
	switch mode {
	case "", model.MilestoneMoveToNext:
		return model.MilestoneMoveToNext, nil
	case model.MilestoneRemove:
		return model.MilestoneRemove, nil
	}
	return "", fmt.Errorf("invalid issues option %q: must be %s or %s", mode, model.MilestoneMoveToNext, model.MilestoneRemove)
}

// --- Writing ---

// milestoneWriter builds one batch of events against a snapshot of the log.
// Milestone events re-project the milestones, so later steps in the batch see
// milestones saved by earlier ones.
type milestoneWriter struct {
	t          *LocalTransport
	base       []model.Event
	batch      []model.Event
	milestones []*model.Milestone
	user       string
	now        time.Time
	messages   []string
}

func (t *LocalTransport) newMilestoneWriter(current []model.Event) *milestoneWriter {
	return &milestoneWriter{
		t:          t,
		base:       current,
		milestones: ProjectMilestones(current),
		user:       t.GetUser(),
		now:        time.Now().UTC(),
	}
}

func (w *milestoneWriter) emit(events ...model.Event) {
	reproject := false
	for _, evt := range events {
		w.batch = append(w.batch, evt)
		reproject = reproject || model.IsMilestoneEvent(evt.Type)
	}
	if reproject {
		all := make([]model.Event, 0, len(w.base)+len(w.batch))
		all = append(all, w.base...)
		all = append(all, w.batch...)
		w.milestones = ProjectMilestones(all)
	}
}

func (w *milestoneWriter) event(id string, typ model.EventType, payload interface{}) model.Event {
	return model.Event{ID: id, Type: typ, Payload: payload, CreatedAt: w.now, CreatedBy: w.user}
}

func (w *milestoneWriter) find(id string) *model.Milestone {
	for _, r := range w.milestones {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// create stores a new OPEN milestone after all saved OPEN milestones.
func (w *milestoneWriter) create(number int, in MilestoneInput) (*model.Milestone, error) {
	var p model.MilestoneCreatePayload
	p.Number = number
	if in.Label != nil {
		p.Label = strings.TrimSpace(*in.Label)
	}
	if in.Goal != nil {
		p.Goal = strings.TrimSpace(*in.Goal)
	}
	if in.TargetDate != nil {
		p.TargetDate = strings.TrimSpace(*in.TargetDate)
	}
	if err := validateMilestoneLabel(w.milestones, p.Label, ""); err != nil {
		return nil, err
	}
	if err := validateTargetDate(p.TargetDate); err != nil {
		return nil, err
	}

	last := ""
	if open := OpenMilestones(w.milestones); len(open) > 0 {
		last = open[len(open)-1].SortOrder
	}
	key, err := sortorder.GenerateKeyBetween(last, "")
	if err != nil {
		return nil, fmt.Errorf("failed to place milestone: %w", err)
	}
	p.SortOrder = key

	suffix, err := gonanoid.Generate("0123456789abcdef", 6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate milestone ID: %w", err)
	}
	id := model.MilestoneIDPrefix + suffix
	w.emit(w.event(id, model.EventTypeMilestoneCreate, p))
	return w.find(id), nil
}

// save stores a placeholder; a saved milestone is returned as is. A locked
// Next is refused: saved before Current, it would become Current itself.
func (w *milestoneWriter) save(r *model.Milestone, in MilestoneInput) (*model.Milestone, error) {
	if !r.Placeholder {
		return r, nil
	}
	if r.Locked {
		return nil, fmt.Errorf("Next (%s) is locked: save Current first", r.Ref())
	}
	saved, err := w.create(r.Number, in)
	if err != nil {
		return nil, err
	}
	w.messages = append(w.messages, fmt.Sprintf("Saved milestone %s", saved.Display()))
	return saved, nil
}

// assign resolves the milestone ref for an issue whose milestone is currentID
// and returns the milestone ID to store, saving a placeholder if needed. Only
// OPEN milestones accept issues; re-sending the issue's own COMPLETED milestone
// is allowed so it can be pruned as a no-op.
func (w *milestoneWriter) assign(ref, currentID string) (string, error) {
	r, err := ResolveMilestone(w.milestones, ref)
	if err != nil || r == nil {
		return "", err
	}
	if r.Placeholder {
		saved, err := w.save(r, MilestoneInput{})
		if err != nil {
			return "", err
		}
		return saved.ID, nil
	}
	if r.State == model.MilestoneCompleted && r.ID != currentID {
		return "", fmt.Errorf("milestone %s is completed: only open milestones accept issues", r.Display())
	}
	return r.ID, nil
}

// saved resolves ref to a saved OPEN milestone, the only kind that can be
// reordered, completed or deleted.
func (w *milestoneWriter) saved(ref, verb string) (*model.Milestone, error) {
	r, err := ResolveMilestone(w.milestones, ref)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("a milestone is required")
	}
	if r.Placeholder {
		return nil, fmt.Errorf("%s is a placeholder: a placeholder cannot be %s", r.Ref(), verb)
	}
	if r.State == model.MilestoneCompleted {
		return nil, fmt.Errorf("milestone %s is completed: it cannot be %s", r.Display(), verb)
	}
	return r, nil
}

// moveTarget is where "move" sends the issues of r when r is completed or
// deleted. With to, it is that OPEN milestone; a placeholder is saved and a
// locked Next refused. Without, it is the open milestone right after r, else
// the one right before it, else a new milestone: the Current placeholder once
// r is gone, saved.
func (w *milestoneWriter) moveTarget(r *model.Milestone, to string) (*model.Milestone, error) {
	if to != "" {
		target, err := ResolveMilestone(w.milestones, to)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, fmt.Errorf("to must name a milestone: use issues=%s to take the issues out of the milestone", model.MilestoneRemove)
		}
		if target.ID == r.ID && !target.Placeholder {
			return nil, fmt.Errorf("cannot move the issues of %s to itself", r.Display())
		}
		if target.State != model.MilestoneOpen {
			return nil, fmt.Errorf("milestone %s is completed: only open milestones accept issues", target.Display())
		}
		return w.save(target, MilestoneInput{})
	}

	open := OpenMilestones(w.milestones)
	for i, o := range open {
		if o.ID != r.ID {
			continue
		}
		if i+1 < len(open) {
			return open[i+1], nil
		}
		if i > 0 {
			return open[i-1], nil
		}
	}
	return w.save(fillSlots(nil, NextMilestoneNumber(w.milestones))[0], MilestoneInput{})
}

// moveMode checks a complete or delete's mode and target: to only goes with
// a move.
func moveMode(mode, to string) (string, error) {
	mode, err := normalizeMilestoneMode(mode)
	if err != nil {
		return "", err
	}
	if mode == model.MilestoneRemove && strings.TrimSpace(to) != "" {
		return "", fmt.Errorf("to cannot be combined with issues=%s", model.MilestoneRemove)
	}
	return mode, nil
}

// moveIssues emits one UPDATE per issue setting its milestone to dest. The
// events are caused by the bulk operation of cause (a milestone or an epic),
// so feeds show that operation once instead of every move.
func (w *milestoneWriter) moveIssues(issues []*model.Issue, dest, cause string) {
	sort.Slice(issues, func(i, j int) bool { return issues[i].ID < issues[j].ID })
	for _, iss := range issues {
		d := dest
		evt := w.event(iss.ID, model.EventTypeUpdate, model.UpdatePayload{MilestoneID: &d})
		evt.Cause = cause
		w.emit(evt)
	}
}

// finish sets started_at on the Current milestone when this batch made a
// saved milestone Current for the first time, then orders the timestamps.
func (w *milestoneWriter) finish() []model.Event {
	touched := false
	for _, evt := range w.batch {
		touched = touched || model.IsMilestoneEvent(evt.Type)
	}
	if touched {
		if open := OpenMilestones(w.milestones); len(open) > 0 && open[0].StartedAt == nil {
			at := w.now
			w.emit(w.event(open[0].ID, model.EventTypeMilestoneUpdate, model.MilestoneUpdatePayload{StartedAt: &at}))
		}
	}
	sequenceTimes(w.batch)
	return w.batch
}

// stamp applies the transport's source and principal to an event.
func (t *LocalTransport) stamp(evt *model.Event) {
	if t.Source != "" {
		evt.Source = t.Source
	}
	if t.OnBehalfOf != "" {
		evt.OnBehalfOf = t.OnBehalfOf
	}
}

// writeBatch appends the events build returns in one locked write.
func (t *LocalTransport) writeBatch(build func(current []model.Event) ([]model.Event, error)) error {
	return storage.AppendEvents(t.Collapse, func(current []model.Event) ([]model.Event, error) {
		batch, err := build(current)
		if err != nil {
			return nil, err
		}
		for i := range batch {
			t.stamp(&batch[i])
		}
		return batch, nil
	})
}

func (t *LocalTransport) autoCommitMilestone(msg string) {
	if t.Config.AutoCommit {
		GitCommit("xpo: " + msg)
	}
}

// milestoneOp runs one milestone write and returns the milestone it names.
func (t *LocalTransport) milestoneOp(commitMsg string, op func(w *milestoneWriter) (string, error)) (*model.Milestone, []string, error) {
	var out *model.Milestone
	var messages []string
	err := t.writeBatch(func(current []model.Event) ([]model.Event, error) {
		w := t.newMilestoneWriter(current)
		id, err := op(w)
		if err != nil {
			return nil, err
		}
		batch := w.finish()
		out = w.find(id)
		messages = w.messages
		return batch, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if out != nil {
		t.autoCommitMilestone(fmt.Sprintf("%s milestone %s", commitMsg, out.Display()))
	}
	return out, messages, nil
}

// CreateMilestone stores a new OPEN milestone after all saved OPEN milestones.
func (t *LocalTransport) CreateMilestone(in MilestoneInput) (*model.Milestone, error) {
	r, _, err := t.milestoneOp("create", func(w *milestoneWriter) (string, error) {
		r, err := w.create(NextMilestoneNumber(w.milestones), in)
		if err != nil {
			return "", err
		}
		return r.ID, nil
	})
	return r, err
}

// UpdateMilestone changes a milestone's label, goal or target date. On a
// placeholder, the write saves it.
func (t *LocalTransport) UpdateMilestone(ref string, in MilestoneInput) (*model.Milestone, error) {
	if in.empty() {
		return nil, fmt.Errorf("no fields set: provide a label, goal or target date")
	}
	r, _, err := t.milestoneOp("update", func(w *milestoneWriter) (string, error) {
		r, err := ResolveMilestone(w.milestones, ref)
		if err != nil {
			return "", err
		}
		if r == nil {
			return "", fmt.Errorf("a milestone is required")
		}
		if r.Placeholder {
			saved, err := w.save(r, in)
			if err != nil {
				return "", err
			}
			return saved.ID, nil
		}

		var p model.MilestoneUpdatePayload
		if in.Label != nil {
			label := strings.TrimSpace(*in.Label)
			if r.State == model.MilestoneOpen {
				if err := validateMilestoneLabel(w.milestones, label, r.ID); err != nil {
					return "", err
				}
			}
			if label != r.Label {
				p.Label = &label
			}
		}
		if in.Goal != nil {
			if goal := strings.TrimSpace(*in.Goal); goal != r.Goal {
				p.Goal = &goal
			}
		}
		if in.TargetDate != nil {
			date := strings.TrimSpace(*in.TargetDate)
			if err := validateTargetDate(date); err != nil {
				return "", err
			}
			if date != r.TargetDate {
				p.TargetDate = &date
			}
		}
		if p.Label != nil || p.Goal != nil || p.TargetDate != nil {
			w.emit(w.event(r.ID, model.EventTypeMilestoneUpdate, p))
		}
		return r.ID, nil
	})
	return r, err
}

// ReorderMilestone moves a saved OPEN milestone directly above (before) or
// below (after) another one.
func (t *LocalTransport) ReorderMilestone(ref, before, after string) (*model.Milestone, error) {
	if (before == "") == (after == "") {
		return nil, fmt.Errorf("set exactly one of before or after")
	}
	r, _, err := t.milestoneOp("reorder", func(w *milestoneWriter) (string, error) {
		r, err := w.saved(ref, "reordered")
		if err != nil {
			return "", err
		}
		anchorRef, below := before, false
		if after != "" {
			anchorRef, below = after, true
		}
		anchor, err := w.saved(anchorRef, "a reorder anchor")
		if err != nil {
			return "", err
		}
		if anchor.ID == r.ID {
			return "", fmt.Errorf("cannot move a milestone relative to itself")
		}

		var rest []*model.Milestone
		for _, o := range OpenMilestones(w.milestones) {
			if o.ID != r.ID {
				rest = append(rest, o)
			}
		}
		idx := 0
		for i, o := range rest {
			if o.ID == anchor.ID {
				idx = i
			}
		}
		lo, hi := "", rest[idx].SortOrder
		if idx > 0 {
			lo = rest[idx-1].SortOrder
		}
		if below {
			lo, hi = rest[idx].SortOrder, ""
			if idx+1 < len(rest) {
				hi = rest[idx+1].SortOrder
			}
		}
		key, err := sortorder.GenerateKeyBetween(lo, hi)
		if err != nil {
			return "", fmt.Errorf("failed to place milestone: %w", err)
		}
		w.emit(w.event(r.ID, model.EventTypeMilestoneUpdate, model.MilestoneUpdatePayload{SortOrder: &key}))
		return r.ID, nil
	})
	return r, err
}

// milestoneMembers splits the issues of a milestone into open and terminal ones.
func milestoneMembers(issues map[string]*model.Issue, milestoneID string) (open, terminal []*model.Issue) {
	for _, iss := range issues {
		if iss.MilestoneID != milestoneID {
			continue
		}
		if model.IsTerminal(iss.Status) {
			terminal = append(terminal, iss)
		} else {
			open = append(open, iss)
		}
	}
	return open, terminal
}

// inScope reports whether an issue in status s counts toward a milestone.
// CANCELED and DUPLICATE issues leave the scope, so a milestone whose other
// issues are done reads as complete.
func inScope(s model.IssueStatus) bool {
	return s != model.StatusCanceled && s != model.StatusDuplicate
}

// parentIDs returns the IDs of issues that have at least one live child.
func parentIDs(issues map[string]*model.Issue) map[string]bool {
	parents := make(map[string]bool)
	for _, iss := range issues {
		if iss.ParentID != "" {
			if _, ok := issues[iss.ParentID]; ok {
				parents[iss.ParentID] = true
			}
		}
	}
	return parents
}

// CompleteMilestone marks a milestone COMPLETED. Its open issues move (mode
// move_to_next, the default) to the milestone to, or by default to its
// neighbour (see moveTarget), or leave the milestone (remove). Done issues
// stay, and their counts are stored in the complete event.
func (t *LocalTransport) CompleteMilestone(ref, mode, to string) (*model.Milestone, []string, error) {
	mode, err := moveMode(mode, to)
	if err != nil {
		return nil, nil, err
	}
	return t.milestoneOp("complete", func(w *milestoneWriter) (string, error) {
		r, err := w.saved(ref, "completed")
		if err != nil {
			return "", err
		}
		issues := ProjectIssues(w.base)
		open, terminal := milestoneMembers(issues, r.ID)

		dest := ""
		if mode == model.MilestoneMoveToNext && len(open) > 0 {
			target, err := w.moveTarget(r, to)
			if err != nil {
				return "", err
			}
			dest = target.ID
			w.messages = append(w.messages, fmt.Sprintf("Moved %d open %s to %s", len(open), pluralIssues(len(open)), target.Display()))
		} else if len(open) > 0 {
			w.messages = append(w.messages, fmt.Sprintf("Removed %d open %s from the milestone", len(open), pluralIssues(len(open))))
		}
		w.moveIssues(open, dest, r.ID)

		parents := parentIDs(issues)
		stats := model.MilestoneStats{Counts: map[string]int{}, Points: &model.MilestonePoints{}, LeafIDs: []string{}}
		for _, iss := range terminal {
			if parents[iss.ID] || !inScope(iss.Status) {
				continue
			}
			stats.Counts[string(iss.Status)]++
			stats.Points.Total += iss.Estimate
			if model.IsCompleted(iss.Status) {
				stats.Points.Done += iss.Estimate
			}
			stats.LeafIDs = append(stats.LeafIDs, iss.ID)
		}
		sort.Strings(stats.LeafIDs)
		w.emit(w.event(r.ID, model.EventTypeMilestoneComplete, model.MilestoneCompletePayload{
			OpenIssues: mode,
			Moved:      len(open),
			MovedTo:    dest,
			Stats:      stats,
		}))
		w.messages = append(w.messages, fmt.Sprintf("Completed %s", r.Display()))
		return r.ID, nil
	})
}

// DeleteMilestone deletes an OPEN milestone. All its issues, done ones included,
// move (move_to_next, the default) to to or its neighbour, as for
// CompleteMilestone, or leave the milestone (remove).
func (t *LocalTransport) DeleteMilestone(ref, mode, to string) ([]string, error) {
	mode, err := moveMode(mode, to)
	if err != nil {
		return nil, err
	}
	_, msgs, err := t.milestoneOp("delete", func(w *milestoneWriter) (string, error) {
		r, err := w.saved(ref, "deleted")
		if err != nil {
			return "", err
		}
		open, terminal := milestoneMembers(ProjectIssues(w.base), r.ID)
		members := append(open, terminal...)

		dest := ""
		if mode == model.MilestoneMoveToNext && len(members) > 0 {
			target, err := w.moveTarget(r, to)
			if err != nil {
				return "", err
			}
			dest = target.ID
			w.messages = append(w.messages, fmt.Sprintf("Moved %d %s to %s", len(members), pluralIssues(len(members)), target.Display()))
		} else if len(members) > 0 {
			w.messages = append(w.messages, fmt.Sprintf("Removed %d %s from the milestone", len(members), pluralIssues(len(members))))
		}
		w.moveIssues(members, dest, r.ID)
		w.emit(w.event(r.ID, model.EventTypeMilestoneDelete, model.MilestoneDeletePayload{Issues: mode, Moved: len(members), MovedTo: dest}))
		w.messages = append(w.messages, fmt.Sprintf("Deleted %s", r.Display()))
		return r.ID, nil
	})
	return msgs, err
}

func pluralIssues(n int) string {
	if n == 1 {
		return "issue"
	}
	return "issues"
}

// --- Reading ---

// MilestoneView is a milestone with its position on the board and its scope.
type MilestoneView struct {
	*model.Milestone
	Slot string // current, next, upcoming or past
	MilestoneScope
}

// MilestoneScope is the size of a milestone: its scope's counts by status, their
// total and done sums, and its story points. Points is nil for a past
// milestone whose snapshot has none.
type MilestoneScope struct {
	Counts map[string]int
	Total  int
	Done   int
	Points *model.MilestonePoints
}

// Board slots.
const (
	SlotCurrent  = "current"
	SlotNext     = "next"
	SlotUpcoming = "upcoming"
	SlotPast     = "past"
)

// BuildMilestoneViews lists Current, Next, the upcoming OPEN milestones in
// order, then the COMPLETED milestones, newest first.
func BuildMilestoneViews(events []model.Event) []MilestoneView {
	milestones := ProjectMilestones(events)
	issues := ProjectIssues(events)

	open := OpenMilestones(milestones)
	slots := fillSlots(open, NextMilestoneNumber(milestones))
	views := []MilestoneView{
		newMilestoneView(slots[0], SlotCurrent, issues),
		newMilestoneView(slots[1], SlotNext, issues),
	}
	for i := 2; i < len(open); i++ {
		views = append(views, newMilestoneView(open[i], SlotUpcoming, issues))
	}

	var past []*model.Milestone
	for _, r := range milestones {
		if !r.Deleted && r.State == model.MilestoneCompleted {
			past = append(past, r)
		}
	}
	sort.SliceStable(past, func(i, j int) bool { return past[i].CompletedAt.After(*past[j].CompletedAt) })
	for _, r := range past {
		views = append(views, newMilestoneView(r, SlotPast, issues))
	}
	return views
}

func newMilestoneView(r *model.Milestone, slot string, issues map[string]*model.Issue) MilestoneView {
	return MilestoneView{Milestone: r, Slot: slot, MilestoneScope: MilestoneCounts(issues, r)}
}

// MilestoneCounts counts the scope of a milestone (leaf issues, not CANCELED or
// DUPLICATE) by status and sums its story points. A COMPLETED milestone uses
// the snapshot stored when it was completed.
func MilestoneCounts(issues map[string]*model.Issue, r *model.Milestone) MilestoneScope {
	s := MilestoneScope{Counts: map[string]int{}}
	if r.Stats != nil {
		for status, n := range r.Stats.Counts {
			s.Counts[status] = n
		}
		if r.Stats.Points != nil {
			points := *r.Stats.Points
			s.Points = &points
		}
	} else {
		s.Points = &model.MilestonePoints{}
		if !r.Placeholder {
			parents := parentIDs(issues)
			for _, iss := range issues {
				if iss.MilestoneID == r.ID && !parents[iss.ID] && inScope(iss.Status) {
					s.Counts[string(iss.Status)]++
					s.Points.Total += iss.Estimate
					if model.IsCompleted(iss.Status) {
						s.Points.Done += iss.Estimate
					}
				}
			}
		}
	}
	for status, n := range s.Counts {
		s.Total += n
		if model.IsCompleted(model.IssueStatus(status)) {
			s.Done += n
		}
	}
	return s
}

// ListMilestones returns the milestone board views.
func (t *LocalTransport) ListMilestones() ([]MilestoneView, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	return BuildMilestoneViews(events), nil
}

// GetMilestone resolves a milestone ref against the current log.
func (t *LocalTransport) GetMilestone(ref string) (*model.Milestone, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	r, err := ResolveMilestone(ProjectMilestones(events), ref)
	if err == nil && r == nil {
		err = fmt.Errorf("a milestone is required")
	}
	return r, err
}

// MilestoneDay is one point of a milestone burndown.
type MilestoneDay struct {
	Date      string `json:"date"`
	Scope     int    `json:"scope"`
	Open      int    `json:"open"`
	Completed int    `json:"completed"`
}

// MilestoneProgress holds the scope and burndown of a milestone.
type MilestoneProgress struct {
	Milestone *model.Milestone
	MilestoneScope
	Days []MilestoneDay
}

// ComputeMilestoneProgress counts the scope of r and replays events into a
// daily burndown of it, from started_at to completed_at (or now). A
// milestone that was never Current has no burndown.
func ComputeMilestoneProgress(events []model.Event, r *model.Milestone, now time.Time) MilestoneProgress {
	p := MilestoneProgress{Milestone: r, MilestoneScope: MilestoneCounts(ProjectIssues(events), r), Days: []MilestoneDay{}}
	if r.StartedAt == nil || r.Placeholder {
		return p
	}

	ordered := append([]model.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })

	type state struct {
		milestone, parent string
		status            model.IssueStatus
		deleted           bool
	}
	states := make(map[string]*state)
	apply := func(evt model.Event) {
		switch evt.Type {
		case model.EventTypeCreate:
			var c model.CreatePayload
			decodePayload(evt.Payload, &c)
			st := model.IssueStatus(c.Status)
			if st == "" {
				st = model.StatusBacklog
			}
			states[evt.ID] = &state{milestone: c.MilestoneID, parent: c.ParentID, status: st}
		case model.EventTypeUpdate:
			s, ok := states[evt.ID]
			if !ok {
				return
			}
			var u model.UpdatePayload
			decodePayload(evt.Payload, &u)
			if u.Status != nil {
				s.status = model.IssueStatus(*u.Status)
			}
			if u.MilestoneID != nil {
				s.milestone = *u.MilestoneID
			}
			if u.ParentID != nil {
				s.parent = *u.ParentID
			}
		case model.EventTypeMerge:
			if s, ok := states[evt.ID]; ok {
				s.status = model.StatusDone
			}
		case model.EventTypeDelete:
			if s, ok := states[evt.ID]; ok {
				s.deleted = true
			}
		}
	}

	day := func(t time.Time) time.Time {
		t = t.UTC()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	end := day(now)
	if r.CompletedAt != nil {
		end = day(*r.CompletedAt)
	}

	idx := 0
	for d := day(*r.StartedAt); !d.After(end); d = d.AddDate(0, 0, 1) {
		dayEnd := d.AddDate(0, 0, 1)
		for idx < len(ordered) && ordered[idx].CreatedAt.Before(dayEnd) {
			apply(ordered[idx])
			idx++
		}
		parents := make(map[string]bool)
		for _, s := range states {
			if !s.deleted && s.parent != "" {
				parents[s.parent] = true
			}
		}
		point := MilestoneDay{Date: d.Format("2006-01-02")}
		for id, s := range states {
			if s.deleted || s.milestone != r.ID || parents[id] || !inScope(s.status) {
				continue
			}
			point.Scope++
			switch {
			case model.IsCompleted(s.status):
				point.Completed++
			case !model.IsTerminal(s.status):
				point.Open++
			}
		}
		p.Days = append(p.Days, point)
	}
	return p
}

// MilestoneProgress computes the progress of the milestone ref names. Archived
// events are included so a past milestone keeps its burndown.
func (t *LocalTransport) MilestoneProgress(ref string) (MilestoneProgress, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return MilestoneProgress{}, fmt.Errorf("failed to read events: %w", err)
	}
	archived, err := storage.ReadArchivedEvents()
	if err != nil {
		return MilestoneProgress{}, fmt.Errorf("failed to read archived events: %w", err)
	}
	r, err := ResolveMilestone(ProjectMilestones(events), ref)
	if err != nil {
		return MilestoneProgress{}, err
	}
	if r == nil {
		return MilestoneProgress{}, fmt.Errorf("a milestone is required")
	}
	return ComputeMilestoneProgress(append(archived, events...), r, time.Now()), nil
}
