## What changed

Tagging an issue with a new label no longer asks for a color. Colors, renames and deletes now happen in the Labels view, through a new Edit label dialog. Built-in labels (`default_labels`) are read-only from the board and through the API, so the meaning the workflow gives them can't drift. Review reshaped the Labels view, the dialog and the palette over several rounds. Each change went into the spec before the code.

## How the pieces fit

### Label picker: one-step create
- `LabelPicker` lost the "Pick a color" sub-view (`creatingLabel`, `handleSelect`, `handleCreateLabel`, `useAddLabel`). The `Create "…"` row calls `onToggle(name)` and clears the search.
  - It no longer needs `keepOpen` (from xpo-2790cd), so the menu closes the same way it does for any other item.
- A new label exists only on the issue, uncolored, until someone colors it in the Labels view. The view already lists labels from issues as well as from config, so nothing else had to change.
- `borderlessBadges` was only used by the color step, so it is gone from the props and from the two NewIssueModal call sites.

### Server
- **`config.UpdateLabel` upserts.** Before, it only touched labels already under `labels:` in `config.yaml`. Coloring an issue-only label therefore wrote nothing, and the color was lost on restart.
  - `AddLabel` and `UpdateLabel` now share three helpers: `loadLabelsNode` (reads the file, creating the doc and the `labels:` mapping if missing), `upsertLabel`, and `writeConfigDoc`.
  - A missing config file is now created rather than returned as an error, matching `AddLabel`.
- **Built-in guard.** `s.isDefaultLabel(name)` is a case-insensitive match against `s.Config.DefaultLabels`. Every write path checks it:
  - `handleAddLabel` rejects a built-in name. `AddLabel` upserts, so without this check a POST could recolor a built-in label.
  - `handleUpdateLabel` rejects any change to a built-in label, color-only included. It also rejects renaming a custom label onto a built-in name, which would merge the two.
  - `handleDeleteLabel` rejects a built-in label.
  - All three return 400 with a message that names the label.

### Shared dialog frame: `ui/Dialog.tsx`
- `ConfirmDialog` was split in two:
  - **`DialogOverlay`**: the portal, the backdrop and the Escape shortcut. It takes `enabled`, so the edit dialog can turn off its own Escape while the delete confirmation is open on top. The keyboard registry runs only the most recently registered overlay binding, so this mainly avoids a dev-mode conflict warning.
  - **`DialogPanel`**: the card. It has the title, `px-5 pt-5 pb-4` body padding, and the `surface-2` footer bar. `footerLeft` defaults to the esc hint.
- `ConfirmDialog` / `ConfirmDialogPanel` are rebuilt on these with no visual change; their existing tests pass unchanged. This puts the edit dialog and the confirm dialogs on the same code, rather than restyling `Modal`.

### Labels view
- **Sections.** Pure helpers in `utils/labels.ts` do the sorting and filtering:
  - `isBuiltinLabel` and `partitionBuiltinLabels` split labels into **Built-in labels**, in `default_labels` order, and **Custom labels**, alphabetical.
  - Each section has a small uppercase heading and no count. `CountBadge` reads "N issues", which is wrong for a list of labels.
  - The search filters both sections, and a section with no matches is hidden.
- **Rows:** color dot, name, and "N issues" on the right, which links to the backlog filtered by the label. With 0 issues it is plain text.
  - **Custom rows** are buttons (click, Enter, Space) that open the edit dialog. The count link stops propagation, which also fixes the original bug where row buttons fell through to the row's navigate handler.
  - **Built-in rows** are inert apart from the count link.
  - The pencil and trash buttons, the inline edit form and the inline Yes/No confirmation are gone.
- The view picks the preview issues with `recentIssuesWithLabel(issues, name, 2)`.

### `EditLabelDialog`
- `EditLabelDialog` holds the state and wraps `EditLabelPanel` in a `DialogOverlay`. The panel has no portal, so it renders to static markup in tests.
- **Layout.** Name and Color come first, in a two-column grid with the field names on the left. The swatches sit in one row, spread across the field column with `justify-between`.
- **Preview.** Below the form is an inset well captioned "Preview". It shows up to two of the label's issues, the second dimmed, each ending in the real `LabelBadge`.
  - The badge's color is overridden through `LabelColorsContext`, so it previews the unsaved color.
  - With no issues, a placeholder row stands in.
- **Footer.** "Delete label" (ghost, danger) on the left, Cancel and Save on the right. Save needs a name and a picked color.
- **Uncolored labels.** They open with no swatch selected. The old code seeded the muted CSS variable as the color, and the server's hex check rejected it.
- **Delete** opens a `ConfirmDialog` that says how many issues lose the label.

### Palette
`LABEL_PRESET_COLORS` holds 12 colors: seven hues the built-ins don't use (pink, orange, yellow, lime, green, teal, gray), plus darker shades of five of them (plum, rust, olive, forest, petrol). `constants.test.ts` requires an RGB distance above 60 between every pair of swatches, and between each swatch and each built-in color (red, purple, indigo, blue).

## Decisions made in review
- **Sections, not a tag.** Built-in and custom labels get separate sections instead of a per-row "Built-in" tag.
- **Click to edit.** Clicking a row opens a dialog, with no row buttons. The issue count is the link to the backlog.
- **Dialog design.** It was picked on a temporary `#/tophat` page (since deleted) from three variants: a preview stage, a preview on issue rows, and a large hero badge. The pick was the preview on issue rows, with Delete label on the left of the footer. After trying it, the user asked for:
  - the form above the preview;
  - full-width swatches;
  - a clearly marked, inset preview.
- **Built-ins fully read-only.** First built-ins could be recolored but not renamed. Then they became fully read-only, so their meaning never drifts. Their colors can still be changed by editing `config.yaml` by hand.
- **Palette.** It went from 12 colors that included the built-in hues, to 10 that looked alike, to 7 distinct hues, to the final 12 with light and dark shades.
- **No explanatory note** on the Built-in section, and no counts in the section headings.

## Acceptance criteria
- [x] Clicking or pressing Enter on `Create "foo"` applies a new, uncolored label in one step, from both the context menu and the issue detail view. The Create row calls `onToggle` directly, without `keepOpen`, and the user tried it on the demo board.
- [x] Choosing an existing uncolored label toggles it directly. The color check in `handleSelect` is gone, and every row calls `onToggle`.
- [x] Coloring an uncolored label in the Labels view is still there after the server restarts. `UpdateLabel` upserts, covered by `TestUpdateLabel_UpsertsMissingLabel`, `_CreatesLabelsSection` and `_NoConfigFile`.
- [x] The Labels view lists built-in and custom labels in separate sections. `partitionBuiltinLabels` has tests in `labels.test.ts`, and the user reviewed the view on the demo board.
- [x] Clicking a custom label row opens the edit dialog, and clicking "k issues" opens the filtered backlog. The user reviewed both on the demo board.
- [x] Built-in labels can't be renamed, recolored or deleted, in the UI or through the API (400).
  - **API:** covered by `TestHandleUpdateLabel_RejectsBuiltinChanges` (rename, case-only rename, color-only; the config is untouched), `_RejectsRenameOntoBuiltin`, `TestHandleAddLabel_RejectsBuiltin` and `TestHandleDeleteLabel_RejectsBuiltin`.
  - **UI:** built-in rows don't open the dialog.
- [x] Palette colors are clearly different from each other and from the built-in colors. Both distance checks are in `constants.test.ts`.
- [x] The edit dialog matches layout B and shares its frame with `ConfirmDialog`.
  - `Dialog.test.tsx` covers `DialogPanel`, and `ConfirmDialog.test.tsx` passes unchanged.
  - `EditLabelDialog.test.tsx` (8 tests) covers the field order, the preview caption and placeholder, Save gating, and the swatch count.
- [x] Go tests: `UpdateLabel` upsert; the handlers reject every change to a built-in label. These are the tests named above.

`make test` passes: lint, the Go suite, and the frontend tests.
