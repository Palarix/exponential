package beats

import (
	"testing"
	"time"

	"github.com/kuyio/beats/internal/config"
	"github.com/kuyio/beats/internal/model"
)

func cycleConfig() config.CycleConfig {
	return config.CycleConfig{
		Enabled:    true,
		Duration:   "2w",
		StartDay:   "monday",
		AnchorDate: "2026-05-25",
	}
}

func TestRollover_DoneStaysInOriginalCycle(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10") // current cycle: 2026-06-08

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDone, CycleID: "2026-05-25"},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "2026-05-25" {
		t.Errorf("done issue should stay in original cycle, got %q", issues["a"].EffectiveCycleID)
	}
}

func TestRollover_NotDonePastCycleRollsForward(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10")

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDoing, CycleID: "2026-05-25"},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "2026-06-08" {
		t.Errorf("not-done past issue should roll to current cycle, got %q", issues["a"].EffectiveCycleID)
	}
}

func TestRollover_CurrentCycleUnchanged(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10")

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusPlanned, CycleID: "2026-06-08"},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "2026-06-08" {
		t.Errorf("current cycle issue should stay, got %q", issues["a"].EffectiveCycleID)
	}
}

func TestRollover_FutureCycleUnchanged(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10")

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusPlanned, CycleID: "2026-06-22"},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "2026-06-22" {
		t.Errorf("future cycle issue should stay, got %q", issues["a"].EffectiveCycleID)
	}
}

func TestRollover_NoCycleIDUntouched(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10")

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusDoing, CycleID: ""},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "" {
		t.Errorf("uncycled issue should have empty effective cycle, got %q", issues["a"].EffectiveCycleID)
	}
}

func TestRollover_MultipleCycleBoundaries(t *testing.T) {
	cc := cycleConfig()
	// 3 cycles after assignment: May 25 → Jun 8 → Jun 22 → Jul 6
	now, _ := time.Parse("2006-01-02", "2026-07-10") // current cycle: 2026-07-06

	issues := map[string]*model.Issue{
		"a": {ID: "a", Status: model.StatusBacklog, CycleID: "2026-05-25"},
	}

	applyCycleRollover(issues, cc, now)

	if issues["a"].EffectiveCycleID != "2026-07-06" {
		t.Errorf("should roll forward across multiple boundaries to current cycle, got %q", issues["a"].EffectiveCycleID)
	}
	if issues["a"].CycleID != "2026-05-25" {
		t.Errorf("original CycleID should be preserved, got %q", issues["a"].CycleID)
	}
}

func TestRollover_MixedIssues(t *testing.T) {
	cc := cycleConfig()
	now, _ := time.Parse("2006-01-02", "2026-06-10")

	issues := map[string]*model.Issue{
		"done-past":     {ID: "done-past", Status: model.StatusDone, CycleID: "2026-05-25"},
		"doing-past":    {ID: "doing-past", Status: model.StatusDoing, CycleID: "2026-05-25"},
		"planned-cur":   {ID: "planned-cur", Status: model.StatusPlanned, CycleID: "2026-06-08"},
		"planned-fut":   {ID: "planned-fut", Status: model.StatusPlanned, CycleID: "2026-06-22"},
		"no-cycle":      {ID: "no-cycle", Status: model.StatusDoing, CycleID: ""},
	}

	applyCycleRollover(issues, cc, now)

	checks := map[string]string{
		"done-past":   "2026-05-25",
		"doing-past":  "2026-06-08",
		"planned-cur": "2026-06-08",
		"planned-fut": "2026-06-22",
		"no-cycle":    "",
	}

	for id, want := range checks {
		if issues[id].EffectiveCycleID != want {
			t.Errorf("%s: EffectiveCycleID = %q, want %q", id, issues[id].EffectiveCycleID, want)
		}
	}
}
