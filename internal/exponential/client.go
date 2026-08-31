package exponential

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// GitCommit stages and commits the issues.db file.
func GitCommit(msg string) {
	hub := storage.HubRoot()
	_ = exec.Command("git", "-C", hub, "add", ".xpo/issues.db").Run()
	_ = exec.Command("git", "-C", hub, "add", ".xpo/artifacts/").Run()
	_ = exec.Command("git", "-C", hub, "commit", "-m", msg).Run()
}

// Client manages the interaction with the xpo issue tracker.
// It delegates data operations to a Transport (local or remote) and
// keeps git-only operations (start, merge, review) as direct methods.
type Client struct {
	Transport    Transport
	Config       *config.Config
	Collapse     bool
	UserOverride string
	OnBehalfOf   string
	Source       string
	local        *LocalTransport
}

// NewClient creates a new Client with the given configuration.
// By default it uses a LocalTransport. When cfg.Remote.URL is set,
// a RemoteTransport is used instead.
func NewClient(cfg *config.Config) *Client {
	c := &Client{Config: cfg}
	if cfg.Remote.URL != "" {
		c.Transport = &RemoteTransport{
			BaseURL:    cfg.Remote.URL,
			Token:      cfg.Remote.Token,
			HTTPClient: &http.Client{Timeout: 30 * time.Second},
		}
	} else {
		lt := &LocalTransport{Config: cfg}
		c.Transport = lt
		c.local = lt
	}
	return c
}

// syncLocal propagates Client-level fields to the underlying
// LocalTransport before each delegation call.
func (c *Client) syncLocal() {
	if c.local != nil {
		c.local.Collapse = c.Collapse
		c.local.UserOverride = c.UserOverride
		c.local.OnBehalfOf = c.OnBehalfOf
		c.local.Source = c.Source
	}
}

const (
	MaxTitleLen       = 500
	MaxDescriptionLen = 100 * 1024
	MaxAssigneeLen    = 200
	MaxLabelLen       = 100
)

func validateAssigneeFormat(assignee string) error {
	if assignee == "" {
		return nil
	}
	open := strings.IndexByte(assignee, '<')
	close := strings.IndexByte(assignee, '>')
	if open >= 0 || close >= 0 {
		if open < 0 || close < 0 || close <= open+1 {
			return fmt.Errorf("invalid assignee format: mismatched angle brackets (expected 'Name <email>')")
		}
		email := assignee[open+1 : close]
		if !strings.Contains(email, "@") {
			return fmt.Errorf("invalid assignee format: email inside angle brackets must contain '@'")
		}
	}
	return nil
}

// ValidateCreatePayload checks and resolves a CreatePayload before it is
// stored. It validates title, status, estimate, and resolves parent_id and
// dependency target IDs to their canonical forms.
func (c *Client) ValidateCreatePayload(p *model.CreatePayload) error {
	if p.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(p.Title) > MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", MaxTitleLen)
	}
	if len(p.Description) > MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum size of %d bytes", MaxDescriptionLen)
	}
	if len(p.Assignee) > MaxAssigneeLen {
		return fmt.Errorf("assignee exceeds maximum length of %d characters", MaxAssigneeLen)
	}
	if err := validateAssigneeFormat(p.Assignee); err != nil {
		return err
	}
	for _, l := range p.Labels {
		if len(l) > MaxLabelLen {
			return fmt.Errorf("label %q exceeds maximum length of %d characters", l, MaxLabelLen)
		}
	}
	if p.Status != "" {
		switch model.IssueStatus(p.Status) {
		case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
			model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		default:
			return fmt.Errorf("invalid status %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, DUPLICATE", p.Status)
		}
	}
	if p.ParentID != "" {
		parent, err := c.GetIssue(p.ParentID)
		if err != nil {
			return fmt.Errorf("parent: %w", err)
		}
		p.ParentID = parent.ID
	}
	for i, dep := range p.Dependencies {
		tgt, err := c.GetIssue(dep.TargetID)
		if err != nil {
			return fmt.Errorf("dependencies[%d]: %w", i, err)
		}
		p.Dependencies[i].TargetID = tgt.ID
	}
	if p.Priority < 0 || p.Priority > 4 {
		return fmt.Errorf("priority must be between 0 and 4 (0=None, 1=Urgent, 2=High, 3=Medium, 4=Low)")
	}
	if p.Estimate > 0 {
		if err := config.ValidateEstimate(c.Config.EstimationSystem, p.Estimate); err != nil {
			return err
		}
	}
	if p.Estimate < 0 {
		return fmt.Errorf("estimate must not be negative")
	}
	if p.CycleID != "" {
		if !c.Config.Cycles.Enabled {
			return fmt.Errorf("cycles are not enabled")
		}
		if _, err := c.Config.Cycles.CycleForID(p.CycleID); err != nil {
			return err
		}
	}
	return nil
}

// ValidateUpdatePayload checks and resolves an UpdatePayload before it is
// stored. It validates status and resolves parent_id and dependency target IDs
// to their canonical forms.
func (c *Client) ValidateUpdatePayload(p *model.UpdatePayload) error {
	if p.Title != nil && len(*p.Title) > MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", MaxTitleLen)
	}
	if p.Description != nil && len(*p.Description) > MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum size of %d bytes", MaxDescriptionLen)
	}
	if p.Assignee != nil && len(*p.Assignee) > MaxAssigneeLen {
		return fmt.Errorf("assignee exceeds maximum length of %d characters", MaxAssigneeLen)
	}
	if p.Assignee != nil {
		if err := validateAssigneeFormat(*p.Assignee); err != nil {
			return err
		}
	}
	for _, l := range p.Labels {
		if len(l) > MaxLabelLen {
			return fmt.Errorf("label %q exceeds maximum length of %d characters", l, MaxLabelLen)
		}
	}
	if p.Status != nil {
		switch model.IssueStatus(*p.Status) {
		case model.StatusBacklog, model.StatusPlanned, model.StatusDoing, model.StatusBlocked,
			model.StatusDone, model.StatusCanceled, model.StatusDuplicate:
		default:
			return fmt.Errorf("invalid status %q: must be one of BACKLOG, PLANNED, DOING, BLOCKED, DONE, CANCELED, DUPLICATE", *p.Status)
		}
	}
	if p.ParentID != nil && *p.ParentID != "" {
		parent, err := c.GetIssue(*p.ParentID)
		if err != nil {
			return fmt.Errorf("parent: %w", err)
		}
		*p.ParentID = parent.ID
	}
	for i, dep := range p.Dependencies {
		tgt, err := c.GetIssue(dep.TargetID)
		if err != nil {
			return fmt.Errorf("dependencies[%d]: %w", i, err)
		}
		p.Dependencies[i].TargetID = tgt.ID
	}
	if p.Priority != nil && (*p.Priority < 0 || *p.Priority > 4) {
		return fmt.Errorf("priority must be between 0 and 4 (0=None, 1=Urgent, 2=High, 3=Medium, 4=Low)")
	}
	if p.Estimate != nil && *p.Estimate > 0 {
		if err := config.ValidateEstimate(c.Config.EstimationSystem, *p.Estimate); err != nil {
			return err
		}
	}
	if p.Estimate != nil && *p.Estimate < 0 {
		return fmt.Errorf("estimate must not be negative")
	}
	if p.CycleID != nil && *p.CycleID != "" {
		if !c.Config.Cycles.Enabled {
			return fmt.Errorf("cycles are not enabled")
		}
		if _, err := c.Config.Cycles.CycleForID(*p.CycleID); err != nil {
			return err
		}
	}
	return nil
}

// --- Transport delegation methods ---

func (c *Client) GetIssue(id string) (*model.Issue, error) {
	c.syncLocal()
	return c.Transport.GetIssue(id)
}

func (c *Client) FindIssue(id string) (*model.Issue, []*model.Issue, bool, error) {
	c.syncLocal()
	return c.Transport.FindIssue(id)
}

func (c *Client) ListIssues(opts FilterOptions) ([]*model.Issue, error) {
	c.syncLocal()
	return c.Transport.ListIssues(opts)
}

func (c *Client) AddIssue(payload model.CreatePayload) (*model.Issue, error) {
	c.syncLocal()
	return c.Transport.AddIssue(payload)
}

func (c *Client) UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error) {
	c.syncLocal()
	return c.Transport.UpdateIssue(id, payload, action)
}

const MaxCommentLen = 100 * 1024

func (c *Client) AddComment(issueID, text string) error {
	if len(text) > MaxCommentLen {
		return fmt.Errorf("comment exceeds maximum size of %d bytes", MaxCommentLen)
	}
	c.syncLocal()
	return c.Transport.AddComment(issueID, text)
}

func (c *Client) DeleteIssue(id string, reason string, cascade ...bool) error {
	c.syncLocal()
	doCascade := len(cascade) > 0 && cascade[0]
	return c.Transport.DeleteIssue(id, reason, doCascade)
}

func (c *Client) CheckDuplicates(title string) ([]*model.Issue, error) {
	c.syncLocal()
	return c.Transport.CheckDuplicates(title)
}

func (c *Client) GetUser() string {
	c.syncLocal()
	return c.Transport.GetUser()
}

func (c *Client) GetInbox(since time.Time) ([]InboxItem, error) {
	c.syncLocal()
	return c.Transport.GetInbox(since)
}

func (c *Client) AddArtifact(issueID, artifactType, filename, content string) error {
	c.syncLocal()
	return c.Transport.AddArtifact(issueID, artifactType, filename, content)
}

func (c *Client) ReadArtifact(issueID, filename string) (string, error) {
	c.syncLocal()
	return c.Transport.ReadArtifact(issueID, filename)
}

func (c *Client) DeleteArtifact(issueID, filename string) error {
	c.syncLocal()
	return c.Transport.DeleteArtifact(issueID, filename)
}

func (c *Client) ListArtifacts(issueID string) ([]model.ArtifactSummary, error) {
	c.syncLocal()
	return c.Transport.ListArtifacts(issueID)
}

func (c *Client) WriteSpec(issueID, content string) error {
	c.syncLocal()
	return c.Transport.WriteSpec(issueID, content)
}

func (c *Client) ReadSpec(issueID string) (string, error) {
	c.syncLocal()
	return c.Transport.ReadSpec(issueID)
}

func (c *Client) DeleteSpec(issueID string) error {
	c.syncLocal()
	return c.Transport.DeleteSpec(issueID)
}

func (c *Client) WriteWalkthrough(issueID, content string) error {
	c.syncLocal()
	return c.Transport.WriteWalkthrough(issueID, content)
}

func (c *Client) ReadWalkthrough(issueID string) (string, error) {
	c.syncLocal()
	return c.Transport.ReadWalkthrough(issueID)
}

func (c *Client) DeleteWalkthrough(issueID string) error {
	c.syncLocal()
	return c.Transport.DeleteWalkthrough(issueID)
}

// resolveIssue is a convenience for git-only methods that need to resolve
// an issue ID from a projected issue map.
func (c *Client) resolveIssue(issues map[string]*model.Issue, id string) (*model.Issue, error) {
	if c.local != nil {
		return c.local.resolveIssue(issues, id)
	}
	issue, ok := issues[id]
	if !ok {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	return issue, nil
}
