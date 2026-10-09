import { describe, expect, it } from "vitest";
import { VIEWS, VIEW_BY_ID, VIEW_BY_GO_KEY, parseRoute, formatRoute, type Route } from "./views";

const route = (overrides: Partial<Route>): Route => ({
  view: "dashboard",
  issueId: null,
  cycleId: null,
  depFocusId: null,
  ...overrides,
});

describe("VIEWS registry", () => {
  it("has unique ids, routes and go keys", () => {
    const unique = (xs: string[]) => new Set(xs).size === xs.length;
    expect(unique(VIEWS.map((v) => v.id))).toBe(true);
    expect(unique(VIEWS.map((v) => v.route))).toBe(true);
    expect(unique(VIEWS.map((v) => v.goKey))).toBe(true);
  });

  it("lists views in sidebar order", () => {
    expect(VIEWS.map((v) => v.id)).toEqual([
      "dashboard", "backlog", "board", "cycles", "timeline", "labels", "dependencies", "my-issues", "inbox",
    ]);
  });

  it("keeps distinct title and nav labels", () => {
    expect(VIEW_BY_ID.dashboard.title).toBe("Dashboard");
    expect(VIEW_BY_ID.dashboard.navLabel).toBe("Overview");
    expect(VIEW_BY_ID.inbox.title).toBe("Inbox");
    expect(VIEW_BY_ID.inbox.navLabel).toBe("Notifications");
    expect(VIEW_BY_ID.backlog.title).toBe("Issues");
  });

  it("maps go keys to views", () => {
    expect(VIEW_BY_GO_KEY).toEqual({
      o: "dashboard", i: "backlog", b: "board", n: "inbox", d: "dependencies",
      l: "labels", c: "cycles", m: "my-issues", t: "timeline",
    });
  });

  it("only cycles requires the cycles feature", () => {
    expect(VIEWS.filter((v) => v.requires).map((v) => v.id)).toEqual(["cycles"]);
  });
});

describe("parseRoute", () => {
  it("parses bare view routes", () => {
    expect(parseRoute("#/issues")).toEqual(route({ view: "backlog" }));
    expect(parseRoute("#/board")).toEqual(route({ view: "board" }));
    expect(parseRoute("#/my-issues")).toEqual(route({ view: "my-issues" }));
    expect(parseRoute("#timeline")).toEqual(route({ view: "timeline" }));
  });

  it("falls back to dashboard for empty or unknown routes", () => {
    expect(parseRoute("")).toEqual(route({ view: "dashboard" }));
    expect(parseRoute("#/")).toEqual(route({ view: "dashboard" }));
    expect(parseRoute("#/backlog")).toEqual(route({ view: "dashboard" }));
    expect(parseRoute("#/nope")).toEqual(route({ view: "dashboard" }));
  });

  it("parses issue, cycle and dependency deep links", () => {
    expect(parseRoute("#/issues/xpo-abc123")).toEqual(route({ view: "backlog", issueId: "xpo-abc123" }));
    expect(parseRoute("#/cycles/2026-10-05")).toEqual(route({ view: "cycles", cycleId: "2026-10-05" }));
    expect(parseRoute("#/dependencies/xpo-abc123")).toEqual(route({ view: "dependencies", depFocusId: "xpo-abc123" }));
  });
});

describe("formatRoute", () => {
  it("formats bare views with their route", () => {
    expect(formatRoute(route({ view: "backlog" }))).toBe("#/issues");
    expect(formatRoute(route({ view: "inbox" }))).toBe("#/inbox");
  });

  it("puts the issue ahead of any view", () => {
    expect(formatRoute(route({ view: "board", issueId: "xpo-1" }))).toBe("#/issues/xpo-1");
  });

  it("only includes cycle and dep focus on their own views", () => {
    expect(formatRoute(route({ view: "cycles", cycleId: "c1" }))).toBe("#/cycles/c1");
    expect(formatRoute(route({ view: "board", cycleId: "c1" }))).toBe("#/board");
    expect(formatRoute(route({ view: "dependencies", depFocusId: "xpo-2" }))).toBe("#/dependencies/xpo-2");
    expect(formatRoute(route({ view: "labels", depFocusId: "xpo-2" }))).toBe("#/labels");
  });

  it("round-trips every view", () => {
    for (const v of VIEWS) {
      const r = route({ view: v.id });
      expect(parseRoute(formatRoute(r))).toEqual(r);
    }
  });
});
