# Spec: Move issue detail scrollbar to right edge with sticky sidebar

## Goal

Reposition the issue detail scrollbar from between content and sidebar to the right edge of the viewport, with the property sidebar sticky-positioned.

## Current layout

```
┌─────────────────────────┬──┬────────────┐
│  Main content           │▒▒│  Sidebar   │
│  (overflow-y-auto)      │▒▒│  (overflow) │
└─────────────────────────┴──┴────────────┘
                           ↑ scrollbar here
```

Two independent scroll containers. Scrollbar sits between content and sidebar.

## Desired layout

```
┌─────────────────────────────────────┬──┐
│  Main content        │  Sidebar     │▒▒│
│  (scrolls)           │  (sticky)    │▒▒│
└─────────────────────────────────────┴──┘
                                       ↑ scrollbar here
```

Single scroll container. Sidebar is sticky and scrollbar is at the right edge.

## Acceptance criteria

- [ ] Single scrollbar at the right edge of the issue detail view
- [ ] Sidebar stays visible (sticky) while scrolling main content
- [ ] Sidebar scrolls independently when its content exceeds viewport height
- [ ] Breadcrumb bar stays fixed at the top (unchanged)
- [ ] No layout regressions on the Details, Spec, or Walkthrough tabs
- [ ] Build passes