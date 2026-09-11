# Spec: Extract `SearchInput` component

## What

Extract the identical search-input pattern from Backlog and Dependencies into a shared controlled `SearchInput` component in the UI layer. The component owns the `/` focus shortcut at `control` priority and the `Escape` clear-and-blur behavior. Consumers place it in `TopBar.center` and pass a value/onChange pair.

## Why

Backlog and Dependencies duplicate the same input markup, icon, kbd hint, and Escape handler. Extracting it keeps them visually and behaviorally consistent and follows the same pattern established by `Tabs` in xpo-d47781.

## Acceptance Criteria

- [ ] `SearchInput` renders the same markup as the current inline inputs: `h-8 pl-8 pr-10` input, magnifying glass SVG, and `<kbd>` hint.
- [ ] The kbd hint shows `/` when the input is empty, `Esc` when populated.
- [ ] `Escape` clears the value and blurs the input.
- [ ] `/` focuses the input, registered at `control` priority through `useKeyboardShortcuts`. No document-level listener.
- [ ] `SearchInput` does not fire when disabled (consumer passes `keyboardNavigationEnabled`).
- [ ] Backlog and Dependencies render `SearchInput` in `TopBar.center`, replacing inline markup.
- [ ] Backlog's `/` handler is removed from its view-level keyboard registration.
- [ ] Dependencies' standalone `/` shortcut registration is removed.
- [ ] The `/` shortcut appears in the keyboard help under the active view's section (control→view folding from xpo-d47781).
- [ ] Tests, build, and lint pass.

## API

```tsx
interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  keyboardNavigationEnabled?: boolean;
}
```

`ref` is managed internally — consumers don't need access to the input element.

## Flow

1. Add `SearchInput` in `web/src/components/ui/SearchInput.tsx`.
2. Render the input with the magnifying glass SVG and kbd hint. Wire `Escape` in the input's `onKeyDown`.
3. Register `/` at `control` priority via `useKeyboardShortcuts` with `allowInEditable: false` (default). Conditionally enable via the `keyboardNavigationEnabled` prop.
4. Replace the Backlog inline search markup with `<SearchInput>`. Remove the `/` case from `handleKeyboard` and the `searchRef`.
5. Replace the Dependencies inline search markup with `<SearchInput>`. Remove its standalone `useKeyboardShortcuts` registration for `/`.
6. Export `SearchInput` from the UI barrel.
7. Add tests for the `/`-to-focus and Escape-to-clear-and-blur logic.

## Decisions

- **Component owns `/` registration:** follows the Tabs pattern — reusable controls register their own shortcuts at `control` priority. The help overlay folds them into the view group automatically.
- **`keyboardNavigationEnabled` prop:** same pattern as Tabs — consumers disable the shortcut when overlays are open.
- **No `ref` forwarding:** neither consumer needs imperative access to the input after extraction. The component manages its own ref internally.
- **Consumers keep their own `value`/`onChange`:** SearchInput is controlled. Backlog uses local state; Dependencies lifts state up. Neither pattern changes.

## Edge Cases

- **LOW — input already focused:** `/` while already in the input is handled by the browser (types a `/`). The keyboard registry's default editable-target guard prevents the shortcut from firing.
- **LOW — Escape with empty input:** blurs without calling onChange (value is already empty).

## Open Questions

None.