import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import type { ComponentProps } from "react";
import Layout from "./Layout";
import type Sidebar from "./Sidebar";

vi.mock("./Sidebar", () => ({ default: () => null }));
vi.mock("../Agent/AgentStatusBar", () => ({ default: () => null }));

const sidebar = {} as ComponentProps<typeof Sidebar>;

function render(version: string, connected: boolean): string {
  return renderToStaticMarkup(
    <Layout sidebar={sidebar} version={version} connected={connected}>
      content
    </Layout>,
  );
}

describe("Layout footer", () => {
  it("shows only the version while connected", () => {
    const html = render("1.5.0-de0d346!", true);
    expect(html).toContain(">v1.5.0-de0d346!<");
    expect(html).not.toContain("Exponential");
  });

  it("shows Disconnected instead of the version when not connected", () => {
    const html = render("1.5.0", false);
    expect(html).toContain(">Disconnected<");
    expect(html).not.toContain("v1.5.0");
  });
});
