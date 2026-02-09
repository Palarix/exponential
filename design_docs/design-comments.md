# Comments Design

## Goal
Enable team collaboration by allowing users to attach comments to issues. Comments should be appended to the issue's timeline and viewable by other team members.

## User Stories
1. **Add Comment**: As a user, I want to add a comment to an issue so that I can provide updates or ask questions.
2. **View Comments**: As a user, I want to view the comment history of an issue so that I can understand the discussion context.

## domain Model Changes

### Events
We will introduce a new event type to the event stream:

```go
const EventTypeComment EventType = "COMMENT"

type CommentPayload struct {
    Text string `json:"text"`
}
```

### State Projection (Issue)
The `Issue` struct will be updated to include a slice of comments. This allows the state projection logic to aggregate comments from the event stream.

```go
type Comment struct {
    ID        string    `json:"id"` // Event ID
    Text      string    `json:"text"`
    CreatedBy string    `json:"created_by"`
    CreatedAt time.Time `json:"created_at"`
}

type Issue struct {
    // ... existing fields ...
    Comments []Comment `json:"comments"`
}
```

### Projection Logic
In `internal/beats/projection.go`, the `ProjectIssues` function will handle `EventTypeComment`:

```go
case model.EventTypeComment:
    // ... find issue ...
    var p model.CommentPayload
    json.Unmarshal(payload, &p)
    
    comment := model.Comment{
        ID:        evt.ID,
        Text:      p.Text,
        CreatedBy: evt.CreatedBy,
        CreatedAt: evt.CreatedAt,
    }
    issue.Comments = append(issue.Comments, comment)
    issue.UpdatedAt = evt.CreatedAt
    issue.Events = append(issue.Events, evt)
```

## CLI Design

### `beats comment`
Adds a new comment to an issue.

**Usage:**
```bash
# Inline comment
beats comment beats-123 "This is a comment"

# From Stdin
echo "This is a longer comment" | beats comment beats-123

# Interactive (Future?)
# beats comment beats-123 (opens $EDITOR)
```

**Implementation:**
- Logic in `cmd/beats/comment.go`.
- Validates issue existence (optional, but good practice).
- Creates `EventTypeComment` event.
- Appends to `issues.db`.

### `beats comments`
Lists comments for an issue in chronological order.

**Usage:**
```bash
beats comments beats-123
```

**Output:**
```text
@nicbet (2 minutes ago):
This is a comment
----------------------------------------
@other (1 minute ago):
Another comment
```

**Implementation:**
- Logic in `cmd/beats/comments.go`.
- Loads issue.
- Iterates over `issue.Comments` and prints them.

## Future Considerations
- **Rich Text/Markdown Rendering**: The CLI could render markdown.
- **Mentions**: Parsing `@user` mentions (already supported in UI data model, but maybe could trigger notifications).
- **Reactions**: Emoji reactions to comments.
