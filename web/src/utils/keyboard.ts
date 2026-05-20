export function isEditableTarget(e: KeyboardEvent): boolean {
  return (
    e.target instanceof HTMLInputElement ||
    e.target instanceof HTMLTextAreaElement ||
    (e.target instanceof HTMLElement && e.target.isContentEditable)
  );
}
