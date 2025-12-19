## Concern
For the "board" case, I don't think every update should be immediately commited and pushed. In a regular standup a number of updates happen (tasks get assigned, new ones created, old and stale ones removed, statuses get updated "oh I forgot to mark that done"). That would create a ton of noisy commits. Maybe the board UI could have a "dirty" flag, and let's you do all the updates/edits locally but then reminds you that you need to "save and sync" your changes to the repo. This could then be done in one single commit and push with a clean commit message like "daily standup 2025-12-19", then everyone on the team knows they can start their workday from this commit which they all synced on

## Reply
This is a **crucial refinement**. You are essentially describing a "Transaction" model.

Treating the standup session as one atomic "Transaction" (Pending State $\rightarrow$ Commit) solves three problems:

1.  **Noise:** It prevents 20 "chore: update status" commits in a 15-minute window.
2.  **Safety:** It allows the team to "Undo" or "Revert" a mistake made during the meeting before it becomes permanent history.
3.  **Context:** The commit message "Daily Standup 2025-12-19" is infinitely more valuable for future auditing than a stream of random updates.

Here is the architectural pattern for the **"Dirty Board" Workflow**.

### 1. The Architecture: "Draft Mode"

#### The "Pending" Buffer

The Go server needs an intermediate state. It shouldn't write to `issues.jsonl` immediately upon receiving an API call.

1.  **In-Memory Buffer:** The Go server maintains a slice `[]Event pendingEvents`.
2.  **Optimistic UI:** When the user moves a card:
      * React sends `POST /api/draft` with the event.
      * Go appends it to `pendingEvents`.
      * Go responds "OK".
      * **Crucial:** React shows the card in the new column *immediately*.
3.  **Visual Feedback:**
      * The UI detects `pendingEvents.length > 0`.
      * A "Floater Bar" appears at the bottom: **"12 Unsaved Changes"**.
      * Buttons: `[Discard]` and `[Save & Sync]`.

-----

### 2. The UX Workflow: "The Standup"

**Step 1: The Gathering**
The team gathers. The manager runs `beats board`.

  * *State:* Clean.
  * *Git:* `main` branch is up to date.

**Step 2: The Updates (Draft Phase)**

  * "Alice, are you done with the API?" $\rightarrow$ Alice says yes.

  * Manager drags "API Task" to DONE.

  * *UI Update:* Task moves. A yellow dot appears on the card (indicating "Unsaved").

  * *Bottom Bar:* **"1 Change to commit."**

  * "Bob, is the DB migration blocked?" $\rightarrow$ Bob says yes.

  * Manager right-clicks "DB Migration" $\rightarrow$ Add Blocked Status.

  * *Bottom Bar:* **"2 Changes to commit."**

**Step 3: The "Commit" (Sync Phase)**
The standup ends. The manager clicks **[Save & Sync]**.

  * **Modal Prompt:** "Commit Message?" (Pre-filled with: `Daily Standup YYYY-MM-DD`).
  * **Action:**
    1.  The Go server takes all `pendingEvents`.
    2.  Appends them to `issues.jsonl` on disk.
    3.  Runs `git add .beats/issues.jsonl`.
    4.  Runs `git commit -m "Daily Standup..."`.
    5.  Runs `git push`.
  * **Success:** The yellow dots disappear. The bottom bar vanishes.

-----

### 3. Implementation: The Go Data Structures

Since we are ready to implement, let's define the **Core Domain Models** in Go. This includes your requirements for:

1.  **Event Sourcing** (The Log)
2.  **Kanban State** (The Projection)
3.  **Tags/Labels** (The Flexible Metadata)

Create a file `internal/domain/types.go`.

```go
package domain

import (
	"time"
)

// --- 1. The Event Log (Storage Layer) ---

// EventType defines what happened
type EventType string

const (
	EventCreate   EventType = "CREATE"
	EventUpdate   EventType = "UPDATE"   // Generic update (title, desc, status)
	EventWorkLog  EventType = "WORK_LOG" // "I spent 2 points of effort"
	EventLabel    EventType = "LABEL"    // Add/Remove labels
	EventArchive  EventType = "ARCHIVE"  // Move to cold storage
)

// Event is the immutable atom of history.
// This matches a single line in issues.jsonl
type Event struct {
	ID        string          `json:"event_id"` // Unique ID for the event itself
	Timestamp int64           `json:"ts"`       // Unix timestamp
	User      string          `json:"user"`     // "alice" or "bob"
	Type      EventType       `json:"type"`     // CREATE, UPDATE, etc.
	EntityID  string          `json:"entity_id"`// The Task or Epic ID
	
	// Payload is polymorphic. It depends on the Type.
	// using map[string]interface{} allows flexibility without complex custom unmarshalling
	Payload   map[string]any  `json:"data"`     
}

// --- 2. The Calculated State (Memory/Board Layer) ---

type IssueStatus string

const (
	StatusBacklog IssueStatus = "BACKLOG"
	StatusPlanned IssueStatus = "PLANNED"
	StatusDoing   IssueStatus = "DOING"
	StatusBlocked IssueStatus = "BLOCKED"
	StatusDone    IssueStatus = "DONE"
)

type IssueKind string

const (
	KindEpic IssueKind = "EPIC"
	KindTask IssueKind = "TASK"
)

// Issue represents the "Current State" of a ticket.
// This is what the React UI consumes.
type Issue struct {
	ID          string      `json:"id"`
	Kind        IssueKind   `json:"kind"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      IssueStatus `json:"status"`
	
	// Hierarchy
	ParentID    string      `json:"parent_id,omitempty"` // For Tasks pointing to Epics
	ChildrenIDs []string    `json:"children,omitempty"`  // Computed field (for Epics)

	// Metadata
	Assignee    string      `json:"assignee,omitempty"`
	Labels      []string    `json:"labels"`              // "bug", "frontend", "v1.0"
	CreatedAt   int64       `json:"created_at"`
	UpdatedAt   int64       `json:"updated_at"`

	// Estimation & Work
	Estimate    int         `json:"estimate"`  // Total Story Points
	WorkLogged  int         `json:"work_logged"` // Sum of all WORK_LOG events
	
	// Blocking Logic
	BlockedBy   string      `json:"blocked_by,omitempty"` // ID of the blocker
	BlockReason string      `json:"block_reason,omitempty"`
}

// Board represents the entire world state
type Board struct {
	Epics map[string]*Issue `json:"epics"`
	Tasks map[string]*Issue `json:"tasks"`
	
	// LastProcessedEvent helps us sync with Snapshots
	LastEventID string      `json:"last_event_id"`
}
```

### 4. The "Projection" Logic (Event to State)

Now, we need the function that turns the `Event` stream into the `Issue` state. This is the heart of the system.

Create `internal/logic/projector.go`.

```go
package logic

import (
	"beats/internal/domain"
	"encoding/json"
	"fmt"
)

// ApplyEvent takes the current state of the board and applies a single event mutation
func ApplyEvent(board *domain.Board, evt domain.Event) error {
	
	// 1. Handle CREATE
	if evt.Type == domain.EventCreate {
		newIssue := &domain.Issue{
			ID:        evt.EntityID,
			CreatedAt: evt.Timestamp,
			UpdatedAt: evt.Timestamp,
			Labels:    []string{},
			// Default values from payload...
		}
		
		// Parse payload safely
		if kind, ok := evt.Payload["kind"].(string); ok {
			newIssue.Kind = domain.IssueKind(kind)
		}
		if title, ok := evt.Payload["title"].(string); ok {
			newIssue.Title = title
		}
		if parent, ok := evt.Payload["parent_id"].(string); ok {
			newIssue.ParentID = parent
		}

		// Store in correct map
		if newIssue.Kind == domain.KindEpic {
			board.Epics[newIssue.ID] = newIssue
		} else {
			board.Tasks[newIssue.ID] = newIssue
		}
		return nil
	}

	// 2. Find the existing issue (for Updates)
	var issue *domain.Issue
	if e, exists := board.Epics[evt.EntityID]; exists {
		issue = e
	} else if t, exists := board.Tasks[evt.EntityID]; exists {
		issue = t
	} else {
		// Event refers to unknown issue (maybe archived?). Ignore or Log.
		return fmt.Errorf("entity not found: %s", evt.EntityID)
	}

	// Always update timestamp
	issue.UpdatedAt = evt.Timestamp

	// 3. Handle Updates based on Type
	switch evt.Type {
	case domain.EventUpdate:
		// Map generic fields
		if val, ok := evt.Payload["status"].(string); ok {
			issue.Status = domain.IssueStatus(val)
		}
		if val, ok := evt.Payload["title"].(string); ok {
			issue.Title = val
		}
		// ... handle other fields
		
	case domain.EventLabel:
		// Add or Remove labels
		if label, ok := evt.Payload["add"].(string); ok {
			// Check distinct to avoid duplicates
			found := false
			for _, l := range issue.Labels {
				if l == label { found = true }
			}
			if !found {
				issue.Labels = append(issue.Labels, label)
			}
		}
		// logic for "remove" ...

	case domain.EventWorkLog:
		// Add to the total
		if amount, ok := evt.Payload["amount"].(float64); ok { // JSON numbers are floats
			issue.WorkLogged += int(amount)
		}
	}

	return nil
}
```

### Next Steps

This gives us the strong typing we need.

Would you like to move on to:

1.  **The Store:** Writing the code that reads/writes the `jsonl` file (and handles the "Draft" buffer)?
2.  **The API:** Setting up the Gin handlers to serve this data to the frontend?