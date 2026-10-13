import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { KeyboardNavProvider } from "./keyboard";
import ToastProvider from "./components/ui/Toast";

/**
 * Renders a component to static HTML inside the app's providers, so tests can
 * assert on its markup (roles, ARIA state) without a DOM environment. Effects
 * don't run and portals can't render, so only the initial, closed state is
 * observable. Pass a `client` seeded with query data to render loaded states.
 */
export function renderMarkup(el: ReactElement, client = new QueryClient()): string {
  return renderToStaticMarkup(
    <QueryClientProvider client={client}>
      <ToastProvider>
        <KeyboardNavProvider>{el}</KeyboardNavProvider>
      </ToastProvider>
    </QueryClientProvider>,
  );
}

/** Opening tags of every element in `html` that carries the given role. */
export function tagsWithRole(html: string, role: string): string[] {
  return html.match(new RegExp(`<[^>]*role="${role}"[^>]*>`, "g")) ?? [];
}
