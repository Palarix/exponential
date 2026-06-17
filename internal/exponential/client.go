package exponential

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
)

// GitCommit stages and commits the issues.db file.
func GitCommit(msg string) {
	_ = exec.Command("git", "add", ".xpo/issues.db").Run()
	_ = exec.Command("git", "commit", "-m", msg).Run()
}

// Client manages the interaction with the xpo issue tracker.
// It delegates data operations to a Transport (local or remote) and
// keeps git-only operations (start, merge, review) as direct methods.
type Client struct {
	Transport    Transport
	Config       *config.Config
	Collapse     bool
	UserOverride string
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
	}
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

func (c *Client) AddComment(issueID, text string) error {
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
