# Spec: Improve prose-exponential markdown rendering

## Goal

Make the rendered markdown in the web UI visually polished — proper table rendering, clear heading hierarchy, sensible list indentation, and consistent vertical rhythm.

## Requirements

### Tables
- All cell borders (not just row separators)
- Cell padding for comfortable reading
- Header row visually distinct: muted text color, semibold, subtle background

### Headings
- Graduated top margins by heading level (h1 largest, h4 smallest) to establish visual hierarchy
- Tighter line-height (1.3) instead of inheriting body's 1.75
- Slightly larger font sizes for h1/h2 to differentiate from body text

### Lists
- First-level lists should have minimal indent — bullets/numbers within the text column, not pushed out
- Small gap between marker and text content
- Nested lists keep full indent for visual nesting
- Fix double-margin stacking when list items contain `<p>` elements

### Vertical rhythm
- Slightly more breathing room between paragraphs, code blocks, blockquotes, and HRs
- Inline code slightly smaller (0.875em)

### Editor parity
- Tiptap editor table styles must match read-only rendering
- All changes to `.prose-exponential` apply to both contexts

## Acceptance criteria

- [ ] Tables render with full grid borders, cell padding, and distinct header row
- [ ] Headings have graduated top margins (h1 > h2 > h3 > h4)
- [ ] First-level lists are nearly flush with body text
- [ ] Bullets/numbers have a small gap before text content
- [ ] No layout shifts between editor and preview for any element type
- [ ] Build passes (`make build`)