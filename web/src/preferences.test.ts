import { describe, expect, it } from "vitest";
import { DEFAULT_PREFERENCES, parsePreferences, resolveMotion } from "./preferences";

describe("parsePreferences", () => {
  it("defaults when nothing is stored", () => {
    expect(parsePreferences(null)).toEqual(DEFAULT_PREFERENCES);
    expect(DEFAULT_PREFERENCES.motion).toBe("system");
    expect(DEFAULT_PREFERENCES.diffView).toBe("unified");
  });

  it("reads a stored diff view", () => {
    expect(parsePreferences('{"diffView":"split"}').diffView).toBe("split");
    expect(parsePreferences('{"diffView":"side"}').diffView).toBe("unified");
  });

  it("keeps each valid field when another is invalid", () => {
    expect(parsePreferences('{"motion":"off","diffView":9}')).toEqual({ motion: "off", diffView: "unified" });
  });

  it("reads a stored motion setting", () => {
    expect(parsePreferences('{"motion":"on"}').motion).toBe("on");
    expect(parsePreferences('{"motion":"off"}').motion).toBe("off");
    expect(parsePreferences('{"motion":"system"}').motion).toBe("system");
  });

  it("falls back per field on unknown values", () => {
    expect(parsePreferences('{"motion":"sometimes"}').motion).toBe("system");
    expect(parsePreferences('{"motion":1}').motion).toBe("system");
    expect(parsePreferences("{}").motion).toBe("system");
  });

  it("falls back on bad JSON and non-objects", () => {
    expect(parsePreferences("{nope")).toEqual(DEFAULT_PREFERENCES);
    expect(parsePreferences("[]")).toEqual(DEFAULT_PREFERENCES);
    expect(parsePreferences('"on"')).toEqual(DEFAULT_PREFERENCES);
    expect(parsePreferences("null")).toEqual(DEFAULT_PREFERENCES);
  });
});

describe("resolveMotion", () => {
  it("plays everything when On, whatever the OS says", () => {
    expect(resolveMotion("on", false)).toBe("full");
    expect(resolveMotion("on", true)).toBe("full");
  });

  it("turns animations off when Off, whatever the OS says", () => {
    expect(resolveMotion("off", false)).toBe("off");
    expect(resolveMotion("off", true)).toBe("off");
  });

  it("follows the OS under System default", () => {
    expect(resolveMotion("system", false)).toBe("full");
    expect(resolveMotion("system", true)).toBe("reduced");
  });
});
