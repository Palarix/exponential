import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import QuestionCard from "./QuestionCard";

const input = JSON.stringify({
  questions: [
    { question: "Which color?", header: "Color", options: [{ label: "Red", description: "Warm" }, { label: "Blue", description: "Cool" }], multiSelect: false },
    { question: "Which pets?", header: "Pets", options: [{ label: "Cat", description: "" }, { label: "Dog", description: "" }], multiSelect: true },
  ],
});

function card(json: string): string {
  return renderMarkup(<QuestionCard chatId="c1" step={{ kind: "tool", id: "t1", name: "AskUserQuestion", arg: "Which color?", done: false, approval: { id: "a1", input: json } }} />);
}

describe("QuestionCard", () => {
  const html = card(input);

  it("shows each question with its header, options and descriptions", () => {
    const order = ["Color", "Which color?", "Red", "Warm", "Blue", "Cool", "Pets", "Which pets?", "Cat", "Dog"].map((t) => html.indexOf(t));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((x, y) => x - y)).toEqual(order);
  });

  it("picks one option of a single-select question, several of a multi-select one", () => {
    expect(html).toContain('role="radiogroup"');
    expect(html.match(/role="radio"/g)).toHaveLength(3); // Red, Blue, Other
    expect(html.match(/role="checkbox"/g)).toHaveLength(3); // Cat, Dog, Other
  });

  it("offers Other for free text, and hides the field until it is picked", () => {
    expect(html.match(/>Other…</g)).toHaveLength(2);
    expect(html).not.toContain("<input");
  });

  it("shows the field at once for a question without options", () => {
    const open = card('{"questions":[{"question":"Why?","header":"Reason"}]}');
    expect(open).toContain("<input");
    expect(open).not.toContain(">Other…<");
  });

  it("offers Skip and Send, Send last and disabled until every question has an answer", () => {
    const skip = html.indexOf(">Skip<");
    const send = html.indexOf(">Send<");
    expect(skip).toBeGreaterThan(-1);
    expect(send).toBeGreaterThan(skip);
    expect(html).toMatch(/<button[^>]*disabled=""[^>]*>Send</);
    expect(html).not.toContain("Allow");
  });
});
