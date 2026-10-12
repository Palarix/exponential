export type IssueIdPart = string | { id: string };

/**
 * Matches issue IDs with the project prefix, e.g. `xpo-a1b2c3`. The API sends
 * the prefix without the hyphen (`xpo`); a trailing hyphen is accepted too.
 */
export function issueIdPattern(prefix: string): RegExp {
  const escaped = (prefix.replace(/-$/, "") + "-").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\b(${escaped}[a-f0-9]{6})\\b`, "g");
}

/** Splits text into plain parts and issue ID parts, in order. */
export function splitIssueIds(text: string, prefix: string): IssueIdPart[] {
  if (!prefix) return [text];
  const parts: IssueIdPart[] = [];
  let last = 0;
  for (const m of text.matchAll(issueIdPattern(prefix))) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    parts.push({ id: m[1] });
    last = m.index + m[0].length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

const ISSUES_HASH = "#/issues/";

export function issueHref(id: string): string {
  return ISSUES_HASH + id;
}

/** The issue ID an `#/issues/<id>` href points to, or null for any other href. */
export function issueIdFromHref(href: string | undefined): string | null {
  if (!href?.startsWith(ISSUES_HASH)) return null;
  const id = href.slice(ISSUES_HASH.length);
  return id && !id.includes("/") ? id : null;
}
