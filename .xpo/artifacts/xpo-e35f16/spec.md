# Labels: create without a color step, manage colors and built-ins in the Labels view

## What
- **Label picker:** `Create "foo"` adds the label to the issue in one step. The label has no color. Choosing an existing uncolored label toggles it. The "Pick a color" step is gone.
- **Server:**
  - `config.UpdateLabel` upserts, so a color set on an uncolored label is saved to `config.yaml`.
  - Built-in labels (`default_labels`) are read-only through the API: renaming, recoloring, re-adding or deleting one, or renaming another label onto one, is rejected with 400.
- **Labels view:**
  - Two sections, "Built-in labels" and "Custom labels", each with its own list.
  - Built-in rows are read-only: no dialog, just the count link.
  - Clicking a custom row opens an "Edit label" dialog. Rows no longer have pencil or trash buttons.
  - "k issues" sits at the far right of the row in normal text size, and is the link to the filtered backlog.
- **Edit label dialog** (custom labels only):
  - It uses the same frame as the confirm dialogs.
  - The name and a 12-color palette come first, with a preview of the label on issue rows below them.
  - Every palette color is clearly different from the others and from the built-in label colors.

## Why
Color belongs to label management, not to tagging an issue. Built-in labels drive how the workflow classifies issues, and their colors carry that meaning, so built-in labels can't be changed from the board. Custom colors stay clearly apart from built-in colors, so a custom label is never mistaken for a bug or a feature. Dialogs should look the same everywhere, so the frame is shared code, not copied.

## How

### LabelPicker (`web/src/components/ui/LabelPicker.tsx`)
- Remove `creatingLabel`, `handleSelect`, `handleCreateLabel`, `useAddLabel`, and the `LABEL_PRESET_COLORS` / `LabelBadge` imports.
- The Create row calls `onToggle(name)` and clears the search. It no longer sets `keepOpen`, so the menu closes the same way it does for any other item.
- `canCreateLabels` still decides whether the Create row is shown. Update its doc comment.
- `borderlessBadges` was only used by the color step, so remove it from the props and from the NewIssueModal call sites.
- A new label exists only on the issue until someone colors it in the Labels view. This is how uncolored labels already work: the Labels view lists labels from config and from issues.

### `config.UpdateLabel` (`internal/config/config.go`)
- If `oldName` isn't under `labels:`, append `newName: "color"`, creating the `labels:` section if needed, the same way `AddLabel` does. A missing config file is created rather than returned as an error, again like `AddLabel`.
- To avoid duplicating code, both functions share helpers: `loadLabelsNode`, `upsertLabel`, `writeConfigDoc`.

### Handlers (`internal/server/handlers.go`)
- Add `s.isDefaultLabel(name)`, a case-insensitive match against `s.Config.DefaultLabels`.
- `handleAddLabel`: if `name` is a default label, return 400 `"built-in label %q cannot be changed"`. `AddLabel` upserts, so without this guard a POST could recolor a built-in label.
- `handleUpdateLabel`:
  - If `old_name` is a default label, return 400 `"built-in label %q cannot be changed"`, whether the request renames it, recolors it, or both.
  - If `new_name` is a default label, return 400 `"label name %q is reserved for a built-in label"`. This stops a custom label from being renamed onto a built-in one.
- `handleDeleteLabel`: if `name` is a default label, return 400 `"built-in label %q cannot be deleted"`.

### Shared dialog frame (`web/src/components/ui/Dialog.tsx`)
- **`DialogOverlay`** `{ scope, onClose, enabled?, children }`: the portal, the `bg-black/30` backdrop, and the Escape shortcut. This is the overlay half of today's `ConfirmDialog`. `enabled` turns off Escape while a dialog is stacked on top.
- **`DialogPanel`** `{ id, role?, title, children, footerLeft?, footer }`: the card, in `surface-3` with border, `radius-lg`, `shadow-lg` and `max-w-md`.
  - The title is `text-sm font-semibold`, and the body has `px-5 pt-5 pb-4` padding.
  - The footer bar is on `surface-2`. `footerLeft` defaults to the `esc` "to cancel" hint, and the `footer` buttons sit on the right.
- **`ConfirmDialog` / `ConfirmDialogPanel`** are rebuilt on these with no visual change: `ConfirmDialogPanel` puts the body and description into `DialogPanel` with `role="alertdialog"`. Their existing tests still pass.

### Palette (`web/src/constants.ts`)
- `LABEL_PRESET_COLORS` has 12 colors: seven hues the built-ins don't use, plus darker shades of five of them. They are ordered by hue, each dark shade after its light one:
  - pink `#e85d9c`, plum `#a33a6e`
  - orange `#f28c38`, rust `#b4501f`
  - yellow `#e6c935`, olive `#86881a`
  - lime `#a3c93a`
  - green `#3fb26b`, forest `#2e7a2c`
  - teal `#1fb5b0`, petrol `#0e6a7c`
  - gray `#8a8f98`
- Every pair, and every palette color against each built-in color (red `#eb5757`, purple `#b36cd9`, indigo `#5e6ad2`, blue `#4da6e8`), is more than 60 apart in RGB distance. Near-duplicates like sand/brown/amber and slate/graphite are gone.
- The "New Label" form uses the same palette, and its default color is the first swatch.

### Labels view (`web/src/components/Labels/Labels.tsx`)
- Read `DefaultLabelsContext`. Add pure helpers to `utils/labels.ts`:
  - `isBuiltinLabel(name, defaultLabels)`: case-insensitive.
  - `partitionBuiltinLabels(labels, defaultLabels)` → `{ builtin, custom }`. `builtin` follows the `default_labels` order, and `custom` keeps the input order, which is alphabetical.
  - `recentIssuesWithLabel(issues, label, limit)`: the most recently updated issues that carry the label.
- **Sections:** two sections, each with a small uppercase muted heading, in the style of the Dashboard section titles. The headings have no count: `CountBadge` reads "N issues", which is wrong for a list of labels.
  - The search filters both lists, and a section with no matches is hidden. If neither section has a match, show "No labels match …".
  - If there are no custom labels and no search, the Custom section shows "No custom labels yet · Create your first label".
- **Row:** color dot, name, and on the far right "k issues" (`1 issue`, `text-sm`, muted).
  - "k issues" is a link to the backlog filtered by the label. With 0 issues it is plain text, not a link.
  - **Custom row:** the whole row is a button (`role="button"`, `tabIndex={0}`, Enter/Space) that opens the edit dialog. The count link stops propagation so it doesn't also open the dialog.
  - **Built-in row:** not clickable and no hover highlight; only the count link is interactive.
  - Pencil and trash buttons are removed, along with the inline edit form and the inline Yes/No delete confirmation.
- The view passes the dialog up to 2 issues that carry the label, most recently updated first.
- The "New Label" create form stays inline, as before.

### `EditLabelDialog` (`web/src/components/Labels/EditLabelDialog.tsx`), layout B from the tophat
- `EditLabelDialog` holds the state and wraps `EditLabelPanel` in a `DialogOverlay`. `EditLabelPanel` is portal-free, so it can be tested. Since only custom labels open it, it has no built-in mode.
- Title: "Edit label".
- **Form** first, then the **Preview** below it.
- **Form:** a two-column grid, with field names (`Name`, `Color`) on the left.
  - The name input autofocuses, and Enter saves.
  - The swatches sit in one row that spans the full width of the field column (`flex justify-between`). The selected swatch gets a ring.
- **Preview:** an inset well (`surface-1` with an inset shadow and padding) with a small uppercase "Preview" caption. Inside it is a card with up to two issue rows (status icon, ID in mono, title, and the label badge on the right) showing the label with the current name and color.
  - The second row is dimmed.
  - The badge is the real `LabelBadge`, with its color overridden through `LabelColorsContext`.
  - With no issues, one placeholder row reads "Issues with this label look like this".
- **Footer:**
  - **Left:** "Delete label" (ghost, danger tone), in place of the esc hint.
  - **Right:** Cancel (secondary) and Save (primary). Save is disabled while the name is empty or no color is picked.
- Uncolored labels open with no color selected. The server rejects the muted CSS var that used to be seeded, and a label's old color may no longer be in the palette.
- "Delete label" opens a `ConfirmDialog` ("Delete label?"). It says how many issues the label will be removed from and that this can't be undone. While it is open, the edit dialog's Escape is turned off. A confirmed delete closes both dialogs.

## Decisions
1. **Built-in labels are always protected, even with `hide_default_labels: true`.** Hiding affects display only; the workflow still classifies issues by these labels.
2. **Built-in labels are fully read-only from the board,** color included, so their meaning never drifts. Their colors can still be changed by editing `config.yaml` by hand. (Review feedback; replaces the earlier "recolor only".)
3. **Labels created in the picker are not written to config.** Writing them would need a placeholder color, and a missing color already means "uncolored".
4. **Sections replace the per-row "Built-in" tag.** The section heading already says which labels are built-in, so a tag on each row would repeat it. (Review feedback.)
5. **Delete moves into the edit dialog,** behind a confirmation, because the row buttons are gone. (Review feedback.)
6. **Layout B, with Delete label on the footer's left,** was picked on the tophat over a preview stage (A) and a large hero badge (C). After trying it, the user moved the form above the preview. (Review feedback.)
7. **The preview uses the label's real issues**, not made-up ones, so it shows where the label actually appears.
8. **The confirm dialog's frame becomes the shared `DialogPanel`** rather than restyling `Modal`. `Modal` (the new-issue dialog and others) keeps its header bar, which is out of scope here.
9. **The palette leaves out built-in hues, and every swatch is clearly different from the others.** It is a fixed list in the frontend, not computed from the project's `default_labels` colors. Seven hues with darker shades of five of them (12 swatches) separate better than more hues that look alike. (Review feedback: 10 similar → 7 → 12.)
11. **No explanatory note on the Built-in section.** The rows being inert says enough. (Review feedback.)
10. **A custom label's existing color is kept if it isn't in the new palette.** Opening the dialog shows no swatch selected until a new one is picked. Saving then needs a pick; Cancel keeps the old color.

## Edge cases
- Renaming `Bug` → `bug` when `bug` is a default label: rejected (400).
- Renaming a custom label `ux` → `Bug`: rejected (400, reserved name).
- POST `/api/config/labels` with a built-in name: rejected (400).
- Recoloring a label that appears only on issues: upserted into config, so it survives a restart.
- `UpdateLabel` rename of a label not in config: inserted under the new name, and issues are relabeled by the existing handler code.
- A label with 0 issues: the count is plain text, the row still opens the dialog, and the preview shows the placeholder row.

## Tests
- `internal/config/labels_test.go`: `UpdateLabel` upserts into an existing `labels:` section, creates the section, and creates the config file when it's missing.
- `internal/server/labels_test.go`:
  - updates to a built-in label are rejected with 400: a rename, a case-only rename, and a color-only change; the config is not touched
  - a custom label renamed to a built-in name → 400
  - adding a label with a built-in name → 400
  - deleting a built-in label → 400
  - renaming a custom label still → 200
- `web/src/utils/labels.test.ts`: `isBuiltinLabel` is case-insensitive. `partitionBuiltinLabels` puts built-ins in `default_labels` order and keeps the custom order. `recentIssuesWithLabel` returns the most recent matches.
- `web/src/constants.test.ts`: every `LABEL_PRESET_COLORS` color is more than 60 RGB apart from every other swatch and from each built-in color.
- `web/src/components/ui/Dialog.test.tsx`: `DialogPanel` renders the title, body, the default esc hint, and `footerLeft` in its place.
- `web/src/components/ui/ConfirmDialog.test.tsx`: unchanged, and still passes.
- `web/src/components/Labels/EditLabelDialog.test.tsx` (static render of `EditLabelPanel`):
  - an editable name, with "Delete label" in place of the esc hint
  - Save is disabled for an uncolored label or a blank name
  - the preview shows the given issues' IDs and titles, or the placeholder when there are none
  - one swatch per `LABEL_PRESET_COLORS` entry
  - the name and color fields come before the preview
  - the preview is captioned "Preview"
- The picker and the row interactions are checked by hand, because the web tests can't drive state.

## Acceptance criteria
- [ ] Clicking or pressing Enter on `Create "foo"` applies a new, uncolored label in one step, from both the context menu and the issue detail view
- [ ] Choosing an existing uncolored label toggles it directly
- [ ] Coloring an uncolored label in the Labels view is still there after the server restarts
- [ ] The Labels view lists built-in and custom labels in separate sections
- [ ] Clicking a custom label row opens the edit dialog, and clicking "k issues" opens the filtered backlog
- [ ] Built-in labels can't be renamed, recolored or deleted, in the UI or through the API (400)
- [ ] Palette colors are clearly different from each other and from the built-in colors
- [ ] The edit dialog matches layout B and shares its frame with `ConfirmDialog`
- [ ] Go tests: `UpdateLabel` upsert; the handlers reject every change to a built-in label
