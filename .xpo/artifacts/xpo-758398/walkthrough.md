# Walkthrough: Issue Drafts

## What was built

The user can save the content of the new-issue modal as an issue draft. An issue draft is a JSON file in `.xpo/drafts/`. It has no issue ID and does not go into the event log. A "Drafts" sidebar item with a count badge appears when issue drafts exist. The Drafts view lists them. A click on a row reopens the issue draft in the new-issue modal. From there the user saves it again, creates the issue, or discards the issue draft.

## Why it is built this way

- **Files, not events.** An issue draft is not an issue. The event log would need a new projection, and the issue draft would show in history. A file per issue draft is simple to read, write and delete.
- **A new name and route.** The codebase already uses "draft" for `POST /api/draft` (`handleDraft`). That endpoint applies write events immediately. The new feature uses `IssueDraft` and `/api/issue-drafts` to keep the two concepts apart. The handler file is `issue_drafts.go`, not `drafts_handlers.go`, so it does not sit next to `draft_test.go` with a similar name.
- **Personal work in progress.** Git ignores `.xpo/drafts/`, and with auth on each user sees only their own issue drafts.

## How the pieces fit together

### Storage: `internal/storage/drafts.go`

- `NewDraftID` returns `draft-<8 hex>`. `ValidDraftID` checks `^draft-[0-9a-f]{8}$`.
- `draftPath` calls `ValidDraftID` before it builds a path. Every read, write and delete goes through `draftPath`. This check blocks path traversal.
- `WriteDraft` writes a temp file in `.xpo/drafts/`, sets mode 0644, then renames it. A reader never sees a partial file.
- `WriteDraft` encodes with `SetEscapeHTML(false)`. Without it, `created_by` would contain `<` instead of `<`.
- `ensureDraftsDir` creates the directory and writes `.gitignore` with `*`. This works in every repository, and the root `.gitignore` does not change.
- `ListDrafts` skips corrupt files with a log warning, so one bad file does not hide the others. It sorts by `UpdatedAt`, newest first.
- `ReadDraft` and `DeleteDraft` return `ErrDraftNotFound` for a missing file.

### Client: `internal/exponential/drafts.go`

- `ValidateIssueDraftPayload` applies the create-payload length limits, the status check and a non-negative estimate. It allows an empty title, because an issue draft can hold only a description.
- `ListIssueDrafts`, `CreateIssueDraft`, `UpdateIssueDraft` and `DeleteIssueDraft` take an `owner`. An empty owner means auth is off and no filter applies.
- `ownedDraft` returns `ErrDraftNotFound` for an issue draft of another owner. The caller cannot tell that the issue draft exists.
- `UpdateIssueDraft` keeps `ID`, `CreatedAt` and `CreatedBy`. It never recreates a deleted issue draft.
- The methods are on `Client`, not on the `Transport` interface. Remote mode cannot reach `.xpo/drafts/` on the hub, so each method returns `ErrLocalOnly` when the client has no `LocalTransport`.

### HTTP: `internal/server/issue_drafts.go`

- `issueDraftClient` builds a client that acts as the request user. With auth on, it also returns that user as the owner.
- `respondIssueDraftError` maps `ErrDraftNotFound` and `ErrInvalidDraftID` to 404, and `ErrLocalOnly` to 501.
- Each write and delete calls `s.broadcastEvent("draft_changed", id)`. `handleSSE` in `sse.go` sends it as the SSE event `draft_changed`.
- `router.go` registers the four routes with the `issue.create` capability.

### Frontend

- **API.** `client.ts` has the four calls. `query-keys.ts` adds `issueDrafts` to `SSE_INVALIDATED_KEYS`. `useSSE.ts` listens to `draft_changed`. `useIssueDrafts` returns `[]` on error, so a user without `issue.create` sees no sidebar item.
- **Mutations.** `mutation-options.ts` adds `saveIssueDraftOptions`, `deleteIssueDraftOptions` and `createIssueFromDraftOptions`. The last one creates the issue, then deletes the issue draft. A failed delete does not fail the mutation, because a retry would create a duplicate issue. It returns `{ issueId, draftDeleted }` instead.
- **Form helpers.** `NewIssueModal/issue-form.ts` holds the form type and converts between form and payload. These pure functions carry the unit tests, because the project has no DOM test setup.
- **Modal.** `NewIssueModal` takes an optional `draft` prop. It reads the form state once at mount. `App.tsx` gives the modal a new `key` on each open, so the form starts from the issue draft or empty. This avoids an effect that copies props into state.
- **Dirty check.** For a new issue, the modal asks for confirmation when title or description has text. For a reopened issue draft, it asks only when the form differs from the saved version.
- **Confirmation dialogs.** `ui/ConfirmDialog.tsx` replaces the inline alertdialog of the modal. The modal and the Drafts view use it. It owns its Escape shortcut scope.
- **Navigation.** `views.ts` adds the `drafts` view with `g r` and `requires: "drafts"`. `Sidebar` hides the item when the count is 0 and passes the count as `badge`. `useAppNav().openDraft` opens the modal with an issue draft.
- **Drafts view.** `Drafts/Drafts.tsx` lists the rows and uses the shared `EmptyState`, as the other full-page views do.

## Decisions made during review

- The first Drafts empty state copied the small Inbox panel style. During tophat, the user asked for the full-page style. The view now uses `EmptyState` with a dashed-outline illustration and a "New issue" action.
- The first smoke test showed draft files with mode 0600 and escaped `<` `>`. `WriteDraft` now sets 0644 and disables HTML escaping. A storage test covers both.

## Acceptance Criteria

- [x] The new-issue modal shows a "Save draft" button next to "Create issue". Evidence: smoke test snapshot.
- [x] "Save draft" writes `.xpo/drafts/<draft-id>.json` and closes the modal. Evidence: smoke test, file `draft-9d66b483.json` on disk.
- [x] The discard confirmation dialog offers "Save draft", "Discard" and "Cancel". Evidence: smoke test screenshot.
- [x] The issue draft JSON holds title, description, labels, additional_labels, status, estimate, parent_id and assignee. Evidence: `TestReadDraft_RoundTrip`, `issue-form.test.ts`.
- [x] The issue draft JSON also holds `id`, `created_at`, `updated_at` and `created_by`. Evidence: `TestCreateIssueDraft_SetsMetadata`, file on disk.
- [x] The sidebar shows a "Drafts" item only when one or more issue drafts exist. Evidence: smoke test before and after the save.
- [x] The "Drafts" item shows a count badge with the number of issue drafts. Evidence: smoke test, badge "1".
- [x] The Drafts view lists issue drafts, newest `updated_at` first. Evidence: `TestListDrafts_NewestFirstAndSkipsBadFiles`, smoke test.
- [x] Each row shows the title (or "Untitled draft"), the primary label and the relative `updated_at`. Evidence: smoke test, `draftDisplayTitle` test.
- [x] A click on a row opens the new-issue modal with the issue draft fields filled in. Evidence: smoke test, "Edit Draft" modal with the saved title.
- [x] In a reopened issue draft, "Save draft" updates the same file. Evidence: smoke test, one file with the new title. `TestUpdateIssueDraft_KeepsIDAndCreatedAt`.
- [x] In a reopened issue draft, "Create issue" creates the issue and then deletes the issue draft file. Evidence: smoke test. `createIssueFromDraft` tests.
- [x] In a reopened issue draft, "Discard draft" deletes the issue draft file after a confirmation. Evidence: code path through `ConfirmDialog` and `useDeleteIssueDraft`. The smoke test did not click it.
- [x] Each row in the Drafts view has a delete action with a confirmation. Evidence: code path and a11y snapshot ("Delete draft" button). The smoke test did not click it.
- [x] When the user deletes the last issue draft, the sidebar hides the "Drafts" item. Evidence: smoke test after "Create issue".
- [x] When no issue drafts exist, the Drafts view shows an empty state. Evidence: smoke test screenshot with `EmptyState`.
- [x] Git does not track `.xpo/drafts/`. Evidence: `TestWriteDraft_CreatesDirWithGitignore`, empty `git status` in the demo repository.
- [x] With auth on, a user sees and changes only their own issue drafts. Evidence: `TestIssueDrafts_AuthScopesToOwner`, `TestIssueDrafts_OwnerFilter`.
- [x] Other open browser tabs receive issue draft changes through SSE. Evidence: `TestIssueDrafts_BroadcastsDraftChanged`, `TestHandleSSE_StreamsDraftChanged`, `query-keys.test.ts`. No manual two-tab test.
- [x] Go tests cover the storage functions and the HTTP handlers. Evidence: `storage/drafts_test.go`, `exponential/drafts_test.go`, `server/issue_drafts_test.go`.
- [x] Frontend tests cover the issue draft behavior of the new-issue modal. Evidence: `issue-form.test.ts`, issue draft cases in `mutation-options.test.ts`.
- [x] `make test` passes. Evidence: exit 0, 651 frontend tests and the Go suite.

## Notes for future readers

- `.xpo/drafts/` is on the hub, through `storage.XpoDir()`. A worktree build reads the same issue drafts as the main checkout.
- No MCP tool or CLI command exists for issue drafts. Add one through the `Client` methods, not through `Transport`.
- Priority and cycle are not in the payload, because the new-issue modal does not show them. If the modal gets them, add them to `IssueDraftPayload` and `issue-form.ts` together.
