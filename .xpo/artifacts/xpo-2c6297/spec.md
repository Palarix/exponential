# Merge view content width

## What

Walkthrough and conversation tabs stretch too wide at large viewports.

## How

- **Walkthrough** (`MergeView.tsx:269`): already has `max-w-3xl`, just add `mx-auto` to center it
- **Conversation** (`MergeView.tsx:586-621`): wrap the content in a `max-w-3xl mx-auto` container

## AC

- Both tabs constrain prose width to `max-w-3xl` (768px) and center horizontally
