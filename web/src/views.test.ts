import { describe, expect, it } from "vitest";
import { VIEWS, VIEW_BY_ID, VIEW_BY_GO_KEY, parseRoute, formatRoute, routeTitle, type Route } from "./views";

const route = (overrides: Partial<Route>): Route => ({
  view: "dashboard",
  issueId: null,
  cycleId: null,
  releaseNumber: null,
  depFocusId: null,
  chatId: null,
  chatHistory: null,
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
      "dashboard", "backlog", "board", "cycles", "releases", "timeline", "labels", "dependencies", "agent", "my-issues", "inbox", "drafts",
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
      l: "labels", c: "cycles", v: "releases", m: "my-issues", t: "timeline", r: "drafts", a: "agent",
    });
  });

  it("gates cycles on the feature and drafts on having issue drafts", () => {
    expect(Object.fromEntries(VIEWS.filter((v) => v.requires).map((v) => [v.id, v.requires]))).toEqual({
      cycles: "cycles", drafts: "drafts", agent: "agent",
    });
  });

  it("puts Agent in the personal section", () => {
    expect(VIEW_BY_ID.agent.section).toBe("personal");
  });
});

describe("parseRoute", () => {
  it("parses bare view routes", () => {
    expect(parseRoute("#/issues")).toEqual(route({ view: "backlog" }));
    expect(parseRoute("#/board")).toEqual(route({ view: "board" }));
    expect(parseRoute("#/my-issues")).toEqual(route({ view: "my-issues" }));
    expect(parseRoute("#timeline")).toEqual(route({ view: "timeline" }));
    expect(parseRoute("#/drafts")).toEqual(route({ view: "drafts" }));
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

  it("parses release links by number, ignoring anything else", () => {
    expect(parseRoute("#/releases")).toEqual(route({ view: "releases" }));
    expect(parseRoute("#/releases/7")).toEqual(route({ view: "releases", releaseNumber: 7 }));
    expect(parseRoute("#/releases/rel-3fa9c1")).toEqual(route({ view: "releases" }));
  });

  it("parses agent chat links", () => {
    expect(parseRoute("#/agent")).toEqual(route({ view: "agent" }));
    expect(parseRoute("#/agent/3f2a")).toEqual(route({ view: "agent", chatId: "3f2a" }));
  });

  it("parses the chat history page, with an issue filter", () => {
    expect(parseRoute("#/agent/history")).toEqual(route({ view: "agent", chatHistory: { issueId: null } }));
    expect(parseRoute("#/agent/history?issue=xpo-1")).toEqual(route({ view: "agent", chatHistory: { issueId: "xpo-1" } }));
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

  it("only includes the release number on the releases view", () => {
    expect(formatRoute(route({ view: "releases", releaseNumber: 7 }))).toBe("#/releases/7");
    expect(formatRoute(route({ view: "board", releaseNumber: 7 }))).toBe("#/board");
  });

  it("only includes the chat on the agent view", () => {
    expect(formatRoute(route({ view: "agent", chatId: "3f2a" }))).toBe("#/agent/3f2a");
    expect(formatRoute(route({ view: "board", chatId: "3f2a" }))).toBe("#/board");
  });

  it("formats the chat history page", () => {
    expect(formatRoute(route({ view: "agent", chatHistory: { issueId: null } }))).toBe("#/agent/history");
    expect(formatRoute(route({ view: "agent", chatHistory: { issueId: "xpo-1" } }))).toBe("#/agent/history?issue=xpo-1");
  });

  it("round-trips every view", () => {
    for (const v of VIEWS) {
      const r = route({ view: v.id });
      expect(parseRoute(formatRoute(r))).toEqual(r);
    }
  });
});

describe("routeTitle", () => {
  it("names the view, the open issue, or the chat history page", () => {
    expect(routeTitle(route({ view: "board" }))).toBe("Board");
    expect(routeTitle(route({ view: "board", issueId: "xpo-1" }))).toBe("xpo-1");
    expect(routeTitle(route({ view: "agent" }))).toBe("Agent");
    expect(routeTitle(route({ view: "agent", chatHistory: { issueId: null } }))).toBe("All Chats");
  });
});
