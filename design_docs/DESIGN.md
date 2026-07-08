# Exponential Design System

Dark theme design system for the Exponential web application.

## Color Tokens

### Backgrounds
| Token | Value | Usage |
|-------|-------|-------|
| `--color-bg-primary` | `#0a0a0a` | Main content area |
| `--color-bg-secondary` | `#0f0f0f` | Tab bars, section headers |
| `--color-bg-tertiary` | `#141414` | Input fields, nested surfaces |
| `--color-bg-elevated` | `#1a1a1a` | Cards, popovers, modals |
| `--color-bg-hover` | `#1e1e1e` | Hover state for rows and buttons |
| `--color-bg-sidebar` | `#0d0d0d` | Sidebar background |

### Borders
| Token | Value |
|-------|-------|
| `--color-border-subtle` | `rgba(255,255,255,0.06)` |
| `--color-border-default` | `rgba(255,255,255,0.1)` |
| `--color-border-focus` | `rgba(94,106,210,0.5)` |

### Text
| Token | Value | Usage |
|-------|-------|-------|
| `--color-text-primary` | `#f5f5f5` | Titles, active nav, primary content |
| `--color-text-secondary` | `#a0a0a0` | Body text, inactive nav |
| `--color-text-muted` | `#555555` | Timestamps, IDs, placeholder text |

### Accent
| Token | Value |
|-------|-------|
| `--color-accent-primary` | `#5e6ad2` |
| `--color-accent-primary-hover` | `#6c75db` |

### Status Colors
| Status | Token | Value | Icon |
|--------|-------|-------|------|
| Backlog | `--color-status-backlog` | `#777777` | Dashed circle |
| Planned | `--color-status-planned` | `#a0a0a0` | Empty circle outline |
| Doing | `--color-status-doing` | `#f2c94c` | Half-filled circle |
| Blocked | `--color-status-blocked` | `#eb5757` | Filled circle + X |
| Done | `--color-status-done` | `#4cb782` | Filled circle + checkmark |

### Label Colors
| Label | Value |
|-------|-------|
| Bug | `#eb5757` |
| Feature | `#b36cd9` |
| Epic | `#5e6ad2` |
| Improvement | `#4da6e8` |

## Typography

- Font family: Inter (sans), JetBrains Mono (mono)
- List row text: 13px
- Body text: 14px (text-sm)
- Section headers: 13px uppercase, tracking-wider, muted color
- View titles: 16px medium
- Issue detail title: 20-24px bold
- Issue IDs: 11-12px monospace, muted color

## Spacing & Layout

- Sidebar width: 240px (w-60)
- List row height: 36px
- Nav item height: 32px
- Compact padding throughout
- Border radius: 4px (sm), 6px (md), 8px (lg), 10px (xl)

## Interactions

- Hover: subtle background shift to `--color-bg-hover`
- Transitions: 100-150ms, no spring/bounce
- No glass morphism, no glow effects, no gradients
- Focus: 2px ring using `--color-border-focus`

## Component Catalog

### Button
Flat styling, no shadows or glow. Variants: primary (indigo), secondary (elevated bg + border), ghost (transparent), danger (red).

### Card
Flat background + subtle border. No glass or glow variants. Tighter corners (--radius-md).

### Badge / LabelBadge
Small colored dot (6px) + text label. Not colored background pills. Title case text.

### StatusIcon
SVG circle-based icons matching the status colors table above.

### Modal / Panel
Flat dark background, no backdrop blur. Issue detail uses a slide-over panel instead of centered modal.

### Popover
Positioned absolutely, dark elevated background, subtle border + shadow. Used for status/label/estimate selection in issue detail.
