# Board view: keyboard help missing shortcuts

## What

The Board view has full keyboard support (navigation, `S`/`L`/`E` quick-set, `1`–`7` column shortcuts) but the `?` help overlay has no Board section at all. Users can't discover these shortcuts.

## Why

The issue description assumed `S`/`L`/`E` were missing from Board, but they already exist (Board.tsx lines 399–413). The column shortcuts (`1`–`7`) also exist (line 414). The only gap is documentation in the help overlay.

## How

Add a `BOARD` shortcut group to `KeyboardHelp.tsx` documenting all Board-specific shortcuts, and include it in `ALL_GROUPS`.

### Board shortcuts to document

| Key | Action |
|-----|--------|
| J / ↓ | Next card |
| K / ↑ | Previous card |
| → | Next column |
| ← | Previous column |
| Enter | Open issue |
| S | Set status |
| L | Set labels |
| E | Set estimate |
| 1–7 | Quick set status by column |
| . | Copy issue ID |

## Acceptance criteria

- [ ] `?` overlay shows a "Board" section with all shortcuts listed above
- [ ] Section appears between Backlog and Issue Detail in the group order
