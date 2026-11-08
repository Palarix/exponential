// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, setupDom } from "../../test-dom";
import { makeIssue } from "../../test-utils";
import { IssueRow } from "./MilestoneIssues";

beforeAll(setupDom);

const issue = makeIssue({ id: "xpo-ab1", title: "Courier map", description: "Shows couriers live." });

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
  vi.useRealTimers();
});

/** Rests the pointer on the element `pick` finds in the row; returns the open hover card, if any. */
function rest(pick: (row: HTMLElement) => Element) {
  vi.useFakeTimers();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, enabled: false } } });
  const mounted = mountDom(
    <QueryClientProvider client={client}>
      <KeyboardNavProvider>
        <TooltipPrimitive.Provider>
          <IssueRow issue={issue} all={[issue]} onIssueClick={() => {}} />
        </TooltipPrimitive.Provider>
      </KeyboardNavProvider>
    </QueryClientProvider>,
  );
  cleanup = mounted.cleanup;
  const row = mounted.host.querySelector<HTMLElement>("[data-backlog-row]")!;
  act(() => { pick(row).dispatchEvent(new MouseEvent("mouseover", { bubbles: true })); });
  act(() => { vi.advanceTimersByTime(500); });
  return [...document.body.querySelectorAll("div")].find((d) => d.textContent?.includes("Shows couriers live.")) ?? null;
}

describe("IssueRow hover card", () => {
  it("opens when the pointer rests on the title", () => {
    expect(rest((row) => [...row.querySelectorAll("span")].filter((s) => s.textContent === "Courier map").at(-1)!)).not.toBeNull();
  });

  it("does not open when the pointer rests on the rest of the row", () => {
    expect(rest((row) => row.querySelector('[title="Copy issue ID"]') ?? row)).toBeNull();
    cleanup?.();
    cleanup = undefined;
    expect(rest((row) => row)).toBeNull();
  });
});
