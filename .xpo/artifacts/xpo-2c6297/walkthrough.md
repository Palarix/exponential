# Merge view content width

## What changed

Walkthrough and conversation tabs in the merge view now constrain their content width and center it, instead of stretching edge-to-edge.

## Changes in `MergeView.tsx`

- **Walkthrough tab**: Added `mx-auto` to the existing prose container and changed `max-w-3xl` to `max-w-[52rem]` (832px)
- **Conversation tab**: Wrapped the comments list and input in a new `max-w-[52rem] mx-auto` container

The diff and commits tabs are unchanged — tabular/code content benefits from full width.

## Width choice

Used `max-w-[52rem]` (832px) to match the effective content width of the issue detail view's main column (~820px measured, the difference being padding).
