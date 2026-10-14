# Walkthrough: Web activity timeline: cleared fields and priority/parent/cycle changes are not described

## What was built

The issue activity timeline now describes every field change in words, including cleared values. Its rows read as muted sentences:

> Nicolas Bettenburg removed this issue from its cycle · 16m ago
> Nicolas Bettenburg set estimate to ▲ 3 Points and updated labels to (Bug) and assigned to Alice and set priority to ▮ High · 16m ago

The CLI timeline (`xpo history`) uses the same wording.

Review shaped most of the presentation, in four rounds: wording, then actor styling, then muted rows with hover, then baseline alignment. The decisions below record the final state and why.

## How the pieces fit

```
ActivityTimeline.tsx        builds entries; renders system rows and comment cards
  └─ describeEvent()        activity-describe.tsx: event → React text fragments
       └─ chips             activity-chips.tsx: StatusChip, EstimateChip, PriorityChip,
                            CycleName, ActivityValue (hover), ChipIcon (icon alignment)
internal/ui/timeline.go     describeUpdate(): the same phrases for the CLI
```

- **`activity-describe.tsx`:** `describeEvent` moved here from `ActivityTimeline.tsx`, so Vitest can render it on its own. It returns fragments joined with " and ".
- **`activity-chips.tsx`:** the small display components. They live in their own file because the `react-refresh/only-export-components` lint rule doesn't allow a file to export both components and a plain function.
- **`ActivityTimeline.tsx`:** the system row is one inline run of text (`data-activity="system"`). The actor name is wrapped in `ActivityValue`.

## Wording

| Payload | Text |
|---|---|
| `estimate: 0` | cleared the estimate |
| `assignee: ""` | removed the assignee |
| `labels: []` | removed all labels |
| `priority: 0` / `n` | cleared the priority / set priority to [icon] High |
| `parent_id: ""` / id | removed the parent issue / set parent to [issue chip] |
| `cycle_id: ""` / id | removed this issue from its cycle / added this issue to Cycle N |

The CLI uses the same phrases. It says "added this issue to cycle 2026-10-12" because it only has the cycle ID. `CycleName` on the web looks up the number with `useCycles()`, and falls back to the ID when the cycle is unknown or cycles are disabled.

## Decisions

- **Every phrase names its object.** Review found that "removed parent" and "removed from cycle" read as if the actor was removed. Phrases like "the parent issue" and "this issue" make "<actor> <phrase>" a sentence.
- **`[]` and `null` mean different things for labels.** Unchanged labels are stored as `null` (the `labels` field has no `omitempty`), and a cleared label list as `[]`. Both timelines say "removed all labels" only for an empty array. No UPDATE event in this repo's log has `labels: []`, so no old entry changes its text.
- **System rows are muted, and the actor and values highlight on hover.** Comment headers keep the actor in primary. In system rows, the actor and every value (status, points, priority, assignee, cycle, artifact filename) stay muted. `ActivityValue` and the chips add `cursor-pointer` and `hover:text-[var(--color-text-primary)]`. The hover hints that a part is interactive; hover cards come in xpo-fd7dc9.
  - The issue chip keeps its shared accent style, so it looks the same everywhere.
  - Agent-on-behalf names keep `cursor-help` on their inner span.
  - The artifact filename uses a plain mono span instead of `<Text>`. `Text` always sets a color (muted by default), and that would override the parent's hover color.
- **Rows are inline text, not flex.** The row used to be `flex flex-wrap gap-2`, so every part sat 8px apart. As one run of text, the parts get normal word spaces, and long rows wrap like a sentence.
- **Chips are inline spans; only their icons are centered.** An `inline-flex` chip with `align-middle` centers the whole chip on the middle of the lowercase letters, which drops its label below the line. Now the label is plain text on the baseline. `ChipIcon` (`inline-block align-middle`, svg as block) centers only the icon.
- **The estimate triangle gets a 1.5px lift.** Lucide's `Triangle` path runs from y≈3 to y=21 in its 24×24 viewBox, plus a 2-unit stroke. When centered, its base hung 1.5px below the baseline. `ChipIcon` takes an optional `lift`, and `EstimateChip` passes `relative -top-[1.5px]`. This is not a Lucide bug: the path is the same in 1.17.0, 1.57.0 and on `main`. The same offset affects the triangle across the app, so xpo-7e98d9 adds a shared `EstimateIcon` and will replace this local lift.

## Measurements (demo board, 13px row text)

| Check | Result |
|---|---|
| Text pieces on the long row (actor, "3 Points", "Alice", "High", action words) | identical boxes: top 613.4, bottom 628.9, so one baseline |
| "Bug" badge text (12px) | same bottom, 628.9 |
| Priority bars | base within 0.2px of the baseline |
| Estimate triangle, before the lift | ink 963.3–973.3; baseline 971.9; cap top 962.3 |
| Estimate triangle, after the lift | base 625.83 vs baseline 625.88; apex 615.83 vs cap top 616.29 |
| Hover | the hovered chip turns `#e4e4e7`, the rest stays `rgb(122,122,125)`; cursor `pointer` |

## Follow-ups

- xpo-dd3cae: event-type icons in the activity rail instead of the actor avatar
- xpo-fd7dc9: hover cards for people and values
- xpo-7e98d9: shared `EstimateIcon` with optical correction across the app

## Tests

- **`activity-describe.test.tsx`** (16 cases): every row of the wording table, the unchanged texts, " and " joining, the cycle name and its fallback, and the artifact filename's muted styling with hover.
- **`ActivityTimeline.test.tsx`:** renders the whole timeline with seeded queries and checks the system row:
  - nothing is primary except on hover
  - the actor and values carry the hover and pointer classes
  - no chip is an `inline-flex` box around its icon
  - the container has no flex or gap, and the text reads "Alice Smith set estimate to 3 Points and set priority to High"
- **`internal/ui/timeline_test.go`:** the cleared-field phrases, and `[]` versus `null` for labels, including payloads decoded from JSON maps.

## Acceptance criteria

- [x] `estimate: 0` renders "cleared the estimate". Evidence: `activity-describe.test.tsx` "describes a cleared estimate".
- [x] `assignee: ""` renders "removed the assignee", including when it is the only field. Evidence: "describes an unassign on its own".
- [x] `labels: []` renders "removed all labels" on web and CLI. Evidence: "describes removing all labels" and `TestDescribeEvent_UpdateClearedLabels`.
- [x] `priority: 0` renders "cleared the priority"; `priority: 2` renders "set priority to" with the icon and "High". Evidence: the two priority cases.
- [x] `parent_id: ""` renders "removed the parent issue"; an ID renders "set parent to" with an issue chip. Evidence: the two parent cases (the chip links to `#/issues/xpo-a1b2c3`).
- [x] `cycle_id: ""` renders "removed this issue from its cycle"; a known cycle renders "added this issue to Cycle N"; an unknown cycle renders its ID. Evidence: the three cycle cases.
- [x] The CLI timeline uses the same wording. Evidence: `TestDescribeEvent_UpdateClearedScalarFields`, and `xpo history` on the demo project.
- [x] The existing status, estimate, title, description, labels and assignee texts are unchanged. Evidence: the "keeps …" cases.
- [x] In system rows, the actor and values are muted and turn primary on hover with a pointer cursor. Evidence: `ActivityTimeline.test.tsx`, plus the hover check in the browser.
- [x] Chip labels and label badge text share the baseline of the row text. Evidence: the measurements table.
- [x] The system row's text has no flex gap, parts are separated by single spaces, and it wraps as text. Evidence: "flows as text without a flex gap", plus a screenshot of the wrapped long row.
- [x] Vitest tests cover each table row, the muted styling with hover, and the row spacing. Evidence: the two test files above.
- [x] `make test` passes. Evidence: exit 0, 899 frontend tests plus the Go suite.
