import { describe, expect, it } from "vitest";
import type { ActiveShortcut } from "../../keyboard";
import { displayKeys, groupShortcuts, keyJoin } from "./keyboard-help-utils";

function shortcut(overrides: Partial<ActiveShortcut> = {}): ActiveShortcut {
  return {
    id: "view.next",
    key: "j",
    label: "Next item",
    group: "View",
    scope: "view",
    priority: "view",
    ...overrides,
  };
}

describe("keyboard help utilities", () => {
  it("groups live registrations and puts Global first", () => {
    const groups = groupShortcuts([
      shortcut(),
      shortcut({ id: "global.help", key: "?", label: "Help", group: "Global", scope: "global", priority: "global" }),
    ]);

    expect(groups.map((group) => group.title)).toEqual(["Global", "View"]);
    expect(groups[1].shortcuts[0].id).toBe("view.next");
    expect(groups[1].shortcuts[0].bindings).toHaveLength(1);
  });

  it("combines alternate bindings with the same label into one row", () => {
    const groups = groupShortcuts([
      shortcut({ id: "view.next.j", key: "j" }),
      shortcut({ id: "view.next.arrow", key: "ArrowDown" }),
    ]);

    expect(groups[0].shortcuts).toHaveLength(1);
    expect(groups[0].shortcuts[0].bindings.map((binding) => binding.key)).toEqual(["j", "ArrowDown"]);
  });

  it("formats sequences, modifiers, and named keys", () => {
    const sequence = shortcut({ key: ["g", "b"] });
    const command = shortcut({ key: "k", modifiers: { meta: true } });
    const arrow = shortcut({ key: "ArrowDown" });

    expect(displayKeys(sequence, true)).toEqual(["G", "B"]);
    expect(keyJoin(sequence)).toBe("seq");
    expect(displayKeys(command, true)).toEqual(["⌘", "K"]);
    expect(keyJoin(command)).toBe("combo");
    expect(displayKeys(arrow, false)).toEqual(["↓"]);
  });

  it("falls back to the registration scope when no group is provided and no view exists", () => {
    expect(groupShortcuts([shortcut({ group: undefined, scope: "tabs", priority: "global" })])[0].title).toBe("tabs");
  });

  it("folds control-priority shortcuts into the active view group", () => {
    const groups = groupShortcuts([
      shortcut({ id: "backlog.next", key: "j", label: "Next issue", group: "Backlog", scope: "backlog", priority: "view" }),
      shortcut({ id: "tabs.prev", key: "[", label: "Previous tab", group: "Tabs", scope: "tabs", priority: "control" }),
      shortcut({ id: "tabs.next", key: "]", label: "Next tab", group: "Tabs", scope: "tabs", priority: "control" }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].title).toBe("Backlog");
    expect(groups[0].shortcuts).toHaveLength(3);
  });

  it("folds control-priority shortcuts without a group into the view group", () => {
    const groups = groupShortcuts([
      shortcut({ id: "board.next", key: "j", label: "Next card", group: "Board", scope: "board", priority: "view" }),
      shortcut({ id: "tabs.prev", key: "[", label: "Previous tab", group: undefined, scope: "tabs", priority: "control" }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].title).toBe("Board");
    expect(groups[0].shortcuts).toHaveLength(2);
  });

  it("keeps control-priority shortcuts separate when no view is active", () => {
    const groups = groupShortcuts([
      shortcut({ id: "global.help", key: "?", label: "Help", group: "Global", scope: "global", priority: "global" }),
      shortcut({ id: "tabs.prev", key: "[", label: "Previous tab", group: "Tabs", scope: "tabs", priority: "control" }),
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0].title).toBe("Global");
    expect(groups[1].title).toBe("Tabs");
  });
});
