export function searchInputKbdHint(value: string): string {
  return value.trim() ? "Esc" : "/";
}
