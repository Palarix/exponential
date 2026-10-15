import { describe, expect, it } from "vitest";
import { renderMarkup, tagsWithRole } from "../../render-markup";
import { emptyPicks, parseQuestions, pickOption, stepInClass } from "./chat-thread-utils";
import QuestionCard, { Choice, QuestionFooter, QuestionReview, QuestionTabs } from "./QuestionCard";

const input = JSON.stringify({
  questions: [
    { question: "Which color?", header: "Color", options: [{ label: "Red", description: "Warm" }, { label: "Blue", description: "Cool" }], multiSelect: false },
    { question: "Which pets?", header: "Pets", options: [{ label: "Cat", description: "" }, { label: "Dog", description: "" }], multiSelect: true },
  ],
});

const one = JSON.stringify({ questions: [{ question: "Which color?", header: "Color", options: [{ label: "Red", description: "Warm" }, { label: "Blue", description: "Cool" }] }] });

function card(json: string): string {
  return renderMarkup(<QuestionCard chatId="c1" step={{ kind: "tool", id: "t1", name: "AskUserQuestion", arg: "Which color?", done: false, approval: { id: "a1", input: json } }} />);
}

const noop = () => {};

describe("QuestionCard with one question", () => {
  const html = card(one);

  it("shows the question with its header at the end of the line, then the options and descriptions", () => {
    const order = ["Which color?", ">Color<", "Red", "Warm", "Blue", "Cool"].map((t) => html.indexOf(t));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((x, y) => x - y)).toEqual(order);
  });

  it("has no tabs and no Previous", () => {
    expect(html).not.toContain('role="tablist"');
    expect(html).not.toContain(">Previous<");
  });

  it("offers Other for free text, and hides the field until it is picked", () => {
    expect(html.match(/role="radio"/g)).toHaveLength(3); // Red, Blue, Other
    expect(html).toContain(">Other…<");
    expect(html).not.toContain("<input");
  });

  it("shows each option's key as a number hint, and none on Other", () => {
    const hints = html.match(/<kbd[^>]*>[^<]*<\/kbd>/g) ?? [];
    expect(hints.map((k) => k.replace(/<[^>]*>/g, ""))).toEqual(["1", "2"]);
    expect(html.indexOf("<kbd")).toBeGreaterThan(html.indexOf(">Red<"));
  });

  it("shows the field at once for a question without options", () => {
    const open = card('{"questions":[{"question":"Why?","header":"Reason"}]}');
    expect(open).toContain("<input");
    expect(open).not.toContain(">Other…<");
  });

  it("offers Chat about this and Submit answers, disabled until the question has an answer, and no review step", () => {
    const chat = html.indexOf(">Chat about this<");
    const submit = html.indexOf(">Submit answers<");
    expect(chat).toBeGreaterThan(-1);
    expect(submit).toBeGreaterThan(chat);
    expect(html).not.toContain("Skip");
    expect(html).not.toContain(">Next<");
    expect(html).not.toContain("Review");
    expect(html).toMatch(/<button[^>]*disabled=""[^>]*>Submit answers</);
    expect(html).not.toContain("Allow");
  });
});

describe("QuestionCard with several questions", () => {
  const html = card(input);

  it("shows a tab per question, a last Review tab, and only the first question", () => {
    const tabs = tagsWithRole(html, "tab");
    expect(tabs).toHaveLength(3);
    expect(tabs[0]).toContain('aria-selected="true"');
    expect(tabs[1]).toContain('aria-selected="false"');
    expect(html.split('role="tab"')[3]).toContain(">Review<");
    expect(html).toContain('role="tabpanel"');
    expect(html).toContain("Which color?");
    expect(html).not.toContain("Which pets?");
    expect(html).not.toContain(">Cat<");
  });

  it("leaves the header to the tab, with no chip on the question line", () => {
    const panel = html.slice(html.indexOf('role="tabpanel"'));
    expect(panel).toContain("Which color?");
    expect(panel).not.toContain(">Color<");
  });

  it("opens with Chat about this and Next, and no Previous", () => {
    expect(html).not.toContain(">Previous<");
    expect(html).toMatch(/>Chat about this<[\s\S]*>Next</);
    expect(html).not.toMatch(/<button[^>]*disabled=""[^>]*>Next</);
    expect(html).not.toContain(">Submit answers<");
  });

  it("can take focus for its keys", () => {
    expect(html).toMatch(/<div[^>]*role="group"[^>]*tabindex="-1"/);
  });

  it("shows the approval card's focus border while focus is in it", () => {
    expect(html).toContain("focus-within:border-[var(--color-border-focus)]");
  });

  it("does not slide the first step in when the card opens", () => {
    expect(html).not.toContain("animate-step-in");
  });
});

describe("stepInClass", () => {
  it("slides a step in from its direction, and not at all without one", () => {
    expect(stepInClass("forward")).toBe("motion-safe:animate-step-in-forward");
    expect(stepInClass("back")).toBe("motion-safe:animate-step-in-back");
    expect(stepInClass(null)).toBe("");
  });
});

describe("Choice", () => {
  it("holds its children inside the row, after the button", () => {
    const html = renderMarkup(
      <Choice multi={false} checked label="Other…" disabled={false} onClick={noop}>
        <input aria-label="answer" />
      </Choice>,
    );
    expect(html).toMatch(/^<div[^>]*><button[^>]*role="radio"[\s\S]*<\/button>[\s\S]*<input aria-label="answer"\/>[\s\S]*<\/div>$/);
  });

  it("pops the mark in when the option is checked", () => {
    const radio = renderMarkup(<Choice multi={false} checked label="Red" disabled={false} onClick={noop} />);
    const box = renderMarkup(<Choice multi checked label="Cat" disabled={false} onClick={noop} />);
    const open = renderMarkup(<Choice multi={false} checked={false} label="Blue" disabled={false} onClick={noop} />);
    expect(radio).toContain("motion-safe:animate-pop-in");
    expect(box).toContain("motion-safe:animate-pop-in");
    expect(open).not.toContain("animate-pop-in");
  });

  it("centers the mark, the label and the hint", () => {
    const html = renderMarkup(<Choice multi={false} checked={false} label="Red" description="Warm" hint={1} disabled={false} onClick={noop} />);
    expect(html).toMatch(/<button[^>]*class="[^"]*items-center/);
  });
});

describe("QuestionTabs", () => {
  const qs = parseQuestions(input);

  it("marks answered, current and open questions", () => {
    const picks = pickOption(emptyPicks(qs), qs, 0, "Red");
    const answeredFirst = renderMarkup(<QuestionTabs questions={qs} picks={picks} current={1} disabled={false} onSelect={noop} />);
    expect(answeredFirst).toMatch(/aria-label="Color, answered"/);
    expect(answeredFirst).toMatch(/aria-label="Pets, current"/);
    const open = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={0} disabled={false} onSelect={noop} />);
    expect(open).toMatch(/aria-label="Pets, not answered"/);
  });

  it("shows a check only on answered tabs, and no other mark", () => {
    const picks = pickOption(emptyPicks(qs), qs, 0, "Red");
    const html = renderMarkup(<QuestionTabs questions={qs} picks={picks} current={1} disabled={false} onSelect={noop} />);
    const [, color, pets, review] = html.split('role="tab"');
    expect(color).toContain("<svg");
    expect(color).toContain("motion-safe:animate-pop-in");
    expect(pets).not.toContain("<svg");
    expect(pets).not.toMatch(/aria-hidden/);
    expect(review).toContain(">Review<");
    expect(review).not.toContain("<svg");
  });

  it("pushes the Review tab to the far right, apart from the question tabs", () => {
    const html = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={0} disabled={false} onSelect={noop} />);
    expect(tagsWithRole(html, "tab").map((t) => /\bml-auto\b/.test(t))).toEqual([false, false, true]);
  });

  it("pads each tab 12px above and below, so the whole band is clickable", () => {
    const html = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={0} disabled={false} onSelect={noop} />);
    expect(tagsWithRole(html, "tab").every((t) => /\bpy-3\b/.test(t))).toBe(true);
  });

  it("selects the Review tab on the review step", () => {
    const html = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={2} disabled={false} onSelect={noop} />);
    const tabs = tagsWithRole(html, "tab");
    expect(tabs.map((t) => t.includes('aria-selected="true"'))).toEqual([false, false, true]);
  });

  it("shows the full header as a title, for long headers", () => {
    const html = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={0} disabled={false} onSelect={noop} />);
    expect(html).toContain('title="Color"');
  });

  it("disables the tabs while the card sends", () => {
    const html = renderMarkup(<QuestionTabs questions={qs} picks={emptyPicks(qs)} current={0} disabled onSelect={noop} />);
    expect(tagsWithRole(html, "tab").every((t) => t.includes('disabled=""'))).toBe(true);
  });
});

/** Opening tags of the review rows, whatever their attribute order. */
function reviewRows(html: string): string[] {
  return html.match(/<div[^>]*class="group [^"]*"[^>]*>/g) ?? [];
}

describe("QuestionReview", () => {
  const qs = parseQuestions(input);
  const picks = pickOption(emptyPicks(qs), qs, 0, "Red");

  it("lists each full question over its answer, or Not answered, and no heading", () => {
    const html = renderMarkup(<QuestionReview questions={qs} picks={picks} disabled={false} onSelect={noop} />);
    const order = [">Which color?<", ">Red<", ">Which pets?<", ">Not answered<"].map((t) => html.indexOf(t));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((x, y) => x - y)).toEqual(order);
    expect(html).not.toContain("Review your answers");
    expect(html).not.toContain(">Color<");
  });

  it("marks an answer with the answered check, an open question with a question mark, and reads as a result, not an option", () => {
    const html = renderMarkup(<QuestionReview questions={qs} picks={picks} disabled={false} onSelect={noop} />);
    const [, color, pets] = html.split('class="group ');
    expect(color).toContain("lucide-check");
    expect(color).not.toContain("question-mark");
    expect(pets).toContain("lucide-circle-question-mark");
    expect(pets).not.toContain("lucide-check");
    expect(html).not.toMatch(/role="(radio|checkbox)"/);
  });

  it("makes only Change clickable, one per question, named for its question", () => {
    const html = renderMarkup(<QuestionReview questions={qs} picks={picks} disabled={false} onSelect={noop} />);
    const buttons = html.match(/<button[^>]*>[\s\S]*?<\/button>/g) ?? [];
    expect(buttons).toHaveLength(2);
    expect(buttons.every((b) => b.includes(">Change<"))).toBe(true);
    expect(buttons[0]).toContain('aria-label="Change answer to: Which color?"');
    expect(buttons.some((b) => b.includes("Which pets?") && b.includes(">Change<"))).toBe(true);
  });

  it("keeps the rows' padding inside the body, so they line up with the option rows and footer buttons", () => {
    const html = renderMarkup(<QuestionReview questions={qs} picks={picks} disabled={false} onSelect={noop} />);
    expect(html).not.toMatch(/[\s"]-mx-/);
    const rows = reviewRows(html);
    expect(rows).toHaveLength(2);
    expect(rows.every((r) => /\bpx-2\.5\b/.test(r))).toBe(true);
  });

  it("disables Change while the card sends", () => {
    const html = renderMarkup(<QuestionReview questions={qs} picks={picks} disabled onSelect={noop} />);
    expect(html.match(/<button[^>]*disabled=""/g)).toHaveLength(2);
  });
});

describe("QuestionFooter", () => {
  const footer = (props: Partial<Parameters<typeof QuestionFooter>[0]>) =>
    renderMarkup(<QuestionFooter primary="Next" primaryDisabled={false} sending={null} error={null} onChat={noop} onPrimary={noop} {...props} />);

  it("puts Chat about this first, then the error and the primary button, and no Previous", () => {
    const html = footer({ error: "Boom" });
    const order = [">Chat about this<", "Boom", ">Next<"].map((t) => html.indexOf(t));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((x, y) => x - y)).toEqual(order);
    expect(html).not.toContain("Previous");
  });

  it("pads like the body, so the buttons line up with the option rows, with 12px above and below", () => {
    expect(footer({})).toMatch(/^<div[^>]*class="[^"]*\bpx-3 py-3\b/);
  });

  it("shows Chat about this as a secondary button, with a border", () => {
    expect(footer({})).toMatch(/<button[^>]*class="[^"]*\bborder\b[^"]*"[^>]*>Chat about this</);
  });

  it("reads Submit answers on the review step, disabled until every question has an answer", () => {
    expect(footer({ primary: "Submit answers", primaryDisabled: true })).toMatch(/<button[^>]*disabled=""[^>]*>Submit answers</);
    expect(footer({ primary: "Submit answers" })).not.toMatch(/<button[^>]*disabled=""[^>]*>Submit answers</);
  });

  it("disables both buttons while the card sends", () => {
    const html = footer({ sending: "answer" });
    expect(html.match(/<button[^>]*disabled=""/g)).toHaveLength(2);
  });
});
