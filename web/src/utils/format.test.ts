import { describe, it, expect, vi, afterEach } from "vitest";
import {
  formatRelativeTime,
  shortName,
  extractEmail,
  displayActor,
  formatShortDate,
  formatTriage,
  stripMarkdown,
  linkifyIssueIds,
  fuzzyMatch,
  fuzzyScore,
  formatDuration,
} from "./format";

describe("shortName", () => {
  it("extracts name before email", () => {
    expect(shortName("John Doe <john@example.com>")).toBe("John Doe");
  });

  it("returns plain name as-is", () => {
    expect(shortName("Alice")).toBe("Alice");
  });

  it("handles empty string", () => {
    expect(shortName("")).toBe("");
  });

  it("handles name with multiple angle brackets", () => {
    expect(shortName("A <B> <C>")).toBe("A");
  });
});

describe("extractEmail", () => {
  it("extracts email from identity string", () => {
    expect(extractEmail("John Doe <john@example.com>")).toBe("john@example.com");
  });

  it("lowercases and trims the email", () => {
    expect(extractEmail("Jane < JANE@EXAMPLE.COM >")).toBe("jane@example.com");
  });

  it("returns empty string for plain name without angle brackets", () => {
    expect(extractEmail("Alice")).toBe("");
  });

  it("returns empty string for empty input", () => {
    expect(extractEmail("")).toBe("");
  });
});

describe("displayActor", () => {
  it("returns principal only without onBehalfOf", () => {
    expect(displayActor("John <j@x.com>")).toEqual({ principal: "John" });
  });

  it("returns principal and via with onBehalfOf", () => {
    expect(displayActor("Bot <b@x.com>", "Alice <a@x.com>")).toEqual({
      principal: "Bot",
      via: "Alice",
    });
  });
});

describe("formatRelativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  const BASE = new Date("2026-06-15T12:00:00Z").getTime();

  it("returns 'just now' for < 1 minute", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(formatRelativeTime(new Date(BASE - 30_000).toISOString())).toBe(
      "just now",
    );
  });

  it("returns minutes ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(formatRelativeTime(new Date(BASE - 5 * 60_000).toISOString())).toBe(
      "5m ago",
    );
  });

  it("returns hours ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(
      formatRelativeTime(new Date(BASE - 3 * 3_600_000).toISOString()),
    ).toBe("3h ago");
  });

  it("returns days ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(
      formatRelativeTime(new Date(BASE - 7 * 86_400_000).toISOString()),
    ).toBe("7d ago");
  });

  it("returns months ago for >= 30 days", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(
      formatRelativeTime(new Date(BASE - 60 * 86_400_000).toISOString()),
    ).toBe("2mo ago");
  });

  it("boundary: exactly 1 minute", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(formatRelativeTime(new Date(BASE - 60_000).toISOString())).toBe(
      "1m ago",
    );
  });

  it("boundary: exactly 60 minutes -> 1h", () => {
    vi.useFakeTimers();
    vi.setSystemTime(BASE);
    expect(
      formatRelativeTime(new Date(BASE - 60 * 60_000).toISOString()),
    ).toBe("1h ago");
  });
});

describe("formatShortDate", () => {
  it("formats ISO date string", () => {
    expect(formatShortDate("2026-03-15T10:00:00Z")).toBe("Mar 15");
  });

  it("formats January", () => {
    expect(formatShortDate("2026-01-01T00:00:00Z")).toBe("Jan 1");
  });

  it("formats December", () => {
    expect(formatShortDate("2026-12-25T00:00:00Z")).toBe("Dec 25");
  });

  it("strips leading zeros from day", () => {
    expect(formatShortDate("2026-04-05T00:00:00Z")).toBe("Apr 5");
  });
});

describe("formatTriage", () => {
  it("returns dash for zero", () => {
    expect(formatTriage(0)).toBe("—");
  });

  it("returns dash for negative", () => {
    expect(formatTriage(-10)).toBe("—");
  });

  it("returns minutes for < 60", () => {
    expect(formatTriage(45)).toBe("45m");
  });

  it("returns rounded hours for < 24h", () => {
    expect(formatTriage(90)).toBe("2h");
  });

  it("returns 1-decimal days for < 10d", () => {
    expect(formatTriage(60 * 24 * 3.5)).toBe("3.5d");
  });

  it("returns rounded days for >= 10d", () => {
    expect(formatTriage(60 * 24 * 15)).toBe("15d");
  });
});

describe("stripMarkdown", () => {
  it("removes headings", () => {
    expect(stripMarkdown("## Title")).toBe("Title");
  });

  it("removes bold", () => {
    expect(stripMarkdown("**bold**")).toBe("bold");
  });

  it("removes italic", () => {
    expect(stripMarkdown("*italic*")).toBe("italic");
  });

  it("removes inline code", () => {
    expect(stripMarkdown("`code`")).toBe("code");
  });

  it("removes code blocks", () => {
    expect(stripMarkdown("```\ncode\n```")).toBe("");
  });

  it("removes images", () => {
    expect(stripMarkdown("![alt](url)")).toBe("");
  });

  it("keeps link text", () => {
    expect(stripMarkdown("[text](url)")).toBe("text");
  });

  it("removes strikethrough", () => {
    expect(stripMarkdown("~~struck~~")).toBe("struck");
  });

  it("removes blockquotes", () => {
    expect(stripMarkdown("> quoted")).toBe("quoted");
  });

  it("removes list markers", () => {
    expect(stripMarkdown("- item\n* item2\n1. numbered")).toBe(
      "item item2 numbered",
    );
  });

  it("collapses whitespace", () => {
    expect(stripMarkdown("a   b\n\nc")).toBe("a b c");
  });
});

describe("linkifyIssueIds", () => {
  it("wraps issue IDs in markdown links", () => {
    expect(linkifyIssueIds("see xpo-a1b2c3 for details", "xpo-")).toBe(
      "see [xpo-a1b2c3](#/issues/xpo-a1b2c3) for details",
    );
  });

  it("handles multiple IDs", () => {
    const result = linkifyIssueIds("xpo-111111 and xpo-222222", "xpo-");
    expect(result).toBe(
      "[xpo-111111](#/issues/xpo-111111) and [xpo-222222](#/issues/xpo-222222)",
    );
  });

  it("does not match partial IDs (fewer than 6 hex chars)", () => {
    expect(linkifyIssueIds("xpo-abc", "xpo-")).toBe("xpo-abc");
  });

  it("does not match non-hex chars", () => {
    expect(linkifyIssueIds("xpo-ghijkl", "xpo-")).toBe("xpo-ghijkl");
  });

  it("escapes regex special chars in prefix", () => {
    expect(linkifyIssueIds("x.o-aabbcc done", "x.o-")).toBe(
      "[x.o-aabbcc](#/issues/x.o-aabbcc) done",
    );
  });
});

describe("formatDuration", () => {
  it("returns minutes for < 1h", () => {
    expect(formatDuration(0.5)).toBe("30m");
  });

  it("returns hours for < 24h", () => {
    expect(formatDuration(2.5)).toBe("2.5h");
  });

  it("returns days for < 7d", () => {
    expect(formatDuration(48)).toBe("2d");
  });

  it("returns weeks for < 5w", () => {
    expect(formatDuration(24 * 14)).toBe("2w");
  });

  it("returns months for >= 5w", () => {
    expect(formatDuration(24 * 60)).toBe("2mo");
  });

  it("rounds to 1 decimal", () => {
    expect(formatDuration(1.23)).toBe("1.2h");
  });
});

describe("fuzzyMatch", () => {
  it("matches exact strings", () => {
    expect(fuzzyMatch("hello", "hello")).toBe(true);
  });

  it("matches subsequences", () => {
    expect(fuzzyMatch("hello world", "hlo")).toBe(true);
    expect(fuzzyMatch("backlog collision", "bgc")).toBe(true);
  });

  it("is case-insensitive", () => {
    expect(fuzzyMatch("Hello World", "hw")).toBe(true);
    expect(fuzzyMatch("hello", "HELLO")).toBe(true);
  });

  it("rejects non-matching queries", () => {
    expect(fuzzyMatch("hello", "xyz")).toBe(false);
    expect(fuzzyMatch("abc", "abdc")).toBe(false);
  });

  it("matches empty query against anything", () => {
    expect(fuzzyMatch("anything", "")).toBe(true);
    expect(fuzzyMatch("", "")).toBe(true);
  });

  it("rejects non-empty query against empty text", () => {
    expect(fuzzyMatch("", "a")).toBe(false);
  });
});

describe("fuzzyScore", () => {
  it("scores exact match highest", () => {
    expect(fuzzyScore("hello", "hello")).toBe(10000);
  });

  it("scores substring match above fuzzy", () => {
    const substr = fuzzyScore("UI Polish", "poli");
    const fuzzy = fuzzyScore("Ghost parent row duplicates popover", "poli");
    expect(substr).toBeGreaterThan(fuzzy);
  });

  it("scores early substring higher", () => {
    const early = fuzzyScore("Polish things", "poli");
    const late = fuzzyScore("UI Polish", "poli");
    expect(early).toBeGreaterThan(late);
  });

  it("scores consecutive chars higher than scattered", () => {
    const consecutive = fuzzyScore("parent issue", "par");
    const scattered = fuzzyScore("play around right", "par");
    expect(consecutive).toBeGreaterThan(scattered);
  });

  it("scores word-boundary matches higher", () => {
    const boundary = fuzzyScore("hello world", "hw");
    const mid = fuzzyScore("showtime", "hw");
    expect(boundary).toBeGreaterThan(mid);
  });

  it("returns 0 for non-matches", () => {
    expect(fuzzyScore("hello", "xyz")).toBe(0);
  });

  it("returns positive for empty query", () => {
    expect(fuzzyScore("anything", "")).toBeGreaterThan(0);
  });
});
