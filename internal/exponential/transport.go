package exponential

import (
	"errors"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// ErrLocalOnly is returned when a local-only operation (git branch, merge,
// archive) is attempted on a remote transport.
var ErrLocalOnly = errors.New("operation requires local mode")

// Transport abstracts issue data operations so the Client can work in both
// local mode (reading .xpo/issues.db directly) and remote mode (proxying
// to an exponential server over HTTP).
type Transport interface {
	ListIssues(opts FilterOptions) ([]*model.Issue, error)
	GetIssue(id string) (*model.Issue, error)
	FindIssue(id string) (issue *model.Issue, children []*model.Issue, archived bool, err error)
	AddIssue(payload model.CreatePayload) (*model.Issue, error)
	UpdateIssue(id string, payload model.UpdatePayload, action string) ([]string, error)
	AddComment(issueID, text string) error
	DeleteIssue(id string, reason string, cascade bool) error
	CheckDuplicates(title string) ([]*model.Issue, error)
	GetUser() string
	GetInbox(since time.Time) ([]InboxItem, error)

	// Artifacts
	AddArtifact(issueID, artifactType, filename, content string) error
	ReadArtifact(issueID, filename string) (string, error)
	DeleteArtifact(issueID, filename string) error
	ListArtifacts(issueID string) ([]model.ArtifactSummary, error)
	WriteSpec(issueID, content string) error
	ReadSpec(issueID string) (string, error)
	DeleteSpec(issueID string) error
	WriteWalkthrough(issueID, content string) error
	ReadWalkthrough(issueID string) (string, error)
	DeleteWalkthrough(issueID string) error
}
