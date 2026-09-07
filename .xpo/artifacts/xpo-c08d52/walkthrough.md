# Walkthrough: Fix all frontend eslint errors

## What was built

Systematic fix of 63 pre-existing eslint errors across the `web/src/` frontend codebase, plus a build system change to gate future commits on lint.

## Approach

Each fix was applied individually, verified with `bun run lint`, and tophatted in the browser before moving to the next. No `eslint-disable` comments were used — every error was fixed at the source.

## Key patterns

### Ref writes during render → useEffect

The React Compiler forbids writing to `ref.current` in the component body. The codebase had a common pattern of `const ref = useRef(callback); ref.current = callback;` to keep a stable ref to the latest callback. All instances were moved into `useEffect`:

```tsx
// Before
const onChangeRef = useRef(onChange);
onChangeRef.current = onChange;

// After
const onChangeRef = useRef(onChange);
useEffect(() => { onChangeRef.current = onChange; }, [onChange]);
```

Affected: `useSSE.ts`, `ContextMenu.tsx`, `MarkdownEditor.tsx`, `MyIssues.tsx`, `Backlog.tsx`

### setState in effects → direct DOM

Popover positioning used `useState` + `useLayoutEffect` to measure the DOM and set position state. The React Compiler flagged this as cascading renders. The fix: write directly to `el.style` properties in the layout effect, and use a static initial style (`visibility: hidden`) in JSX.

Affected: `Popover.tsx`, `ContextMenu.tsx`, `FilterMenu.tsx`

### Non-component exports → separate files

`react-refresh/only-export-components` requires files with a default component export to only export components. Contexts, hooks, and constants were extracted:

- `Badge.tsx` → `BadgeContexts.ts` (3 React contexts)
- `Toast.tsx` → `ToastContext.ts` (context + `useToast` hook)
- `DndComponents.tsx` → `backlogCollision.ts` (collision detection function)
- `Section.tsx` → `sectionIcons.ts` (SVG path constants)

### Static constants → module scope

Objects defined inside component bodies that never change were moved to module scope to avoid being flagged as missing dependencies:

- `App.tsx`: `GO_TARGETS`
- `Dashboard.tsx`: `STATUS_ORDER_MAP`, `ALL_STATUSES`

### Inbox derived state

The inbox auto-select effect (`if (!selectedId) setSelectedId(first)`) was replaced with derived values that compute the effective selection inline, eliminating a synchronous setState in an effect and a flash of no-selection on first render.

### MarkdownEditor typing

Replaced `ed.storage as Record<string, any>` (3 occurrences) with a typed `MarkdownStorage` interface for the tiptap-markdown extension's storage shape.

## Build system

- `make lint` now runs `go vet` + `cd web && bun run lint`
- `make test` depends on `lint` — both must pass before declaring implementation done
- `CLAUDE.md` documents the pre-completion gate