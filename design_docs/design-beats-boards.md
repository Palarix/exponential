# Beats Boards
This feature describes the design of the `beats board` command. The board is an interactive, reactive browser based view on beats issues and states. It allows users to create, modify and delete issues and beats, and to change the relative order of issues, as well as changing their state by moving them between status columns.

With the `beats board` command, the CLI starts a Go-based webserver on port `9132`, serving a React app that let's users interact with beats issues as a Kanban board.

## User Interface
The Kanban board is rendered as a Single Page Application (SPA) with a column-based Kanban Board as the main interface.
- Epics are rendered as "swimlanes" on the Kanban board grouping their child issues. Epics aggregate statistics of the child epics: in addition to showing the total number of children in the epic, we also display the number of issues in (OPEN, DOING, DONE) states, as well as the sum of all estimated story points across all children and the sum of story points burned down.
- Based on the event stream, we can also offer a "Statistics" butto that shows project statistics like burndown chart over time, velocity, etc.
- Issues are rendered as cards with their title on the card, as well as metadata in the corners like story points.
- The left borders of the issue cards are colored by issue type: red for bugs, blue for features, grey for tasks. 
- Uses TailwindCSS v4 and ShadCN for UI components and styling

## Operations
User can perform all the CRUD operations that are also available on the commandline:
- Add a new issues (select issue type, specify title and metadata, optionally parent, story point, blocking or blocked by, etc.)
- Clicking on an existing issue shows the issue details in a modal dialog and lets the user interactively update issue fields (title, description, metadata, select blockers or blocking issues (with autocomplete))
- Move issues to another state by dragging and dropping them across the columns of the Kanban board
- Re-order issues to change their relative priority (we need to support a new `order` field for issues, his will also cause the sorting of our CLI projection to change for the `beats ls` commands, which is currently sorting by `created_at` in ascending order)
- Delete obsolete issues (with confirmation dialog)
- When board state is changed by the user, shows "Changes (unsaved)" in the header, with an option to save all changes to the board (only then writes `issues.db`)
- Issues can be assigned to a parent EPIC by dragging and dropping the issue into an epic swimlane.
- Issues can be reordered by dragging and dropping them above or beyond other issues.
- Issues can be assigned a new status by dragging and dropping them in a new state column on the Kanban board.
- In addition to a Kanban board view, there is a list view that allows for easier planning by showing all issues in a table-like manner with configurable columns shown (settings are stored using a browser local db so they are specific to each user).
- The planning view allows to create a number of filters and store those filters with a name in browser local storage (so they are specific to a user as well).



## Interactivity / Live Board
For `beats` to work as a bridge between Humans and Agents, the board **must be "Live"**.

If an AI Agent running in the background (or a colleague via CLI) adds a task to `issues.db`, your browser should update instantly without you hitting refresh.

Here is how we implement this **"Real-Time Reactivity"** using the file system as the trigger.

### The Architecture: File Watcher + SSE

We don't need complex WebSockets. **Server-Sent Events (SSE)** are perfect for this one-way data flow (Server to Client).

1.  **The Trigger:** An Agent (or CLI) appends a line to `.beats/issues.db`.
2.  **The Watcher:** The Go server (running `beats board`) has a file watcher (`fsnotify`) monitoring that specific file.
3.  **The Delta:** The server reads *only the new lines* (events) added since the last read.
4.  **The Push:** The server sends a JSON payload via an open HTTP connection (SSE) to the React frontend.
5.  **The React:** The frontend Reducer takes the new event, applies it to the local state, and the board re-renders.

### 1. The Go Implementation (`fsnotify`)

In your Go server, you run a background go routine that watches the file.

```go
// internal/server/watcher.go
import (
    "github.com/fsnotify/fsnotify"
)

func WatchFile(filePath string, broadcast chan<- Event) {
    watcher, _ := fsnotify.NewWatcher()
    defer watcher.Close()

    watcher.Add(filePath)

    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok { return }

            if event.Op&fsnotify.Write == fsnotify.Write {
                // 1. File changed! Read new lines.
                newEvents := readNewLines(filePath)

                // 2. Broadcast to all connected clients
                for _, evt := range newEvents {
                    broadcast <- evt
                }
            }
        case err := <-watcher.Errors:
            // handle error
        }
    }
}
```

### 2. The Server-Sent Events (SSE) Handler

In Gin (or standard `net/http`), you expose a stream endpoint.

```go
// GET /api/stream
func StreamHandler(c *gin.Context) {
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")

    // Listen to the broadcast channel
    clientChan := make(chan Event)
    registerClient(clientChan)
    defer unregisterClient(clientChan)

    for {
        select {
        case evt := <-clientChan:
            // Send Data to Browser
            // Format: "data: { ...json... }\n\n"
            c.SSEvent("message", evt)
            c.Writer.Flush()
        case <-c.Request.Context().Done():
            return // Client disconnected
        }
    }
}
```

### 3. The React "Listener"

In your React app (`useEffect`), you listen for these messages.

```javascript
useEffect(() => {
  const eventSource = new EventSource('/api/stream');

  eventSource.onmessage = (e) => {
    const newEvent = JSON.parse(e.data);

    // "Dispatch" this event to your Reducer
    // This uses the exact same logic as your initial load!
    dispatch({ type: 'APPLY_EVENT', payload: newEvent });

    // Optional: Show a subtle "toast" notification
    if (newEvent.user === "DevinAI") {
        toast.info("AI Agent added a new task");
    }
  };

  return () => eventSource.close();
}, []);
```

### 4. Handling "Draft" Conflicts

There is one UX edge case: **What if I am dragging a card, and the AI moves it at the same time?**

Since you are using the **Draft/Transaction** model we discussed:

1.  **Scenario:** You have "Task A" in *Draft Move* state (Pending).
2.  **Incoming Event:** The AI updates "Task A" status to "DONE".
3.  **Reaction:**
      * The React App receives the event.
      * It detects you have a pending draft on that same ID.
      * **Strategy:** It pauses the auto-update for that specific card and shows a **"Stale Data" warning** on the card.
      * **User Action:** You see "Conflict". You can choose to **"Accept Incoming"** (AI wins) or **"Force Push"** (You win).

For **New Items** (Backlog additions), there is never a conflict. They just pop onto the board instantly. This creates a really magical "multiplayer" feel, even if the other player is a bot.
