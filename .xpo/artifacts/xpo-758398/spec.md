# Issue Drafts

## What

The user can save the content of the new-issue modal as an issue draft. The server stores each issue draft as a JSON file in `.xpo/drafts/`. An issue draft has no issue ID. A "Drafts" sidebar item with a count badge appears when one or more issue drafts exist. The Drafts view lists the issue drafts. The user opens an issue draft in the new-issue modal, then creates the issue or discards the issue draft.

## Why

Today the new-issue modal has two exits: create the issue or lose the text. Users who are not ready to file an issue must keep the modal open or copy the text elsewhere. Issue drafts keep unfinished issues out of the event log and out of the backlog.

## Terminology

- "issue draft" is the new feature. In code, use the `IssueDraft` type and the `/api/issue-drafts` route.
- `/api/draft` and `handleDraft` are the existing write endpoint. This work does not change them.
- "new-issue modal" is `NewIssueModal`. "discard confirmation dialog" is the alertdialog inside it.
- "Drafts view" is the page at `#/drafts`.

## Acceptance Criteria

- [ ] The new-issue modal shows a "Save draft" button next to "Create issue".
- [ ] "Save draft" writes `.xpo/drafts/<draft-id>.json` and closes the modal.
- [ ] The discard confirmation dialog offers "Save draft", "Discard" and "Cancel".
- [ ] The issue draft JSON holds title, description, labels, additional_labels, status, estimate, parent_id and assignee.
- [ ] The issue draft JSON also holds `id`, `created_at`, `updated_at` and `created_by`.
- [ ] The sidebar shows a "Drafts" item only when one or more issue drafts exist.
- [ ] The "Drafts" item shows a count badge with the number of issue drafts.
- [ ] The Drafts view lists issue drafts, newest `updated_at` first.
- [ ] Each row shows the title (or "Untitled draft"), the primary label and the relative `updated_at`.
- [ ] A click on a row opens the new-issue modal with the issue draft fields filled in.
- [ ] In a reopened issue draft, "Save draft" updates the same file. It does not create a second file.
- [ ] In a reopened issue draft, "Create issue" creates the issue and then deletes the issue draft file.
- [ ] In a reopened issue draft, "Discard draft" deletes the issue draft file after a confirmation.
- [ ] Each row in the Drafts view has a delete action with a confirmation.
- [ ] When the user deletes the last issue draft, the sidebar hides the "Drafts" item.
- [ ] When no issue drafts exist, the Drafts view shows an empty state.
- [ ] Git does not track `.xpo/drafts/`.
- [ ] With auth on, a user sees and changes only their own issue drafts.
- [ ] Other open browser tabs receive issue draft changes through SSE.
- [ ] Go tests cover the storage functions and the HTTP handlers.
- [ ] Frontend tests cover the issue draft behavior of the new-issue modal.
- [ ] `make test` passes.

## Flow

### Backend

1. Add `model.IssueDraft` to `internal/model/types.go`. Fields: `ID`, `CreatedAt`, `UpdatedAt`, `CreatedBy`, `Payload`.
2. Add `model.IssueDraftPayload` with the modal fields: title, description, labels, additional_labels, status, estimate, parent_id, assignee.
3. Add `internal/storage/drafts.go` with `ListDrafts`, `ReadDraft`, `WriteDraft` and `DeleteDraft`.
4. Generate draft IDs as `draft-<8 hex>`. This ID cannot collide with an issue ID.
5. When `WriteDraft` creates `.xpo/drafts/`, write `.xpo/drafts/.gitignore` with the content `*`.
6. Write each file atomically: write to a temp file in `.xpo/drafts/`, then rename it.
7. Write the file with mode 0644. Do not HTML-escape the JSON, so `created_by` stays readable.
8. Validate the draft ID against `^draft-[0-9a-f]{8}$` before any file access. This check blocks path traversal.
9. Add `Client` methods for issue drafts in `internal/exponential/drafts.go`.
10. Do not add the methods to the `Transport` interface. Each method returns `ErrLocalOnly` when the client has no `LocalTransport`.
11. Add `ValidateIssueDraftPayload`. It applies the create-payload length limits, the status check and a non-negative estimate.
12. `ValidateIssueDraftPayload` allows an empty title. An issue draft can hold only a description.
13. Add handlers in `internal/server/issue_drafts.go` for `GET`, `POST`, `PUT` and `DELETE`.
14. Use the routes `/api/issue-drafts` and `/api/issue-drafts/{id}`.
15. Register the routes in `SetupRoutes` with the `issue.create` capability.
16. Set `created_by` with `Client.GetUser`: the auth user, else the configured user, else the git user.
17. With auth on, filter the `GET` result by `created_by`.
18. With auth on, return 404 for `PUT` and `DELETE` on an issue draft of another user.
19. Return 404 for an invalid draft ID. Return 501 for `ErrLocalOnly`.
20. Call `s.broadcastEvent("draft_changed", id)` after each write and each delete.
21. In `handleSSE`, send the `draft_changed` broadcast as the SSE event `draft_changed`.

### Frontend

22. Add `fetchIssueDrafts`, `createIssueDraft`, `updateIssueDraft` and `deleteIssueDraft` to `web/src/api/client.ts`.
23. Add the `issueDrafts` query key. Add it to `SSE_INVALIDATED_KEYS`.
24. Add the `useIssueDrafts` query. It returns `[]` while it loads or when the user has no access.
25. In `useSSE.ts`, listen to the `draft_changed` event.
26. Add `saveIssueDraftOptions`, `deleteIssueDraftOptions` and `createIssueFromDraftOptions` to `mutation-options.ts`.
27. `createIssueFromDraftOptions` creates the issue, then deletes the issue draft. It returns `{ issueId, draftDeleted }`.
28. Add a `drafts` view to `VIEWS` with the `g r` shortcut and `requires: "drafts"`.
29. Add the view icon to `VIEW_ICONS` and the view component to `VIEW_COMPONENTS`.
30. Hide the `drafts` sidebar item when the issue draft count is 0. Pass the count as `badge`.
31. Add `web/src/components/Drafts/Drafts.tsx` with the list, the row delete action and the empty state.
32. Add `ConfirmDialog` to `web/src/components/ui/`. Move the alertdialog markup of the modal into it.
33. Add `issue-form.ts` with the form type and the conversion between form and issue draft payload.
34. Add an optional `draft` prop to `NewIssueModal`. Fill the form state from `draft.payload`.
35. In `App.tsx`, remount the modal on each open. The form then starts from the issue draft, or empty.
36. For a reopened issue draft, set the modal title to "Edit Draft". Show a "Discard draft" button.
37. Add the "Save draft" button. It calls update when `draft` is set, and create when it is not.
38. Add "Save draft" to the discard confirmation dialog.
39. Add `useAppNav().openDraft(draft)` in `App.tsx`. It opens the new-issue modal with the `draft` prop.

## Decisions

- **Server files in `.xpo/drafts/`, not `localStorage`.** The issue description asks for `.xpo/drafts`. Server files survive a browser change, and every browser can open them.
- **Not in the event log.** An issue draft is not an issue. Event-log entries would need a new projection and would show the issue draft in history.
- **A separate `/api/issue-drafts` route, not `/api/draft`.** The old name means "write event". A new route keeps the two concepts apart.
- **Draft ID prefix `draft-`.** The prefix makes the file name and the URL clearly different from an issue ID.
- **The sidebar item is conditional.** The issue description asks for this. The `g r` shortcut opens the Drafts view also when no issue drafts exist.
- **Git ignores `.xpo/drafts/`.** Issue drafts are personal work in progress. A `.gitignore` inside the directory works in every repository. The root `.gitignore` does not change.
- **With auth on, each user sees only their own issue drafts.** The handlers filter by `created_by`. With auth off, all users see all issue drafts. A request for an issue draft of another user gets 404, not 403. This hides that the issue draft exists.
- **Client methods, not `Transport` methods.** Remote mode cannot reach `.xpo/drafts/` on the hub. A `Transport` method would need a stub that only returns an error.
- **The modal remounts on each open.** The form reads the issue draft once at mount. This avoids an effect that copies props into state.
- **A shared `ConfirmDialog`.** The modal and the Drafts view need the same confirmation. One component keeps them consistent.
- **No MCP tool or CLI command in this issue.** The scope is the web UI. Agents can create issues directly.

## Edge Cases

### HIGH (developer decides)

- **The issue create succeeds, but the issue draft delete fails.** The issue exists and the issue draft stays. The modal closes, so the user cannot create the issue twice. The UI shows "Issue created, but its draft could not be deleted". The user can delete the issue draft in the Drafts view.

### MEDIUM (agent proposes, developer confirms)

- **Empty issue draft.** The modal disables "Save draft" when title and description are both empty. This rule matches `hasContent`.
- **A reopened issue draft without changes.** Close does not show the discard confirmation dialog. With changes, the dialog asks "Discard changes?". "Discard" keeps the saved issue draft.
- **Another tab deletes the open issue draft.** "Save draft" gets a 404. The handler does not recreate the file. The UI shows "This draft no longer exists" and the modal stays open. The user can still create the issue.
- **Another user deletes the parent issue after the user saves the issue draft.** "Create issue" fails validation. The UI shows the validation error. The issue draft stays.
- **Corrupt JSON file in `.xpo/drafts/`.** `ListDrafts` skips the file and logs a warning. `GET /api/issue-drafts` still returns the other issue drafts.
- **A user without the `issue.create` capability.** `GET /api/issue-drafts` returns 403. The UI treats this as no issue drafts and hides the sidebar item.

### LOW (agent handles silently)

- If `.xpo/drafts/` does not exist, `ListDrafts` returns an empty list. `WriteDraft` creates the directory.
- `ListDrafts` ignores files that do not match `draft-<8 hex>.json`.
- If the count is above 99, the existing `NavItem` shows "99+".

## Assumptions

- Only local mode supports issue drafts. In remote mode, the client methods return `ErrLocalOnly`. The web UI talks to the local server.
- The `issue.create` capability is sufficient for all four issue draft operations.
- No auto-save. The user saves an issue draft explicitly.
- Priority and cycle stay out of scope, because the new-issue modal does not show them today.

## Open Questions

None.
