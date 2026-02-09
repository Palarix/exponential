# Beats Issues

Beats is an opinionated issue tracking system. However, opinionated does not mean strict or rigid: beats aims to distill best practices from other issue tracking and project management tools that came before it (Jira, Trello, Linear, ...). Beats gaol is to offer a flexible enough work plane that supports the workflows of many different teams. However, beats is not a configure-everything one-tool-fits-all system like Jira.

## Issue Types

Unlike other issue tracking systems that let users define any number of issue types, beats has a pre-determined set of issue types.

- `EPIC` is a type of issue that contains children issues (cannot be epics). They are used to group related work together that would otherwise be too complex or lengthy for a single issue. The work in epics is intended to be completed together as a unit to deliver the intended functionality. Since epics are a container, they inherit state and metadata from their children if present.

  Epics may have their own title and description, and can have a story points estimate, but this estimate is only relevant for planning purposes when no children have been added yet. As soon as children are present in the Epic, the story point estimate for an epic should be the sum of all story point estimates over all children issues.

- `FEATURE` describes a new feature or functionality in a system. They should be backed by user stories and acceptance criteria or requirements for the feature and estimated with a relative story points metrics.

- `TASK` describes a generic task or unit of work that needs to be completed, such as adding documentation, refactoring the code, creating supporting source code artifacts such as CI/CD files, Makefiles, test case fixtures, etc.

- `BUG` describes an error, problem or misbehaviour in the source code, program or supporting files that should be repaired. Bugs generally have precedence over other issue types when selecting the next piece of work to be worked on.

## Issue Metadata

In addition to the issue `type`, issues in beats have various metadata attached to further describe the issue, their dependencies and state.

All issues have the following mandatory metadata:

- **ID**: a randomly generated, unique id for this issue. Consists of a project-specific prefix (e.g., `beats-`) and an nano-id component (e.g., `a7bzde`).
- **Title**: a descriptive short summary of the issues in 80-120 characters

Issues may have the following optional metadata:

- **Description**: a longer explanation or description of the issue, written in Markdown
- **Status**: a state flag describing the current lifecycle status of the issue, for example `BACKLOG`, `PLANNED`, `DOING`, `DONE` or `BLOCKED`.
- **Checklist**: An array of checklist items of the shape `{t: "<title>", s: "<state>"}` , where the `state` can be one of `open`, `done`.
- **Labels**: a comma-separated list of tags assigned to the issue.
- **Estimate**: a story points estimate of the complexity or size of the issue
- **LoggedEffort**: a record of story points logged against working on the issue
- **Dependencies**: describe how issues relate to each other in the workflow. Stored as an array of `Dependency` objects with the shape `{s: "<id">, t: "<id>", k: "<kind>"}` where `s` denotes the dependency source, `t` the dependency target, and `k` the dependency kind.

Issues of type `EPIC` are a special case, as they have both direct and inherited metadata. Users may initially set a story point estimate for the Epic. As child issues are added their sum of story point estimates is shown instead of the initial estimate stored on the epic. Similarly, the status of an Epic can be initially set, but a child issues are added and change state, the state of the epic is derived from its children.

An Epic should never be "Done" if any child is not "Done." An Epic automatically moves to "Doing" the moment a child moves to "Doing" or "Planned." This prevents stale epics that appear in the Backlog despite work having started. When all children are done beats shows a prompt/hint that the epic can be marked as done, but does not mark as done automatically to allow for manual approval / sign-off.

As soon as children are present, the Epic's status and story points are derived. An Epic is `IN PROGRESS` if any child is `PLANNED` or `DOING`, and is only `DONE` when all children are `DONE`.

### Dependencies

- `blocks` / `blocked_by`: one issue must be completed before the other issue can start. This creates a hard stop where the "blocked" issue can not be moved to "doing" or "done" until the "blocker" issue is resolved.
- `precedes` / `follows`: a softer version of "blocking", suggesting a logical sequence without necessary locking of state.
- `parent` / `child`: a hierarchical relationship that allows to break down larger items (Epics) into smaller manageable pieces (Features, Tasks, Bugs, Stories). A parent cannot be marked "done" until all the children are closed.
- `relates_to`: a generic link indicating that one issue is connected to another issue but not strictly dependent on the other.
- `duplicates` / `duplicated_by`: used when two issues describe the same task, feature, or bug. One is usually resolved as "duplicate" while the other remains as the "master" ticket for tracking.
- `causes` / `caused_by`: often used when a bug was introduced by another change, this information is vital for root cause analysis.
- `fixes` / `fixed_by`: used when one issue fixes another bug.

### Metadata Keys

To keep file size minimal, metadata keys are the following:

- `id` the story unique ID
- `t` the story title
- `d` the story description
- `s` the story status
- `spe` the story points estimate for the issue
- `spl` the story points logged against the issue
- `l` the labels attached to the story
- `c` a Checklist (`[{t: "...", s: "..."}]`)
- `dep` the Dependencies (`[{s: "...", t: "...", k: "..."}]`)

### Comments

To foster collaboration in beats, users may comment on issues to provide additional context and information. Comments are recorded as events in the `issues.db` event stream and attached to the issue ordered temporally.

## Work Modes

Beats unlike previous issues tracking systems aims to be equally consumable and driven by humans and AI agents alike.

For this purpose, beats offers three distinct experiences to interact with the beats system:

1. a **browser-based** experience that is primarily meant to be consumed by non-technical humans
2. an interactive, **TUI-based** terminal experience that is primarily meant to be consumed by technical humans
3. a **CLI interface** that is primarily meant to be consumed by AI agents and automation scripts (but can be consumed by humans, too)

### The Browser Experience

By running the `beats board` command users can start a built-in web-server hosting a React app and a set of API endpoints.

The browser-based experience is divided into four distinct work modes aimed at supporting different project stakeholders:

- a Dashboard view,
- a Backlog view,
- a Board view,
- a Dependencies view.

Users can switch between these views as Tabs in the application top bar, which also hosts the Search bar, Notifications and the "Save updates", "Sync" buttons.

Below the top application bar sits the view-specific toolbar, which shows the view name "Dashboard", "Backlog", "Board", "Dependencies" on the left-hand side, and filtering, grouping dropdown menus on the right-hand side, along with view-dependent tool buttons (like "Add issue" in the Board view").

Below the view toolbar sits the main view content window that displays the view-dependent main content.

#### The Dashboard View

The dashboard view aims at making summary statistics about the project available to managerial stakeholders. It displays common software work statistics such as velocity, burn-down charts, average story points, total story points, issue statistics such as number by type, number by state and so on.

#### The Backlog View

The backlog view aims at offering team leads and scrum masters an easy way to view and groom the backlog, prioritize work, add details to existing issues, update issues and create new issues, as well as working with epics.

The backlog view divides the entirety of beats issues into 3 tables:

1. The "Backlog" table contains all issues that have been added and have a `BACKLOG` state.
2. The "Active" table contains all issues that have a `PLANNED`, `DOING` or `DONE` state.
3. The "Archived" table contains all issues that have a `DONE` state and have been moved to the archive database. The "Archived" table is collapsed by default, with only the table header and a collapse icon button showing. Table headers show a count of issues in this table, for example: "Archived (14)".

The Backlog view allows for bulk editing and updating of issues. The view toolbar has an "Edit" button that changes to an "Update all" button when multiple rows (issues) are selected. Clicking on this button either opens a modal dialog for editing a single issue, or a modal dialog that lets users select from possible bulk operations - depending on the selection.

Users can change the relative order of issues within their table with drag and drop. Users can add one or more selected issues to the Sprint by clicking the "Add to Sprint" button in the view toolbar (this will also change their status from "BACKLOG" to "PLANNED").

#### The Board View

The board view is aimed at developers working on the project. The main feature of this view is the Kanban board with columns representing issue state and swimlanes representing epics. On the board are issue cards that users interact with. By default, the `BACKLOG` column is hidden but can be enabled in the options.

Users can drag and drop cards to either change their relative position (order) to each other, or to set a specific state on an issue by dragging the issue to the corresponding state column. Clicking on an issue opens a modal dialog that allows users to edit the title, description, and other metadata. They can assign tags to issues from this UI with an autocomplete function that suggests existing tags (to help with consistency in the project) and allows creating new tags.

Users can also comment on an issue in this dialog. The commenting function does not allow for file attachments, but can link to existing files in the project repository via `@/file/path` links. They can also mention other users using the `@username` link. For both cases, the UI offers suggestions and autocomplete options.

Within the modal dialog, users can edit the `Checklist` of the issue, add items to the checklist, remove items from the checklist and mark items in the checklists as `open` or `done`. If a checklist exists on an issue, the card on the board shows a progress bar beneath the card title corresponding to the open/done number of checklist items.

Users can add new issues to the board by clicking the "Add issue" button in the board header. This opens up a new modal dialog which lets users pick the issue type, a parent issue (if the issue type is `EPIC`), enter a title (required) and description (optional), as well as any additional metadata like story point estimates, tags, state.

Users can delete issues by clicking on an issue and selecting the trash can icon from the issue detail dialog. Deletion happens only after user confirmation. Bulk deletion should be done from the backlog view and we don't offer support for bulk actions in the Kanban board view as its intention is to focus on individual issues.

#### The Dependencies View

The dependencies view displays all unarchived `beats` issues as a dependency graph / tree. It is rendered based on the `dependency` metadata of issues. Edges between issues are rendered differently on the dependency type: `blocking` edges are red, `required` edges are blue, `related` edges are gray.
