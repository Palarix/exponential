# Spec: Web activity timeline: cleared fields and priority/parent/cycle changes are not described

## What

The web activity timeline (`web/src/components/IssueDetail/ActivityTimeline.tsx`) will describe every UPDATE field the CLI timeline describes, and describe cleared values in words. Each phrase names its object ("the parent issue", "this issue"), so "<actor> <phrase>" reads as a sentence. The web and CLI (`internal/ui/timeline.go`) use the same wording. System rows stay muted throughout, with a hover highlight on the actor and values, and get normal word spacing.

## Why

Today the web timeline:

- shows "set estimate to ▲ 0 Points" for a cleared estimate;
- skips a cleared assignee (`""` is falsy), so an event that only unassigns has no text;
- shows "updated labels to" with nothing after it for `labels: []`;
- never describes `priority`, `parent_id` or `cycle_id`.

Since xpo-328824 and xpo-bbfcf9, agents clear these fields with `null`, so cleared values now show up in the event log more often.

## How

1. Move `describeEvent` from `ActivityTimeline.tsx` to a new `activity-describe.tsx` in the same folder, so tests can render it. `ActivityTimeline.tsx` imports it. Move `StatusChip` and `EstimateChip` to `activity-chips.tsx`, next to the new `PriorityChip` and `CycleName`. The `react-refresh/only-export-components` lint rule doesn't allow a file to export both components and a plain function.
2. Change the UPDATE fragments:

   | Payload | Text |
   |---|---|
   | `estimate: 0` | cleared the estimate |
   | `estimate: n` | set estimate to ▲ n Points (unchanged) |
   | `assignee: ""` | removed the assignee |
   | `assignee: "…"` | assigned to **Name** (unchanged) |
   | `labels: []` | removed all labels |
   | `labels: [...]` | updated labels to [badges] (unchanged) |
   | `priority: 0` | cleared the priority |
   | `priority: n` | set priority to [PriorityIcon] **High** (labels from `PRIORITY_LABELS`) |
   | `parent_id: ""` | removed the parent issue |
   | `parent_id: "xpo-…"` | set parent to [IssueRef chip] |
   | `cycle_id: ""` | removed this issue from its cycle |
   | `cycle_id: "2026-10-12"` | added this issue to **Cycle 12** |

3. Add a `CycleName` component in `activity-chips.tsx`. It looks up the cycle with `useCycles()` and shows `Cycle {number}`. If the cycle isn't in the list (cycles disabled or still loading), it shows the raw cycle ID.
4. In `internal/ui/timeline.go`, use the same wording as the table. Add "removed all labels" for `labels: []`; today the CLI shows nothing for it. This also replaces the CLI's earlier "unassigned", "cleared estimate", "cleared priority", "removed parent", "removed from cycle" and "added to cycle X".
5. **System rows are muted, with a hover highlight.** Only comment headers show the actor in primary. In a system row, the actor name and every value (status name, points, priority name, assignee name, cycle name, artifact filename) use the row's muted color. Today the values use primary, medium weight. Each of these parts turns primary on hover (`hover:text-[var(--color-text-primary)]`) and shows a pointer cursor, which hints that it is interactive. Names with an agent on their behalf keep their `cursor-help` underline. Icons in chips keep their own colors. The issue chip (`IssueRef`) keeps its shared styling, so it looks the same as everywhere else. Hover cards for people and values are a separate feature.
6. **System rows flow as text.** The system row's text container is `flex flex-wrap gap-2`. That makes the name, each text fragment, each chip, each "and" and the "·" separate flex items with 8px between them, instead of a word space. Render the row text as one inline run of text with normal spaces between parts. Chips (`StatusChip`, `EstimateChip`, `PriorityChip`) are plain inline spans, not flex boxes, so their label sits on the line's text baseline. Only the icon gets a vertical nudge to center it on the text. The estimate triangle (lucide `Triangle`, 24×24 viewBox, path from y≈3 to y=21 plus a 2-unit stroke) needs 1.5px more lift: when centered, its base hangs 1.5px below the text baseline. With the lift, its base sits on the baseline and its apex at the digits' cap height. An `inline-flex` chip with `align-middle` lines up the chip's middle with the lowercase letters' middle, which drops the label below the surrounding words. Label badges (`LabelBadge`) keep their pill shape and align the same way.

## Not changed

- Link (`dependencies`) changes stay undescribed in both timelines. They come from a separate code path with their own events.
- The fragment order and the " and " joining stay as they are.

## Acceptance criteria

- [ ] `estimate: 0` renders "cleared the estimate"
- [ ] `assignee: ""` renders "removed the assignee", including when it is the only field
- [ ] `labels: []` renders "removed all labels" on web and CLI
- [ ] `priority: 0` renders "cleared the priority"; `priority: 2` renders "set priority to" with the priority icon and "High"
- [ ] `parent_id: ""` renders "removed the parent issue"; `parent_id: "xpo-…"` renders "set parent to" with an issue chip
- [ ] `cycle_id: ""` renders "removed this issue from its cycle"; a known cycle renders "added this issue to Cycle N"; an unknown cycle renders its ID
- [ ] The CLI timeline uses the same wording for these fields
- [ ] The existing status, estimate, title, description, labels and assignee texts are unchanged
- [ ] In system rows, the actor name and values are muted, not primary, and turn primary on hover with a pointer cursor
- [ ] Chip labels and label badge text share the baseline of the surrounding row text
- [ ] The system row's text has no flex gap; parts are separated by single spaces, and the row wraps as text
- [ ] Vitest tests cover each row in the table, the muted styling with hover highlight, and the row spacing
- [ ] `make test` passes
