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

// Releases are event-sourced like issues: RELEASE_* events in issues.db,
// projected by ProjectReleases. Current and Next are the first two OPEN
// releases in the manual order; missing slots are placeholders that are
// stored only on their first write. See the specs of xpo-4c92d4 and
// xpo-f89e55.

// ReleaseInput holds the editable fields of a release. A nil field is left
// unchanged; an empty string clears it.
type ReleaseInput struct {
	Label      *string
	Goal       *string
	TargetDate *string
}

func (in ReleaseInput) empty() bool {
	return in.Label == nil && in.Goal == nil && in.TargetDate == nil
}

// --- Projection ---

// ProjectReleases returns every release in the log, deleted ones included,
// ordered by number. Deleted releases are kept so their numbers are never
// given out again.
func ProjectReleases(events []model.Event) []*model.Release {
	byID := make(map[string]*model.Release)
	var all []*model.Release
	for _, evt := range events {
		if !model.IsReleaseEvent(evt.Type) {
			continue
		}
		r := byID[evt.ID]
		switch evt.Type {
		case model.EventTypeReleaseCreate:
			if r != nil {
				continue
			}
			var p model.ReleaseCreatePayload
			decodePayload(evt.Payload, &p)
			r = &model.Release{
				ID:         evt.ID,
				Number:     p.Number,
				Label:      p.Label,
				Goal:       p.Goal,
				TargetDate: p.TargetDate,
				SortOrder:  p.SortOrder,
				State:      model.ReleaseOpen,
				CreatedAt:  evt.CreatedAt,
				CreatedBy:  principalOf(evt),
			}
			byID[evt.ID] = r
			all = append(all, r)
		case model.EventTypeReleaseUpdate:
			if r == nil {
				continue
			}
			var p model.ReleaseUpdatePayload
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
			// started_at is set once: the first time the release became Current.
			if p.StartedAt != nil && r.StartedAt == nil {
				at := *p.StartedAt
				r.StartedAt = &at
			}
		case model.EventTypeReleaseComplete:
			if r == nil || r.State == model.ReleaseReleased {
				continue
			}
			var p model.ReleaseCompletePayload
			decodePayload(evt.Payload, &p)
			at := evt.CreatedAt
			r.State = model.ReleaseReleased
			r.ReleasedAt = &at
			stats := p.Stats
			r.Stats = &stats
		case model.EventTypeReleaseDelete:
			if r != nil {
				r.Deleted = true
			}
		}
	}
	renumberReleases(all)
	sort.Slice(all, func(i, j int) bool { return all[i].Number < all[j].Number })
	return all
}

// renumberReleases resolves duplicate numbers, which a git merge of two
// branches that each created a release can produce. The earlier release
// keeps the number; the later one gets the next free number.
func renumberReleases(all []*model.Release) {
	byAge := append([]*model.Release(nil), all...)
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

// OpenReleases returns the saved OPEN releases in the manual order.
func OpenReleases(releases []*model.Release) []*model.Release {
	var open []*model.Release
	for _, r := range releases {
		if !r.Deleted && !r.Placeholder && r.State == model.ReleaseOpen {
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

// NextReleaseNumber is the number the next saved release gets.
func NextReleaseNumber(releases []*model.Release) int {
	n := 0
	for _, r := range releases {
		n = max(n, r.Number)
	}
	return n + 1
}

// CurrentNext returns the Current and Next slots. A slot with no saved OPEN
// release holds a placeholder with the next free number; a Next placeholder
// is locked while Current is a placeholder.
func CurrentNext(releases []*model.Release) (current, next *model.Release) {
	slots := fillSlots(OpenReleases(releases), NextReleaseNumber(releases))
	return slots[0], slots[1]
}

func fillSlots(open []*model.Release, nextNumber int) [2]*model.Release {
	var slots [2]*model.Release
	for i := range slots {
		if i < len(open) {
			slots[i] = open[i]
			continue
		}
		slots[i] = &model.Release{Number: nextNumber, State: model.ReleaseOpen, Placeholder: true}
		nextNumber++
	}
	slots[1].Locked = slots[1].Placeholder && slots[0].Placeholder
	return slots
}

var releaseNumberRef = regexp.MustCompile(`^#(\d+)$`)

// ResolveRelease finds the release a ref names: "#7" (number), a rel- ID,
// "@current", "@next", or else a label (case-insensitive; an OPEN release
// wins over RELEASED ones). "none" and "" return nil. A ref that names
// an unsaved Current or Next returns its placeholder.
func ResolveRelease(releases []*model.Release, ref string) (*model.Release, error) {
	ref = strings.TrimSpace(ref)
	lower := strings.ToLower(ref)
	switch lower {
	case "", "none":
		return nil, nil
	case "@current":
		cur, _ := CurrentNext(releases)
		return cur, nil
	case "@next":
		_, next := CurrentNext(releases)
		return next, nil
	}

	live := make([]*model.Release, 0, len(releases))
	for _, r := range releases {
		if !r.Deleted {
			live = append(live, r)
		}
	}

	if strings.HasPrefix(ref, model.ReleaseIDPrefix) {
		for _, r := range live {
			if r.ID == ref {
				return r, nil
			}
		}
		return nil, fmt.Errorf("release %s not found", ref)
	}

	if m := releaseNumberRef.FindStringSubmatch(ref); m != nil {
		n, _ := strconv.Atoi(m[1])
		for _, r := range live {
			if r.Number == n {
				return r, nil
			}
		}
		cur, next := CurrentNext(releases)
		for _, slot := range []*model.Release{cur, next} {
			if slot.Placeholder && slot.Number == n {
				return slot, nil
			}
		}
		return nil, fmt.Errorf("release #%d not found", n)
	}

	var released *model.Release
	for _, r := range live {
		if strings.ToLower(r.Label) != lower {
			continue
		}
		if r.State == model.ReleaseOpen {
			return r, nil
		}
		if released == nil || (r.ReleasedAt != nil && released.ReleasedAt != nil && r.ReleasedAt.After(*released.ReleasedAt)) {
			released = r
		}
	}
	if released != nil {
		return released, nil
	}
	return nil, fmt.Errorf("release %q not found", ref)
}

// --- Validation ---

func validateReleaseLabel(releases []*model.Release, label, selfID string) error {
	if label == "" {
		return nil
	}
	if len(label) > MaxLabelLen {
		return fmt.Errorf("release label exceeds maximum length of %d characters", MaxLabelLen)
	}
	lower := strings.ToLower(label)
	if strings.HasPrefix(label, "#") || strings.HasPrefix(label, "@") || lower == "none" || strings.HasPrefix(lower, model.ReleaseIDPrefix) {
		return fmt.Errorf("invalid release label %q: labels may not start with # or @, start with rel-, or be none", label)
	}
	for _, r := range OpenReleases(releases) {
		if r.ID != selfID && strings.ToLower(r.Label) == lower {
			return fmt.Errorf("an open release is already labelled %q (%s)", r.Label, r.Ref())
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

func normalizeReleaseMode(mode string) (string, error) {
	switch mode {
	case "", model.ReleaseMoveToNext:
		return model.ReleaseMoveToNext, nil
	case model.ReleaseRemove:
		return model.ReleaseRemove, nil
	}
	return "", fmt.Errorf("invalid issues option %q: must be %s or %s", mode, model.ReleaseMoveToNext, model.ReleaseRemove)
}

// --- Writing ---

// releaseWriter builds one batch of events against a snapshot of the log.
// Release events re-project the releases, so later steps in the batch see
// releases saved by earlier ones.
type releaseWriter struct {
	t        *LocalTransport
	base     []model.Event
	batch    []model.Event
	releases []*model.Release
	user     string
	now      time.Time
	messages []string
}

func (t *LocalTransport) newReleaseWriter(current []model.Event) *releaseWriter {
	return &releaseWriter{
		t:        t,
		base:     current,
		releases: ProjectReleases(current),
		user:     t.GetUser(),
		now:      time.Now().UTC(),
	}
}

func (w *releaseWriter) emit(events ...model.Event) {
	reproject := false
	for _, evt := range events {
		w.batch = append(w.batch, evt)
		reproject = reproject || model.IsReleaseEvent(evt.Type)
	}
	if reproject {
		all := make([]model.Event, 0, len(w.base)+len(w.batch))
		all = append(all, w.base...)
		all = append(all, w.batch...)
		w.releases = ProjectReleases(all)
	}
}

func (w *releaseWriter) event(id string, typ model.EventType, payload interface{}) model.Event {
	return model.Event{ID: id, Type: typ, Payload: payload, CreatedAt: w.now, CreatedBy: w.user}
}

func (w *releaseWriter) find(id string) *model.Release {
	for _, r := range w.releases {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// create stores a new OPEN release after all saved OPEN releases.
func (w *releaseWriter) create(number int, in ReleaseInput) (*model.Release, error) {
	var p model.ReleaseCreatePayload
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
	if err := validateReleaseLabel(w.releases, p.Label, ""); err != nil {
		return nil, err
	}
	if err := validateTargetDate(p.TargetDate); err != nil {
		return nil, err
	}

	last := ""
	if open := OpenReleases(w.releases); len(open) > 0 {
		last = open[len(open)-1].SortOrder
	}
	key, err := sortorder.GenerateKeyBetween(last, "")
	if err != nil {
		return nil, fmt.Errorf("failed to place release: %w", err)
	}
	p.SortOrder = key

	suffix, err := gonanoid.Generate("0123456789abcdef", 6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate release ID: %w", err)
	}
	id := model.ReleaseIDPrefix + suffix
	w.emit(w.event(id, model.EventTypeReleaseCreate, p))
	return w.find(id), nil
}

// save stores a placeholder; a saved release is returned as is. A locked
// Next is refused: saved before Current, it would become Current itself.
func (w *releaseWriter) save(r *model.Release, in ReleaseInput) (*model.Release, error) {
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
	w.messages = append(w.messages, fmt.Sprintf("Saved release %s", saved.Display()))
	return saved, nil
}

// assign resolves the release ref for an issue whose release is currentID
// and returns the release ID to store, saving a placeholder if needed. Only
// OPEN releases accept issues; re-sending the issue's own RELEASED release
// is allowed so it can be pruned as a no-op.
func (w *releaseWriter) assign(ref, currentID string) (string, error) {
	r, err := ResolveRelease(w.releases, ref)
	if err != nil || r == nil {
		return "", err
	}
	if r.Placeholder {
		saved, err := w.save(r, ReleaseInput{})
		if err != nil {
			return "", err
		}
		return saved.ID, nil
	}
	if r.State == model.ReleaseReleased && r.ID != currentID {
		return "", fmt.Errorf("release %s is released: only open releases accept issues", r.Display())
	}
	return r.ID, nil
}

// saved resolves ref to a saved OPEN release, the only kind that can be
// reordered, completed or deleted.
func (w *releaseWriter) saved(ref, verb string) (*model.Release, error) {
	r, err := ResolveRelease(w.releases, ref)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("a release is required")
	}
	if r.Placeholder {
		return nil, fmt.Errorf("%s is a placeholder: a placeholder cannot be %s", r.Ref(), verb)
	}
	if r.State == model.ReleaseReleased {
		return nil, fmt.Errorf("release %s is released: it cannot be %s", r.Display(), verb)
	}
	return r, nil
}

// moveTarget is where "move" sends the issues of r when r is completed or
// deleted. With to, it is that OPEN release; a placeholder is saved and a
// locked Next refused. Without, it is the open release right after r, else
// the one right before it, else a new release: the Current placeholder once
// r is gone, saved.
func (w *releaseWriter) moveTarget(r *model.Release, to string) (*model.Release, error) {
	if to != "" {
		target, err := ResolveRelease(w.releases, to)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, fmt.Errorf("to must name a release: use issues=%s to take the issues out of the release", model.ReleaseRemove)
		}
		if target.ID == r.ID && !target.Placeholder {
			return nil, fmt.Errorf("cannot move the issues of %s to itself", r.Display())
		}
		if target.State != model.ReleaseOpen {
			return nil, fmt.Errorf("release %s is released: only open releases accept issues", target.Display())
		}
		return w.save(target, ReleaseInput{})
	}

	open := OpenReleases(w.releases)
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
	return w.save(fillSlots(nil, NextReleaseNumber(w.releases))[0], ReleaseInput{})
}

// moveMode checks a complete or delete's mode and target: to only goes with
// a move.
func moveMode(mode, to string) (string, error) {
	mode, err := normalizeReleaseMode(mode)
	if err != nil {
		return "", err
	}
	if mode == model.ReleaseRemove && strings.TrimSpace(to) != "" {
		return "", fmt.Errorf("to cannot be combined with issues=%s", model.ReleaseRemove)
	}
	return mode, nil
}

// moveIssues emits one UPDATE per issue setting its release to dest. The
// events are caused by the bulk operation of cause (a release or an epic),
// so feeds show that operation once instead of every move.
func (w *releaseWriter) moveIssues(issues []*model.Issue, dest, cause string) {
	sort.Slice(issues, func(i, j int) bool { return issues[i].ID < issues[j].ID })
	for _, iss := range issues {
		d := dest
		evt := w.event(iss.ID, model.EventTypeUpdate, model.UpdatePayload{ReleaseID: &d})
		evt.Cause = cause
		w.emit(evt)
	}
}

// finish sets started_at on the Current release when this batch made a
// saved release Current for the first time, then orders the timestamps.
func (w *releaseWriter) finish() []model.Event {
	touched := false
	for _, evt := range w.batch {
		touched = touched || model.IsReleaseEvent(evt.Type)
	}
	if touched {
		if open := OpenReleases(w.releases); len(open) > 0 && open[0].StartedAt == nil {
			at := w.now
			w.emit(w.event(open[0].ID, model.EventTypeReleaseUpdate, model.ReleaseUpdatePayload{StartedAt: &at}))
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

func (t *LocalTransport) autoCommitRelease(msg string) {
	if t.Config.AutoCommit {
		GitCommit("xpo: " + msg)
	}
}

// releaseOp runs one release write and returns the release it names.
func (t *LocalTransport) releaseOp(commitMsg string, op func(w *releaseWriter) (string, error)) (*model.Release, []string, error) {
	var out *model.Release
	var messages []string
	err := t.writeBatch(func(current []model.Event) ([]model.Event, error) {
		w := t.newReleaseWriter(current)
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
		t.autoCommitRelease(fmt.Sprintf("%s release %s", commitMsg, out.Display()))
	}
	return out, messages, nil
}

// CreateRelease stores a new OPEN release after all saved OPEN releases.
func (t *LocalTransport) CreateRelease(in ReleaseInput) (*model.Release, error) {
	r, _, err := t.releaseOp("create", func(w *releaseWriter) (string, error) {
		r, err := w.create(NextReleaseNumber(w.releases), in)
		if err != nil {
			return "", err
		}
		return r.ID, nil
	})
	return r, err
}

// UpdateRelease changes a release's label, goal or target date. On a
// placeholder, the write saves it.
func (t *LocalTransport) UpdateRelease(ref string, in ReleaseInput) (*model.Release, error) {
	if in.empty() {
		return nil, fmt.Errorf("no fields set: provide a label, goal or target date")
	}
	r, _, err := t.releaseOp("update", func(w *releaseWriter) (string, error) {
		r, err := ResolveRelease(w.releases, ref)
		if err != nil {
			return "", err
		}
		if r == nil {
			return "", fmt.Errorf("a release is required")
		}
		if r.Placeholder {
			saved, err := w.save(r, in)
			if err != nil {
				return "", err
			}
			return saved.ID, nil
		}

		var p model.ReleaseUpdatePayload
		if in.Label != nil {
			label := strings.TrimSpace(*in.Label)
			if r.State == model.ReleaseOpen {
				if err := validateReleaseLabel(w.releases, label, r.ID); err != nil {
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
			w.emit(w.event(r.ID, model.EventTypeReleaseUpdate, p))
		}
		return r.ID, nil
	})
	return r, err
}

// ReorderRelease moves a saved OPEN release directly above (before) or
// below (after) another one.
func (t *LocalTransport) ReorderRelease(ref, before, after string) (*model.Release, error) {
	if (before == "") == (after == "") {
		return nil, fmt.Errorf("set exactly one of before or after")
	}
	r, _, err := t.releaseOp("reorder", func(w *releaseWriter) (string, error) {
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
			return "", fmt.Errorf("cannot move a release relative to itself")
		}

		var rest []*model.Release
		for _, o := range OpenReleases(w.releases) {
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
			return "", fmt.Errorf("failed to place release: %w", err)
		}
		w.emit(w.event(r.ID, model.EventTypeReleaseUpdate, model.ReleaseUpdatePayload{SortOrder: &key}))
		return r.ID, nil
	})
	return r, err
}

// releaseMembers splits the issues of a release into open and terminal ones.
func releaseMembers(issues map[string]*model.Issue, releaseID string) (open, terminal []*model.Issue) {
	for _, iss := range issues {
		if iss.ReleaseID != releaseID {
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

// inScope reports whether an issue in status s counts toward a release.
// CANCELED and DUPLICATE issues leave the scope, so a release whose other
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

// CompleteRelease marks a release RELEASED. Its open issues move (mode
// move_to_next, the default) to the release to, or by default to its
// neighbour (see moveTarget), or leave the release (remove). Done issues
// stay, and their counts are stored in the complete event.
func (t *LocalTransport) CompleteRelease(ref, mode, to string) (*model.Release, []string, error) {
	mode, err := moveMode(mode, to)
	if err != nil {
		return nil, nil, err
	}
	return t.releaseOp("complete", func(w *releaseWriter) (string, error) {
		r, err := w.saved(ref, "completed")
		if err != nil {
			return "", err
		}
		issues := ProjectIssues(w.base)
		open, terminal := releaseMembers(issues, r.ID)

		dest := ""
		if mode == model.ReleaseMoveToNext && len(open) > 0 {
			target, err := w.moveTarget(r, to)
			if err != nil {
				return "", err
			}
			dest = target.ID
			w.messages = append(w.messages, fmt.Sprintf("Moved %d open %s to %s", len(open), pluralIssues(len(open)), target.Display()))
		} else if len(open) > 0 {
			w.messages = append(w.messages, fmt.Sprintf("Removed %d open %s from the release", len(open), pluralIssues(len(open))))
		}
		w.moveIssues(open, dest, r.ID)

		parents := parentIDs(issues)
		stats := model.ReleaseStats{Counts: map[string]int{}, LeafIDs: []string{}}
		for _, iss := range terminal {
			if parents[iss.ID] || !inScope(iss.Status) {
				continue
			}
			stats.Counts[string(iss.Status)]++
			stats.LeafIDs = append(stats.LeafIDs, iss.ID)
		}
		sort.Strings(stats.LeafIDs)
		w.emit(w.event(r.ID, model.EventTypeReleaseComplete, model.ReleaseCompletePayload{
			OpenIssues: mode,
			Moved:      len(open),
			MovedTo:    dest,
			Stats:      stats,
		}))
		w.messages = append(w.messages, fmt.Sprintf("Released %s", r.Display()))
		return r.ID, nil
	})
}

// DeleteRelease deletes an OPEN release. All its issues, done ones included,
// move (move_to_next, the default) to to or its neighbour, as for
// CompleteRelease, or leave the release (remove).
func (t *LocalTransport) DeleteRelease(ref, mode, to string) ([]string, error) {
	mode, err := moveMode(mode, to)
	if err != nil {
		return nil, err
	}
	_, msgs, err := t.releaseOp("delete", func(w *releaseWriter) (string, error) {
		r, err := w.saved(ref, "deleted")
		if err != nil {
			return "", err
		}
		open, terminal := releaseMembers(ProjectIssues(w.base), r.ID)
		members := append(open, terminal...)

		dest := ""
		if mode == model.ReleaseMoveToNext && len(members) > 0 {
			target, err := w.moveTarget(r, to)
			if err != nil {
				return "", err
			}
			dest = target.ID
			w.messages = append(w.messages, fmt.Sprintf("Moved %d %s to %s", len(members), pluralIssues(len(members)), target.Display()))
		} else if len(members) > 0 {
			w.messages = append(w.messages, fmt.Sprintf("Removed %d %s from the release", len(members), pluralIssues(len(members))))
		}
		w.moveIssues(members, dest, r.ID)
		w.emit(w.event(r.ID, model.EventTypeReleaseDelete, model.ReleaseDeletePayload{Issues: mode, Moved: len(members), MovedTo: dest}))
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

// ReleaseView is a release with its position on the board and its counts.
type ReleaseView struct {
	*model.Release
	Slot   string // current, next, upcoming or past
	Counts map[string]int
	Total  int
	Done   int
}

// Board slots.
const (
	SlotCurrent  = "current"
	SlotNext     = "next"
	SlotUpcoming = "upcoming"
	SlotPast     = "past"
)

// BuildReleaseViews lists Current, Next, the upcoming OPEN releases in
// order, then the RELEASED releases, newest first.
func BuildReleaseViews(events []model.Event) []ReleaseView {
	releases := ProjectReleases(events)
	issues := ProjectIssues(events)

	open := OpenReleases(releases)
	slots := fillSlots(open, NextReleaseNumber(releases))
	views := []ReleaseView{
		newReleaseView(slots[0], SlotCurrent, issues),
		newReleaseView(slots[1], SlotNext, issues),
	}
	for i := 2; i < len(open); i++ {
		views = append(views, newReleaseView(open[i], SlotUpcoming, issues))
	}

	var past []*model.Release
	for _, r := range releases {
		if !r.Deleted && r.State == model.ReleaseReleased {
			past = append(past, r)
		}
	}
	sort.SliceStable(past, func(i, j int) bool { return past[i].ReleasedAt.After(*past[j].ReleasedAt) })
	for _, r := range past {
		views = append(views, newReleaseView(r, SlotPast, issues))
	}
	return views
}

func newReleaseView(r *model.Release, slot string, issues map[string]*model.Issue) ReleaseView {
	counts, total, done := ReleaseCounts(issues, r)
	return ReleaseView{Release: r, Slot: slot, Counts: counts, Total: total, Done: done}
}

// ReleaseCounts counts the scope of a release (leaf issues, not CANCELED or
// DUPLICATE) by status. A RELEASED release uses the snapshot stored when it
// was completed.
func ReleaseCounts(issues map[string]*model.Issue, r *model.Release) (counts map[string]int, total, done int) {
	counts = map[string]int{}
	if r.Stats != nil {
		for status, n := range r.Stats.Counts {
			counts[status] = n
		}
	} else if !r.Placeholder {
		parents := parentIDs(issues)
		for _, iss := range issues {
			if iss.ReleaseID == r.ID && !parents[iss.ID] && inScope(iss.Status) {
				counts[string(iss.Status)]++
			}
		}
	}
	for status, n := range counts {
		total += n
		if model.IsCompleted(model.IssueStatus(status)) {
			done += n
		}
	}
	return counts, total, done
}

// ListReleases returns the release board views.
func (t *LocalTransport) ListReleases() ([]ReleaseView, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	return BuildReleaseViews(events), nil
}

// GetRelease resolves a release ref against the current log.
func (t *LocalTransport) GetRelease(ref string) (*model.Release, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	r, err := ResolveRelease(ProjectReleases(events), ref)
	if err == nil && r == nil {
		err = fmt.Errorf("a release is required")
	}
	return r, err
}

// ReleaseDay is one point of a release burndown.
type ReleaseDay struct {
	Date      string `json:"date"`
	Scope     int    `json:"scope"`
	Open      int    `json:"open"`
	Completed int    `json:"completed"`
}

// ReleaseProgress holds the counts and burndown of a release.
type ReleaseProgress struct {
	Release *model.Release
	Counts  map[string]int
	Total   int
	Done    int
	Days    []ReleaseDay
}

// ComputeReleaseProgress counts the scope of r and replays events into a
// daily burndown of it, from started_at to released_at (or now). A
// release that was never Current has no burndown.
func ComputeReleaseProgress(events []model.Event, r *model.Release, now time.Time) ReleaseProgress {
	p := ReleaseProgress{Release: r, Days: []ReleaseDay{}}
	p.Counts, p.Total, p.Done = ReleaseCounts(ProjectIssues(events), r)
	if r.StartedAt == nil || r.Placeholder {
		return p
	}

	ordered := append([]model.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })

	type state struct {
		release, parent string
		status          model.IssueStatus
		deleted         bool
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
			states[evt.ID] = &state{release: c.ReleaseID, parent: c.ParentID, status: st}
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
			if u.ReleaseID != nil {
				s.release = *u.ReleaseID
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
	if r.ReleasedAt != nil {
		end = day(*r.ReleasedAt)
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
		point := ReleaseDay{Date: d.Format("2006-01-02")}
		for id, s := range states {
			if s.deleted || s.release != r.ID || parents[id] || !inScope(s.status) {
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

// ReleaseProgress computes the progress of the release ref names. Archived
// events are included so a past release keeps its burndown.
func (t *LocalTransport) ReleaseProgress(ref string) (ReleaseProgress, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return ReleaseProgress{}, fmt.Errorf("failed to read events: %w", err)
	}
	archived, err := storage.ReadArchivedEvents()
	if err != nil {
		return ReleaseProgress{}, fmt.Errorf("failed to read archived events: %w", err)
	}
	r, err := ResolveRelease(ProjectReleases(events), ref)
	if err != nil {
		return ReleaseProgress{}, err
	}
	if r == nil {
		return ReleaseProgress{}, fmt.Errorf("a release is required")
	}
	return ComputeReleaseProgress(append(archived, events...), r, time.Now()), nil
}
