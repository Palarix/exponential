import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import { renderMarkup } from "../../render-markup";
import { queryKeys } from "../../api/query-keys";
import type { Release } from "../../api/types";
import { makeIssue } from "../../test-utils";
import ReleaseChip from "./ReleaseChip";

const rel = (number: number, o: Partial<Release> = {}): Release => ({
  id: `rel-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
});

function chip(issue = makeIssue({ release_id: "rel-2", release: "1.2" }), client = new QueryClient(), expanded = false) {
  return renderMarkup(<TooltipPrimitive.Provider><ReleaseChip issue={issue} expanded={expanded} onOpen={() => {}} /></TooltipPrimitive.Provider>, client);
}

describe("ReleaseChip", () => {
  it("shows the Package icon and the release name in a picker trigger", () => {
    const html = chip();
    expect(html).toContain("lucide-package");
    expect(html).toContain(">1.2<");
    expect(html).toContain('aria-label="Release 1.2, change release"');
    expect(html).toMatch(/<button[^>]*data-release-trigger[^>]*rounded-full|<button[^>]*rounded-full[^>]*data-release-trigger/);
    expect(html).toContain('aria-haspopup="menu"');
    expect(html).toContain('aria-expanded="false"');
  });

  it("marks the trigger expanded while its picker is open", () => {
    expect(chip(undefined, undefined, true)).toContain('aria-expanded="true"');
  });

  it("renders nothing without a release", () => {
    expect(chip(makeIssue())).toBe("");
  });

  it("marks a released release in its name", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.releases, {
      current: rel(1), next: rel(2), upcoming: [], past: [rel(9, { label: "1.0", state: "RELEASED" })],
    });
    const html = chip(makeIssue({ release_id: "rel-9", release: "1.0" }), client);
    expect(html).toContain('aria-label="Release 1.0 · released, change release"');
  });

  it("truncates a long label", () => {
    expect(chip(makeIssue({ release_id: "rel-2", release: "A very long release label indeed" }))).toMatch(/truncate/);
  });
});
