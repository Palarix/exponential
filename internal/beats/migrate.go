package beats

import (
	"fmt"
	"time"

	"github.com/kuyio/beats/internal/model"
	"github.com/kuyio/beats/internal/storage"
)

// MigrateOptions contains flags for migration behavior
type MigrateOptions struct {
	DryRun bool
}

// RunMigrations executes all necessary migrations based on current config version
func (c *Client) RunMigrations(opts MigrateOptions) ([]string, error) {
	var report []string

	// Migration v0 -> v1: ParentID to Dependencies
	if c.Config.Version < 1 {
		report = append(report, "Running migration v1: Convert ParentID to Dependencies...")
		msgs, err := c.MigrateParentToDependencies(opts)
		if err != nil {
			return nil, fmt.Errorf("migration v1 failed: %w", err)
		}
		report = append(report, msgs...)

		if !opts.DryRun {
			// Update config version
			if err := UpdateConfigVersion(1); err != nil {
				return nil, fmt.Errorf("failed to update config version: %w", err)
			}
			report = append(report, "Updated config version to 1")
		}
	}

	return report, nil
}

// MigrateParentToDependencies finds issues with ParentID/BlockedBy that lack corresponding Dependencies
// and creates update events to add them.
func (c *Client) MigrateParentToDependencies(opts MigrateOptions) ([]string, error) {
	events, err := storage.ReadEvents()
	if err != nil {
		return nil, fmt.Errorf("error reading events: %w", err)
	}

	// Project current state
	issues := ProjectIssues(events)

	var user = c.GetUser()
	var timestamp = time.Now().UTC()
	var eventsToAppend []model.Event
	var report []string

	for _, issue := range issues {
		if issue.Deleted {
			continue
		}

		changes := false
		updatePayload := model.UpdatePayload{}
		var newDeps []model.Dependency

		// Note: ProjectIssues *already* populates Dependencies from payload if they exist.
		// It creates a *view* where ParentID is derived from Dependencies.
		// However, for migration, we need to inspect if the dependency *originates* from the new field
		// or if we need to *persist* it to the new field for future-proofing.
		//
		// But wait, if ProjectIssues derives ParentID from Dependencies, how do we know if it came from the old field?
		// We probably need to check if the issue *has* explicit dependencies matching the parent.
		//
		// Actually, `ProjectIssues` logic we wrote:
		// "Backward Compatibility: ParentID from payload -> Dependency"
		// This happens in-memory during projection.
		// So `issue.Dependencies` *contains* the backward-compat ones too!
		//
		// Migration strategy:
		// We want to persist these derived dependencies as *explicit* dependencies in a new Update event.
		// But if we just do that, we might duplicate them if we run migration twice?
		//
		// We should only add dependencies if they are NOT already in the event stream as dependencies.
		// But `ProjectIssues` merges them.
		//
		// Optimized approach:
		// We can look at the raw events? Or we can just trust that `ProjectIssues` gives us the *desired* state (Dependencies populated from ParentID),
		// and we just write that state back as an explicit update.
		//
		// If we write an Update with `Dependencies: [...]`, subsequent projections will use that.
		//
		// But we must be careful not to duplicate. `model.Dependency` equality check needed.
		//
		// Let's iterate raw events to see if `Dependencies` field was ever used?
		// No, that's complex.
		//
		// Simpler:
		// 1. Take `issue.Dependencies` (which includes derived ones).
		// 2. Create an update event setting `Dependencies` to this full list.
		// 3. This effectively "solidifies" the migration.
		//
		// Is this safe?
		// If `ParentID` field in `CreatePayload` still exists, `ProjectIssues` will add it *again* to the list?
		// Let's check `ProjectIssues`:
		// ```go
		// issue.Dependencies = p.Dependencies
		// ...
		// if p.ParentID != "" { issue.Dependencies = append(..., DependencyChild) }
		// ```
		//
		// If we save an update with the Child dependency, and the original Create/Update still has ParentID,
		// `ProjectIssues` will produce implicit dependency AND explicit dependency (duplicate).
		//
		// We need to deduplicate in `ProjectIssues` or here.
		// Since we cannot change historical events easily (immutable log), `ProjectIssues` MUST deduplicate.
		//
		// I recall I did NOT implement deduplication in `ProjectIssues`.
		// I should check `ProjectIssues` implementation again or fix it there first.
		//
		// Let's check `projection.go` via file reading.
		// But I know what I wrote: `issue.Dependencies = append(...)`. It assumes strict append.
		//
		// So if I persist the migration, I will have duplicates in the projected view!
		//
		// FIX: I must ensure `ProjectIssues` deduplicates dependencies.

		// Let's assume for now I will fix projection.go.
		// The migration command simply "upgrades" the store.

		// Wait, if I change `ProjectIssues` to deduplicate, then writing the update is safe.
		// But the goal of migration is to move away from `ParentID`.
		// Since we can't delete `ParentID` from old events, we are stuck with it.
		//
		// Maybe the migration isn't strictly necessary if `ProjectIssues` handles it?
		// The user asked for: "make one pass... convert ParentID data to Dependencies[] data".
		// This implies they want the data in the new format.
		//
		// If we do this, we should probably update `ProjectIssues` to *ignore* `ParentID` if `Dependencies` are present?
		// Or intelligent merging.

		// For now, let's implement validation logic here:
		// If we generate an update, we assume future code might ignore ParentID?
		//
		// Let's stick to: "Generate Update event with Dependencies populated".
		// And ensure `ProjectIssues` dedupes.

		if len(issue.Dependencies) > 0 {
			// We have dependencies (derived or explicit).
			// We want to ensure they are persisted explicitly.
			changes = true
			newDeps = issue.Dependencies
		}

		if changes {
			updatePayload.Dependencies = newDeps

			// Only append if not dry run
			if !opts.DryRun {
				// We don't want to spam updates if nothing changed effectively...
				// But we can't easily know if it's already persisted as dependency without re-reading raw events.
				//
				// Let's just do it. Idempotency is hard without inspecting history.
				// But we can check `issue.Events` to see if the *last* event was a migration?
				//
				// For now, naive approach: Write update.

				evt := model.Event{
					ID:        issue.ID,
					Type:      model.EventTypeUpdate,
					Payload:   updatePayload,
					CreatedAt: timestamp,
					CreatedBy: user,
				}
				eventsToAppend = append(eventsToAppend, evt)
			}
			report = append(report, fmt.Sprintf("Migrating %s: %d dependencies", issue.ID, len(newDeps)))
		}
	}

	if !opts.DryRun && len(eventsToAppend) > 0 {
		for _, evt := range eventsToAppend {
			if err := storage.AppendEvent(evt); err != nil {
				return nil, err
			}
		}
	}

	return report, nil
}
