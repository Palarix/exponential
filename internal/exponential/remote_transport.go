package exponential

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// RemoteTransport implements Transport by proxying to a xpo server
// over its existing REST API.
type RemoteTransport struct {
	BaseURL    string
	Token      string
	User       string
	HTTPClient *http.Client
}

// apiIssue mirrors the server's IssueResponse JSON shape.
type apiIssue struct {
	ID               string          `json:"id"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Status           string          `json:"status"`
	IsInferred       bool            `json:"is_inferred"`
	ParentID         string          `json:"parent_id"`
	Estimate         int             `json:"estimate"`
	Priority         int             `json:"priority"`
	SortOrder        string          `json:"sort_order"`
	Assignee         string          `json:"assignee"`
	CycleID          string          `json:"cycle_id"`
	EffectiveCycleID string          `json:"effective_cycle_id"`
	BranchStats      *apiBranchStats `json:"branch_stats"`
	Labels           []string        `json:"labels"`
	Dependencies     []apiDependency `json:"dependencies"`
	Comments         []apiComment    `json:"comments"`
	CreatedAt        time.Time       `json:"created_at"`
	CreatedBy        string          `json:"created_by"`
	UpdatedAt        time.Time       `json:"updated_at"`
	IsPending        bool            `json:"is_pending"`
}

type apiDependency struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Kind     string `json:"kind"`
}

type apiComment struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type apiUser struct {
	User string `json:"user"`
}

type apiBranchStats struct {
	Branch       string `json:"branch"`
	HeadSHA      string `json:"head_sha"`
	Commits      int    `json:"commits"`
	FilesChanged int    `json:"files_changed"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
}

type draftRequest struct {
	IssueID string      `json:"issue_id,omitempty"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type draftResponse struct {
	Status  string `json:"status"`
	IssueID string `json:"issue_id"`
}

func apiIssueToModel(a apiIssue) *model.Issue {
	issue := &model.Issue{
		ID:               a.ID,
		Title:            a.Title,
		Description:      a.Description,
		Status:           model.IssueStatus(a.Status),
		InferredStatus:   a.IsInferred,
		ParentID:         a.ParentID,
		Estimate:         a.Estimate,
		Priority:         a.Priority,
		SortOrder:        a.SortOrder,
		Assignee:         a.Assignee,
		CycleID:          a.CycleID,
		EffectiveCycleID: a.EffectiveCycleID,
		Labels:           a.Labels,
		CreatedAt:        a.CreatedAt,
		CreatedBy:        a.CreatedBy,
		UpdatedAt:        a.UpdatedAt,
	}

	if a.BranchStats != nil {
		issue.BranchStats = &model.BranchStats{
			Branch:       a.BranchStats.Branch,
			HeadSHA:      a.BranchStats.HeadSHA,
			Commits:      a.BranchStats.Commits,
			FilesChanged: a.BranchStats.FilesChanged,
			Insertions:   a.BranchStats.Insertions,
			Deletions:    a.BranchStats.Deletions,
		}
	}

	for _, d := range a.Dependencies {
		issue.Dependencies = append(issue.Dependencies, model.Dependency{
			SourceID: d.SourceID,
			TargetID: d.TargetID,
			Kind:     model.DependencyKind(d.Kind),
		})
	}

	for _, c := range a.Comments {
		issue.Comments = append(issue.Comments, model.Comment{
			ID:        c.ID,
			Text:      c.Text,
			CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt,
		})
	}

	return issue
}

func (r *RemoteTransport) do(req *http.Request) (*http.Response, error) {
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
	req.Header.Set("Content-Type", "application/json")
	return r.HTTPClient.Do(req)
}

func (r *RemoteTransport) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", r.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return r.do(req)
}

func (r *RemoteTransport) post(path string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", r.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return r.do(req)
}

func decodeJSON[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil && errBody.Error != "" {
			return result, fmt.Errorf("server returned %d: %s", resp.StatusCode, errBody.Error)
		}
		return result, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode server response: %w", err)
	}
	return result, nil
}

func (r *RemoteTransport) ListIssues(opts FilterOptions) ([]*model.Issue, error) {
	resp, err := r.get("/api/issues")
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	apiIssues, err := decodeJSON[[]apiIssue](resp)
	if err != nil {
		return nil, err
	}

	issues := make([]*model.Issue, 0, len(apiIssues))
	for _, a := range apiIssues {
		issues = append(issues, apiIssueToModel(a))
	}

	return FilterIssues(issues, opts, r.GetUser()), nil
}

func (r *RemoteTransport) GetIssue(id string) (*model.Issue, error) {
	resp, err := r.get("/api/issues/" + id)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}
	a, err := decodeJSON[apiIssue](resp)
	if err != nil {
		return nil, err
	}
	return apiIssueToModel(a), nil
}

func (r *RemoteTransport) FindIssue(id string) (*model.Issue, []*model.Issue, bool, error) {
	issue, err := r.GetIssue(id)
	if err != nil {
		return nil, nil, false, err
	}

	allResp, err := r.get("/api/issues")
	if err != nil {
		return issue, nil, false, nil
	}
	apiIssues, err := decodeJSON[[]apiIssue](allResp)
	if err != nil {
		return issue, nil, false, nil
	}

	var children []*model.Issue
	for _, a := range apiIssues {
		if a.ParentID == issue.ID {
			children = append(children, apiIssueToModel(a))
		}
	}

	return issue, children, false, nil
}

func (r *RemoteTransport) AddIssue(payload model.CreatePayload) (*model.Issue, error) {
	resp, err := r.post("/api/draft", draftRequest{
		Type:    string(model.EventTypeCreate),
		Payload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("add issue: %w", err)
	}
	dr, err := decodeJSON[draftResponse](resp)
	if err != nil {
		return nil, err
	}

	return r.GetIssue(dr.IssueID)
}

func (r *RemoteTransport) UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error) {
	resp, err := r.post("/api/draft", draftRequest{
		IssueID: id,
		Type:    string(model.EventTypeUpdate),
		Payload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("update issue: %w", err)
	}
	_, err = decodeJSON[draftResponse](resp)
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("Updated %s", id)}, nil
}

func (r *RemoteTransport) AddComment(issueID, text string) error {
	resp, err := r.post("/api/draft", draftRequest{
		IssueID: issueID,
		Type:    string(model.EventTypeComment),
		Payload: model.CommentPayload{Text: text},
	})
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	_, err = decodeJSON[draftResponse](resp)
	return err
}

func (r *RemoteTransport) DeleteIssue(id string, reason string, cascade bool) error {
	resp, err := r.post("/api/draft", draftRequest{
		IssueID: id,
		Type:    string(model.EventTypeDelete),
		Payload: model.DeletePayload{Reason: reason, Cascade: cascade},
	})
	if err != nil {
		return fmt.Errorf("delete issue: %w", err)
	}
	_, err = decodeJSON[draftResponse](resp)
	return err
}

func (r *RemoteTransport) CheckDuplicates(title string) ([]*model.Issue, error) {
	resp, err := r.get("/api/issues")
	if err != nil {
		return nil, fmt.Errorf("check duplicates: %w", err)
	}
	apiIssues, err := decodeJSON[[]apiIssue](resp)
	if err != nil {
		return nil, err
	}

	issues := make([]*model.Issue, 0, len(apiIssues))
	for _, a := range apiIssues {
		issues = append(issues, apiIssueToModel(a))
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].CreatedAt.After(issues[j].CreatedAt)
	})

	titleTokens := tokenize(title)
	var duplicates []*model.Issue

	for _, issue := range issues {
		issueTokens := tokenize(issue.Title)
		intersection := 0
		for t := range titleTokens {
			if issueTokens[t] {
				intersection++
			}
		}
		minLen := len(titleTokens)
		if len(issueTokens) < minLen {
			minLen = len(issueTokens)
		}
		if minLen > 0 {
			ratio := float64(intersection) / float64(minLen)
			if ratio >= 0.75 {
				duplicates = append(duplicates, issue)
			}
		}
	}

	return duplicates, nil
}

// GetInbox fetches the authenticated user's inbox from the server, which
// filters events server-side by identity and the optional since cursor.
func (r *RemoteTransport) GetInbox(since time.Time) ([]InboxItem, error) {
	path := "/api/inbox"
	if !since.IsZero() {
		path += "?since=" + url.QueryEscape(since.UTC().Format(time.RFC3339))
	}
	resp, err := r.get(path)
	if err != nil {
		return nil, fmt.Errorf("get inbox: %w", err)
	}
	return decodeJSON[[]InboxItem](resp)
}

func (r *RemoteTransport) GetUser() string {
	if r.User != "" {
		return r.User
	}
	resp, err := r.get("/api/user")
	if err != nil {
		return "Unknown <unknown@example.com>"
	}
	u, err := decodeJSON[apiUser](resp)
	if err != nil {
		return "Unknown <unknown@example.com>"
	}
	r.User = u.User
	return r.User
}
