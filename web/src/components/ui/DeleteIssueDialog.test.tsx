import { Children, isValidElement, type ReactElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderMarkup } from "../../render-markup";
import ConfirmDialog from "./ConfirmDialog";
import Button from "./Button";
import DeleteIssueDialog from "./DeleteIssueDialog";

type DialogProps = Parameters<typeof ConfirmDialog>[0];
type ButtonProps = { variant?: string; onClick: () => void; children: ReactNode };

const issue = { id: "xpo-a1b2c3", title: "Fix login redirect loop", status: "DOING" };

/** The ConfirmDialog the component returns; portals can't render to static markup. */
function dialog(subIssueCount: number) {
  const onDelete = vi.fn();
  const onCancel = vi.fn();
  const el = DeleteIssueDialog({ issue, subIssueCount, onDelete, onCancel }) as ReactElement<DialogProps>;
  expect(el.type).toBe(ConfirmDialog);
  const buttons = Children.toArray(el.props.children).filter(isValidElement) as ReactElement<ButtonProps>[];
  for (const b of buttons) expect(b.type).toBe(Button);
  const button = (label: string) => {
    const b = buttons.find(b => b.props.children === label);
    expect(b, label).toBeDefined();
    return b!;
  };
  return { props: el.props, buttons, button, onDelete, onCancel };
}

describe("DeleteIssueDialog", () => {
  it("names the issue in a card: status, ID and title", () => {
    const body = renderMarkup(<>{dialog(0).props.body}</>);
    expect(body).toContain("xpo-a1b2c3");
    expect(body).toContain("Fix login redirect loop");
    expect(body).toContain("<svg");
    expect(body).toContain("bg-[var(--color-surface-2)]");
  });

  it("asks to delete an issue without sub-issues: Cancel, then Delete", () => {
    const d = dialog(0);
    expect(d.props.title).toBe("Delete issue?");
    expect(d.props.description).toBe("This can't be undone.");
    expect(d.buttons.map(b => b.props.children)).toEqual(["Cancel", "Delete"]);
    expect(d.button("Delete").props.variant).toBe("danger");
    d.button("Delete").props.onClick();
    expect(d.onDelete).toHaveBeenCalledWith(false);
  });

  it("offers to keep or delete sub-issues: Cancel, Keep sub-issues, Delete all", () => {
    const d = dialog(20);
    expect(d.props.title).toBe("Delete issue?");
    expect(d.props.description).toBe("It has 20 sub-issues. Keep them as top-level issues, or delete them too?");
    expect(d.buttons.map(b => b.props.children)).toEqual(["Cancel", "Keep sub-issues", "Delete all"]);
    expect(d.button("Delete all").props.variant).toBe("danger");
    // Keep sub-issues still deletes the parent: soft red, not neutral.
    expect(d.button("Keep sub-issues").props.variant).toBe("secondary");
    expect(d.button("Keep sub-issues").props).toHaveProperty("tone", "danger");
    d.button("Keep sub-issues").props.onClick();
    expect(d.onDelete).toHaveBeenLastCalledWith(false);
    d.button("Delete all").props.onClick();
    expect(d.onDelete).toHaveBeenLastCalledWith(true);
  });

  it("counts a single sub-issue in the singular", () => {
    expect(dialog(1).props.description).toBe("It has 1 sub-issue. Keep it as a top-level issue, or delete it too?");
  });

  it("cancels without deleting from the Cancel button, Escape and the backdrop", () => {
    for (const count of [0, 3]) {
      const d = dialog(count);
      // ConfirmDialog wires Escape and the backdrop click to onCancel.
      expect(d.props.onCancel).toBe(d.onCancel);
      expect(d.button("Cancel").props.variant).toBe("secondary");
      expect(d.button("Cancel").props).not.toHaveProperty("bordered");
      d.button("Cancel").props.onClick();
      expect(d.onCancel).toHaveBeenCalledOnce();
      expect(d.onDelete).not.toHaveBeenCalled();
    }
  });
});
