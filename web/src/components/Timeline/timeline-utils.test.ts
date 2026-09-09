import { describe, it, expect, vi, afterEach } from "vitest";
import type { TimelineEntry } from "../../api/types";
import {
  categorizeEntry,
  groupByDay,
  dayLabel,
  daySummary,
  entryActor,
  extractContributors,
} from "./timeline-utils";

function makeEntry(overrides: Partial<TimelineEntry> = {}): TimelineEntry {
  return {
    kind: "issue_event",
    timestamp: "2026-03-15T10:00:00Z",
    issue_id: "xpo-abc123",
    ...overrides,
  };
}

describe("categorizeEntry", () => {
  it("categorizes commits", () => {
    expect(categorizeEntry(makeEntry({ kind: "commit" }))).toBe("commits");
  });

  it("categorizes comments", () => {
    expect(categorizeEntry(makeEntry({ event_type: "COMMENT" }))).toBe("comments");
  });

  it("categorizes merges", () => {
    expect(categorizeEntry(makeEntry({ event_type: "MERGE" }))).toBe("merges");
  });

  it("categorizes artifacts", () => {
    expect(categorizeEntry(makeEntry({ event_type: "ARTIFACT" }))).toBe("artifacts");
  });

  it("categorizes terminal status updates as closed", () => {
    expect(categorizeEntry(makeEntry({ event_type: "UPDATE", payload: { status: "DONE" } }))).toBe("closed");
    expect(categorizeEntry(makeEntry({ event_type: "UPDATE", payload: { status: "CANCELED" } }))).toBe("closed");
    expect(categorizeEntry(makeEntry({ event_type: "UPDATE", payload: { status: "DUPLICATE" } }))).toBe("closed");
  });

  it("categorizes non-terminal updates as issues", () => {
    expect(categorizeEntry(makeEntry({ event_type: "UPDATE", payload: { status: "DOING" } }))).toBe("issues");
    expect(categorizeEntry(makeEntry({ event_type: "UPDATE", payload: { title: "changed" } }))).toBe("issues");
  });

  it("defaults to issues for unknown event types", () => {
    expect(categorizeEntry(makeEntry({ event_type: "CREATE" }))).toBe("issues");
    expect(categorizeEntry(makeEntry({}))).toBe("issues");
  });
});

describe("groupByDay", () => {
  it("groups entries by date prefix", () => {
    const entries = [
      makeEntry({ timestamp: "2026-03-15T10:00:00Z" }),
      makeEntry({ timestamp: "2026-03-15T14:00:00Z" }),
      makeEntry({ timestamp: "2026-03-16T09:00:00Z" }),
    ];
    const groups = groupByDay(entries);
    expect(groups.size).toBe(2);
    expect(groups.get("2026-03-15")).toHaveLength(2);
    expect(groups.get("2026-03-16")).toHaveLength(1);
  });

  it("returns empty map for empty input", () => {
    expect(groupByDay([]).size).toBe(0);
  });
});

describe("dayLabel", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns 'Today' for today's date", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-15T12:00:00Z"));
    expect(dayLabel("2026-03-15")).toBe("Today");
  });

  it("returns 'Yesterday' for yesterday's date", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-15T12:00:00Z"));
    expect(dayLabel("2026-03-14")).toBe("Yesterday");
  });

  it("returns formatted date for older dates", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-15T12:00:00Z"));
    const label = dayLabel("2026-03-10");
    expect(label).not.toBe("Today");
    expect(label).not.toBe("Yesterday");
    expect(typeof label).toBe("string");
  });
});

describe("daySummary", () => {
  it("counts events and commits separately", () => {
    const entries = [
      makeEntry({ kind: "issue_event" }),
      makeEntry({ kind: "issue_event" }),
      makeEntry({ kind: "commit" }),
    ];
    expect(daySummary(entries)).toBe("2 events, 1 commit");
  });

  it("handles only commits", () => {
    const entries = [
      makeEntry({ kind: "commit" }),
      makeEntry({ kind: "commit" }),
    ];
    expect(daySummary(entries)).toBe("2 commits");
  });

  it("handles only events", () => {
    const entries = [makeEntry({ kind: "issue_event" })];
    expect(daySummary(entries)).toBe("1 event");
  });

  it("returns empty string for no entries", () => {
    expect(daySummary([])).toBe("");
  });
});

describe("entryActor", () => {
  it("returns author name for commits", () => {
    expect(entryActor(makeEntry({ kind: "commit", author: "Alice <alice@example.com>" }))).toBe("Alice");
  });

  it("returns creator name for issue events", () => {
    expect(entryActor(makeEntry({ created_by: "Bob <bob@example.com>" }))).toBe("Bob");
  });

  it("handles on_behalf_of by returning the principal", () => {
    const entry = makeEntry({ created_by: "Agent <agent@bot.local>", on_behalf_of: "Carol <carol@example.com>" });
    expect(entryActor(entry)).toBe("Agent");
  });

  it("returns empty string for missing author/creator", () => {
    expect(entryActor(makeEntry({ kind: "commit" }))).toBe("");
  });
});

describe("extractContributors", () => {
  it("collects unique actor names sorted", () => {
    const entries = [
      makeEntry({ created_by: "Bob <bob@example.com>" }),
      makeEntry({ kind: "commit", author: "Alice <alice@example.com>" }),
      makeEntry({ created_by: "Bob <bob@example.com>" }),
    ];
    expect(extractContributors(entries)).toEqual(["Alice", "Bob"]);
  });

  it("returns empty for no entries", () => {
    expect(extractContributors([])).toEqual([]);
  });
});
