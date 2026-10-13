package exponential

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// MergeRecord is an issue's MERGE event, decoded.
type MergeRecord struct {
	Payload    model.MergePayload
	At         time.Time
	By         string
	OnBehalfOf string
}

// LatestMerge returns the issue's most recent MERGE event, if any.
func LatestMerge(issue *model.Issue) (MergeRecord, bool) {
	for i := len(issue.Events) - 1; i >= 0; i-- {
		evt := issue.Events[i]
		if evt.Type != model.EventTypeMerge {
			continue
		}
		// After a JSON roundtrip the payload is a map.
		var p model.MergePayload
		b, _ := json.Marshal(evt.Payload)
		json.Unmarshal(b, &p)
		return MergeRecord{Payload: p, At: evt.CreatedAt, By: evt.CreatedBy, OnBehalfOf: evt.OnBehalfOf}, true
	}
	return MergeRecord{}, false
}

// FindMergeCommit returns the commit on the default branch that merged
// issueID. The MERGE event can't carry it: the bookkeeping is amended into
// the merge commit (or, for ff, the branch tip) after the event is written,
// which changes the SHA. So it is the first first-parent commit after the
// base that adds this issue's MERGE event to issues.db. Older events that
// did record merge_sha use it directly.
func FindMergeCommit(issueID string, p model.MergePayload) (string, error) {
	if p.MergeSHA != "" {
		if out, err := hubGit("rev-parse", "--verify", "--quiet", p.MergeSHA+"^{commit}").Output(); err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	if p.BaseSHA == "" {
		return "", fmt.Errorf("the merge of %s has no base commit", issueID)
	}
	marker := fmt.Sprintf(`"id":%q,"type":"MERGE"`, issueID)
	out, err := hubGit("log", "--first-parent", "--reverse", "--format=%H",
		"-S"+marker, p.BaseSHA+".."+DefaultBranch(), "--", ".xpo/issues.db").Output()
	if err != nil {
		return "", fmt.Errorf("failed to search for the merge commit of %s", issueID)
	}
	sha, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if sha == "" {
		return "", fmt.Errorf("merge commit for %s not found on %s", issueID, DefaultBranch())
	}
	return sha, nil
}

// MergedChanges is what an issue's merge brought into the default branch.
type MergedChanges struct {
	MergeSHA         string           `json:"merge_sha"`
	BaseSHA          string           `json:"base_sha"`
	Branch           string           `json:"branch"`
	Strategy         string           `json:"strategy"`
	MergedAt         time.Time        `json:"merged_at"`
	MergedBy         string           `json:"merged_by"`
	MergedOnBehalfOf string           `json:"merged_on_behalf_of,omitempty"`
	Commits          []DetailedCommit `json:"commits"`
	Diff             string           `json:"diff"`
}

// GetMergedChanges returns the commits and diff of the issue's latest merge.
func GetMergedChanges(issue *model.Issue) (*MergedChanges, error) {
	rec, ok := LatestMerge(issue)
	if !ok {
		return nil, fmt.Errorf("%s has not been merged", issue.ID)
	}
	sha, err := FindMergeCommit(issue.ID, rec.Payload)
	if err != nil {
		return nil, err
	}
	commits := listCommitsDetailed("--no-merges", rec.Payload.BaseSHA+".."+sha)
	if commits == nil {
		commits = []DetailedCommit{}
	}
	return &MergedChanges{
		MergeSHA:         sha,
		BaseSHA:          rec.Payload.BaseSHA,
		Branch:           rec.Payload.Branch,
		Strategy:         rec.Payload.Strategy,
		MergedAt:         rec.At,
		MergedBy:         rec.By,
		MergedOnBehalfOf: rec.OnBehalfOf,
		Commits:          commits,
		Diff:             GetDiffText(sha, rec.Payload.BaseSHA),
	}, nil
}
